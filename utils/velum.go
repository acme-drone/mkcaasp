package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/sclevine/agouti"
)

const (
	domain = "nip.io"
	user   = "test@test.com"
	passwd = "123456789"
)

// InstallUI handles Velum interactions
func InstallUI(nodes *CAASPOut) {
	hosts := len(nodes.IPMastersExt.Value) + len(nodes.IPWorkersExt.Value)
	go func() {
		log.Println("Bootstrapping the cluster")
	}()
	// Uncomment for disabling headless
	driver := agouti.ChromeDriver()
	//driver := agouti.ChromeDriver(
	//	agouti.ChromeOptions("args", []string{"--headless", "--disable-gpu", "--no-sandbox"}),
	//)
	if err := driver.Start(); err != nil {
		log.Fatal(err)
	}
	page, err := driver.NewPage(agouti.Browser("chrome"))
	if err != nil {
		log.Fatal(err)
	}
	t := time.Now()
	page.Session().SetImplicitWait(1000)
	if err := page.Navigate(fmt.Sprintf("https://%v.%s/users/sign_up", nodes.IPAdminExt.Value, domain)); err != nil {
		log.Fatal(err)
	}
	if err := page.Find("#user_email").Fill(user); err != nil {
		log.Fatal(err)
	}
	if err := page.Find("#user_password").Fill(passwd); err != nil {
		log.Fatal(err)
	}
	if err := page.Find("#user_password_confirmation").Fill(passwd); err != nil {
		log.Fatal(err)
	}
	if err := page.Find(".btn-block").Click(); err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Printf("Admin user created %s %s\n", user, passwd)
	}()
	page.Session().SetImplicitWait(3000)
	if err := page.Find("#settings_dashboard").Fill(nodes.IPAdminInt.Value); err != nil {
		log.Fatal(err)
	}
	if err := page.Find("#settings_tiller").Click(); err != nil {
		log.Fatal(err)
	}
	if err := page.Find(".steps-container .pull-right").Click(); err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Printf("Fill Admin internal ip: %s; selected Tiller\n", nodes.IPAdminInt.Value)
	}()
	if err := page.Find(".btn-primary").Click(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Duration(hosts) * time.Second * 20)
	if err := page.Find(".panel-heading #accept-all").Click(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(time.Duration(hosts) * time.Second * 20)
	for i := 2; i < hosts+2; i++ {
		path := fmt.Sprintf(".minion_%d .minion-hostname", i)
		text, err := page.Find(path).Text()
		if err != nil {
			log.Fatal(err)
		}
		if strings.Contains(text, "master") {
			a := fmt.Sprintf(".minion_%d .master-btn", i)
			err = page.Find(a).Click()
			if err != nil {
				log.Fatal(err)
			}
		} else {
			a := fmt.Sprintf(".minion_%d .worker-btn", i)
			err = page.Find(a).Click()
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	go func() {
		log.Printf("Roles assigned\tMasters: %v\tWorkers: %v\n", nodes.IPMastersExt, nodes.IPWorkersExt)
	}()
	page.Session().SetImplicitWait(3 * 1000)
	err = page.Find(".steps-container #set-roles").Click()
	if err != nil {
		log.Fatal(err)
	}
	page.Session().SetImplicitWait(3 * 1000)
	apiserver := fmt.Sprintf("%s.%s", nodes.IPMastersExt.Value[0], domain)
	err = page.Find("#settings_apiserver").Fill(apiserver)
	if err != nil {
		log.Fatal(err)
	}
	apiserver = fmt.Sprintf("%s.%s", nodes.IPAdminExt.Value, domain)
	err = page.Find("#settings_dashboard_external_fqdn").Fill(apiserver)
	if err != nil {
		log.Fatal(err)
	}
	err = page.Find("#bootstrap").Click()
	if err != nil {
		log.Fatal(err)
	}
	page.Session().SetImplicitWait(5 * 1000)
	for {
		page.Session().SetImplicitWait(30 * 1000)
		selection := page.All(".fa-check-circle-o, .fa-times-circle")
		count, _ := selection.Count()
		if count == hosts {
			break
		}
		go func() {
			log.Printf("Bootstrapping cluster for %2.2f seconds now", time.Since(t).Seconds())
		}()
	}
	// Check if bootstrap was successfull
	selection := page.All(".fa-check-circle-o")
	count, _ := selection.Count()
	if count == hosts {
		log.Printf("Bootstrap Successful, bootstrap time: %2.2f minutes\n", time.Since(t).Minutes())
	} else {
		log.Fatal("Bootstrap failed")
	}
	if err := driver.Stop(); err != nil {
		log.Fatal(err)
	}
}

// CheckVelumUp returns Velum worm up time in Seconds
func CheckVelumUp(page string) float64 {
	t := time.Now()
	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 10*time.Minute)
	defer cancel()
	req, err := http.NewRequest(http.MethodGet, page, nil)
	if err != nil {
		log.Fatal(err)
	}
	req = req.WithContext(ctx)
	tr := &http.Transport{
		TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
	}
	var DefaultClient = &http.Client{Transport: tr}
	var resp *http.Response
	for {
		resp, err = DefaultClient.Do(req)
		if err != nil || resp.StatusCode != http.StatusOK {
			time.Sleep(10 * time.Second)
			continue
		} else {
			break
		}
	}
	defer resp.Body.Close()
	return time.Since(t).Seconds()
}
