package utils

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/sclevine/agouti"
)

const (
	domain = "nip.io"
	user   = "test@test.com"
	passwd = "123456789"
)

func VelumUpdater(nodes *CAASPOut) {
	t := time.Now()
	hosts := len(nodes.IPMastersExt.Value) + len(nodes.IPWorkersExt.Value)
	driver := agouti.ChromeDriver(
		agouti.ChromeOptions("args", []string{"--no-sandbox"}), //[]string{"--headless", "--disable-gpu", "--no-sandbox"}
	)
	if err := driver.Start(); err != nil {
		log.Fatal(err)
	}
	page, err := driver.NewPage(agouti.Browser("chrome"))
	if err != nil {
		log.Fatal(err)
	}
	page.Session().SetImplicitWait(3000)
	//---------------Visiting directly the "signup" page of velum to create a user
	if err := page.Navigate(fmt.Sprintf("https://%v.%s/users/sign_up", nodes.IPAdminExt.Value, domain)); err != nil {
		log.Fatal(err)
	}

	//---------------Logging in
	time.Sleep(2 * time.Second)
	if err := page.Find("#user_email").Fill(user); err != nil {
		log.Fatal(err)
	}
	if err := page.Find("#user_password").Fill(passwd); err != nil {
		log.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	//------------------Login button
	if err := page.Find(".btn-block").Click(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(4 * time.Second)

	//-----------------UPDATE ADMIN NODE
	if err := page.Find(".update-admin-btn").Click(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(2 * time.Second)
	//---------------Reboot to update
	if err := page.Find(".reboot-update-btn").Click(); err != nil {
		log.Fatal(err)
	}

	for {
		out, er := AdminOrchCmd(nodes, "refresh", "")
		if !strings.Contains(er, "nil") {
			fmt.Printf("%s\n%s\n", out, er)
		} else {
			fmt.Printf("%s\n", out)
		}
		time.Sleep(2 * time.Second)
		if err := page.Find(".reboot-update-btn"); err != nil {
			if err := page.Find("#update-all-nodes").Click(); err == nil {
				break
			}
		}
		time.Sleep(5 * time.Second)
		go func() {
			log.Printf("Updating Admin for %2.2f seconds now...", time.Since(t).Seconds())
		}()
	}

	for {
		page.Session().SetImplicitWait(30 * 1000)
		selection := page.All(".fa-check-circle-o, .fa-times-circle")
		count, _ := selection.Count()
		if count >= hosts {
			break
		} else {
			selection := page.All(".fa-arrow-circle-up")
			count, _ := selection.Count()
			if count > 0 {
				err = page.Find("#retry-cluster-upgrade").Click()
				if err != nil {
					fmt.Fprintf(os.Stdout, "Cannot find Retry Cluster Update Button:%s\n", err)
				}
			}
		}
		go func() {
			log.Printf("Updating cluster for %2.2f seconds now", time.Since(t).Seconds())
		}()
		time.Sleep(20 * time.Second)
	}
	page.CloseWindow()
}

func CreateAcc(nodes *CAASPOut) {
	go func() {
		log.Println("Bootstrapping the cluster")
	}()
	//driver := agouti.ChromeDriver()
	driver := agouti.ChromeDriver(
	//agouti.ChromeOptions("args", []string{"--no-sandbox"}), //[]string{"--headless", "--disable-gpu", "--no-sandbox"}
	)
	if err := driver.Start(); err != nil {
		log.Fatal(err)
	}
	page, err := driver.NewPage(agouti.Browser("chrome"))
	if err != nil {
		log.Fatal(err)
	}
	page.Session().SetImplicitWait(3000)
	//---------------Visiting directly the "signup" page of velum to create a user
	if err := page.Navigate(fmt.Sprintf("https://%v.%s/users/sign_up", nodes.IPAdminExt.Value, domain)); err != nil {
		log.Fatal(err)
	}
	//---------------Filling in user data
	time.Sleep(10 * time.Second)
	if err := page.Find("#user_email").Fill(user); err != nil {
		fmt.Fprintf(os.Stdout, "Createacc->Fill error: %s\n", err)
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
	page.CloseWindow()
	time.Sleep(3 * time.Second)
}

func FirstSetup(nodes *CAASPOut) {
	hosts := len(nodes.IPMastersExt.Value) + len(nodes.IPWorkersExt.Value)
	t := time.Now()
	go func() {
		log.Println("Adding nodes to the cluster...")
	}()
	driver := agouti.ChromeDriver(
		agouti.ChromeOptions("args", []string{"--headless", "--disable-gpu", "--no-sandbox"}), // "--disable-gpu"   "--headless"
	)
	if err := driver.Start(); err != nil {
		log.Fatal(err)
	}
	page, err := driver.NewPage(agouti.Browser("chrome"))
	if err != nil {
		log.Fatal(err)
	}
	if err := page.Navigate(fmt.Sprintf("https://%v", nodes.IPAdminExt.Value)); err != nil {
		log.Fatal(err)
	}
	//---------------Logging in
	if err := page.Find("#user_email").Fill(user); err != nil {
		log.Fatal(err)
	}
	if err := page.Find("#user_password").Fill(passwd); err != nil {
		log.Fatal(err)
	}
	time.Sleep(2 * time.Second)

	//------------------Login button
	if err := page.Find(".btn-block").Click(); err != nil {
		log.Fatal(err)
	}
	page.Session().SetImplicitWait(5 * 1000)
	time.Sleep(3 * time.Second)
	//------------Filling in the Admin IP on the dashboard
	if err := page.Find("#settings_dashboard").Fill(nodes.IPAdminInt.Value); err != nil {
		log.Fatal(err)
	}
	go func() {
		log.Printf("Fill Admin internal ip: %s; selected Tiller\n", nodes.IPAdminInt.Value)
	}()
	//------Checking the Tiller
	if err := page.Find("#settings_tiller").Click(); err != nil {
		log.Fatal(err)
	}
	//--------Clicking Next
	if err := page.Find(".steps-container .pull-right").Click(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(3 * time.Second)
	//-------Clicking Next
	if err := page.Find(".steps-container .btn-primary").Click(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(3 * time.Second)

	//--------------counting so all newly added nodes will be pending to be accepted
	itercounter := 0
	for {
		time.Sleep(20 * time.Second)
		if itercounter >= 10 {
			break
			log.Printf("Not all nodes are visible to the admin. Something is wrong with the cluster/cluster.tf...")
		}
		selection := page.All(".pending-accept-link")
		count, _ := selection.Count()
		if count >= Cluster.Diff {
			break
		}
		go func() {
			log.Printf("Waiting for new nodes to be accepted for %2.2f seconds now", time.Since(t).Seconds())
		}()
		log.Printf("All new nodes accepted at %2.2f seconds!", time.Since(t).Seconds())
		itercounter++
		time.Sleep(2 * time.Second)
	}

	//----------------Accept All Nodes Button---------------------------------
	if err := page.Find(".pending-nodes-container .pull-right").Click(); err != nil {
		log.Fatal(err)
	}
	time.Sleep(3 * time.Second)

	time.Sleep(time.Duration(hosts) * time.Second * 20)

	//------distributing roles to nodes based on their names
	for i := 2; i < hosts+2; i++ {
		path := fmt.Sprintf(".minion_%d .minion-hostname", i)
		text, err := page.Find(path).Text()
		if err != nil {
			log.Printf("Node %s already registered or unexistent...\n", path)
		}
		if strings.Contains(text, "master") {
			a := fmt.Sprintf(".minion_%d .master-btn", i)
			err = page.Find(a).Click()
			if err != nil {
				log.Printf("the minion %d couldn't be found:%s\n", i, err)
			}
		} else {
			a := fmt.Sprintf(".minion_%d .worker-btn", i)
			err = page.Find(a).Click()
			if err != nil {
				log.Printf("the minion %d couldn't be found:%s\n", i, err)
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
	time.Sleep(5 * time.Second)
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
	//-------Waiting for all nodes to have a green check status "ready"
	for {
		page.Session().SetImplicitWait(30 * 1000)
		selection := page.All(".fa-check-circle-o, .fa-times-circle")
		count, _ := selection.Count()
		if count >= hosts {
			log.Printf("The Cluster is properly set up.")
			break
		}
		go func() {
			log.Printf("Bootstrapping cluster for %2.2f seconds now", time.Since(t).Seconds())
		}()
		time.Sleep(10 * time.Second)
	}
	page.CloseWindow()
	time.Sleep(3 * time.Second)
}

// InstallUI handles Velum interactions
func InstallUI(nodes *CAASPOut, Cluster *CaaSPCluster) {
	go func() {
		log.Println("Adding nodes to the cluster...")
	}()
	driver := agouti.ChromeDriver(
		agouti.ChromeOptions("args", []string{"--no-sandbox"}), // "--disable-gpu"   "--headless"
	)
	if err := driver.Start(); err != nil {
		log.Fatal(err)
	}
	page, err := driver.NewPage(agouti.Browser("chrome"))
	if err != nil {
		log.Fatal(err)
	}
	t := time.Now()
	hosts := len(nodes.IPMastersExt.Value) + len(nodes.IPWorkersExt.Value) + Cluster.Diff
	if err := page.Navigate(fmt.Sprintf("https://%v", nodes.IPAdminExt.Value)); err != nil {
		log.Fatal(err)
	}

	//--------------------Logging in
	if err := page.Find("#user_email").Fill(user); err != nil {
		log.Fatal(err)
	}
	if err := page.Find("#user_password").Fill(passwd); err != nil {
		log.Fatal(err)
	}

	if err := page.Find(".btn-block").Click(); err != nil {
		log.Fatal(err)
	}

	//-------------Counting the number of pending nodes (boot+autoyast might take a while) to accept
	for {
		time.Sleep(10 * time.Second)
		selection := page.All(".pending-accept-link")
		count, _ := selection.Count()
		if count >= Cluster.Diff {
			break
		}
		go func() {
			log.Printf("Bootstrapping new nodes to the cluster for %2.2f seconds now", time.Since(t).Seconds())
		}()
	}
	if err := page.Find(".panel-heading #accept-all").Click(); err != nil {
		log.Fatal(err)
	}

	time.Sleep(60 * time.Second)

	for {
		if err := page.Find(".assign-nodes-link").Click(); err != nil {
			log.Println(err)
		} else {
			break
		}
		log.Printf("Bootstrapping new nodes to the cluster for %2.2f seconds now", time.Since(t).Seconds())
	}
	time.Sleep(3 * time.Second)

	//-----------Adding minions based on their name (master, if not -> worker)
	fmt.Println(hosts)
	for i := 2; i < hosts+2; i++ {
		path := fmt.Sprintf(".minion_%d .minion-hostname", i)
		text, err := page.Find(path).Text()
		if err != nil {
			log.Printf("Node %s already registered or unexistent...\n", path)
		} else {
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
		time.Sleep(3 * time.Second)
	}

	time.Sleep(1 * time.Second)
	if err := page.Find(".add-nodes-btn").Click(); err != nil {
		log.Fatal(err)
	}

	//-------waiting for all nodes to have a green checkmark "ready" (e.g. to be bootstrapped)
	for {
		page.Session().SetImplicitWait(30 * 1000)
		selection := page.All(".fa-check-circle-o, .fa-times-circle")
		count, _ := selection.Count()
		if count == hosts {
			break
		}
		go func() {
			log.Printf("Bootstrapping the added node(s) %2.2f seconds now", time.Since(t).Seconds())
		}()
		time.Sleep(10 * time.Second)
	}

	go func() {
		log.Printf("Roles assigned\tMasters: %v\tWorkers: %v\n", nodes.IPMastersExt, nodes.IPWorkersExt)
	}()
	if err := driver.Stop(); err != nil {
		log.Fatal(err)
	}
}

// Download downloads the kubeconfig file
func Download(url string) error {
	http.DefaultTransport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	out, err := os.Create("kubeconfig")
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, resp.Body)
	return err
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
