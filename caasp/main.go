package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"os"

	"mkcaasp/utils"
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

			 run the utility: caasp -repo $HOME/automation -createcaasp -caaspuiinst -createses -action apply -auth openstack.json
			 `
)

var (
	openstack     = flag.String("auth", "openstack.json", "name of the json file containing openstack variables")
	action        = flag.String("action", "apply", "terraform action to run, example: apply, destroy")
	caasp         = flag.Bool("createcaasp", false, "enables/disables caasp terraform openstack setup")
	ses           = flag.Bool("createses", false, "enables/disables ses terraform openstack setup")
	howto         = flag.Bool("usage", false, "prints usage information")
	caasptfoutput = flag.Bool("caasptfoutput", false, "loads in memory caasp terraform ouput json")
	sestfoutput   = flag.Bool("sestfoutput", false, "loads in memory ses terraform ouput json")
	caaspUIInst   = flag.Bool("caaspuiinst", false, "Configures caasp using Velum UI")

	home = flag.String("repo", "automation", "kubic automation repo location")
)

const (
	caaspDir = "caasp-openstack-terraform"
	sesDir   = "ses-openstack-terraform"
	output   = "terraform output -json"
)

func main() {
	flag.Parse()
	if *howto {
		fmt.Fprintf(os.Stdout, "%v\n", howtouse)
		os.Exit(0)
	}
	os.Chdir(*home)
	if *caasp {
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
		utils.InstallUI(&a)
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
}
