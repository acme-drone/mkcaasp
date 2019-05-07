package utils

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
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
		"OS_PASSWORD=" + Dehashinator("./../", "./"),
		"OS_PROJECT_ID=" + auth.ProjectID,
	}
	return env, nil
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
		log.Fatalf("Search didn't work...%s", err)
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
		fmt.Printf("%s", err)
	}
	f.Close()
	out, err := exec.Command("cat", "terraform.tfvars").CombinedOutput()
	if err != nil {
		log.Printf("Error! %s\n", err)
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
	time.Sleep(10 * time.Second)
	env, err := SetOSEnv(openstackAPIauth)
	if err != nil {
		log.Fatal(err)
	}
	outstd, outstderr := RunScript(command, env)
	return outstd, outstderr
}

func Hashinator(pass string, homedir string, caasporsespdir string) {
	v := OSAPI{}
	var tempkey string
	file, _ := os.Open(homedir + "/key.json")
	decoder := json.NewDecoder(file)
	defer file.Close()
	err := decoder.Decode(&tempkey)
	if err != nil {
		fmt.Printf("This is bad! .json decoding didn't work at opening %s key.json:  %s\n", homedir, err)
	}

	file, _ = os.Open(homedir + "/" + caasporsespdir + "/openstack.json")
	decoder = json.NewDecoder(file)
	defer file.Close()
	err = decoder.Decode(&v)
	if err != nil {
		fmt.Printf("This is bad! .json opening at %s openstack.json didn't work:  %s\n", homedir+"/"+caasporsespdir, err)
	}
	block, _ := aes.NewCipher([]byte(Hasher(tempkey)))
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		panic(err.Error())
	}
	ciphertext := gcm.Seal(nonce, nonce, []byte(pass), nil)
	f, err := ioutil.ReadFile(homedir + "/" + caasporsespdir + "/openstack.json")
	if err != nil {
		fmt.Printf("Error at opening json file!...%s\n", err)
	}
	v.Password = ciphertext[:]
	f, _ = json.MarshalIndent(v, "", " ")
	err = ioutil.WriteFile(homedir+"/"+caasporsespdir+"/openstack.json", f, 0644)
	if err != nil {
		fmt.Printf("Error at writing to json file!...%s\n", err)
	}
}

func Dehashinator(homedir string, caasporsespdir string) string {
	v := OSAPI{}
	file, _ := os.Open(caasporsespdir + "/openstack.json")
	decoder := json.NewDecoder(file)
	defer file.Close()
	err := decoder.Decode(&v)
	if err != nil {
		fmt.Printf("This is bad! .json decoding didn't work @opening openstack.json: %s", err)
	}

	var tempkey string
	file, _ = os.Open(homedir + "/key.json")
	decoder = json.NewDecoder(file)
	defer file.Close()
	err = decoder.Decode(&tempkey)
	if err != nil {
		fmt.Printf("This is bad! .json decoding didn't work @opening key.json  %s:", err)
	}

	block, err := aes.NewCipher([]byte(Hasher(tempkey)))
	if err != nil {
		panic(err.Error())
	}
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		panic(err.Error())
	}
	nonceSize := gcm.NonceSize()
	nonce, ciphertext := v.Password[:nonceSize], v.Password[nonceSize:]
	decoded, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		panic(err.Error())
	}
	return string(decoded)
}

func Hasher(key string) string {
	hasher := md5.New()
	hasher.Write([]byte(key))
	return hex.EncodeToString(hasher.Sum(nil))
}

func OpenstackCmd(dir string, openstackAPIauth string) (string, string) {
	var stdoutBuf, stderrBuf bytes.Buffer
	var a, b int
	var command []string
	hashtable := make(map[string]bool)
	array := []string{"-repo", "-auth", "-action", "-createses", "-createcaasp", "-usage", "-sestfoutput", "-caasptfoutput", "-caaspuiinst", "-addnodes", "-nodes"}
	for i := 0; i < len(array); i++ {
		hashtable[array[i]] = true
	}
	for i := 0; i < len(os.Args); i++ {
		if os.Args[i] == "-ostkcmd" {
			a = i
		} else {
			if a > 0 && i > a && hashtable[os.Args[i]] {
				b = i
			}
		}
	}
	command = append([]string{}, os.Args[a+1:b]...)
	wd, _ := os.Getwd()
	os.Chdir(filepath.Join(wd, dir))
	env, err := SetOSEnv(openstackAPIauth)
	if err != nil {
		log.Fatal(err)
	}
	command = append([]string{"-c"}, command...)
	cmd := exec.Command("bash", command...)
	newEnv := append(os.Environ(), env...)
	cmd.Env = newEnv

	stdoutIn, _ := cmd.StdoutPipe()
	stderrIn, _ := cmd.StderrPipe()

	var errStdout, errStderr error
	stdout := io.MultiWriter(os.Stdout, &stdoutBuf)
	stderr := io.MultiWriter(os.Stderr, &stderrBuf)

	err = cmd.Start()
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
