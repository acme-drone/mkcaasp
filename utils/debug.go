package utils

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
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
