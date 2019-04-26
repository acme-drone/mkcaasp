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
	"io"
	"io/ioutil"
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
	path, err := filepath.Abs(filepath.Dir(file))
	if err != nil {
		fmt.Printf("key.json or openstack.json path not properly set...%s", err)
	}
	env := EnvOS{
		"OS_AUTH_URL=" + auth.AuthURL,
		"OS_REGION_NAME=" + auth.RegionName,
		"OS_PROJECT_NAME=" + auth.ProjectName,
		"OS_USER_DOMAIN_NAME=" + auth.UserDomainName,
		"OS_IDENTITY_API_VERSION=" + auth.IdentityAPIVersion,
		"OS_INTERFACE=" + auth.Interface,
		"OS_USERNAME=" + auth.Username,
		"OS_PASSWORD=" + Dehashinator(path+"/../", path),
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

func Hashinator(pass string, homedir string, caasporsespdir string) {
	v := OSAPI{}
	var tempkey string
	file, _ := os.Open(homedir + "/key.json")
	decoder := json.NewDecoder(file)
	defer file.Close()
	err := decoder.Decode(&tempkey)
	if err != nil {
		fmt.Println("This is bad! .json decoding didn't work:", err)
	}

	file, _ = os.Open(homedir + caasporsespdir + "/openstack.json")
	decoder = json.NewDecoder(file)
	defer file.Close()
	err = decoder.Decode(&v)
	if err != nil {
		fmt.Println("This is bad! .json decoding didn't work:", err)
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
	f, err := ioutil.ReadFile(homedir + caasporsespdir + "/openstack.json")
	if err != nil {
		fmt.Printf("Error at opening json file!...%s")
	}
	v.Password = ciphertext[:]
	f, _ = json.MarshalIndent(v, "", " ")
	err = ioutil.WriteFile(homedir+caasporsespdir+"/openstack.json", f, 0644)
	if err != nil {
		fmt.Printf("Error at writing to json file!...%s")
	}
}

func Dehashinator(homedir string, caasporsespdir string) string {

	v := OSAPI{}
	file, _ := os.Open(caasporsespdir + "/openstack.json")
	decoder := json.NewDecoder(file)
	defer file.Close()
	err := decoder.Decode(&v)
	if err != nil {
		fmt.Println("This is bad! .json decoding didn't work:", err)
	}

	var tempkey string
	file, _ = os.Open(homedir + "/key.json")
	decoder = json.NewDecoder(file)
	defer file.Close()
	err = decoder.Decode(&tempkey)
	if err != nil {
		fmt.Println("This is bad! .json decoding didn't work:", err)
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
