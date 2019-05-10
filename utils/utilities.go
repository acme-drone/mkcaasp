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
var ENV EnvOS

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
	ENV = env
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

func CAASPOutReturner(openstack string, homedir string, caaspDir string) *CAASPOut {
	os.Chdir(filepath.Join(homedir, caaspDir))
	a := CAASPOut{}
	env, err := SetOSEnv(openstack)
	if err != nil {
		log.Fatal(err)
	}
	newEnv := append(os.Environ(), env...)
	cmd := exec.Command("terraform", "output", "-json")
	cmd.Env = newEnv
	out, err := cmd.CombinedOutput()
	if err != nil {
		fmt.Printf("%s\n", err)
	}
	err = json.Unmarshal([]byte(out), &a)
	if err != nil {
		log.Printf("Error while unmarshalling: %s\n", err)
	}
	return &a
}

func AdminOrchCmd(s *CAASPOut, option string, command string) (string, string) {
	var err error
	alias := []string{"docker", "exec", "$(docker ps -q --filter name=salt-master)", "salt", "-P", "\"roles:admin|kube-master|kube-minion\""}
	//------------sweeping through options...
	if option == "refresh" {
		cmd := append(alias, "saltutil.refresh_grains")
		out, err := s.SSHCommand(cmd...).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stdout, "ssh command didn't run as expected: %s\n", err)
		}
		return fmt.Sprintf("%s", string(out)), fmt.Sprintf("%s", err)
	}
	if option == "command" {
		cmd := append(alias, "cmd.run", "'"+command+"'")
		out, err := s.SSHCommand(cmd...).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stdout, "ssh command didn't run as expected: %s\n", err)
		}
		return fmt.Sprintf("%s", string(out)), fmt.Sprintf("%s", err)
	}
	if option == "disable" {
		cmd := append(alias, []string{"cmd.run", "'systemctl disable --now transactional-update.timer'"}...)
		out, err := s.SSHCommand(cmd...).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stdout, "ssh command didn't run as expected: %s\n", err)
		}
		return fmt.Sprintf("%s", string(out)), fmt.Sprintf("%s", err)
	}
	if option == "register" {
		cmdtorun := "'transactional-update register -r " + command + "'"
		cmd := append(alias, []string{"cmd.run", cmdtorun}...)
		out, err := s.SSHCommand(cmd...).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stdout, "ssh command didn't run as expected: %s\n", err)
		}
		return fmt.Sprintf("%s", string(out)), fmt.Sprintf("%s", err)
	}
	if option == "addrepo" {
		cmdtorun := append(alias, "cmd.run 'zypper ar "+command+"'")
		out, err := s.SSHCommand(cmdtorun...).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stdout, "AdmOrchCmd -> addrepo: ssh command didn't run as expected: %s\n", err)
		}
		fmt.Printf(fmt.Sprintf("%s", string(out)))
		return fmt.Sprintf("%s", string(out)), fmt.Sprintf("%s", err)
	}
	if option == "update" || option == "packupdate" {
		var cmd *exec.Cmd
		var stdoutBuf, stderrBuf bytes.Buffer
		//----------system update and updating a development package have slightly different workflow
		if option == "update" {
			cmdtorun := append(alias, []string{"cmd.run", "'transactional-update cleanup dup salt'"}...)
			cmd = s.SSHCommand(cmdtorun...)
		} else {
			//-------if package -> first setting transact-up.conf to allow automatic -y accept development packages
			transactupdconf := []string{"REBOOT_METHOD=salt", "ZYPPER_AUTO_IMPORT_KEYS=1"}
			for i := 0; i < len(transactupdconf); i++ {
				if i == 0 {
					AdminOrchCmd(s, "command", "echo "+transactupdconf[i]+" > /etc/transactional-update.conf")
				} else {
					AdminOrchCmd(s, "command", "echo "+transactupdconf[i]+" >> /etc/transactional-update.conf")
				}
			}
			out, err := AdminOrchCmd(s, "command", "cat /etc/transactional-update.conf")
			if !strings.Contains(err, "nil") {
				return out, err
			}
			if strings.Contains(out, "REBOOT_METHOD=salt") && strings.Contains(out, "ZYPPER_AUTO_IMPORT_KEYS=1") {
				cmdtorun := append(alias, []string{"cmd.run", "'transactional-update", "reboot", "pkg", "install", "-y", command + "'"}...)
				cmd = s.SSHCommand(cmdtorun...)
			} else {
				log.Fatalf("AdminOrchCmd->package update: the trans-update.conf file not properly set up: %s\n", out)
			}
		}
		newEnv := append(os.Environ(), ENV...)
		cmd.Env = newEnv
		stdoutIn, _ := cmd.StdoutPipe()
		stderrIn, _ := cmd.StderrPipe()
		var errStdout, errStderr error
		stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
		stderr := io.MultiWriter(os.Stderr, &stderrBuf)
		err := cmd.Start()
		if err != nil {
			return fmt.Sprintf("%s", os.Stdout), fmt.Sprintf("%s", err)
		}
		go func() {
			_, errStdout = io.Copy(stdout, stdoutIn)
		}()
		go func() {
			_, errStderr = io.Copy(stderr, stderrIn)
		}()
		err = cmd.Wait()
		if err != nil {
			return fmt.Sprintf("%s", os.Stdout), fmt.Sprintf("%s", err)
		}
		if errStdout != nil || errStderr != nil {
			log.Fatal("AdminOrchCmd -> update: failed to capture stdout or stderr\n")
		}
		return stdoutBuf.String(), stderrBuf.String()
	}
	if option == "new" {
		AdminOrchCmd(s, "register", command)
		AdminOrchCmd(s, "disable", "")
		AdminOrchCmd(s, "update", "")
		AdminOrchCmd(s, "refresh", "")
	}
	return fmt.Sprintf("%s", os.Stdout), fmt.Sprintf("%s", err)
}

func NodesAdder(dir string, append string, nodes *CAASPOut, Firsttime bool) *CaaSPCluster {
	temp := strings.Split(append, "")
	if len(temp) > 4 {
		log.Fatalf("Check your syntaxis...there must be just four symbols in -addnodes argument")
	} else {
		//-------------------PARSING the argument of -addnodes or -nodes
		for i := 0; i < len(temp); i++ {
			if temp[i] == "w" {
				if len(temp) >= i+2 {
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

	//------------
	if Firsttime == true {
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
		log.Fatalf("utils.NodesAdder: couldn't create the file...%s", err)
	}
	err = templ.Execute(f, Cluster)
	if err != nil {
		log.Fatalf("utils.NodesAdder: couldn't execute the Cluster template %s", err)
	}
	f.Close()
	out, err := exec.Command("cat", "terraform.tfvars").CombinedOutput()
	if err != nil {
		log.Fatalf("utils.NodesAdder: Couldn't execute the command %s", err)
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
