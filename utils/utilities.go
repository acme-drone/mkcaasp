package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
)

var Cluster CaaSPCluster

// SetOSEnv sets up Openstack environment variables
func SetOSEnv(file string) (EnvOS, error) {
	var auth = OSAPI{}
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(f).Decode(&auth); err != nil {
		return nil, err
	}
	env := EnvOS{
		"OS_AUTH_URL=" + auth.AuthURL,
		"OS_REGION_NAME=" + auth.RegionName,
		"OS_PROJECT_NAME=" + auth.ProjectName,
		"OS_USER_DOMAIN_NAME=" + auth.UserDomainName,
		"OS_IDENTITY_API_VERSION=" + auth.IdentityAPIVersion,
		"OS_INTERFACE=" + auth.Interface,
		"OS_USERNAME=" + auth.Username,
		"OS_PASSWORD=" + Dehashinator("./../", "./"), //Dehashinator("./../", "./"),  auth.Password
		"OS_PROJECT_ID=" + auth.ProjectID,
	}
	return env, nil
}

func (s *CAASPOut) SSHCommand(cmd ...string) *exec.Cmd {
	arg := append(
		[]string{"-o", "StrictHostKeyChecking=no",
			fmt.Sprintf("root@%s", s.IPAdminExt.Value),
		},
		cmd...,
	)
	return exec.Command("ssh", arg...)
}

func AdminOrchCmd(s *CAASPOut, option string) {
	alias := []string{"docker", "exec", "$(docker ps -q --filter name=salt-master)", "salt", "-P", "\"roles:admin|kube-master|kube-minion\""}
	//------------sweeping through options...
	if option == "refresh" {
		cmd := append(alias, "saltutil.refresh_grains")
		out, err := s.SSHCommand(cmd...).CombinedOutput()
		if err != nil {
			log.Printf("ssh command didn't run as expected: %s\n", err)
		}
		fmt.Printf("%s", fmt.Sprintf("%s", string(out)))
	}
	//	withalias := append(
	//		[]string{"docker", "exec", "$(docker ps -q --filter name=salt-master)", "salt", "-P", "\"roles:admin|kube-master|kube-minion\""}, cmd...)
}

func NodesAdder(dir string, append string, nodes *CAASPOut, Firstornot bool) *CaaSPCluster {
	temp := strings.Split(append, "")
	if len(temp) > 4 {
		log.Fatalf("Check your syntaxis...there must be just four symbols in -addnodes argument")
	} else {
		for i := 0; i < len(temp); i++ {
			if temp[i] == "w" {
				if len(temp) >= i+2 {
					//fmt.Printf("there are %s workers.\n", temp[i+1])
					Cluster.WorkCount, _ = strconv.Atoi(temp[i+1])
					fmt.Printf("Adding %v workers.\n", Cluster.WorkCount)
				}
			}
			if temp[i] == "m" {
				if len(temp) >= i+2 {
					Cluster.MastCount, _ = strconv.Atoi(temp[i+1])
					fmt.Printf("Adding %v masters.\n", Cluster.MastCount)
				}
			}
		}
	}

	if Firstornot == true {
		Cluster.Diff = len(nodes.IPMastersExt.Value) + len(nodes.IPWorkersExt.Value)
		if Cluster.Diff == 0 {
			if Cluster.MastCount == 0 {
				Cluster.MastCount += 1
			}
			if Cluster.WorkCount == 0 {
				Cluster.WorkCount += 2
			}
		}
	} else {
		if Cluster.MastCount >= 0 || Cluster.WorkCount >= 0 {
			Cluster.Diff = Cluster.MastCount + Cluster.WorkCount
			Cluster.MastCount += len(nodes.IPMastersExt.Value)
			Cluster.WorkCount += len(nodes.IPWorkersExt.Value)
		}
	}
	templ, err := template.New("AddingNodes").Parse(CulsterTempl)
	if err != nil {
		log.Fatalf("Error parsin ClusterTempl constant...%s", err)
	}
	var f *os.File
	wd, _ := os.Getwd()
	os.Chdir(filepath.Join(wd, dir))
	f, err = os.Create("terraform.tfvars")
	if err != nil {
		log.Fatalf("couldn't create the file...%s", err)
	}
	err = templ.Execute(f, Cluster)
	if err != nil {
		log.Fatalf("couldn't execute the Cluster template %s", err)
	}
	f.Close()
	out, err := exec.Command("cat", "terraform.tfvars").CombinedOutput()
	if err != nil {
		log.Fatalf("Couldn't execute the command %s", err)
	}
	log.Printf("That's the modified cluster config:\n%s\n", fmt.Sprintf("%s", string(out)))
	return &Cluster
}

// RunScript accepts 4 inputs and a runs terraform script
func RunScript(command string, env EnvOS) (string, string) {
	var stdoutBuf, stderrBuf bytes.Buffer
	cmd := exec.Command("bash", "-c", command)
	newEnv := append(os.Environ(), env...)
	cmd.Env = newEnv

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)

	err := cmd.Start()
	if err != nil {
		log.Fatalf("cmd.Start() failed with '%s'\n", err)
	}

	go func() {
		_, errStdout = io.Copy(stdout, stdoutIn)
	}()

	go func() {
		_, errStderr = io.Copy(stderr, stderrIn)
	}()

	err = cmd.Wait()
	if err != nil {
		log.Fatalf("cmd.Run() failed with %s\n", err)
	}
	if errStdout != nil || errStderr != nil {
		log.Fatal("failed to capture stdout or stderr\n")
	}
	return stdoutBuf.String(), stderrBuf.String()
}

// TfInit tinitializes the needed terraform modules
func TfInit(dir string) {
	wd, _ := os.Getwd()
	os.Chdir(filepath.Join(wd, dir))
	cmd := exec.Command("terraform", "init")
	err := cmd.Run()
	if err != nil {
		fmt.Fprintf(os.Stderr, "terraform init failed: %v", err)
	}
}

// CmdRun takes a directory, openstack api auth, command and runs RunScript
func CmdRun(dir, openstackAPIauth, command string) (string, string) {
	wd, _ := os.Getwd()
	os.Chdir(filepath.Join(wd, dir))
	env, err := SetOSEnv(openstackAPIauth)
	if err != nil {
		log.Fatal(err)
	}
	outstd, outstderr := RunScript(command, env)
	return outstd, outstderr
}

func OpenstackCmd(dir string, openstackAPIauth string) (string, string) {
	var a, b int
	hashtable := make(map[string]bool)
	array := []string{"-repo", "-auth", "-action", "-createses", "-createcaasp", "-usage", "-sestfoutput", "-caasptfoutput", "-caaspuiinst", "-addnodes", "-nodes"}
	for i := 0; i < len(array); i++ {
		hashtable[array[i]] = true
	}

	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "-ostkcmd" {
			a = i + 1
		} else {
			if !hashtable[os.Args[i]] {
				b = i
			}
		}
	}

	wd, _ := os.Getwd()
	os.Chdir(filepath.Join(wd, dir))
	env, err := SetOSEnv(openstackAPIauth)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(os.Args[a : b+1])
	newEnv := append(os.Environ(), env...)
	cmd := exec.Command("openstack", os.Args[a:b+1]...)
	cmd.Env = newEnv
	out, err := cmd.CombinedOutput()
	return fmt.Sprintf("%s", string(out)), fmt.Sprintf("%s", err)
}
