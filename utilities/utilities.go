package utilities

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"time"
)

// SetOSEnv sets up Openstack environment variables
func SetOSEnv(file string) (EnvOS, error) {
	var auth = OpenStackAPI{}
	f, err := os.Open(file)
	defer f.Close()
	if err != nil {
		return nil, err
	}
	if err := json.NewDecoder(f).Decode(&auth); err != nil {
		return nil, err
	}
	env := EnvOS{
		"OS_AUTH_URL=" + auth.OSAuthURL,
		"OS_REGION_NAME=" + auth.OSRegionName,
		"OS_PROJECT_NAME=" + auth.OSProjectName,
		"OS_USER_DOMAIN_NAME=" + auth.OSUserDomainName,
		"OS_IDENTITY_API_VERSION=" + auth.OSIdentityAPIVersion,
		"OS_INTERFACE=" + auth.OSInterface,
		"OS_USERNAME=" + auth.OSUsername,
		"OS_PASSWORD=" + auth.OSPassword,
		"OS_PROJECT_ID=" + auth.OSProjectID,
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

// CheckVelumUp returns Velum worm up time in Seconds
func CheckVelumUp(page string) float64 {
	t := time.Now()
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	req, err := http.NewRequest(http.MethodGet, page, nil)
	if err != nil {
		log.Fatal(err)
	}
	req = req.WithContext(ctx)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	var DefaultClient = &http.Client{Transport: tr}
	var resp *http.Response
	for {
		resp, err = DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			time.Sleep(10 * time.Second)
			continue
		} else {
			break
		}
	}
	defer resp.Body.Close()

	return time.Since(t).Seconds()
}
