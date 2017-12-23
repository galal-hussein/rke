package pki

const (
	CertPathPrefix          = "/etc/kubernetes/ssl/"
	CertificatesServiceName = "certificates"
	CrtDownloaderContainer  = "cert-deployer"
	CertificatesSecretName  = "k8s-certs"

	CACertName             = "kube-ca"
	KubeAPICertName        = "kube-apiserver"
	KubeControllerCertName = "kube-controller-manager"
	KubeSchedulerCertName  = "kube-scheduler"
	KubeProxyCertName      = "kube-proxy"
	EtcdCertName           = "kube-etcd"
	KubeNodeCertName       = "kube-node"

	KubeNodeCommonName       = "system:node"
	KubeNodeOrganizationName = "system:nodes"

	KubeAdminCertName         = "kube-admin"
	KubeAdminOrganizationName = "system:masters"
	KubeAdminConfigPrefix     = ".kube_config_"
)
