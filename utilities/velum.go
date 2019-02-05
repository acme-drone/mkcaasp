package utilities

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/sclevine/agouti"
)

// InstallUI handles Velum interactions
func InstallUI(nodes *CAASPTFOutput) {
	go func() {
		log.Println("Bootstrapping the cluster")
	}()
	driver := agouti.ChromeDriver()
	if err := driver.Start(); err != nil {
		log.Fatal("Failed to start ChromeDriver:", err)
	}
	page, err := driver.NewPage(agouti.Browser("chrome"))
	if err != nil {
		log.Fatal("Failed to open page:", err)
	}
	t := time.Now()
	if err := page.Navigate(fmt.Sprintf("https://%v.nip.io/users/sign_up", nodes.CAASPIPAdminExt.Value)); err != nil {
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
	go func() {
		log.Printf("Admin user created test@test.com/123456789\n")
	}()
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
	go func() {
		log.Printf("Fill Admin internal ip: %s; selected Tiller\n", nodes.CAASPIPAdminInt.Value)
	}()
	if err := page.FindByClass("btn-primary").Click(); err != nil {
		log.Fatal("Setup - Next failed:", err)
	}
	machines := len(nodes.IPMastersExt.Value) + len(nodes.IPWorkersExt.Value)
	time.Sleep(time.Duration(machines) * time.Minute)
	if err := page.FindByID("accept-all").Click(); err != nil {
		log.Fatal("Accepting nodes failed:", err)
	}
	time.Sleep(time.Duration(machines) * 20 * time.Second)
	for i := 2; i < machines+2; i++ {
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
	go func() {
		log.Printf("Node discovery finished; roles assigned\n\t\tMasters: %v\n\t\tWorkers: %v\n", nodes.IPMastersExt, nodes.IPWorkersExt)
	}()
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
	time.Sleep(5 * time.Second)
	for {
		time.Sleep(30 * time.Second)
		selection := page.All(".fa-check-circle-o, .fa-times-circle")
		count, _ := selection.Count()
		if count == machines {
			break
		}
		go func() {
			log.Printf("Bootstrapping cluster for %v seconds now", time.Since(t).Seconds())
		}()
	}
	// Check if bootstrap was successfull
	selection := page.All(".fa-check-circle-o")
	count, _ := selection.Count()
	if count == machines {
		log.Printf("Bootstrap Successful, bootstrap time: %v\n", time.Since(t).Minutes())
	} else {
		log.Fatal("Bootstrap failed")
	}
	if err := driver.Stop(); err != nil {
		log.Fatal("Failed to strop ChromeDriver:", err)
	}
}
