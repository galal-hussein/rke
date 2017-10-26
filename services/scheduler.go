package services

import (
	"fmt"
	"strings"

	"github.com/Sirupsen/logrus"
	"github.com/rancher/rke/hosts"
)

type Scheduler struct {
	Version string `yaml:"version"`
	Image   string `yaml:"image"`
}

func runScheduler(host hosts.Host, schedulerService Scheduler) error {
	isRunning, err := isSchedulerRunning(host, schedulerService)
	if err != nil {
		return err
	}
	if isRunning {
		logrus.Infof("Scheduler is already running on host [%s]", host.Hostname)
		return nil
	}
	err = runSchedulerContainer(host, schedulerService)
	if err != nil {
		return err
	}
	return nil
}

func isSchedulerRunning(host hosts.Host, schedulerService Scheduler) (bool, error) {
	cmd := "docker ps -a | grep " + SchedulerContainerName + " | wc -l"
	logrus.Infof("Check if Scheduler is running on host [%s]", host.Hostname)
	stdout, stderr := host.RunSSHCommand(cmd)
	isRunning := strings.TrimSuffix(stdout, "\n")
	if stderr != nil {
		return false, fmt.Errorf("Failed to check if Scheduler is running: %v", stderr)
	}
	if isRunning == "1" {
		return true, nil
	}
	return false, nil
}

func runSchedulerContainer(host hosts.Host, schedulerService Scheduler) error {
	err := pullSchedulerImage(host, schedulerService)
	if err != nil {
		return err
	}
	cmd := constructSchedulerCommand(host, schedulerService)
	_, stderr := host.RunSSHCommand(cmd)

	if stderr != nil {
		return fmt.Errorf("Failed to run Scheduler container on host [%s]: %v", host.Hostname, stderr)
	}
	logrus.Infof("Successfully ran Scheduler container on host [%s]", host.Hostname)
	return nil
}

func pullSchedulerImage(host hosts.Host, schedulerService Scheduler) error {
	pullCmd := "docker pull " + schedulerService.Image + ":" + schedulerService.Version
	stdout, stderr := host.RunSSHCommand(pullCmd)
	if stderr != nil {
		return fmt.Errorf("Failed to pull Scheduler image on host [%s]: %v", host.Hostname, stderr)
	}
	logrus.Debugf("Pulling Image on host [%s]: %v", host.Hostname, stdout)
	logrus.Infof("Successfully pulled Scheduler image on host [%s]", host.Hostname)
	return nil
}

func constructSchedulerCommand(host hosts.Host, schedulerService Scheduler) string {
	return `docker run -d --name=` + SchedulerContainerName + `\
          ` + schedulerService.Image + `:` + schedulerService.Version + ` /hyperkube scheduler \
					--master=http://` + host.IP + `:8080 \
          --address=0.0.0.0 \
          --v=2`
}
