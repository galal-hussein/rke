package cmd

import (
	"context"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"

	"github.com/rancher/rke/cluster"
	"github.com/rancher/rke/hosts"
	v3 "github.com/rancher/types/apis/management.cattle.io/v3"
	"github.com/urfave/cli"
)

const (
	installDockerCmd = "curl -sSL https://releases.rancher.com/install-docker/DOCKER_VERSION.sh | sudo sh -"
	addUserToDockerCmd = "sudo usermod -aG docker USER"
)

func UpQuickCommand() cli.Command {
	upQuickFlags := []cli.Flag{
		cli.BoolFlag{
			Name:  "disable-port-check",
			Usage: "Disable port check validation between nodes",
		},
		cli.StringFlag{
			Name:  "user",
			Usage: "SSH user for quick connection",
		},
		cli.StringFlag{
			Name:  "control-nodes,c",
			Usage: "Comma separated list of control plane nodes addresses",
		},
		cli.StringFlag{
			Name:  "worker-nodes,w",
			Usage: "Comma separated list of worker plane nodes addresses",
		},
		cli.StringFlag{
			Name:  "etcd-nodes,e",
			Usage: "Comma separated list of etcd plane nodes addresses",
		},
		cli.StringFlag{
			Name: "nodes,n",
			Usage: "Comma separated list of nodes addresses with all roles",
		},
		cli.StringFlag{
			Name: "docker-version",
			Usage: "Docker version to be installed on the nodes",
			Value: "19.03",
		},
		cli.StringFlag{
			Name: "name",
			Usage: "Name of the config file to be saved after building the cluster",
			Value: "cluster.yml",
		},
	}

	upQuickFlags = append(upQuickFlags, commonFlags...)

	return cli.Command{
		Name:   "up-quick",
		Usage:  "Bring the cluster up quickly with no config",
		Action: clusterUpQuickFromCli,
		Flags:  upQuickFlags,
	}
}

func clusterUpQuickFromCli(ctx *cli.Context) error {
	logrus.Infof("Running RKE version: %v", ctx.App.Version)

	rkeConfig, err := getQuickRKEConfig(ctx)
	if err != nil {
		return fmt.Errorf("Failed Get Quick Config for the cluster: %v", err)
	}

	rkeConfig, err = setOptionsFromCLI(ctx, rkeConfig)
	if err != nil {
		return err
	}
	disablePortCheck := ctx.Bool("disable-port-check")
	// setting up the flags
	flags := cluster.GetExternalFlags(false, false, disablePortCheck, "", "")

	dockerVersion := ctx.String("docker-version")
	if err := installAndConfigureDocker(context.Background(), rkeConfig, flags, dockerVersion); err != nil {
		return err
	}

	if err := ClusterInit(context.Background(), rkeConfig, hosts.DialersOptions{}, flags); err != nil {
		return err
	}

	_, _, _, _, _, err = ClusterUp(context.Background(), hosts.DialersOptions{}, flags, map[string]interface{}{})

	configFile := ctx.String("name")
	return writeConfig(rkeConfig,configFile,false)
}


func installAndConfigureDocker(ctx context.Context,rkeConfig *v3.RancherKubernetesEngineConfig, flags cluster.ExternalFlags, dockerVersion string) error {
	kubeCluster, err := cluster.InitClusterObject(ctx, rkeConfig, flags, "")
	if err != nil {
		return err
	}
	allHosts := hosts.GetUniqueHostList(kubeCluster.EtcdHosts, kubeCluster.ControlPlaneHosts, kubeCluster.WorkerHosts)
	for _, host := range allHosts {
		err := host.ExecuteDirectCommand(ctx, strings.Replace(installDockerCmd,"DOCKER_VERSION",dockerVersion,1))
		if err != nil {
			return err
		}
		err = host.ExecuteDirectCommand(ctx, strings.Replace(addUserToDockerCmd,"USER",host.User,1))
		if err != nil {
			return err
		}
	}
	return nil
}
func getQuickRKEConfig(ctx *cli.Context) (*v3.RancherKubernetesEngineConfig, error) {
	user := ctx.String("user")
	if user == "" {
		return nil, fmt.Errorf("User must be defined")
	}
	allnodes := strings.Split(ctx.String("nodes"), ",")
	controlNodes := strings.Split(ctx.String("control-nodes"), ",")
	workerNodes := strings.Split(ctx.String("worker-nodes"), ",")
	etcdNodes := strings.Split(ctx.String("etcd-nodes"), ",")

	rkeNodes := map[string]v3.RKEConfigNode{}
	getQuickNodeList(rkeNodes, allnodes, "all", user)
	getQuickNodeList(rkeNodes, controlNodes, "controlplane", user)
	getQuickNodeList(rkeNodes, workerNodes, "worker", user)
	getQuickNodeList(rkeNodes, etcdNodes, "etcd", user)

	rkeConfigNodes := []v3.RKEConfigNode{}
	for _, node := range rkeNodes {
		rkeConfigNodes = append(rkeConfigNodes, node)
	}

	return &v3.RancherKubernetesEngineConfig{
		Nodes:    rkeConfigNodes,
	}, nil

}

func getQuickNodeList(rkeNodes map[string]v3.RKEConfigNode, nodes []string, role, user string) {
	for _, node := range nodes {
		if node == "" {
			continue
		}
		assignRole := []string{role}
		if role == "all" {
			assignRole = []string{"controlplane","worker","etcd"}
		}
		if _, ok := rkeNodes[node]; ok {
			nodeRole := rkeNodes[node].Role
			if role != "all" {
				nodeRole = append(nodeRole, assignRole...)
			}
			continue
		}
		rkeNodes[node] = v3.RKEConfigNode{
			Address: node,
			User:    user,
			Role:    assignRole,
		}
	}
	fmt.Println(rkeNodes)
}

