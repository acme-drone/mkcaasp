package tests

import (
	"log"
	"mkcaasp/utils"
	"strings"
)

func HealthChecks(a *utils.CAASPOut, homedir string, caaspdir string) map[string]string {
	//-----------------CHECKING THE MINION VERSIONS------------------
	log.Println("Checking minion versions...")
	cmdargs := []string{"docker", "exec", "$(docker ps -q --filter name=salt-master)", "salt", "-P", "\"roles:admin|kube-master|kube-minion\"", "cmd.run", "'salt-minion", "--version'"}
	cmdtoexec := a.SSHCommand(a.IPAdminExt.Value, homedir, caaspdir, cmdargs...)
	_, err := utils.NiceBufRunner(cmdtoexec)
	if err != "%!s(<nil>)" && !strings.Contains(err, "list of known hosts") {
		log.Printf("salt cmd.run -checking salt-minion versions failed: %s", err)
	}

	//------------------CHECKING FOR DEAD SERVICES------------------
	log.Println("Checking for dead services...")
	cmdargs = []string{"docker", "exec", "$(docker ps -q --filter name=salt-master)", "salt", "-P", "\"roles:admin|kube-master|kube-minion\"", "cmd.run", "'systemctl list-units --state=dead --all'"}
	cmdtoexec = a.SSHCommand(a.IPAdminExt.Value, homedir, caaspdir, cmdargs...)
	_, err = utils.NiceBufRunner(cmdtoexec)
	if err != "%!s(<nil>)" && !strings.Contains(err, "list of known hosts") {
		log.Printf("salt cmd.run -checking failed services failed: %s", err)
	}

	//------------------CHECKING FOR FAILED SERVICES------------------
	log.Println("Checking for failed services...")
	cmdargs = []string{"docker", "exec", "$(docker ps -q --filter name=salt-master)", "salt", "-P", "\"roles:admin|kube-master|kube-minion\"", "cmd.run", "'systemctl list-units --state=failed --all'"}
	cmdtoexec = a.SSHCommand(a.IPAdminExt.Value, homedir, caaspdir, cmdargs...)
	out, err := utils.NiceBufRunner(cmdtoexec)
	if err != "%!s(<nil>)" && !strings.Contains(err, "list of known hosts") {
		log.Printf("salt cmd.run -checking failed services failed: %s", err)
	}
	var b = make(map[string]string)
	temp := strings.Split(out, "\n")
	for i := 0; i < len(temp); i++ {
		if strings.Contains(temp[i], ":") && i+1 < len(temp) && strings.Contains(temp[i+1], "loaded units") {
			b[strings.Replace(temp[i], ":", "", 10)] = temp[i+1]
		}
	}
	return b
}
