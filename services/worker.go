package services

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/rke/hosts"
)

func RunWorkerPlane(workerHosts []hosts.Host, workerServices Services) error {
	logrus.Infof("Building up Worker Plane..")
	for _ = range workerHosts {
		// setup networking
		// run kubelet
		// run kubeproxy
	}
	return nil
}
