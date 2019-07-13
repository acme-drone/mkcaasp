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
	"strconv"
	"sync"
	"time"
)

var (
	Mkcaasproot = ""
	Config, err = CaaSP4CFG(Mkcaasproot)
	skubaroot   = Config.Skubaroot
	clustername = "my-cluster"
	//workdir      = filepath.Join(skubaroot, "test/lib/prototyping")
	vmwaretfdir  = filepath.Join(skubaroot, "ci/infra/vmware")
	myclusterdir = filepath.Join(vmwaretfdir, clustername)
	testdir      = filepath.Join(skubaroot, "test/core-features")
	ENV2         = os.Environ()
)

func VMWareexporter() {
	a := []string{
		"GOVC_URL=" + Config.Vmware.GOVC_URL,
		"GOVC_USERNAME=" + Config.Vmware.GOVC_USERNAME,
		"GOVC_PASSWORD=" + Config.Vmware.GOVC_PASSWORD,
		"GOVC_INSECURE=" + string(Config.Vmware.GOVC_INSECURE),
		//-------------
		"VSPHERE_SERVER=" + Config.Vmware.VSPHERE_SERVER,
		"VSPHERE_USER=" + Config.Vmware.VSPHERE_USER,
		"VSPHERE_PASSWORD=" + Config.Vmware.VSPHERE_PASSWORD,
		"VSPHERE_ALLOW_UNVERIFIED_SSL=" + strconv.FormatBool(Config.Vmware.VSPHERE_ALLOW_UNVERIFIED_SSL),
	}
	for _, k := range a {
		exec.Command("export", k).Run()
	}
	ENV2 = append(ENV2, a...)
}

func CaaSP4CFG(mkcaasproot string) (*MKCaaSPCfg, error) {
	var a *MKCaaSPCfg
	file, err := os.Open(filepath.Join(mkcaasproot, "mkcaaspcfg.json"))
	defer file.Close()
	if err != nil {
		fmt.Printf("Coudn't open the file! %s\n", err)
		return nil, err
	}
	if err := json.NewDecoder(file).Decode(&a); err != nil {
		fmt.Printf("Coudn't decode! %s", err)
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

func NodeOSExporter(a *TFOutput) []string {
	for _, elem := range a.IP_Load_Balancer.Value {
		ENV2 = append(ENV2, fmt.Sprintf("CONTROLPLANE=%s", elem))
	}
	for index, elem := range a.IP_Masters.Value {
		ENV2 = append(ENV2, fmt.Sprintf("MASTER0%v=%s", index, elem))
	}
	for index, elem := range a.IP_Workers.Value {
		ENV2 = append(ENV2, fmt.Sprintf("WORKER0%v=%s", index, elem))
	}
	return ENV2
}

func CreateCaasp4(action string) (string, string) {
	var suffix string
	var cmd *exec.Cmd
	//----------------Deploying With Terraform-------------------
	if action == "apply" {
		suffix = "-auto-approve"
	}
	if action == "destroy" {
		suffix = "-auto-approve"
		_, err := exec.Command("rm", "-R", myclusterdir).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error removing the %s folder...%v", myclusterdir, err )
		}
	}
	if suffix != "" {
		cmd = exec.Command("terraform", action, suffix)
	} else {
		cmd = exec.Command("terraform", action)
	}
	fmt.Println(ENV2)
	cmd.Env = ENV2
	out, errstr := NiceBuffRunner(cmd, vmwaretfdir)
	if errstr != "%!s(<nil>)" && errstr != "" {
		log.Printf("Error while running \"terraform command\":  %s", errstr)
		return out, errstr
	}
	if action == "apply" {
		log.Printf("The Nodes were successfully Deployed by %s on %s Platform", Config.Deploy, Config.Platform)
		time.Sleep(10 * time.Second)
	}
	return out, errstr
}

func JoinWorkers(tf *TFOutput, b map[string]Node) (string, string){
	var out, errstr string
	for index, k := range tf.IP_Workers.Value {
		fmt.Println(tf.IP_Workers.Value)
		time.Sleep(10*time.Second)
		k8sname := fmt.Sprintf("worker-pimp-comrade-0%v", index)
		cmd := exec.Command("skuba", "node", "join", "--role", "worker", "--user", "sles", "--sudo", "--target", k, k8sname)
		node := b[k]
		node.K8sName = k8sname
		b[k] = node
		cmd.Dir = filepath.Join(vmwaretfdir, clustername)
		cmd.Env = ENV2
		out, errstr = NiceBuffRunner(cmd, filepath.Join(vmwaretfdir, clustername))
		if errstr != "%!s(<nil>)" && errstr != "" {
			log.Printf("Error while running \"skuba join worker command\":  %s", errstr)
			return out, errstr
		}
	}
	return out, errstr
}

func DeployCaasp4() (string, string) {
	//---------------Deploying With Skuba-----------------------
	//var out, errstr string
	tf := TFParser()
	NodeOSExporter(tf)
	b := ClusterCheckBuilder(tf, "setup")
	cmd := exec.Command("skuba", "cluster", "init", "--control-plane", tf.IP_Load_Balancer.Value[0], clustername)
	cmd.Env = ENV2
	out, errstr := NiceBuffRunner(cmd, vmwaretfdir)
	if errstr != "%!s(<nil>)" && errstr != "" {
		log.Printf("Error while running \"skuba init command\":  %s", errstr)
		return out, errstr
	}	
	//fmt.Println(b)
	log.Printf("Successfully initiated the cluster load balancer: %s ...\n", tf.IP_Load_Balancer.Value[0])
	time.Sleep(20 * time.Second)
	for index, k := range tf.IP_Masters.Value {
		k8sname := fmt.Sprintf("master-pimp-general-0%v", index)
		cmd := exec.Command("skuba", "node", "bootstrap", "--user", "sles", "--sudo", "--target", k, k8sname)
		node := b[k]
		node.K8sName = k8sname
		b[k] = node
		cmd.Dir = filepath.Join(vmwaretfdir, clustername)
		fmt.Println(cmd.Dir)
		cmd.Env = ENV2
		out, errstr := NiceBuffRunner(cmd, filepath.Join(vmwaretfdir, clustername))
		if errstr != "%!s(<nil>)" && errstr != "" {
			log.Printf("Error while running \"skuba node bootstrap %s\":  %s", k8sname, errstr)
			return out, errstr
		}
		log.Printf("Successfully installed %s ->IP: %s in the cluster...\n", k8sname, k)
	}
	JoinWorkers(tf, b)
 return out, errstr
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
