package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

var (
	Mkcaasproot = ""
	config, err = CaaSP4CFG(Mkcaasproot)
	skubaroot   = config.Skubaroot
	clustername = "my-cluster"
	//workdir      = filepath.Join(skubaroot, "test/lib/prototyping")
	vmwaretfdir  = filepath.Join(skubaroot, "ci/infra/vmware")
	myclusterdir = filepath.Join(vmwaretfdir, clustername)
	testdir      = filepath.Join(skubaroot, "test/core-features")
)

func CaaSP4CFG(mkcaasproot string) (*MKCaaSPCfg, error) {
	var a *MKCaaSPCfg
	file, err := os.Open(filepath.Join(mkcaasproot, "mkcaaspcfg.json"))
	defer file.Close()
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(file).Decode(&a); err != nil {
		return nil, err
	}
	return a, err
}

func TFParser() *TFOutput {
	var a *TFOutput
	cmd := exec.Command("terraform", "output", "-json")
	out, errstr := NiceBuffRunner(cmd, vmwaretfdir)
	if errstr != "%!s(<nil>)" && errstr != "" {
		log.Printf("Error while running \"terraform output -json\":  %s", errstr)
	}
	err := json.Unmarshal([]byte(out), &a)
	if err != nil {
		log.Printf("Error while unmarshalling: %s", err)
	}
	return a
}

func OSExporter(a *TFOutput) {
	ENV := os.Environ()
	for _, elem := range a.IP_Load_Balancer.Value {
		ENV = append(ENV, fmt.Sprintf("CONTROLPLANE=%s", elem))
	}
	for index, elem := range a.IP_Masters.Value {
		ENV = append(ENV, fmt.Sprintf("MASTER0%v=%s", index, elem))
	}
	for index, elem := range a.IP_Workers.Value {
		ENV = append(ENV, fmt.Sprintf("WORKER0%v=%s", index, elem))
	}
}

func CheckIPSSH(node Node) Node {
	count := 0
	//----------Checking if Node has network connection
	command := []string{"ping", "-c", "3", node.IP}
	out, err := exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error while running ping %s: %s", node.IP, err)
	}
	temp := strings.Split(fmt.Sprintf("%s", string(out)), "\n")
	for _, k := range temp {
		if strings.Contains(k, "ttl") {
			count += 1
		}
	}
	if count >= 3 {
		node.Network = true
	}
	//----------Checking if Node has port 22 opened
	command = []string{"nc", "-zvw3", node.IP, "22"}
	out, err = exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error while running nc -zvw3 %s 22: %s", node.IP, err)
	}
	if strings.Contains(fmt.Sprintf("%s", string(out)), "succeeded") {
		node.Port22 = true
	}
	//---------Checking if Node ssh service is fine
	command = []string{"echo", "KEYWORD"}
	cmd := node.SSHCmd("", command)
	out, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error while SSH-ing into the node %s:  %s", node.IP, err)
	}
	if strings.Contains(fmt.Sprintf("%s", string(out)), "KEYWORD") {
		node.SSH = true
	}
	//
	return node
}

func CheckNode(node Node) Node {
	b := make(map[string]Node)
	command := []string{"hostname"}
	out, err := node.SSHCmd("", command).CombinedOutput()
	if err != nil {
		log.Printf("error while running SSH command: %s", err)
	}
	temp := strings.Split(fmt.Sprintf("%s", string(out)), "\n")
	//------------Checking in the output of hostname
	for _, k := range temp {
		if k != " " && k != "" && !strings.Contains(k, "known hosts") {
			node.NodeName = k
		}
	}
	if node.Role != "Load_Balancer" {
		checkpath := "/var/log/pods"
		//---checking in the logs for "Local node-name: xxxxxxxxxxxx" to find K8s Name of the node
		command = []string{"sudo", "grep", "-R", "node-name", checkpath}
		out, err = node.SSHCmd("", command).CombinedOutput()
		if err != nil {
			log.Printf("error while running SSH sudo grep command: %s", err)
		}
		temp = strings.Split(fmt.Sprintf("%s", string(out)), " ")
		for index, _ := range temp {
			//fmt.Println(temp[index])
			if strings.Contains(temp[index], "pimp") {
				//fmt.Printf("Here is the k8s name: %s \n", strings.Replace(temp[index], "\"", "", 10))
				node.K8sName = strings.Replace(temp[index], "\"", "", 10)
				b[node.IP] = node
				break
			}
		}
	}
	return node
}

func ClusterCheckBuilder(a *TFOutput) {
	var node Node
	node.Username = "sles" //to be improved if different user for different roles...
	for _, k := range a.IP_Load_Balancer.Value {
		node.IP = k
		node := CheckIPSSH(node)
		node.Role = "Load_Balancer"
		node = CheckNode(node)
		fmt.Println(node)
	}
	for _, k := range a.IP_Masters.Value {
		node.IP = k
		node := CheckIPSSH(node)
		node.Role = "Master"
		node = CheckNode(node)
		fmt.Println(node)
	}
	for _, k := range a.IP_Workers.Value {
		node.IP = k
		node := CheckIPSSH(node)
		node.Role = "Worker"
		node = CheckNode(node)
		fmt.Println(node)
	}
}

func (node *Node) SSHCmd(workdir string, command []string) *exec.Cmd {
	args := append(
		[]string{"-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile /dev/null", "-i", filepath.Join(skubaroot, "ci/infra/id_shared"),
			fmt.Sprintf("%s@%s", node.Username, node.IP),
		},
		command...,
	)
	return exec.Command("ssh", args...)
}

func NiceBuffRunner(cmd *exec.Cmd, workdir string) (string, string) {
	var stdoutBuf, stderrBuf bytes.Buffer
	//newEnv := append(os.Environ(), ENV...)
	//cmd.Env = newEnv
	cmd.Dir = workdir
	pipe, _ := cmd.StdoutPipe()
	errpipe, _ := cmd.StderrPipe()
	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)
	err := cmd.Start()
	if err != nil {
		return fmt.Sprintf("%s", os.Stdout), fmt.Sprintf("%s", err)
	}
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		_, errStdout = io.Copy(stdout, pipe)
		wg.Done()
	}()
	go func() {
		_, errStderr = io.Copy(stderr, errpipe)
		wg.Wait()
	}()
	err = cmd.Wait()
	if err != nil {
		return fmt.Sprintf("%s", os.Stdout), fmt.Sprintf("%s", err)
	}
	if errStdout != nil || errStderr != nil {
		log.Fatal("Command runninng error: failed to capture stdout or stderr\n")
	}
	return stdoutBuf.String(), stderrBuf.String()
}
