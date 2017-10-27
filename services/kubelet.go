package services

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/rke/hosts"
)

type Kubelet struct {
	Version string `yaml:"version"`
	Image   string `yaml:"image"`
}

func runKubelet(host hosts.Host, masterHost hosts.Host, kubeletService Kubelet, isMaster bool) error {
	isRunning, err := isKubeletRunning(host, masterHost, kubeletService)
	if err != nil {
		return err
	}
	if isRunning {
		logrus.Infof("Kubelet is already running on host [%s]", host.Hostname)
		return nil
	}
	err = runKubeletContainer(host, masterHost, kubeletService, isMaster)
	if err != nil {
		return err
	}
	return nil
}

func isKubeletRunning(host hosts.Host, masterHost hosts.Host, kubeletService Kubelet) (bool, error) {
	cmd := "docker ps -a | grep " + KubeletContainerName + " | wc -l"
	logrus.Infof("Check if Kubelet is running on host [%s]", host.Hostname)
	stdout, stderr := host.RunSSHCommand(cmd)
	isRunning := strings.TrimSuffix(stdout, "\n")
	if stderr != nil {
		return false, fmt.Errorf("Failed to check if Kubelet is running: %v", stderr)
	}
	if isRunning == "1" {
		return true, nil
	}
	return false, nil
}

func runKubeletContainer(host hosts.Host, masterHost hosts.Host, kubeletService Kubelet, isMaster bool) error {
	err := pullKubeletImage(host, kubeletService)
	if err != nil {
		return err
	}
	cmd := constructKubeletCommand(host, masterHost, kubeletService, isMaster)
	_, stderr := host.RunSSHCommand(cmd)

	if stderr != nil {
		return fmt.Errorf("Failed to run Kubelet container on host [%s]: %v", host.Hostname, stderr)
	}
	logrus.Infof("Successfully ran Kubelet container on host [%s]", host.Hostname)
	return nil
}

func pullKubeletImage(host hosts.Host, kubeletService Kubelet) error {
	pullCmd := "docker pull " + kubeletService.Image + ":" + kubeletService.Version
	stdout, stderr := host.RunSSHCommand(pullCmd)
	if stderr != nil {
		return fmt.Errorf("Failed to pull Kubelet image on host [%s]: %v", host.Hostname, stderr)
	}
	logrus.Debugf("Pulling Image on host [%s]: %v", host.Hostname, stdout)
	logrus.Infof("Successfully pulled Kubelet image on host [%s]", host.Hostname)
	return nil
}

func constructKubeletCommand(host hosts.Host, masterHost hosts.Host, kubeletService Kubelet, isMaster bool) string {
	var masterArgs string
	if isMaster {
		masterArgs = "--register-with-taints=node-role.kubernetes.io/master=:NoSchedule --node-labels=node-role.kubernetes.io/master=true"
	}
	return `docker run -d \
			  --net=host \
			  --pid=host \
			  --privileged \
			  --name=` + KubeletContainerName + ` \
			  --restart=on-failure:5 \
			  -v /etc/cni:/etc/cni:ro \
			  -v /opt/cni:/opt/cni:ro \
			  -v /etc/resolv.conf:/etc/resolv.conf \
			  -v /sys:/sys:ro \
			  -v /var/lib/docker:/var/lib/docker:rw \
			  -v /var/lib/kubelet:/var/lib/kubelet:shared \
			  -v /var/run:/var/run:rw \
				-v /run:/run \
				-v /dev:/host/dev \
			  ` + kubeletService.Image + `:` + kubeletService.Version + ` \
			  ./hyperkube kubelet \
				--v=2 \
				--address=0.0.0.0 \
				--cluster-domain=cluster.local \
				--hostname-override=` + host.Hostname + ` \
				--pod-infra-container-image=gcr.io/google_containers/pause-amd64:3.0 \
				--cgroup-driver=cgroupfs \
				--cgroups-per-qos=True \
				--enforce-node-allocatable=""  \
				--cluster-dns=10.233.0.3 \
				--resolv-conf=/etc/resolv.conf \
				--network-plugin=cni --cni-conf-dir=/etc/cni/net.d --cni-bin-dir=/opt/cni/bin \
				--allow-privileged=true \
				--cloud-provider="" ` + masterArgs + ` \
				--api-servers=http://` + masterHost.IP + `:8080/ `
}
