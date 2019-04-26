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
)

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
		"OS_PASSWORD=" + auth.Password,
		"OS_PROJECT_ID=" + auth.ProjectID,
	}
	return env, nil
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

func Hashinator(s string, caasporsespdir string) {
	h := md5.New()
	h.Write([]byte(s))

	f, err := ioutil.ReadFile(caasporsespdir + "/openstack.json")
	if err != nil {
		fmt.Printf("Error at opening json file!...%s", err)
	}

	if strings.Contains(string(f), ":\"pass\",") || strings.Contains(string(f), "\"Password\":\"\",") {
		f = []byte(strings.Replace(string(f), "pass", hex.EncodeToString(h.Sum(nil)), -1))
		err = ioutil.WriteFile(caasporsespdir+"/openstack.json", f, 0644)
		if err != nil {
			fmt.Printf("Error at writing to json file!...%s", err)
		}
	}
}
