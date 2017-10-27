package services

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
