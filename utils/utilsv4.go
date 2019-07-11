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
	"sync"
)

var (
	skubaroot    = "/home/atighineanu/golang/src/github.com/skuba"
	clustername  = "my-cluster"
	workdir      = filepath.Join(skubaroot, "test/lib/prototyping")
	vmwaretfdir  = filepath.Join(skubaroot, "ci/infra/vmware")
	myclusterdir = filepath.Join(vmwaretfdir, clustername)
	testdir      = filepath.Join(skubaroot, "test/core-features")
)

type TFOutput struct {
	IP_Load_Balancer *TFTag `json: ip_load_balancer`
	IP_Masters       *TFTag `json: ip_masters`
	IP_Workers       *TFTag `json: ip_workers`
}

type TFTag struct {
	Sensitive bool     `json: sensitive`
	Type      string   `json: type`
	Value     []string `json: value`
}

type ClusterCheck map[string]Node

type Node struct {
	IP         string
	NodeName   string
	Username   string
	Network    bool
	SSH        bool
	ContHealth bool
	PackHealth bool
	RepoHealth bool
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

func CheckIPSSH(node *Node) {
	command := []string{"ping", "-c", "3", node.IP}
	out, err := exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Error while running ping %s: %s", node.IP, err)
	}
	fmt.Println(fmt.Sprintf("%s", string(out)))
}

func ClusterCheckBuilder(a *TFOutput) {
	var IPSlice []string
	for _, k := range a.IP_Load_Balancer.Value {
		IPSlice = append(IPSlice, k)
	}
	for _, k := range a.IP_Masters.Value {
		IPSlice = append(IPSlice, k)
	}
	for _, k := range a.IP_Workers.Value {
		IPSlice = append(IPSlice, k)
	}

	for _, k := range IPSlice {
		var node Node
		node.IP = k
		node.Username = "sles"
		//command := []string{"ls -alh /var/log/pods"}
		command := []string{"hostname"}
		out, err := node.SSHCmd("", command).CombinedOutput()
		if err != nil {
			log.Printf("error while running SSH command: %s", err)
		}
		fmt.Println(fmt.Sprintf("%s", string(out)))
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
