package pki

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"fmt"
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
	logrus.Debugf("[certificates] CA Certificate: %s", string(cert.EncodeCertPEM(caCrt)))
	certs[CACertName] = CertificatePKI{
		Certificate: caCrt,
		Key:         caKey,
		Name:        CACertName,
		EnvName:     CACertENVName,
		KeyEnvName:  CAKeyENVName,
		Path:        CACertPath,
		KeyPath:     CAKeyPath,
	}

	// generate API certificate and key
	log.Infof(ctx, "[certificates] Generating Kubernetes API server certificates")
	kubeAPIAltNames := GetAltNames(cpHosts, clusterDomain, KubernetesServiceIP)
	kubeAPICrt, kubeAPIKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, kubeAPIAltNames, KubeAPICertName, nil, true)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("[certificates] Kube API Certificate: %s", string(cert.EncodeCertPEM(kubeAPICrt)))
	certs[KubeAPICertName] = CertificatePKI{
		Certificate: kubeAPICrt,
		Key:         kubeAPIKey,
		Name:        KubeAPICertName,
		EnvName:     KubeAPICertENVName,
		KeyEnvName:  KubeAPIKeyENVName,
		Path:        KubeAPICertPath,
		KeyPath:     KubeAPIKeyPath,
	}

	// generate Kube controller-manager certificate and key
	log.Infof(ctx, "[certificates] Generating Kube Controller certificates")
	kubeControllerCrt, kubeControllerKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, nil, KubeControllerCommonName, nil, false)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("[certificates] Kube Controller Certificate: %s", string(cert.EncodeCertPEM(kubeControllerCrt)))
	certs[KubeControllerName] = CertificatePKI{
		Certificate:   kubeControllerCrt,
		Key:           kubeControllerKey,
		Config:        getKubeConfigX509("https://127.0.0.1:6443", KubeControllerName, CACertPath, KubeControllerCertPath, KubeControllerKeyPath),
		Name:          KubeControllerName,
		CommonName:    KubeControllerCommonName,
		EnvName:       KubeControllerCertENVName,
		KeyEnvName:    KubeControllerKeyENVName,
		Path:          KubeControllerCertPath,
		KeyPath:       KubeControllerKeyPath,
		ConfigEnvName: KubeControllerConfigENVName,
		ConfigPath:    KubeControllerConfigPath,
	}

	// generate Kube scheduler certificate and key
	log.Infof(ctx, "[certificates] Generating Kube Scheduler certificates")
	kubeSchedulerCrt, kubeSchedulerKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, nil, KubeSchedulerCommonName, nil, false)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("[certificates] Kube Scheduler Certificate: %s", string(cert.EncodeCertPEM(kubeSchedulerCrt)))
	certs[KubeSchedulerName] = CertificatePKI{
		Certificate:   kubeSchedulerCrt,
		Key:           kubeSchedulerKey,
		Config:        getKubeConfigX509("https://127.0.0.1:6443", KubeSchedulerName, CACertPath, KubeSchedulerCertPath, KubeSchedulerKeyPath),
		Name:          KubeSchedulerName,
		CommonName:    KubeSchedulerCommonName,
		EnvName:       KubeSchedulerCertENVName,
		KeyEnvName:    KubeSchedulerKeyENVName,
		Path:          KubeSchedulerCertPath,
		KeyPath:       KubeSchedulerKeyPath,
		ConfigEnvName: KubeSchedulerConfigENVName,
		ConfigPath:    KubeSchedulerConfigPath,
	}

	// generate Kube Proxy certificate and key
	log.Infof(ctx, "[certificates] Generating Kube Proxy certificates")
	kubeProxyCrt, kubeProxyKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, nil, KubeProxyCommonName, nil, false)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("[certificates] Kube Proxy Certificate: %s", string(cert.EncodeCertPEM(kubeProxyCrt)))
	certs[KubeProxyName] = CertificatePKI{
		Certificate:   kubeProxyCrt,
		Key:           kubeProxyKey,
		Config:        getKubeConfigX509("https://127.0.0.1:6443", KubeProxyName, CACertPath, KubeProxyCertPath, KubeProxyKeyPath),
		Name:          KubeProxyName,
		CommonName:    KubeProxyCommonName,
		EnvName:       KubeProxyCertENVName,
		Path:          KubeProxyCertPath,
		KeyEnvName:    KubeProxyKeyENVName,
		KeyPath:       KubeProxyKeyPath,
		ConfigEnvName: KubeProxyConfigENVName,
		ConfigPath:    KubeProxyConfigPath,
	}

	// generate Kubelet certificate and key
	log.Infof(ctx, "[certificates] Generating Node certificate")
	nodeCrt, nodeKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, nil, KubeNodeCommonName, []string{KubeNodeOrganizationName}, false)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("[certificates] Node Certificate: %s", string(cert.EncodeCertPEM(kubeProxyCrt)))
	certs[KubeNodeName] = CertificatePKI{
		Certificate:   nodeCrt,
		Key:           nodeKey,
		Config:        getKubeConfigX509("https://127.0.0.1:6443", KubeNodeName, CACertPath, KubeNodeCertPath, KubeNodeKeyPath),
		Name:          KubeNodeName,
		CommonName:    KubeNodeCommonName,
		OUName:        KubeNodeOrganizationName,
		EnvName:       KubeNodeCertENVName,
		KeyEnvName:    KubeNodeKeyENVName,
		Path:          KubeNodeCertPath,
		KeyPath:       KubeNodeKeyPath,
		ConfigEnvName: KubeNodeConfigENVName,
		ConfigPath:    KubeNodeCommonName,
	}
	log.Infof(ctx, "[certificates] Generating admin certificates and kubeconfig")
	kubeAdminCrt, kubeAdminKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, nil, KubeAdminCommonName, []string{KubeAdminOrganizationName}, false)
	if err != nil {
		return nil, err
	}
	logrus.Debugf("[certificates] Admin Certificate: %s", string(cert.EncodeCertPEM(kubeAdminCrt)))
	certs[KubeAdminCommonName] = CertificatePKI{
		Certificate: kubeAdminCrt,
		Key:         kubeAdminKey,
		Config: GetKubeConfigX509WithData(
			"https://"+cpHosts[0].Address+":6443",
			KubeAdminCommonName,
			string(cert.EncodeCertPEM(caCrt)),
			string(cert.EncodeCertPEM(kubeAdminCrt)),
			string(cert.EncodePrivateKeyPEM(kubeAdminKey))),
		CommonName:    KubeAdminCommonName,
		OUName:        KubeAdminOrganizationName,
		ConfigEnvName: KubeAdminConfigENVName,
		ConfigPath:    localConfigPath,
	}
	etcdAltNames := GetAltNames(etcdHosts, clusterDomain, KubernetesServiceIP)
	for i := range etcdHosts {
		logrus.Infof("[certificates] Generating etcd-%d certificates and key", i)
		etcdCrt, etcdKey, err := GenerateSignedCertAndKey(caCrt, caKey, nil, etcdAltNames, EtcdCertName, nil, true)
		if err != nil {
			return nil, err
		}
		etcdName := getEtcdCrtName(i)
		certs[etcdName] = CertificatePKI{
			Certificate: etcdCrt,
			Key:         etcdKey,
			Name:        etcdName,
			CommonName:  etcdName,
			EnvName:     fmt.Sprintf("%s_%d", EtcdCertENVName, i),
			KeyEnvName:  fmt.Sprintf("%s_%d_KEY", EtcdCertENVName, i),
			Path:        fmt.Sprintf("%s-%d.pem", EtcdKeyCertPathPrefix, i),
			KeyPath:     fmt.Sprintf("%s-%d-key.pem", EtcdKeyCertPathPrefix, i),
		}
	}
	return certs, nil
}
