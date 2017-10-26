package services

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/rke/hosts"
)

type KubeController struct {
	Version string `yaml:"version"`
	Image   string `yaml:"image"`
}

func runKubeController(host hosts.Host, kubeControllerService KubeController) error {
	isRunning, err := isKubeControllerRunning(host, kubeControllerService)
	if err != nil {
		return err
	}
	if isRunning {
		logrus.Infof("Kube-Controller is already running on host [%s]", host.Hostname)
		return nil
	}
	err = runKubeControllerContainer(host, kubeControllerService)
	if err != nil {
		return err
	}
	return nil
}

func isKubeControllerRunning(host hosts.Host, kubeControllerService KubeController) (bool, error) {
	cmd := "docker ps -a | grep " + KubeControllerContainerName + " | wc -l"
	logrus.Infof("Check if Kube Controller is running on host [%s]", host.Hostname)
	stdout, stderr := host.RunSSHCommand(cmd)
	isRunning := strings.TrimSuffix(stdout, "\n")
	if stderr != nil {
		return false, fmt.Errorf("Failed to check if Kube Controller is running: %v", stderr)
	}
	if isRunning == "1" {
		return true, nil
	}
	return false, nil
}

func runKubeControllerContainer(host hosts.Host, kubeControllerService KubeController) error {
	err := pullKubeControllerImage(host, kubeControllerService)
	if err != nil {
		return err
	}
	cmd := constructKubeControllerCommand(host, kubeControllerService)
	_, stderr := host.RunSSHCommand(cmd)

	if stderr != nil {
		return fmt.Errorf("Failed to run Kube Controller container on host [%s]: %v", host.Hostname, stderr)
	}
	logrus.Infof("Successfully ran Kube Controller container on host [%s]", host.Hostname)
	return nil
}

func pullKubeControllerImage(host hosts.Host, kubeControllerService KubeController) error {
	pullCmd := "docker pull " + kubeControllerService.Image + ":" + kubeControllerService.Version
	stdout, stderr := host.RunSSHCommand(pullCmd)
	if stderr != nil {
		return fmt.Errorf("Failed to pull Kube Controller image on host [%s]: %v", host.Hostname, stderr)
	}
	logrus.Debugf("Pulling Image on host [%s]: %v", host.Hostname, stdout)
	logrus.Infof("Successfully pulled Kube Controller image on host [%s]", host.Hostname)
	return nil
}

func constructKubeControllerCommand(host hosts.Host, kubeControllerService KubeController) string {
	return `docker run -d --name=` + KubeControllerContainerName + `\
          ` + kubeControllerService.Image + `:` + kubeControllerService.Version + ` /hyperkube controller-manager \
					--address=0.0.0.0 \
					--cloud-provider="" \
					--master=http://` + host.IP + `:8080 \
          --enable-hostpath-provisioner=false \
          --node-monitor-grace-period=40s \
          --node-monitor-period=5s \
          --pod-eviction-timeout=5m0s \
          --v=2 \
          --allocate-node-cidrs=true \
          --cluster-cidr=10.233.64.0/18 \
          --service-cluster-ip-range=10.233.0.0/18`
}
