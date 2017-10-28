package hosts

import (
	"fmt"
	"io"
	"io/ioutil"
	"net"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/client"
	"github.com/urfave/cli"
	"golang.org/x/crypto/ssh"
)

const (
	SocketsDir       = ".sockets"
	remoteSocketAddr = "/var/run/docker.sock"
	TunnelTimeout    = 30
	DockerAPIVersion = "1.24"
)

func (h Host) WaitForSocketTunnel() error {
	exists := make(chan bool, 1)
	go func() {
		for {
			if _, err := os.Stat(h.SocketPath); !os.IsNotExist(err) {
				exists <- true
				break
			}
			time.Sleep(time.Second * 1)
		}
	}()
	select {
	case <-exists:
		return nil
	case <-time.After(time.Second * TunnelTimeout):
		return fmt.Errorf("Timeout waiting for socket to be created for host [%s]", h.Hostname)
	}
}

func (h *Host) SetSocketPath() {
	h.SocketPath = SocketsDir + "/docker-" + h.Hostname + ".sock"
}

func (h *Host) TunnelUp(ctx *cli.Context) error {
	logrus.Infof("[SSH] Start tunnel for host [%s]", h.Hostname)
	// Connection settings
	sshAddr := h.IP + ":22"
	localAddr := h.SocketPath
	remoteAddr := remoteSocketAddr
	// Build SSH client configuration
	cfg, err := makeSSHConfig(h.User)
	if err != nil {
		logrus.Fatalln(err)
	}
	// Establish connection with SSH server
	conn, err := ssh.Dial("tcp", sshAddr, cfg)
	if err != nil {
		logrus.Fatalln(err)
	}
	defer conn.Close()

	// Establish connection with remote server
	remote, err := conn.Dial("unix", remoteAddr)
	if err != nil {
		logrus.Fatalln(err)
	}

	// Start local server to forward traffic to remote connection
	local, err := net.Listen("unix", localAddr)
	if err != nil {
		logrus.Fatalln(err)
	}
	defer local.Close()

	// set Docker client
	socketURL := "unix://" + h.SocketPath
	logrus.Debugf("Connecting to Docker socket [%s]", socketURL)
	h.DClient, err = client.NewClient(socketURL, DockerAPIVersion, nil, nil)
	if err != nil {
		return fmt.Errorf("Can't connect to Docker for host [%s]: %v", h.Hostname, err)
	}
	defer h.DClient.Close()

	// Handle incoming connections
	for {
		client, err := local.Accept()
		if err != nil {
			logrus.Fatalln(err)
		}
		handleClient(client, remote)
	}
}
func (h Host) RemoveSocketFile() {
	logrus.Debugf("Removing socket file: %s", h.SocketPath)
	err := os.Remove(h.SocketPath)
	if err != nil {
		logrus.Errorf("Failed to remove socket file: %v", err)
	}
}

func privateKeyPath() string {
	return os.Getenv("HOME") + "/.ssh/id_rsa"
}
func handleClient(client net.Conn, remote net.Conn) {
	defer client.Close()
	defer remote.Close()
	chDone := make(chan bool)

	// Start remote -> local data transfer
	go func() {
		_, err := io.Copy(client, remote)
		if err != nil {
			logrus.Errorf("error while copy remote->local: %v", err)
		}
		chDone <- true
	}()

	// Start local -> remote data transfer
	go func() {
		_, err := io.Copy(remote, client)
		if err != nil {
			logrus.Errorf("error while copy local->remote: %v", err)
		}
		chDone <- true
	}()

	<-chDone
}

// Get private key for ssh authentication
func parsePrivateKey(keyPath string) (ssh.Signer, error) {
	buff, _ := ioutil.ReadFile(keyPath)
	return ssh.ParsePrivateKey(buff)
}

func makeSSHConfig(user string) (*ssh.ClientConfig, error) {
	key, err := parsePrivateKey(privateKeyPath())
	if err != nil {
		return nil, err
	}

	config := ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(key),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	return &config, nil
}

func CreateSocketDirIfNotExist() {
	logrus.Debugf("Creating sockets directory if not exist")
	if _, err := os.Stat(SocketsDir); os.IsNotExist(err) {
		os.Mkdir(SocketsDir, 0755)
	}
}

func RemoveSocketDir() {
	logrus.Debugf("Removing socket directory: %s", SocketsDir)
	err := os.RemoveAll(SocketsDir)
	if err != nil {
		logrus.Errorf("Failed to remove socket dir: %v", err)
	}
}
