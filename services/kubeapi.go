package services

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/rke/hosts"
)

type KubeAPI struct {
	Version string `yaml:"version"`
	Image   string `yaml:"image"`
}

func runKubeAPI(host hosts.Host, etcdHosts []hosts.Host, kubeAPIService KubeAPI) error {
	isRunning, err := isKubeAPIRunning(host, kubeAPIService)
	if err != nil {
		return err
	}
	if isRunning {
		logrus.Infof("KubeAPI is already running on host [%s]", host.Hostname)
		return nil
	}
	etcdConnString := getEtcdConnString(etcdHosts)
	err = runKubeAPIContainer(host, kubeAPIService, etcdConnString)
	if err != nil {
		return err
	}
	return nil
}

func isKubeAPIRunning(host hosts.Host, kubeAPIService KubeAPI) (bool, error) {
	cmd := "docker ps -a | grep " + KubeAPIContainerName + " | wc -l"
	logrus.Infof("Check if KubeAPI is running on host [%s]", host.Hostname)
	stdout, stderr := host.RunSSHCommand(cmd)
	isRunning := strings.TrimSuffix(stdout, "\n")
	if stderr != nil {
		return false, fmt.Errorf("Failed to check if KubeAPI is running: %v", stderr)
	}
	if isRunning == "1" {
		return true, nil
	}
	return false, nil
}

func runKubeAPIContainer(host hosts.Host, kubeAPIService KubeAPI, etcdConnString string) error {
	err := pullKubeAPIImage(host, kubeAPIService)
	if err != nil {
		return err
	}
	cmd := constructKubeAPICommand(host, kubeAPIService, etcdConnString)
	_, stderr := host.RunSSHCommand(cmd)

	if stderr != nil {
		return fmt.Errorf("Failed to run Kube API container on host [%s]: %v", host.Hostname, stderr)
	}
	logrus.Infof("Successfully ran Kube API container on host [%s]", host.Hostname)
	return nil
}

func pullKubeAPIImage(host hosts.Host, kubeAPIService KubeAPI) error {
	pullCmd := "docker pull " + kubeAPIService.Image + ":" + kubeAPIService.Version
	stdout, stderr := host.RunSSHCommand(pullCmd)
	if stderr != nil {
		return fmt.Errorf("Failed to pull Kube API image on host [%s]: %v", host.Hostname, stderr)
	}
	logrus.Debugf("Pulling Image on host [%s]: %v", host.Hostname, stdout)
	logrus.Infof("Successfully pulled Kube API image on host [%s]", host.Hostname)
	return nil
}

func constructKubeAPICommand(host hosts.Host, kubeAPIService KubeAPI, etcdConnString string) string {
	return `docker run -d -p 8080:8080 --net=host --name=` + KubeAPIContainerName + `\
          ` + kubeAPIService.Image + `:` + kubeAPIService.Version + ` /hyperkube apiserver \
          --insecure-bind-address=0.0.0.0 --insecure-port=8080 \
          --cloud-provider="" \
          --allow_privileged=true \
          --service-cluster-ip-range=10.233.0.0/18 \
          --admission-control=NamespaceLifecycle,LimitRanger,PersistentVolumeLabel,DefaultStorageClass,ResourceQuota,DefaultTolerationSeconds \
          --runtime-config=batch/v2alpha1 \
          --runtime-config=authentication.k8s.io/v1beta1=true \
          --storage-backend=etcd3 \
          --etcd-servers=` + etcdConnString + ` \
          --advertise-address=` + host.IP
}
