package services

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/rke/hosts"
)

type Etcd struct {
	Version string `yaml:"version"`
	Image   string `yaml:"image"`
}

func RunEtcdPlane(etcdHosts []hosts.Host, etcdService Etcd) error {
	logrus.Infof("[Etcd] Building up Etcd Plane..")
	for _, host := range etcdHosts {
		isRunning, err := isEtcdRunning(host, etcdService)
		if err != nil {
			return err
		}
		if isRunning {
			logrus.Infof("Etcd is already running on host [%s]", host.Hostname)
			return nil
		}
		err = runEtcdContainer(host, etcdService)
		if err != nil {
			return err
		}
	}
	return nil
}

func isEtcdRunning(host hosts.Host, etcdService Etcd) (bool, error) {
	cmd := "docker ps -a | grep " + EtcdContainerName + " | wc -l"
	logrus.Infof("Check if Etcd is running on host [%s]", host.Hostname)
	stdout, stderr := host.RunSSHCommand(cmd)
	isRunning := strings.TrimSuffix(stdout, "\n")
	if stderr != nil {
		return false, fmt.Errorf("Failed to check if Etcd is running: %v", stderr)
	}
	if isRunning == "1" {
		return true, nil
	}
	return false, nil
}

func runEtcdContainer(host hosts.Host, etcdService Etcd) error {
	err := pullEtcdImage(host, etcdService)
	if err != nil {
		return err
	}

	cmd := constructEtcdCommand(host, etcdService)
	_, stderr := host.RunSSHCommand(cmd)

	if stderr != nil {
		return fmt.Errorf("Failed to run Etcd container on host [%s]: %v", host.Hostname, stderr)
	}
	logrus.Infof("Successfully ran Etcd container on host [%s]", host.Hostname)
	return nil
}

func pullEtcdImage(host hosts.Host, etcdService Etcd) error {
	pullCmd := "docker pull " + etcdService.Image + ":" + etcdService.Version
	stdout, stderr := host.RunSSHCommand(pullCmd)

	if stderr != nil {
		return fmt.Errorf("Failed to pull Etcd image on host [%s]: %v", host.Hostname, stderr)
	}
	logrus.Debugf("Pulling Image on host [%s]: %v", host.Hostname, stdout)
	logrus.Infof("Successfully pulled Etcd image on host [%s]", host.Hostname)
	return nil
}

func constructEtcdCommand(host hosts.Host, etcdService Etcd) string {
	return `docker run -d -p 2379:2379 -p 2380:2380 \
					--volume=/var/lib/etcd:/etcd-data \
          --name ` + EtcdContainerName + ` ` + etcdService.Image + `:` + etcdService.Version + ` \
					/usr/local/bin/etcd --name etcd-` + host.Hostname + ` \
					--data-dir=/etcd-data \
          --advertise-client-urls http://` + host.IP + `:2379,http://` + host.IP + `:4001 \
          --listen-client-urls http://0.0.0.0:2379 \
          --initial-advertise-peer-urls http://` + host.IP + `:2380 \
          --listen-peer-urls http://0.0.0.0:2380 \
          --initial-cluster-token etcd-cluster-1 \
          --initial-cluster etcd-` + host.Hostname + `=http://` + host.IP + `:2380`
}

func getEtcdConnString(hosts []hosts.Host) string {
	connString := ""
	for i, host := range hosts {
		connString += "http://" + host.IP + ":2379"
		if i < (len(hosts) - 1) {
			connString += ","
		}
	}
	return connString
}
