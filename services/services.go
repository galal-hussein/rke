package services

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/rancher/rke/hosts"
	"golang.org/x/net/context"
)

type Container struct {
	Services Services `yaml:"services"`
}

type Services struct {
	Etcd           Etcd           `yaml:"etcd"`
	KubeAPI        KubeAPI        `yaml:"kube-api"`
	KubeController KubeController `yaml:"kube-controller"`
	Scheduler      Scheduler      `yaml:"scheduler"`
	Kubelet        Kubelet        `yaml:"kubelet"`
	Kubeproxy      Kubeproxy      `yaml:"kubeproxy"`
}

type HealthCheck struct {
	Command        string         `yaml:"command"`
	Interval       time.Duration  `yaml:"interval"`
	Timeout        time.Duration  `yaml:"timeout"`
	StartPeriod    time.Duration  `yaml:"start_period"`
	Retries        int            `yaml:"retries"`
}
const (
	ETCDRole                    = "etcd"
	MasterRole                  = "controlplane"
	WorkerRole                  = "worker"
	KubeAPIContainerName        = "kube-api"
	KubeletContainerName        = "kubelet"
	KubeproxyContainerName      = "kube-proxy"
	KubeControllerContainerName = "kube-controller"
	SchedulerContainerName      = "scheduler"
	EtcdContainerName           = "etcd"
)

func IsContainerRunning(host hosts.Host, containerName string) (bool, error) {
	logrus.Debugf("Checking if container %s is running on host [%s]", containerName, host.Hostname)
	containers, err := host.DClient.ContainerList(context.Background(), types.ContainerListOptions{})
	if err != nil {
		return false, fmt.Errorf("Can't get Docker containers for host [%s]: %v", host.Hostname, err)

	}
	for _, container := range containers {
		if container.Names[0] == "/"+containerName {
			return true, nil
		}
	}
	return false, nil
}

func PullImage(host hosts.Host, containerImage string) error {
	out, err := host.DClient.ImagePull(context.Background(), containerImage, types.ImagePullOptions{})
	if err != nil {
		return fmt.Errorf("Can't pull Docker image %s for host [%s]: %v", containerImage, host.Hostname, err)
	}
	defer out.Close()
	if logrus.GetLevel() == logrus.DebugLevel {
		io.Copy(os.Stdout, out)
	} else {
		io.Copy(ioutil.Discard, out)
	}

	return nil
}

func BuildHealthCheckConfig(healthCheck HealthCheck) *container.HealthConfig {
	if healthCheck != (HealthCheck{}) {
		return &container.HealthConfig{
							Test: []string{"CMD-SHELL",
								healthCheck.Command},
							Interval: healthCheck.Interval,
							Timeout: healthCheck.Timeout,
							StartPeriod: healthCheck.StartPeriod,
							Retries: healthCheck.Retries}
	} else {
		logrus.Debugf("[HealthCheck] No Docker Health check defined..")
		return nil
	}
}
