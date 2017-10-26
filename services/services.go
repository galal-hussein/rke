package services

type Container struct {
	Services Services `yaml:"services"`
}

type Services struct {
	Etcd    Etcd    `yaml:"etcd"`
	KubeAPI KubeAPI `yaml:"kubeapi"`
}

const (
	ETCDRole             = "etcd"
	MasterRole           = "controlplane"
	WorkerRole           = "worker"
	KubeAPIContainerName = "kubeapi"
	EtcdContainerName    = "etcd"
)
