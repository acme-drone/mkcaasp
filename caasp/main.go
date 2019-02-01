package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"

	"mkcaasp/utilities"
)

const (
	command  = "terraform %s -var auth_url=$OS_AUTH_URL -var domain_name=$OS_USER_DOMAIN_NAME -var region_name=$OS_REGION_NAME -var project_name=$OS_PROJECT_NAME -var user_name=$OS_USERNAME -var password=$OS_PASSWORD -var-file=terraform.tfvars -auto-approve"
	howtouse = `
			 Make sure you have terraform installed and in $PATH
			 git clone https://github.com/kubic-project/automation.git
			 
			 cd automation
			 
			 run terraform init in the directories you want to use, for example in caasp-openstack-terraform and/or ses-openstack-terraform

			 put openstack.json in the directories you want to use, for example in caasp-openstack-terraform and/or ses-openstack-terraform
			 
			 openstack.json (should reside in caasp-openstack-terraform and ses-openstack-terraform folders) should look like this:
			 {
				"OSAuthURL":"https://smtg:5000/v3",
				"OSRegionName":"Region",
				"OSProjectName":"caasp",
				"OSUserDomainName":"users",
				"OSIdentityAPIVersion":"3",
				"OSInterface":"public",
				"OSUsername":"user",
				"OSPassword":"pass",
				"OSProjectID":"00000000000000000000000000"
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

func main() {
	flag.Parse()
	if *howto {
		fmt.Fprintf(os.Stdout, "%v\n", howtouse)
		os.Exit(0)
	}
	os.Chdir(*home)
	if *caasp {
		utilities.TfInit("caasp-openstack-terraform")
		utilities.CmdRun("caasp-openstack-terraform", *openstack, fmt.Sprintf(command, *action))
	}
	os.Chdir(*home)
	if *ses {
		utilities.TfInit("ses-openstack-terraform")
		utilities.CmdRun("ses-openstack-terraform", *openstack, fmt.Sprintf(command, *action))
	}
	os.Chdir(*home)
	if *caasptfoutput {
		utilities.CmdRun("caasp-openstack-terraform", *openstack, "terraform output -json")
	}
	os.Chdir(*home)
	if *sestfoutput {
		out, _ := utilities.CmdRun("ses-openstack-terraform", *openstack, "terraform output -json")
		a := utilities.SESTFOutput{}
		err := json.Unmarshal([]byte(out), &a)
		if err != nil {
			fmt.Printf("%s\n", err)
		}
		s := a.K8SStorageClass.Value
		fmt.Println(a.SESIPAdminExt.Value, a.SESIPAdminInt.Value, a.SESIPMonsExt.Value, a.SESIPMonsExt.Value, a.SESIPOsdsInt.Value, "\n", a.K8SCephSecret.Value, "\n", fmt.Sprintf("%s", s[0]))

	}
	os.Chdir(*home)
	if *caaspUIInst {
		out, _ := utilities.CmdRun("caasp-openstack-terraform", *openstack, "terraform output -json")
		a := utilities.CAASPTFOutput{}
		err := json.Unmarshal([]byte(out), &a)
		if err != nil {
			fmt.Printf("%s\n", err)
		}
		// wait for velum initialization
		velumURL := fmt.Sprintf("https://%s.nip.io", a.CAASPIPAdminExt.Value)
		fmt.Fprintf(os.Stdout, "Velum warm up time: %2.2f Seconds\n", utilities.CheckVelumUp(velumURL))
		utilities.InstallUI(&a)
		utilities.CmdRun("caasp-openstack-terraform", *openstack, "terraform output -json")
		fmt.Fprintf(os.Stdout, "%v\n", velumURL)
	}
}
