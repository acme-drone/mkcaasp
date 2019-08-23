package utils

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
	"unicode"
)

func CheckOS() (string, error) {
	var sysos string
	out, err := exec.Command("uname", "-a").CombinedOutput()
	if err != nil {
		fmt.Printf("utils.CheckOS -> Error while running uname -a")
		return "", err
	}
	tmp := fmt.Sprintf("%s", string(out))
	if strings.Contains(strings.ToLower(tmp), "darwin") || strings.Contains(strings.ToLower(tmp), "mac") {
		sysos = "mac"
	} else {
		out, err = exec.Command("cat", "/etc/os-release").CombinedOutput()
		if err != nil {
			fmt.Printf("utils.CheckOS -> Error while running uname -a")
			return "", err
		}
		if strings.Contains(strings.ToLower(fmt.Sprintf("%s", string(out))), "suse") {
			sysos = "suse"
		}
	}
	return sysos, err
}

func FolderFinder(sysos string) (string, string) {
	var mkcaasproot, homefolder string
	switch sysos {
	case "suse":
		cmd := []string{"whoami"}
		out, err := exec.Command(cmd[0]).CombinedOutput()
		if err != nil {
			log.Printf("Error occured: %s", err)
		}
		//fmt.Println(fmt.Sprintf("%q", string(out)))
		username := strings.Replace(fmt.Sprintf("%s", string(out)), "\n", "", 1)
		if username == "root" {
			homefolder = "/root"
		} else {
			homefolder = filepath.Join("/home", username)
		}
		folderslice := []string{"go/src/mkcaasp", "work/src/mkcaasp", "golang/src/mkcaasp", "go/src/github.com/atighineanu/mkcaasp", "golang/src/github.com/atighineanu/mkcaasp", "work/src/github.com/atighineanu/mkcaasp"}
		for _, k := range folderslice {
			cmd = []string{"ls", "-alh", filepath.Join(homefolder, k)}
			out, err = exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
			if err != nil {
				log.Printf("A foreseen little error occured (finding folders): %s", err)
			}
			if !strings.Contains(fmt.Sprintf("%s", string(out)), "No such file") {
				mkcaasproot = filepath.Join(homefolder, k)
				break
			}
		}
	}
	if mkcaasproot == "" {
		log.Fatalf("You picked an exotic location for your goroot folder. Please, indicate in the main.go:110-140 the hardcoded value of Mkcaasproot...")
	}
	return mkcaasproot, homefolder
}

func (cluster *SkubaCluster) CheckSkuba() (string, string) {
	cmd := exec.Command("skuba", "cluster", "status")
	cmd.Dir = Myclusterdir
	out, errstr := NiceBuffRunner(cmd, Myclusterdir)
	if errstr != "%!s(<nil>)" && errstr != "" {
		log.Printf("Error while running \"skuba cluster status\":  %s", errstr)
	}
	return out, errstr
}

func CheckIPSSH(node Node) Node {
	count := 0
	//----------Checking if Node has network connection
	command := []string{"ping", "-c", "3", node.IP}
	out, err := exec.Command(command[0], command[1:]...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stdout, "func CheckIPSSH -> Error while running ping %s: %s", node.IP, err)
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
		fmt.Fprintf(os.Stdout, "Func CheckIPSSH -> Error while running nc -zvw3 %s 22: %s", node.IP, err)
	}
	if strings.Contains(fmt.Sprintf("%s", string(out)), "succeeded") {
		node.Port22 = true
	}
	//---------Checking if Node ssh service is fine
	command = []string{"echo", "KEYWORD"}
	cmd := node.SSHCmd("", command)
	out, err = cmd.CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stdout, "Func CheckIPSSH -> Error while SSH-ing into the node %s:  %s", node.IP, err)
	}
	if strings.Contains(fmt.Sprintf("%s", string(out)), "KEYWORD") {
		node.SSH = true
	}
	//
	return node
}

func CheckNodename(node Node, mode string) Node {
	command := []string{"hostname"}
	out, err := node.SSHCmd("", command).CombinedOutput()
	if err != nil {
		log.Printf("func CheckNode -> cmd.hostname: error while running SSH command: %s", err)
	}
	temp := strings.Split(fmt.Sprintf("%s", string(out)), "\n")
	//------------Checking in the output of hostname
	for _, k := range temp {
		if k != " " && k != "" && !strings.Contains(k, "known hosts") {
			node.NodeName = k
		}
	}
	if node.Role != "Load_Balancer" && mode != "setup" {
		checkpath := "/var/log/pods"
		//---checking in the logs for "Local node-name: xxxxxxxxxxxx" to find K8s Name of the node
		command = []string{"sudo", "grep", "-R", "node-name", checkpath}
		out, err = node.SSHCmd("", command).CombinedOutput()
		if err != nil {
			log.Printf("func CheckNode -> error while running SSH sudo grep command: %s", err)
		}
		temp = strings.Split(fmt.Sprintf("%s", string(out)), " ")
		for index, _ := range temp {
			if strings.Contains(temp[index], "pimp") {
				node.K8sName = strings.Replace(temp[index], "\"", "", 10)
				break
			}
		}
	}
	return node
}

func CheckSystemd(node Node) Node {
	var systemd Systemd

	//-----------------critical-chain---------------------
	command := []string{"sudo", "systemd-analyze", "critical-chain"}
	out, err := node.SSHCmd("", command).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Func CheckSystemd -> error running SSH + systemd-analyze: %s", err)
	}
	sysdresult := strings.Split(fmt.Sprintf("%s", string(out)), "\n")
	for i := len(sysdresult) - 1; i >= 0; i-- {
		if sysdresult[i] == "" || strings.Contains(sysdresult[i], "known host") || strings.Contains(sysdresult[i], "the unit") {
			continue
		} else {
			tmp := strings.Split(sysdresult[i], " ")
			for index, k := range tmp {
				if strings.Contains(k, "└") {
					if index+2 < len(tmp) {
						systemd.CriticalChain = append(systemd.CriticalChain, CriticalChain{tmp[index], tmp[index+1], tmp[index+2]})
					} else {
						//unitlist = append(unitlist, tmp[index:])
						systemd.CriticalChain = append(systemd.CriticalChain, CriticalChain{tmp[index], "", ""})
					}
				}
			}
		}
	}
	//----------------------Printing all the Critical Chain-------------------------
	/*
		for _, k := range systemd.CriticalChain {
			fmt.Printf("Unit: %s  TimeAt: %s  TimeDelay: %s\n", k.Unit, k.TimeAt, k.TimeDelay)
		}
	*/

	//-----------------------Analyze-Blame---------------------------------------
	if node.Role == "Load_Balancer" && strings.Contains(systemd.CriticalChain[len(systemd.CriticalChain)-1].Unit, "haproxy") {
		systemd.AllFine = true
	}
	if node.Role == "Master" && strings.Contains(systemd.CriticalChain[len(systemd.CriticalChain)-1].Unit, "crio.service") {
		systemd.AllFine = true
	}
	if node.Role == "Worker" && strings.Contains(systemd.CriticalChain[len(systemd.CriticalChain)-1].Unit, "crio.service") {
		systemd.AllFine = true
	}
	node.Systemd = systemd
	return node
}

///-----------------------WORKS ONLY FOR IPV4--------------------------------------
func CheckIfIP(IP string) {
	for _, k := range IP {
		if unicode.IsDigit(rune(k)) {
			continue
		} else {
			if string(k) == "." {
				continue
			} else {
				log.Fatalf("Please, check your input (should be a [valid] IP adress without pre- or suffixes")
			}
		}
	}
}

func (cluster *SkubaCluster) RebootNodes(flagg string) (string, string) {
	cmdargs := []string{"sudo", "reboot", "--reboot"}
	var out, errstr string
	switch flagg {
	case "workers":
		var cluster SkubaCluster
		cluster.RefreshSkubaCluster()
		if Config.Platform == "openstack" {
			for _, k := range cluster.TF_ostack.IP_Workers.Value {
				CheckIfIP(k)
				var a Node
				a.IP = k
				a.Username = "sles"
				command := a.SSHCmd("", cmdargs)
				out, errstr = NiceBuffRunner(command, "")
				time.Sleep(60 * time.Second)
			}
		}
		if Config.Platform == "vmware" {
			for _, k := range cluster.TF_vmware.IP_Workers.Value {
				CheckIfIP(k)
				var a Node
				a.IP = k
				a.Username = "sles"
				command := a.SSHCmd("", cmdargs)
				out, errstr = NiceBuffRunner(command, "")
			}
		}
	case "masters":
		var cluster SkubaCluster
		cluster.RefreshSkubaCluster()
		if Config.Platform == "openstack" {
			for _, k := range cluster.TF_ostack.IP_Masters.Value {
				CheckIfIP(k)
				var a Node
				a.IP = k
				a.Username = "sles"
				command := a.SSHCmd("", cmdargs)
				out, errstr = NiceBuffRunner(command, "")
			}
		}
		if Config.Platform == "vmware" {
			for _, k := range cluster.TF_vmware.IP_Masters.Value {
				CheckIfIP(k)
				var a Node
				a.IP = k
				a.Username = "sles"
				command := a.SSHCmd("", cmdargs)
				out, errstr = NiceBuffRunner(command, "")
			}
		}
	case "loadbalancer":
		var cluster SkubaCluster
		cluster.RefreshSkubaCluster()
		if Config.Platform == "openstack" {
			k := cluster.TF_ostack.IP_Load_Balancer.Value
			CheckIfIP(k)
			var a Node
			a.IP = k
			a.Username = "sles"
			command := a.SSHCmd("", cmdargs)
			out, errstr = NiceBuffRunner(command, "")

		}
		if Config.Platform == "vmware" {
			for _, k := range cluster.TF_vmware.IP_Load_Balancer.Value {
				CheckIfIP(k)
				var a Node
				a.IP = k
				a.Username = "sles"
				command := a.SSHCmd("", cmdargs)
				out, errstr = NiceBuffRunner(command, "")
			}
		}
	default:
		CheckIfIP(flagg)
		var a Node
		a.IP = flagg
		a.Username = "sles"
		command := a.SSHCmd("", cmdargs)
		out, errstr = NiceBuffRunner(command, "")
	}
	return out, errstr
}

func (cluster *SkubaCluster) ClusterCheckBuilder(mode string) map[string]Node {
	b := make(map[string]Node)
	var node Node
	node.Username = "sles" //to be improved if different user for different roles...
	if mode == "checks" {
		//-------setting load balancer
		switch Config.Platform {
		case "openstack":
			k := cluster.TF_ostack.IP_Load_Balancer.Value
			node.IP = k
			node := CheckIPSSH(node)
			node.Role = "Load_Balancer"
			node = CheckNodename(node, mode)
			node = CheckSystemd(node)
			b[k] = node
			//---------setting masters------------
			for _, k := range cluster.TF_ostack.IP_Masters.Value {
				node.IP = k
				node := CheckIPSSH(node)
				node.Role = "Master"
				node = CheckNodename(node, mode)
				node = CheckSystemd(node)
				b[k] = node
			}
			//---------setting workers--------------
			for _, k := range cluster.TF_ostack.IP_Workers.Value {
				node.IP = k
				node := CheckIPSSH(node)
				node.Role = "Worker"
				node = CheckNodename(node, mode)
				node = CheckSystemd(node)
				b[k] = node
			}

		case "vmware":
			for _, k := range cluster.TF_vmware.IP_Load_Balancer.Value {
				node.IP = k
				node := CheckIPSSH(node)
				node.Role = "Load_Balancer"
				node = CheckNodename(node, mode)
				node = CheckSystemd(node)
				b[k] = node
			}
			//---------setting masters------------
			for _, k := range cluster.TF_vmware.IP_Masters.Value {
				node.IP = k
				node := CheckIPSSH(node)
				node.Role = "Master"
				node = CheckNodename(node, mode)
				node = CheckSystemd(node)
				b[k] = node
			}
			//---------setting workers--------------
			for _, k := range cluster.TF_vmware.IP_Workers.Value {
				node.IP = k
				node := CheckIPSSH(node)
				node.Role = "Worker"
				node = CheckNodename(node, mode)
				node = CheckSystemd(node)
				b[k] = node
			}
		}
		for key, value := range b {
			fmt.Printf("b[%s] = {Role:%s;  %s  %s  %s  %s}\n", key, value.Role, value.K8sName, value.IP, value.NodeName, value.Systemd.AllFine)
		}
	}
	if mode == "setup" {
		switch Config.Platform {
		case "openstack":
			k := cluster.TF_ostack.IP_Load_Balancer.Value
			node.IP = k
			node := CheckIPSSH(node)
			node.Role = "Load_Balancer"
			node = CheckNodename(node, mode)
			b[k] = node
			//------------init masers---------------
			for _, k := range cluster.TF_ostack.IP_Masters.Value {
				node.IP = k
				node := CheckIPSSH(node)
				node.Role = "Master"
				node = CheckNodename(node, mode)
				b[k] = node
			}
			//-------------init workers-------------
			for _, k := range cluster.TF_ostack.IP_Workers.Value {
				node.IP = k
				node := CheckIPSSH(node)
				node.Role = "Worker"
				node = CheckNodename(node, mode)
				b[k] = node
			}
		case "vmware":
			for _, k := range cluster.TF_vmware.IP_Load_Balancer.Value {
				node.IP = k
				node := CheckIPSSH(node)
				node.Role = "Load_Balancer"
				node = CheckNodename(node, mode)
				b[k] = node
			}
			for _, k := range cluster.TF_vmware.IP_Masters.Value {
				node.IP = k
				node := CheckIPSSH(node)
				node.Role = "Master"
				node = CheckNodename(node, mode)
				b[k] = node
			}
			//-------------init workers-------------
			for _, k := range cluster.TF_vmware.IP_Workers.Value {
				node.IP = k
				node := CheckIPSSH(node)
				node.Role = "Worker"
				node = CheckNodename(node, mode)
				b[k] = node
			}
		}
	}
	return b
}
