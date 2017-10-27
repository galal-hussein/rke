package services

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/rke/hosts"
)

type Kubeproxy struct {
	Version string `yaml:"version"`
	Image   string `yaml:"image"`
}

func runKubeproxy(host hosts.Host, masterHost hosts.Host, kubeproxyService Kubeproxy) error {
	isRunning, err := isKubeproxyRunning(host, masterHost, kubeproxyService)
	if err != nil {
		return err
	}
	if isRunning {
		logrus.Infof("Kubeproxy is already running on host [%s]", host.Hostname)
		return nil
	}
	err = runKubeproxyContainer(host, masterHost, kubeproxyService)
	if err != nil {
		return err
	}
	return nil
}

func isKubeproxyRunning(host hosts.Host, masterHost hosts.Host, kubeproxyService Kubeproxy) (bool, error) {
	cmd := "docker ps -a | grep " + KubeproxyContainerName + " | wc -l"
	logrus.Infof("Check if Kubeproxy is running on host [%s]", host.Hostname)
	stdout, stderr := host.RunSSHCommand(cmd)
	isRunning := strings.TrimSuffix(stdout, "\n")
	if stderr != nil {
		return false, fmt.Errorf("Failed to check if Kubeproxy is running: %v", stderr)
	}
	if isRunning == "1" {
		return true, nil
	}
	return false, nil
}

func runKubeproxyContainer(host hosts.Host, masterHost hosts.Host, kubeproxyService Kubeproxy) error {
	err := pullKubeproxyImage(host, kubeproxyService)
	if err != nil {
		return err
	}
	cmd := constructKubeproxyCommand(host, masterHost, kubeproxyService)
	_, stderr := host.RunSSHCommand(cmd)

	if stderr != nil {
		return fmt.Errorf("Failed to run Kubeproxy container on host [%s]: %v", host.Hostname, stderr)
	}
	logrus.Infof("Successfully ran Kubeproxy container on host [%s]", host.Hostname)
	return nil
}

func pullKubeproxyImage(host hosts.Host, kubeproxyService Kubeproxy) error {
	pullCmd := "docker pull " + kubeproxyService.Image + ":" + kubeproxyService.Version
	stdout, stderr := host.RunSSHCommand(pullCmd)
	if stderr != nil {
		return fmt.Errorf("Failed to pull Kubeproxy image on host [%s]: %v", host.Hostname, stderr)
	}
	logrus.Debugf("Pulling Image on host [%s]: %v", host.Hostname, stdout)
	logrus.Infof("Successfully pulled Kubeproxy image on host [%s]", host.Hostname)
	return nil
}

func constructKubeproxyCommand(host hosts.Host, masterHost hosts.Host, kubeproxyService Kubeproxy) string {
	return `docker run -d \
			  --net=host \
			  --privileged \
			  --name=` + KubeproxyContainerName + ` \
			  --restart=on-failure:5 \
			  ` + kubeproxyService.Image + `:` + kubeproxyService.Version + ` \
			  ./hyperkube proxy \
				--v=2 \
				--healthz-bind-address=0.0.0.0 \
				--master=http://` + masterHost.IP + `:8080/ `
}
