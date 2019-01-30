package utilities

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/sclevine/agouti"
)

// InstallUI handles Velum interactions
func InstallUI(nodes *CAASPTFOutput) {
	driver := agouti.ChromeDriver()
	if err := driver.Start(); err != nil {
		log.Fatal("Failed to start ChromeDriver:", err)
	}
	page, err := driver.NewPage(agouti.Browser("chrome"))
	if err != nil {
		log.Fatal("Failed to open page:", err)
	}
	fmt.Fprintf(os.Stdout, "Creating admin user")
	fmt.Fprintf(os.Stdout, "Creating admin user")
	if err := page.Navigate(fmt.Sprintf("https://%v/users/sign_up", nodes.CAASPIPAdminExt.Value)); err != nil {
		log.Fatal("Failed to navigate:", err)
	}
	if err := page.FindByID("user_email").Fill("test@test.com"); err != nil {
		log.Fatal("Filling user failed:", err)
	}
	if err := page.FindByID("user_password").Fill("123456789"); err != nil {
		log.Fatal("Filling password failed:", err)
	}
	if err := page.FindByID("user_password_confirmation").Fill("123456789"); err != nil {
		log.Fatal("Filling password confirmation failed:", err)
	}
	if err := page.FindByClass("btn-block").Click(); err != nil {
		log.Fatal("Creating Admin Failed:", err)
	}
	fmt.Fprintf(os.Stdout, "Setting dashboard internal ip, enabling tiller installation")
	time.Sleep(4 * time.Second)
	if err := page.FindByID("settings_dashboard").Fill(nodes.CAASPIPAdminInt.Value); err != nil {
		log.Fatal("Failed inserting Internal Dashboard Location:", err)
	}
	if err := page.FindByID("settings_tiller").Click(); err != nil {
		log.Fatal("Selecting tiller failed:", err)
	}
	if err := page.FindByName("commit").Click(); err != nil {
		log.Fatal("Setup - Next failed:", err)
	}
	if err := page.FindByClass("btn-primary").Click(); err != nil {
		log.Fatal("Setup - Next failed:", err)
	}
	fmt.Fprintf(os.Stdout, "Accepting Nodes")
	time.Sleep(60 * time.Second)
	if err := page.FindByID("accept-all").Click(); err != nil {
		log.Fatal("Accepting nodes failed:", err)
	}
	time.Sleep(30 * time.Second)
	fmt.Fprintf(os.Stdout, "Configuring roles: masters, workers")
	time.Sleep(4 * time.Second)
	for i := 2; i < len(nodes.IPMastersExt.Value)+len(nodes.IPWorkersExt.Value)+2; i++ {
		path := fmt.Sprintf("//tr[@class='minion_%d']", i)
		text, err := page.FindByXPath(path).Text()
		if err != nil {
			log.Fatal(err)
		}
		if strings.Contains(text, "master") {
			a := fmt.Sprintf("//tr[@class='minion_%d']/td[@class='role-column']//*[contains(@class,'master-btn')]", i)
			err := page.FindByXPath(a).Click()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			a := fmt.Sprintf("//tr[@class='minion_%d']/td[@class='role-column']//*[contains(@class,'worker-btn')]", i)
			err := page.FindByXPath(a).Click()
			if err != nil {
				log.Fatal(err)
			}
		}

	}
	fmt.Fprintf(os.Stdout, "Configuring apiserver, dashboard")
	time.Sleep(4 * time.Second)
	a := fmt.Sprintf("//*[contains(@class,'steps-container')]/*[contains(@id,'set-roles')]")
	err = page.FindByXPath(a).Click()
	if err != nil {
		log.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	apiserver := fmt.Sprintf("%s.nip.io", nodes.IPMastersExt.Value[0])
	err = page.FindByID("settings_apiserver").Fill(apiserver)
	if err != nil {
		log.Fatal(err)
	}
	apiserver = fmt.Sprintf("%s.nip.io", nodes.CAASPIPAdminExt.Value)
	err = page.FindByID("settings_dashboard_external_fqdn").Fill(apiserver)
	if err != nil {
		log.Fatal(err)
	}
	a = fmt.Sprintf("//*[contains(@class,'steps-container')]/*[contains(@id,'bootstrap')]")
	err = page.FindByXPath(a).Click()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Fprintf(os.Stdout, "Bootstrapping cluster")
}
