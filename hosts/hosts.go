package hosts

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/Sirupsen/logrus"

	"golang.org/x/crypto/ssh"
)

type Hosts struct {
	Hosts []Host `yaml:"hosts"`
}

type Host struct {
	IP       string   `yaml:"ip"`
	Role     []string `yaml:"role"`
	Hostname string   `yaml:"hostname"`
	User     string   `yaml:"user"`
	Sudo     bool     `yaml:"sudo"`
}

type SignerContainer struct {
	signers []ssh.Signer
}

func getKey(keyname string) (signer ssh.Signer, err error) {
	fp, err := os.Open(keyname)
	if err != nil {
		return
	}
	defer fp.Close()

	buf, _ := ioutil.ReadAll(fp)
	signer, _ = ssh.ParsePrivateKey(buf)
	return
}

func getSSHClientConfig(host Host) (ssh.ClientConfig, error) {
	key := os.Getenv("HOME") + "/.ssh/id_rsa"

	signer, err := getKey(key)
	if err != nil {
		return ssh.ClientConfig{}, err
	}
	config := &ssh.ClientConfig{
		User: host.User,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(signer),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}
	return *config, nil
}

func (h Host) RunSSHCommand(cmd string) (string, error) {
	if h.Sudo {
		cmd = cmdWithSudo(cmd)
	}
	logrus.Debugf("Running SSH command: %s", cmd)
	config, err := getSSHClientConfig(h)
	if err != nil {
		return "", fmt.Errorf("Can't construct SSH config: %v", err)
	}
	conn, err := ssh.Dial("tcp", fmt.Sprintf("%s:%s", h.IP, "22"), &config)
	if err != nil {
		return "", fmt.Errorf("Can't connect to host [%s]: %v", h.Hostname, err)
	}
	session, err := conn.NewSession()
	if err != nil {
		return "", fmt.Errorf("Can't open SSH session with host [%s]: %v", h.Hostname, err)
	}
	defer session.Close()

	var stdoutBuf bytes.Buffer
	var stderrBuf bytes.Buffer
	session.Stdout = &stdoutBuf
	session.Stderr = &stderrBuf
	session.Run(cmd)

	if len(stderrBuf.String()) != 0 {
		return "", fmt.Errorf("Failed to run SSH command on [%s]: %v", h.Hostname, stderrBuf.String())
	}
	return stdoutBuf.String(), nil
}

func DivideHosts(hosts []Host) ([]Host, []Host, []Host) {
	etcdHosts := []Host{}
	cpHosts := []Host{}
	workerHosts := []Host{}
	for _, host := range hosts {
		for _, role := range host.Role {
			if role == "etcd" {
				etcdHosts = append(etcdHosts, host)
			}
			if role == "controlplane" {
				cpHosts = append(cpHosts, host)
			}
			if role == "worker" {
				workerHosts = append(workerHosts, host)
			}
		}
	}
	return etcdHosts, cpHosts, workerHosts
}

func cmdWithSudo(cmd string) string {
	return "sudo " + cmd
}
