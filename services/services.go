package services

type Container struct {
	Services Services `yaml:"services"`
}

type Services struct {
	Etcd           Etcd           `yaml:"etcd"`
	KubeAPI        KubeAPI        `yaml:"kube-api"`
	KubeController KubeController `yaml:"kube-controller"`
	Scheduler      Scheduler      `yaml:"scheduler"`
}

const (
	ETCDRole                    = "etcd"
	MasterRole                  = "controlplane"
	WorkerRole                  = "worker"
	KubeAPIContainerName        = "kube-api"
	KubeControllerContainerName = "kube-controller"
	SchedulerContainerName      = "scheduler"
	EtcdContainerName           = "etcd"
)
