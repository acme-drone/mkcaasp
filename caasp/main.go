package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"mkcaasp/utils"
	"os"
	"strings"
)

const (
	command  = "terraform %s -var auth_url=$OS_AUTH_URL -var domain_name=$OS_USER_DOMAIN_NAME -var region_name=$OS_REGION_NAME -var project_name=$OS_PROJECT_NAME -var user_name=$OS_USERNAME -var password=$OS_PASSWORD -var-file=terraform.tfvars -auto-approve"
	howtouse = `
			 Make sure you have terraform installed and in $PATH
			 git clone https://github.com/kubic-project/automation.git
			 
			 cd automation

			 put openstack.json in the directories you want to use, for example in caaspDir and/or sesDir
			 
			 openstack.json (should reside in caaspDir and sesDir folders) should look like this:
			 {
				"AuthURL":"https://smtg:5000/v3",
				"RegionName":"Region",
				"ProjectName":"caasp",
				"UserDomainName":"users",
				"IdentityAPIVersion":"3",
				"Interface":"public",
				"Username":"user",
				"Password":"pass",
				"ProjectID":"00000000000000000000000000"
			 }

			 --------------------------------------->>>IMPORTANT!<<<-----------------------------------------------------------
			 run the utility: caasp -repo $HOME/automation -createcaasp -caaspuiinst -createses -action apply -auth openstack.json
			 ------------------------------------------------------------------------------------------------------------------


			 CHECK also the data template in utils/data.go, which sets up the parameters of your cluster in engcloud, 
			 and looks like this:
			 
			var CulsterTempl = 	image_name = "SUSE-CaaS-Platform-3.0-for-OpenStack-Cloud.x86_64-3.0.0-GM.qcow2"
							 	internal_net = "INGSOC-net"
							 	external_net = "floating"
								admin_size = "m1.large"
								master_size = "m1.medium"
								masters = {{.MastCount}}
								worker_size = "m1.medium"
								workers = {{.WorkCount}}
								workers_vol_enabled = 0
								workers_vol_size = 5
								dnsdomain = "testing.qa.caasp.suse.net"
								dnsentry = 0
								stack_name = "INGSOC"
				
			Last but not least: make sure you put your SCC-key whether in a variable in package utils which isn't part of the project
			(like key.go), or hardcoded... in main.go. But we aware: with 1st commit your key will be visible on github.`
)

var (
	libvirt       = flag.String("tflibvirt", "", "switch for terraform-libvirt option")
	openstack     = flag.String("auth", "openstack.json", "name of the json file containing openstack variables")
	action        = flag.String("action", "apply", "terraform action to run, example: apply, destroy")
	caasp         = flag.Bool("createcaasp", false, "enables/disables caasp terraform openstack setup")
	ses           = flag.Bool("createses", false, "enables/disables ses terraform openstack setup")
	howto         = flag.Bool("usage", false, "prints usage information")
	caasptfoutput = flag.Bool("caasptfoutput", false, "loads in memory caasp terraform ouput json")
	sestfoutput   = flag.Bool("sestfoutput", false, "loads in memory ses terraform ouput json")
	caaspUIInst   = flag.Bool("caaspuiinst", false, "Configures caasp using Velum UI")
	ostkcmd       = flag.String("ostkcmd", "", "openstack command to run")
	nodes         = flag.String("nodes", "", "what is the cluster starting configuration. How many masters/workers? w1m1 or w3m1 or m3w5")
	addnodes      = flag.String("addnodes", "", `how many more nodes to add, usage m2w2 -2 more masters, 2 more workers
	Argument must be not longer than 4 symbols (e.g. workers or masters with count more than 1 digit cannot be added; like w10m1)`)
	home = flag.String("repo", "automation", "kubic automation repo location")
	//pass     = flag.String("pass", "password", "the password for cloud to be hashed (and be exported into openstack.json)")
	//hash     = flag.String("key", "default", "chose which string is going to be your hash key")
	cmd      = flag.String("cmd", "", "the orchestration command to run from admin using salt-master container")
	refresh  = flag.Bool("ref", false, "refreshing the salt grains from admin")
	disable  = flag.Bool("dis", false, "disabling transactional-update from admin")
	register = flag.Bool("reg", false, "registering the cluster to SCC")
	addrepo  = flag.String("ar", "", "adding a repository (based on an URL) to the cluster")
	sysupd   = flag.Bool("sysupd", false, "triggers transactional-update cleanup dup salt")
	packupd  = flag.String("packupd", "", "triggers transactional-update with auto-approve for 1 single given package")
	new      = flag.Bool("new", false, "setting up & updating the fresh spawned cluster")
	uiupd    = flag.Bool("uiupd", false, "triggers updating of the cluster through Velum")
)

const (
	caaspDir = "caasp-openstack-terraform"
	sesDir   = "ses-openstack-terraform"
	output   = "terraform output -json"
)

var Cluster *utils.CaaSPCluster

func main() {
	flag.Parse()
	if *howto {
		fmt.Fprintf(os.Stdout, "%v\n", howtouse)
		os.Exit(0)
	}
	os.Chdir(*home)
	if *ostkcmd != "" {
		out1, out2 := utils.CmdRun(caaspDir, *openstack, *ostkcmd)
		fmt.Printf("%s\n  %s\n", out1, out2)
	}
	os.Chdir(*home)
	/*	if *pass != "password" {
		utils.Hashinator(*pass, *hash, *home, caaspDir)
		utils.Hashinator(*pass, *hash, *home, sesDir)
	} */
	os.Chdir(*home)
	if *caasp {
		out, _ := utils.CmdRun(caaspDir, *openstack, output)
		a := utils.CAASPOut{}
		err := json.Unmarshal([]byte(out), &a)
		if err != nil {
			log.Fatal(err)
		}
		if *nodes == "" {
			*nodes = "m1w2"
		}
		Cluster = utils.NodesAdder(caaspDir, *nodes, &a, true)
		utils.TfInit(caaspDir)
		utils.CmdRun(caaspDir, *openstack, fmt.Sprintf(command, *action))
	}
	os.Chdir(*home)
	if *caaspUIInst {
		out, _ := utils.CmdRun(caaspDir, *openstack, output)
		a := utils.CAASPOut{}
		err := json.Unmarshal([]byte(out), &a)
		if err != nil {
			log.Fatal(err)
		}
		velumURL := fmt.Sprintf("https://%s.nip.io", a.IPAdminExt.Value)
		fmt.Fprintf(os.Stdout, "Velum warm up time: %2.2f Seconds\n", utils.CheckVelumUp(velumURL))
		utils.CreateAcc(&a)
		utils.FirstSetup(&a)
	}
	os.Chdir(*home)
	if *addnodes != "" {
		out, _ := utils.CmdRun(caaspDir, *openstack, output)
		a := utils.CAASPOut{}
		err := json.Unmarshal([]byte(out), &a)
		if err != nil {
			log.Fatal(err)
		}
		velumURL := fmt.Sprintf("https://%s.nip.io", a.IPAdminExt.Value)
		fmt.Fprintf(os.Stdout, "Velum warm up time: %2.2f Seconds\n", utils.CheckVelumUp(velumURL))
		Cluster := utils.NodesAdder(caaspDir, *addnodes, &a, false)
		utils.CmdRun(caaspDir, *openstack, fmt.Sprintf(command, *action))
		a = utils.CAASPOut{}
		err = json.Unmarshal([]byte(out), &a)
		if err != nil {
			log.Fatal(err)
		}
		utils.InstallUI(&a, Cluster)
	}
	os.Chdir(*home)
	if *ses {
		utils.TfInit(sesDir)
		utils.CmdRun(sesDir, *openstack, fmt.Sprintf(command, *action))
	}
	os.Chdir(*home)
	if *caasptfoutput {
		utils.CmdRun(caaspDir, *openstack, output)
	}
	os.Chdir(*home)
	if *sestfoutput {
		out, _ := utils.CmdRun(sesDir, *openstack, output)
		a := utils.SESOut{}
		err := json.Unmarshal([]byte(out), &a)
		if err != nil {
			fmt.Printf("%s\n", err)
		}
		s := a.K8SSC.Value
		fmt.Println(a.IPAdminExt.Value, a.IPAdminInt.Value, a.IPMonsExt.Value, a.IPMonsExt.Value, a.IPOsdsInt.Value, "\n", a.K8SCS.Value, "\n", fmt.Sprintf("%s", s[0]))
	}
	//----------------------Cluster - Related - Commands (orchestration)
	os.Chdir(*home)
	if *refresh {
		out, err := utils.AdminOrchCmd(utils.CAASPOutReturner(*openstack, *home, caaspDir), "refresh", "")
		if !strings.Contains(err, "nil") {
			fmt.Printf("%s\n%s\n", out, err)
		} else {
			fmt.Printf("%s\n", out)
		}
	}
	os.Chdir(*home)
	if *cmd != "" {
		out, err := utils.AdminOrchCmd(utils.CAASPOutReturner(*openstack, *home, caaspDir), "command", *cmd)
		if !strings.Contains(err, "nil") {
			fmt.Printf("%s\n%s\n", out, err)
		} else {
			fmt.Printf("%s\n", out)
		}
	}
	os.Chdir(*home)
	if *disable {
		out, err := utils.AdminOrchCmd(utils.CAASPOutReturner(*openstack, *home, caaspDir), "disable", "")
		if !strings.Contains(err, "nil") {
			fmt.Printf("%s\n%s\n", out, err)
		} else {
			fmt.Printf("%s\n", out)
		}
	}
	os.Chdir(*home)
	if *register {
		utils.AdminOrchCmd(utils.CAASPOutReturner(*openstack, *home, caaspDir), "register", utils.RegCode) // <<----------- unexistent variable! put your SCC regcode here!!!!!
	}
	os.Chdir(*home)
	if *addrepo != "" {
		utils.AdminOrchCmd(utils.CAASPOutReturner(*openstack, *home, caaspDir), "addrepo", *addrepo)
	}
	os.Chdir(*home)
	if *sysupd {
		utils.AdminOrchCmd(utils.CAASPOutReturner(*openstack, *home, caaspDir), "update", "")
	}
	os.Chdir(*home)
	if *packupd != "" {
		utils.AdminOrchCmd(utils.CAASPOutReturner(*openstack, *home, caaspDir), "packupdate", *packupd)
	}
	os.Chdir(*home)
	if *new {
		utils.AdminOrchCmd(utils.CAASPOutReturner(*openstack, *home, caaspDir), "new", utils.RegCode) // <<----------- unexistent variable! put your SCC regcode here!!!!!
	}
	os.Chdir(*home)
	if *uiupd {
		a := utils.CAASPOutReturner(*openstack, *home, caaspDir)
		velumURL := fmt.Sprintf("https://%s.nip.io", a.IPAdminExt.Value)
		fmt.Fprintf(os.Stdout, "Velum warm up time: %2.2f Seconds\n", utils.CheckVelumUp(velumURL))
		utils.VelumUpdater(a)
	}
}
