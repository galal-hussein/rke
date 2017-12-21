package pki

import (
	"crypto/rsa"
	"crypto/x509"
	"fmt"
	"net"

	"github.com/rancher/rke/hosts"
	"k8s.io/client-go/util/cert"
)

func GenerateSignedCertAndKey(
	caCrt *x509.Certificate,
	caKey, reusedKey *rsa.PrivateKey,
	altNames *cert.AltNames,
	commonName string,
	orgs []string,
	serverCrt bool) (*x509.Certificate, *rsa.PrivateKey, error) {
	// Generate a generic signed certificate
	var rootKey *rsa.PrivateKey
	var err error
	rootKey = reusedKey
	if reusedKey == nil {
		rootKey, err = cert.NewPrivateKey()
		if err != nil {
			return nil, nil, fmt.Errorf("Failed to generate private key for %s certificate: %v", commonName, err)
		}
	}
	usages := []x509.ExtKeyUsage{x509.ExtKeyUsageClientAuth}
	if serverCrt {
		usages = append(usages, x509.ExtKeyUsageServerAuth)
	}
	if altNames == nil {
		altNames = &cert.AltNames{}
	}
	caConfig := cert.Config{
		CommonName:   commonName,
		Organization: orgs,
		Usages:       usages,
		AltNames:     *altNames,
	}
	clientCert, err := cert.NewSignedCert(caConfig, rootKey, caCrt, caKey)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to generate %s certificate: %v", commonName, err)
	}
	return clientCert, rootKey, nil
}

func generateCACertAndKey() (*x509.Certificate, *rsa.PrivateKey, error) {
	rootKey, err := cert.NewPrivateKey()
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to generate private key for CA certificate: %v", err)
	}
	caConfig := cert.Config{
		CommonName: CACertName,
	}
	kubeCACert, err := cert.NewSelfSignedCACert(caConfig, rootKey)
	if err != nil {
		return nil, nil, fmt.Errorf("Failed to generate CA certificate: %v", err)
	}

	return kubeCACert, rootKey, nil
}

func GetAltNames(cpHosts []*hosts.Host, clusterDomain string, KubernetesServiceIP net.IP) *cert.AltNames {
	ips := []net.IP{}
	dnsNames := []string{}
	for _, host := range cpHosts {
		// Check if node address is a valid IP
		if nodeIP := net.ParseIP(host.Address); nodeIP != nil {
			ips = append(ips, nodeIP)
		} else {
			dnsNames = append(dnsNames, host.Address)
		}

		// Check if node internal address is a valid IP
		if len(host.InternalAddress) != 0 && host.InternalAddress != host.Address {
			if internalIP := net.ParseIP(host.InternalAddress); internalIP != nil {
				ips = append(ips, internalIP)
			} else {
				dnsNames = append(dnsNames, host.InternalAddress)
			}
		}
		// Add hostname to the ALT dns names
		if len(host.HostnameOverride) != 0 && host.HostnameOverride != host.Address {
			dnsNames = append(dnsNames, host.HostnameOverride)
		}
	}
	ips = append(ips, net.ParseIP("127.0.0.1"))
	ips = append(ips, KubernetesServiceIP)
	dnsNames = append(dnsNames, []string{
		"localhost",
		"kubernetes",
		"kubernetes.default",
		"kubernetes.default.svc",
		"kubernetes.default.svc." + clusterDomain,
	}...)
	return &cert.AltNames{
		IPs:      ips,
		DNSNames: dnsNames,
	}
}

func (c *CertificatePKI) ToEnv() []string {
	env := []string{
		c.CertToEnv(),
		c.KeyToEnv(),
	}
	if c.Config != "" {
		env = append(env, c.ConfigToEnv())
	}
	return env
}

func (c *CertificatePKI) CertToEnv() string {
	encodedCrt := cert.EncodeCertPEM(c.Certificate)
	return fmt.Sprintf("%s=%s", c.EnvName, string(encodedCrt))
}

func (c *CertificatePKI) KeyToEnv() string {
	encodedKey := cert.EncodePrivateKeyPEM(c.Key)
	return fmt.Sprintf("%s=%s", c.KeyEnvName, string(encodedKey))
}

func (c *CertificatePKI) ConfigToEnv() string {
	return fmt.Sprintf("%s=%s", c.ConfigEnvName, c.Config)
}

func GetEtcdEnvNodeName(index int) string {
	return fmt.Sprintf("%s-%d", EtcdKeyCertPathPrefix, index)
}

func getEtcdCrtName(index int) string {
	return fmt.Sprintf("%s-%d", EtcdCertName, index)
}
