package utils

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"mkcaasp/utils"
	"os"
	"os/exec"
	"strings"
)

type PackageData struct {
	Name         string
	Version      string
	Release      string
	Architecture string
}

func RpmOutputParser(output string) *PackageData {
	var pack PackageData
	temp := strings.Split(output, "\n")
	for _, k := range temp {
		if strings.Contains(k, "Name") {
			pack.Name = strings.Split(k, " ")[len(strings.Split(k, " "))-1]
		}
		if strings.Contains(k, "Version") {
			pack.Version = strings.Split(k, " ")[len(strings.Split(k, " "))-1]
		}
		if strings.Contains(k, "Release") {
			pack.Release = strings.Split(k, " ")[len(strings.Split(k, " "))-1]
		}
		if strings.Contains(k, "Architecture") {
			pack.Architecture = strings.Split(k, " ")[len(strings.Split(k, " "))-1]
		}

	}
	return &pack
}

func execTmpl(name string) (string, error) {
	var t *template.Template
	var packdata PackageData
	buf := &bytes.Buffer{}
	err := t.ExecuteTemplate(buf, name, packdata)
	return buf.String(), err
}

func CheckVersions() {
	var chromedata, chromedriverdata PackageData
	ChromeVers, err := exec.Command("rpm", []string{"-qi", "google-chrome-stable"}...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stdout, "CheckChromiumVersion->Cmd(rpm -qi) error: %s\n", err)
	}
	chromedata = *RpmOutputParser(fmt.Sprintf("%s", string(ChromeVers)))
	ChromeDriverVers, err := exec.Command("chromedriver", "--version").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stdout, "CheckChromiumVersion->Cmd(rpm -qi) error: %s\n", err)
	}
	temp := strings.Split(fmt.Sprintf("%s", string(ChromeDriverVers)), " ")
	chromedriverdata.Name = temp[0]
	chromedriverdata.Version = temp[1]
	chromedriverdata.Release = temp[2]
	chromever := strings.Split(chromedata.Version, ".")
	chromedrivever := strings.Split(chromedriverdata.Version, ".")
	for i := 0; i < len(chromever); i++ {
		if chromever[i] != chromedrivever[i] {
			if i > 1 {
				log.Printf("Driver state, Chrome state are fine...\nChrome Version: %s\nChromiumDriverVersion: %s", chromedata.Version, chromedriverdata.Version)
			} else {
				log.Fatalf("Different Chrome and ChromiumDriver versions. Please update your testenv.\nChrome Version: %s\nChromiumDriverVersion: %s", chromedata.Version, chromedriverdata.Version)
			}
			break
		}
	}
}

func CheckSaltMinions() {
	b := make(map[string]utils.SaltCluster)
	var saltcluster utils.SaltCluster
	args := []string{"-repo", "/home/atighineanu/work/CaaSP_kube/automation", "-auth", "openstack.json", "-cmd", "\"hostname\""}
	out, err := exec.Command("caasp", args...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stdout, "running mkcaasp command failed: %s\n", err)
	}
	temp := strings.Split(fmt.Sprintf("%s", string(out)), "\n")
	if strings.Contains(temp[0], "arning:") {
		temp[0] = ""
	}
	for i := 0; i < len(temp)-1; i++ {
		if strings.Contains(temp[i], ":") {
			saltcluster.Name = strings.Replace(temp[i+1], " ", "", 1)
			b[strings.Replace(temp[i], " ", "", 1)] = saltcluster
		}
		//	if temp[i] != "" && strings.Contains(temp[i], ":") {
		//		fmt.Printf("b[%s]= %s\n", temp[i], b[temp[i]])
		//	}
	}

	args = []string{"-repo", "/home/atighineanu/work/CaaSP_kube/automation", "-auth", "openstack.json", "-cmd", "\"cat /etc/salt/grains\""}
	out, err = exec.Command("caasp", args...).CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stdout, "running mkcaasp command failed: %s\n", err)
	}
	temp = strings.Split(fmt.Sprintf("%s", string(out)), "\n")
	if strings.Contains(temp[0], "arning:") {
		temp[0] = ""
	}
	for i := 1; i < len(temp)-1; i++ {
		if _, ok := b[strings.Replace(temp[i], " ", "", 1)]; ok {
			saltcluster = b[strings.Replace(temp[i], " ", "", 1)]
			for j := 1; j < len(temp)-i; j++ {
				if _, ok := b[strings.Replace(temp[i+j], " ", "", 1)]; ok {
					if strings.Contains(temp[i+j-1], "reboot_needed: true") {
						saltcluster.RebootNeeded = true
						b[strings.Replace(temp[i], " ", "", 1)] = saltcluster
					}
					break
				}
				if temp[i+j] == "" && strings.Contains(temp[i+j-1], "reboot_needed: true") {
					saltcluster.RebootNeeded = true
					b[strings.Replace(temp[i], " ", "", 1)] = saltcluster
					break
				}
			}
			//fmt.Println(b[strings.Replace(temp[i], " ", "", 1)])
		}
		//fmt.Printf("temp[%v] = %s\n", i, temp[i])
		//if temp[i] != "" && strings.Contains(temp[i], ":") && !strings.Contains(temp[i], "reboot") && !strings.Contains(temp[i], "roles") && !strings.Contains(temp[i], "bootstrap") {
		//	fmt.Printf("b[%s]= %s\n", temp[i], b[temp[i]])
		//}
	}

	a := utils.CAASPOutReturner("openstack.json", "/home/atighineanu/work/CaaSP_kube/automation", "caasp-openstack-terraform")
	var IPslicer []string
	IPslicer = append(IPslicer, a.IPAdminExt.Value)
	for _, k := range a.IPMastersExt.Value {
		IPslicer = append(IPslicer, k)
	}
	for _, k := range a.IPWorkersExt.Value {
		IPslicer = append(IPslicer, k)
	}

	for _, k := range IPslicer {
		temp1 := ""
		out, err := a.SSHCommand(k, "hostname").CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stdout, "running SSHCommand when debugging salt crashed... %s", err)
		}
		tmp := strings.Split(fmt.Sprintf("%s", string(out)), "\n")
		for i := 0; i < len(tmp); i++ {
			if strings.Contains(tmp[i], "caasp-") {
				temp1 = tmp[i]
			}
		}
		for key, value := range b {
			if strings.Contains(value.Name, strings.Replace(temp1, " ", "", 1)) {
				value.IP = k
				b[key] = value
			}
		}
	}

	args = []string{"cat", "/etc/salt/grains"}
	for _, value := range b {
		out, err := value.SSHCmd(value.IP, args...).CombinedOutput()
		if err != nil {
			fmt.Fprintf(os.Stdout, "running SSHCommand when debugging salt crashed... %s", err)
		}
		tmp := strings.Split(fmt.Sprintf("%s", string(out)), "\n")
		c := 0
		for _, k := range tmp {
			fmt.Println(k)
			if strings.Contains(k, "reboot_needed") {
				c++
			}
		}
		if c == 0 && value.RebootNeeded == false {
			log.Println("This node needs to be updated!")
			log.Printf("Updating...")

			out := value.SSHCmd(value.IP, "/usr/sbin/transactional-update cleanup dup reboot")
			utils.NiceBufRunner(out)
		}
	}

	//---------------------final checker
	for key, value := range b {
		fmt.Printf("b[%s] = %s\n", key, value)
	}
}
