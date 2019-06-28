package utils

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"
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

func CheckRebootNeeded(IP string, a *CAASPOut, homedir string, caaspdir string, list map[string]SaltCluster) {
	temp1 := ""
	temp2 := false
	out, err := a.SSHCommand(IP, homedir, caaspdir, "hostname; cat /etc/salt/grains").CombinedOutput()
	if err != nil {
		fmt.Fprintf(os.Stdout, "running SSHCommand when debugging salt crashed... %s", err)
	}
	tmp := strings.Split(fmt.Sprintf("%s", string(out)), "\n")
	for i := 0; i < len(tmp); i++ {
		if strings.Contains(tmp[i], "caasp-") {
			temp1 = tmp[i]
		}
		if strings.Contains(tmp[i], "reboot_needed: true") {
			temp2 = true
		}
	}

	for key, value := range list {
		if strings.Contains(value.Name, strings.Replace(temp1, " ", "", 1)) {
			value.IP = IP
			value.RebootNeeded = temp2
			list[key] = value
		}
	}

}

func CheckSaltMinions(homedir string, caaspdir string) { //nodes *CAASPOut) {
	a := CAASPOutReturner("openstack.json", homedir, caaspdir)
	b := make(map[string]SaltCluster)
	var saltcluster SaltCluster

	out, err := AdminOrchCmd(homedir, caaspdir, a, "command", "hostname")
	if err != "%!s(<nil>)" {
		fmt.Fprintf(os.Stdout, "running mkcaasp command failed: %v\n", err)
	}
	temp := strings.Split(out, "\n")
	if strings.Contains(temp[0], "arning:") {
		temp[0] = ""
	}
	for i := 0; i < len(temp)-1; i++ {
		if strings.Contains(temp[i], ":") {
			saltcluster.Name = strings.Replace(temp[i+1], " ", "", 1)
			b[strings.Replace(temp[i], " ", "", 1)] = saltcluster
		}
	}

	var IPslicer []string
	IPslicer = append(IPslicer, a.IPAdminExt.Value)
	for _, k := range a.IPMastersExt.Value {
		IPslicer = append(IPslicer, k)
	}
	for _, k := range a.IPWorkersExt.Value {
		IPslicer = append(IPslicer, k)
	}

	for _, k := range IPslicer {
		CheckRebootNeeded(k, a, homedir, caaspdir, b)
	}

	for key, value := range b {
		if value.RebootNeeded == false {
			log.Printf("node %s (%s) needs to be updated. Updating...", key, value.Name)
			time.Sleep(1 * time.Second)
			out := value.SSHCmd(value.IP, homedir, caaspdir, "zypper -n --gpg-auto-import-keys ref")
			NiceBufRunner(out)
			out = value.SSHCmd(value.IP, homedir, caaspdir, "/usr/sbin/transactional-update cleanup dup reboot")
			NiceBufRunner(out)
		}
	}

	for _, k := range IPslicer {
		CheckRebootNeeded(k, a, homedir, caaspdir, b)
	}

	//---------------------final checker
	for key, value := range b {
		fmt.Printf("b[%s] = %v\n", key, value)
	}
}
