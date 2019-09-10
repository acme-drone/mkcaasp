package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	Homedir        string
	Mkcaasproot    string
	Config         *MKCaaSPCfg //CaaSP4CFG(Mkcaasproot)
	err            error
	Skubaroot      string
	Vmwaretfdir    string
	Openstacktfdir string
	Myclusterdir   string
	Testworkdir    string
	ENV2           = os.Environ()
	Workdir        string
	//workdir      = filepath.Join(skubaroot, "test/lib/prototyping")
)

//---------------------------INITIATING OS.ENV, VARIABLES, TESTCLUSTER DATA STRUCTURE-----------------------

func OpenstackExporter(Mkcaasproot string) {
	fmt.Println(filepath.Join(Mkcaasproot, "openstack.json"))
	tmpenv, _ := SetOSEnv(filepath.Join(Mkcaasproot, "openstack.json"))
	ENV2 = append(ENV2, tmpenv...)
}

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

func (cluster *SkubaCluster) TFParser() error {
	var a *TFOutput_vmware
	var b *TFOutput_openstack
	/*	cmd := exec.Command("terraform", "init")
		out, errstr := NiceBuffRunner(cmd, Workdir)
		if errstr != "%!s(<nil>)" && errstr != "" {
			return nil, err
			log.Printf("Error while running \"terraform output -json\":  %s", errstr)
		}
	*/
	cmd := exec.Command("terraform", "output", "-json")
	cmd.Env = ENV2
	out, errstr := NiceBuffRunner(cmd, Workdir)
	if errstr != "%!s(<nil>)" && errstr != "" {
		return err
		log.Printf("Error while running \"terraform output -json\":  %s", errstr)
	}
	switch Config.Platform {
	case "vmware":
		err := json.Unmarshal([]byte(out), &a)
		if err != nil {
			return err
			log.Printf("Error while unmarshalling: %s", err)
		}
		cluster.TF_vmware = a
	case "openstack":
		err := json.Unmarshal([]byte(out), &b)
		if err != nil {
			return err
			log.Printf("Error while unmarshalling: %s", err)
		}
		cluster.TF_ostack = b
	}
	return nil
}

func (cluster *SkubaCluster) EnvOSExporter() []string {
	//--------appending to os.env var all the node names...
	cluster.TFParser()
	switch Config.Platform {
	case "openstack":
		elem := cluster.TF_ostack.IP_Load_Balancer.Value
		ENV2 = append(ENV2, fmt.Sprintf("CONTROLPLANE=%s", elem))
		for index, elem := range cluster.TF_ostack.IP_Masters.Value {
			ENV2 = append(ENV2, fmt.Sprintf("MASTER0%v_PIMP_GENERAL=%s", index, elem))
		}
		for index, elem := range cluster.TF_ostack.IP_Workers.Value {
			ENV2 = append(ENV2, fmt.Sprintf("WORKER0%v_PIMP_COMRADE=%s", index, elem))
		}
	case "vwmare":
		for _, elem := range cluster.TF_vmware.IP_Load_Balancer.Value {
			ENV2 = append(ENV2, fmt.Sprintf("CONTROLPLANE=%s", elem))
		}
		for index, elem := range cluster.TF_vmware.IP_Masters.Value {
			ENV2 = append(ENV2, fmt.Sprintf("MASTER0%v_PIMP_GENERAL=%s", index, elem))
		}
		for index, elem := range cluster.TF_vmware.IP_Workers.Value {
			ENV2 = append(ENV2, fmt.Sprintf("WORKER0%v_PIMP_COMRADE=%s", index, elem))
		}

	}
	//--------appending to os.env var the clustername...
	ENV2 = append(ENV2, fmt.Sprintf("CLUSTERNAME=%s", Config.ClusterName))
	//--------appending skuba root dir
	ENV2 = append(ENV2, fmt.Sprintf("CLUSTERNAME=%s", Config.ClusterName))
	return ENV2
}

func (cluster *SkubaCluster) RefreshSkubaCluster() {
	cluster.ClusterName = Config.ClusterName
	cluster.TFParser()
	if err != nil {
		log.Printf("TF parsing did not work: %s", err)
	}
	//cluster.Diagnosis = ClusterCheckBuilder(cluster.TF, "setup")
	cluster.Diagnosis = make(map[string]Node)
}

//------------------------------------------------ EXECUTING ACTUAL CHANGES, TF-DEPLOY, ADDNODE, RUN TEST, SKUBA -COMMAND---------------

func (cluster *SkubaCluster) RunGinkgo() (string, string) {
	//cmd := exec.Command("go", "test")
	cmd := exec.Command("ginkgo" /*"-mod=vendor",*/, "-v", "-r", Testworkdir)
	cmd.Env = ENV2
	log.Println("Running ginko test now...")
	out, errstr := NiceBuffRunner(cmd, Testworkdir)
	if errstr != "" {
		fmt.Printf("%s", errstr)
	}
	return out, errstr
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
		_, err := exec.Command("rm", "-R", Myclusterdir).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stdout, "Error removing the %s folder...%v", Myclusterdir, err)
		}
	}
	if suffix != "" {
		cmd = exec.Command("terraform", action, suffix)
	} else {
		cmd = exec.Command("terraform", action)
	}
	cmd.Env = ENV2
	out, errstr := NiceBuffRunner(cmd, Workdir)
	if errstr != "%!s(<nil>)" && errstr != "" {
		log.Printf("Error while running \"terraform command\":  %s", errstr)
	}
	if action == "apply" {
		log.Printf("The Nodes were successfully Deployed by %s on %s Platform", Config.Deploy, Config.Platform)
		time.Sleep(10 * time.Second)
	}
	return out, errstr
}

//------------------Initializing CaaSP4 Cluster with Skuba-----------------
func (cluster *SkubaCluster) SkubaInit() (string, string) {
	var out, errstr string
	var cmd *exec.Cmd
	switch Config.Platform {
	case "openstack":
		cmd = exec.Command("skuba", "cluster", "init", "--control-plane", cluster.TF_ostack.IP_Load_Balancer.Value, cluster.ClusterName)
	case "vmware":
		cmd = exec.Command("skuba", "cluster", "init", "--control-plane", cluster.TF_vmware.IP_Load_Balancer.Value[0], cluster.ClusterName)
	}
	cmd.Env = append(cmd.Env, cluster.EnvOSExporter()...)
	if Config.Deploy == "terraform" && Config.Platform == "vmware" {
		out, errstr := NiceBuffRunner(cmd, Testworkdir)
		if errstr != "%!s(<nil>)" && errstr != "" {
			return out, errstr
		}
		log.Printf("Successfully initiated the cluster load balancer: %s ...\n", cluster.TF_vmware.IP_Load_Balancer.Value[0])
		time.Sleep(20 * time.Second)
	}
	if Config.Deploy == "terraform" && Config.Platform == "openstack" {
		out, errstr := NiceBuffRunner(cmd, Testworkdir)
		if errstr != "%!s(<nil>)" && errstr != "" {
			return out, errstr
		}
		log.Printf("Successfully initiated the cluster load balancer: %s ...\n", cluster.TF_ostack.IP_Load_Balancer.Value)
		time.Sleep(20 * time.Second)
	}
	return out, errstr
}

//------------------Bootstrapping Masters on CaaSP4 with Skuba---------------
func (cluster *SkubaCluster) BootstrapMaster(mode string) (string, string) {
	var out, errstr string
	if strings.Contains(mode, "selective:") {
		k8sname := fmt.Sprintf("master-pimp-general-0%v", rand.Intn(1000))
		ip := strings.Replace(mode, "selective:", "", 10)
		cmd := exec.Command("skuba", "node", "bootstrap", "--user", "sles", "--sudo", "--target", ip, k8sname)
		node := cluster.Diagnosis[ip]
		node.K8sName = k8sname
		cluster.Diagnosis[ip] = node
		cmd.Dir = filepath.Join(Testworkdir, cluster.ClusterName)
		cmd.Env = append(cmd.Env, ENV2...)
		_, errstr := NiceBuffRunner(cmd, filepath.Join(Testworkdir, cluster.ClusterName))
		if errstr != "%!s(<nil>)" && errstr != "" {
			return out, errstr
		}
		log.Printf("Successfully installed %s ->IP: %s in the cluster...\n", k8sname, ip)
	}
	if mode == "sequential" {
		switch Config.Platform {
		case "openstack":
			count := 0
			var cmd *exec.Cmd
			for index, k := range cluster.TF_ostack.IP_Masters.Value {
				k8sname := fmt.Sprintf("master-pimp-general-0%v", index)
				if count == 0 {
					cmd = exec.Command("skuba", "node", "bootstrap", "--user", "sles", "--sudo", "--target", k, k8sname)
				} else {
					cmd = exec.Command("skuba", "node", "join", "--role", "master", "--user", "sles", "--sudo", "--target", k, k8sname)
				}
				node := cluster.Diagnosis[k]
				node.K8sName = k8sname
				cluster.Diagnosis[k] = node
				cmd.Env = append(cmd.Env, ENV2...)
				_, errstr := NiceBuffRunner(cmd, filepath.Join(Testworkdir, cluster.ClusterName))
				if errstr != "%!s(<nil>)" && errstr != "" {
					log.Printf("Looks like an error: %s\n", errstr)
				}
				log.Printf("Successfully installed %s ->IP: %s in the cluster...\n", k8sname, k)
				count++
			}
		case "vmware":
			count := 0
			var cmd *exec.Cmd
			for index, k := range cluster.TF_vmware.IP_Masters.Value {
				k8sname := fmt.Sprintf("master-pimp-general-0%v", index)
				if count == 0 {
					cmd = exec.Command("skuba", "node", "bootstrap", "--user", "sles", "--sudo", "--target", k, k8sname)
				} else {
					cmd = exec.Command("skuba", "node", "join", "--role", "master", "--user", "sles", "--sudo", "--target", k, k8sname)
				}
				node := cluster.Diagnosis[k]
				node.K8sName = k8sname
				cluster.Diagnosis[k] = node
				cmd.Env = append(cmd.Env, ENV2...)
				_, errstr := NiceBuffRunner(cmd, filepath.Join(Testworkdir, cluster.ClusterName))
				if errstr != "%!s(<nil>)" && errstr != "" {
					log.Printf("Looks like an error: %s\n", errstr)
				}
				log.Printf("Successfully installed %s ->IP: %s in the cluster...\n", k8sname, k)
				count++
			}
		}

	}
	return out, errstr
}

//------------Joining workers with Skuba-------------------------
func (cluster *SkubaCluster) JoinWorkers() (string, string) {
	var out, errstr string
	switch Config.Platform {
	case "openstack":
		for index, k := range cluster.TF_ostack.IP_Workers.Value {
			k8sname := fmt.Sprintf("worker-pimp-comrade-0%v", index)
			node := cluster.Diagnosis[k]
			node.K8sName = k8sname
			cluster.Diagnosis[k] = node
			cmd := exec.Command("skuba", "node", "join", "--role", "worker", "--user", "sles", "--sudo", "--target", k, k8sname)
			cmd.Dir = Myclusterdir
			cmd.Env = append(cmd.Env, ENV2...)
			out, errstr = NiceBuffRunner(cmd, Myclusterdir)
			if errstr != "%!s(<nil>)" && errstr != "" && errstr != " " {
				log.Printf("Looks like an error: %s\n", errstr)
			}
		}
	case "vmware":
		for index, k := range cluster.TF_vmware.IP_Workers.Value {
			k8sname := fmt.Sprintf("worker-pimp-comrade-0%v", index)
			node := cluster.Diagnosis[k]
			node.K8sName = k8sname
			cluster.Diagnosis[k] = node
			cmd := exec.Command("skuba", "node", "join", "--role", "worker", "--user", "sles", "--sudo", "--target", k, k8sname)
			cmd.Env = append(cmd.Env, ENV2...)
			out, errstr = NiceBuffRunner(cmd, Myclusterdir)
			if errstr != "%!s(<nil>)" && errstr != "" && errstr != " " {
				log.Printf("Looks like an error: %s\n", errstr)
			}
		}
	}

	return out, errstr
}

//---------copying the admin conf to .kube/conf ...
func (cluster *SkubaCluster) CopyAdminConf() (string, string) {
	out, err := exec.Command("cp", filepath.Join(Myclusterdir, "admin.conf"), filepath.Join(Homedir, ".kube/config")).CombinedOutput()
	if err != nil {
		log.Printf("There is an error copying the admin.conf file: %s", err)
	}
	return fmt.Sprintf("%s", string(out)), fmt.Sprintf("%s", err)
}

func (node *Node) SSHCmd(workdir string, command []string) *exec.Cmd {
	args := append(
		[]string{"-o", "StrictHostKeyChecking=no", "-o", "UserKnownHostsFile /dev/null", "-i", filepath.Join(Skubaroot, "ci/infra/id_shared"),
			fmt.Sprintf("%s@%s", node.Username, node.IP),
		},
		command...,
	)
	return exec.Command("ssh", args...)
}

func (cluster *SkubaCluster) NodesAdderV4() {
	//---------------Adding the new nodes to the cluster.tf file....
	switch Config.Platform {
	case "vmware":
		templ, err := template.New("AddingNodesSkuba").Parse(VmwareVarsTempl)
		if err != nil {
			log.Fatalf("Error parsin ClusterTempl constant...%s", err)
		}
		var f *os.File
		f, err = os.Create(filepath.Join(Vmwaretfdir, "terraform.tfvars"))
		if err != nil {
			log.Fatalf("utils.NodesAdder: couldn't create the file...%s", err)
		}
		err = templ.Execute(f, cluster.Setup)
		if err != nil {
			log.Fatalf("utils.NodesAdder: couldn't execute the Cluster template %s", err)
		}
		f.Close()
		_, err = exec.Command("cat", filepath.Join(Vmwaretfdir, "terraform.tfvars")).CombinedOutput()
		if err != nil {
			log.Fatalf("utils.NodesAdder: Couldn't execute the cat terraform.tfvars command %s", err)
		}
		log.Printf("That's the modified cluster config...\n Masters: %v    Workers: %v\n", cluster.Setup.MastCount, cluster.Setup.WorkCount)
	case "openstack":
		templ, err := template.New("AddingNodesSkuba").Parse(OpenstackVarsTempl)
		if err != nil {
			log.Fatalf("Error parsin ClusterTempl constant...%s", err)
		}
		var f *os.File
		f, err = os.Create(filepath.Join(Openstacktfdir, "terraform.tfvars"))
		if err != nil {
			log.Fatalf("utils.NodesAdder: couldn't create the file...%s", err)
		}
		err = templ.Execute(f, cluster.Setup)
		if err != nil {
			log.Fatalf("utils.NodesAdder: couldn't execute the Cluster template %s", err)
		}
		f.Close()
		_, err = exec.Command("cat", filepath.Join(Openstacktfdir, "terraform.tfvars")).CombinedOutput()
		if err != nil {
			log.Fatalf("utils.NodesAdder: Couldn't execute the cat terraform.tfvars command %s", err)
		}
		log.Printf("That's the modified cluster config...\n Masters: %v    Workers: %v\n", cluster.Setup.MastCount, cluster.Setup.WorkCount)
	}
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
