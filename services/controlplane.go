package services

import (
	"github.com/Sirupsen/logrus"
	"github.com/rancher/rke/hosts"
)

func RunControlPlane(masterHosts []hosts.Host, etcdHosts []hosts.Host, masterServices Services) error {
	logrus.Infof("[Controlplane] Building up Controller Plane..")
	for _, host := range masterHosts {
		// run kubeapi
		err := runKubeAPI(host, etcdHosts, masterServices.KubeAPI)
		if err != nil {
			return err
		}
		// run kubecontroller
		// run scheduler
	}
	return nil
}
