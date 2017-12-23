package pki

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"net"

	"github.com/rancher/rke/hosts"
	"github.com/rancher/rke/log"
	"github.com/sirupsen/logrus"
	"k8s.io/client-go/util/cert"
)

type CertificatePKI struct {
	Certificate   *x509.Certificate
	Key           *rsa.PrivateKey
	Config        string
	Name          string
	CommonName    string
	OUName        string
	EnvName       string
	Path          string
	KeyEnvName    string
	KeyPath       string
	ConfigEnvName string
	ConfigPath    string
}

// StartCertificatesGeneration ...
func StartCertificatesGeneration(ctx context.Context, cpHosts, etcdHosts []*hosts.Host, clusterDomain, localConfigPath string, KubernetesServiceIP net.IP) (map[string]CertificatePKI, error) {
	logrus.Infof("[certificates] Generating kubernetes certificates")
	certs, err := generateCerts(ctx, cpHosts, etcdHosts, clusterDomain, localConfigPath, KubernetesServiceIP)
	if err != nil {
		return nil, err
	}
	return certs, nil
}

func generateCerts(ctx context.Context, cpHosts, etcdHosts []*hosts.Host, clusterDomain, localConfigPath string, KubernetesServiceIP net.IP) (map[string]CertificatePKI, error) {
	certs := make(map[string]CertificatePKI)
	// generate CA certificate and key
	log.Infof(ctx, "[certificates] Generating CA kubernetes certificates")
	caCrt, caKey, err := generateCACertAndKey()
	if err != nil {
		return nil, err
	}
	certs[CACertName] = ToCertObject(CACertName, "", "", caCrt, caKey)

	// generate API certificate and key
	log.Infof(ctx, "[certificates] Generating Kubernetes API server certificates")
	kubeAPIAltNames := GetAltNames(cpHosts, clusterDomain, KubernetesServiceIP)
	kubeAPICrt, kubeAPIKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, kubeAPIAltNames, KubeAPICertName, nil, true)
	if err != nil {
		return nil, err
	}
	certs[KubeAPICertName] = ToCertObject(KubeAPICertName, "", "", kubeAPICrt, kubeAPIKey)

	// generate Kube controller-manager certificate and key
	log.Infof(ctx, "[certificates] Generating Kube Controller certificates")
	kubeControllerCrt, kubeControllerKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, nil, getDefaultCN(KubeControllerCertName), nil, false)
	if err != nil {
		return nil, err
	}
	certs[KubeControllerCertName] = ToCertObject(KubeControllerCertName, "", "", kubeControllerCrt, kubeControllerKey)

	// generate Kube scheduler certificate and key
	log.Infof(ctx, "[certificates] Generating Kube Scheduler certificates")
	kubeSchedulerCrt, kubeSchedulerKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, nil, getDefaultCN(KubeSchedulerCertName), nil, false)
	if err != nil {
		return nil, err
	}
	certs[KubeSchedulerCertName] = ToCertObject(KubeSchedulerCertName, "", "", kubeSchedulerCrt, kubeSchedulerKey)

	// generate Kube Proxy certificate and key
	log.Infof(ctx, "[certificates] Generating Kube Proxy certificates")
	kubeProxyCrt, kubeProxyKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, nil, getDefaultCN(KubeProxyCertName), nil, false)
	if err != nil {
		return nil, err
	}
	certs[KubeProxyCertName] = ToCertObject(KubeProxyCertName, "", "", kubeProxyCrt, kubeProxyKey)

	// generate Kubelet certificate and key
	log.Infof(ctx, "[certificates] Generating Node certificate")
	nodeCrt, nodeKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, nil, KubeNodeCommonName, []string{KubeNodeOrganizationName}, false)
	if err != nil {
		return nil, err
	}
	certs[KubeNodeCertName] = ToCertObject(KubeNodeCertName, KubeNodeCommonName, KubeNodeOrganizationName, nodeCrt, nodeKey)

	// generate Admin certificate and key
	logrus.Infof("[certificates] Generating admin certificates and kubeconfig")
	kubeAdminCrt, kubeAdminKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, nil, KubeAdminCertName, []string{KubeAdminOrganizationName}, false)
	if err != nil {
		return nil, err
	}
	kubeAdminConfig := GetKubeConfigX509WithData(
		"https://"+cpHosts[0].Address+":6443",
		KubeAdminCertName,
		string(cert.EncodeCertPEM(caCrt)),
		string(cert.EncodeCertPEM(kubeAdminCrt)),
		string(cert.EncodePrivateKeyPEM(kubeAdminKey)))

	certs[KubeAdminCertName] = CertificatePKI{
		Certificate: kubeAdminCrt,
		Key:         kubeAdminKey,
		Config:      kubeAdminConfig,
		CommonName:  KubeAdminCertName,
		OUName:      KubeAdminOrganizationName,
		ConfigPath:  localConfigPath,
	}

	etcdAltNames := GetAltNames(etcdHosts, clusterDomain, KubernetesServiceIP)
	for i := range etcdHosts {
		logrus.Infof("[certificates] Generating etcd-%d certificate and key", i)
		etcdCrt, etcdKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, etcdAltNames, EtcdCertName, nil, true)
		if err != nil {
			return nil, err
		}
		etcdName := GetEtcdCrtName(i)
		certs[etcdName] = ToCertObject(etcdName, "", "", etcdCrt, etcdKey)
	}
	return certs, nil
}
