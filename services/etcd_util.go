package services

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"time"

	etcdclient "github.com/coreos/etcd/client"
	"github.com/docker/docker/client"
	"github.com/rancher/rke/docker"
	"github.com/rancher/rke/hosts"
	"github.com/rancher/rke/log"
	"github.com/sirupsen/logrus"
)

func getEtcdClient(ctx context.Context, etcdHost *hosts.Host, localConnDialerFactory hosts.DialerFactory, cert, key []byte) (etcdclient.Client, error) {
	dialer, err := getEtcdDialer(localConnDialerFactory, etcdHost)
	if err != nil {
		return nil, fmt.Errorf("Failed to create a dialer for host [%s]: %v", etcdHost.Address, err)
	}
	tlsConfig, err := getEtcdTLSConfig(cert, key)
	if err != nil {
		return nil, err
	}

	var DefaultEtcdTransport etcdclient.CancelableTransport = &http.Transport{
		Dial:                dialer,
		TLSClientConfig:     tlsConfig,
		TLSHandshakeTimeout: 10 * time.Second,
	}

	cfg := etcdclient.Config{
		Endpoints: []string{"https://" + etcdHost.InternalAddress + ":2379"},
		Transport: DefaultEtcdTransport,
	}

	return etcdclient.New(cfg)
}

func isEtcdHealthy(ctx context.Context, localConnDialerFactory hosts.DialerFactory, host *hosts.Host, cert, key []byte, url string) bool {
	logrus.Debugf("[etcd] Check etcd cluster health")
	for i := 0; i < 3; i++ {
		dialer, err := getEtcdDialer(localConnDialerFactory, host)
		if err != nil {
			return false
		}
		tlsConfig, err := getEtcdTLSConfig(cert, key)
		if err != nil {
			logrus.Debugf("[etcd] Failed to create etcd tls config for host [%s]: %v", host.Address, err)
			return false
		}

		hc := http.Client{
			Transport: &http.Transport{
				Dial:                dialer,
				TLSClientConfig:     tlsConfig,
				TLSHandshakeTimeout: 10 * time.Second,
			},
		}
		healthy, err := getHealthEtcd(hc, host, url)
		if err != nil {
			logrus.Debug(err)
			time.Sleep(5 * time.Second)
			continue
		}
		if healthy == "true" {
			logrus.Debugf("[etcd] etcd cluster is healthy")
			return true
		}
	}
	return false
}

func getHealthEtcd(hc http.Client, host *hosts.Host, url string) (string, error) {
	healthy := struct{ Health string }{}
	resp, err := hc.Get(url)
	if err != nil {
		return healthy.Health, fmt.Errorf("Failed to get /health for host [%s]: %v", host.Address, err)
	}
	bytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return healthy.Health, fmt.Errorf("Failed to read response of /health for host [%s]: %v", host.Address, err)
	}
	resp.Body.Close()
	if err := json.Unmarshal(bytes, &healthy); err != nil {
		return healthy.Health, fmt.Errorf("Failed to unmarshal response of /health for host [%s]: %v", host.Address, err)
	}
	return healthy.Health, nil
}

func GetEtcdInitialCluster(hosts []*hosts.Host) string {
	initialCluster := ""
	for i, host := range hosts {
		initialCluster += fmt.Sprintf("etcd-%s=https://%s:2380", host.HostnameOverride, host.InternalAddress)
		if i < (len(hosts) - 1) {
			initialCluster += ","
		}
	}
	return initialCluster
}

func getEtcdDialer(localConnDialerFactory hosts.DialerFactory, etcdHost *hosts.Host) (func(network, address string) (net.Conn, error), error) {
	etcdHost.LocalConnPort = 2379
	var etcdFactory hosts.DialerFactory
	if localConnDialerFactory == nil {
		etcdFactory = hosts.LocalConnFactory
	} else {
		etcdFactory = localConnDialerFactory
	}
	return etcdFactory(etcdHost)
}

func GetEtcdConnString(hosts []*hosts.Host) string {
	connString := ""
	for i, host := range hosts {
		connString += "https://" + host.InternalAddress + ":2379"
		if i < (len(hosts) - 1) {
			connString += ","
		}
	}
	return connString
}

func getEtcdTLSConfig(certificate, key []byte) (*tls.Config, error) {
	// get tls config
	x509Pair, err := tls.X509KeyPair([]byte(certificate), []byte(key))
	if err != nil {
		return nil, err

	}
	tlsConfig := &tls.Config{
		InsecureSkipVerify: true,
		Certificates:       []tls.Certificate{x509Pair},
	}
	if err != nil {
		return nil, err
	}
	return tlsConfig, nil
}

func RecoverEtcdNodes(ctx context.Context, etcdHosts []*hosts.Host, dialerFactory hosts.DialerFactory, clusterPrefixPath string) error {
	log.Infof(ctx, "[reconcile] Check for stopped [etcd] containers")
	oldEtcdContainerName := "old-" + EtcdContainerName
	for _, host := range etcdHosts {
		if err := host.TunnelUp(ctx, dialerFactory, clusterPrefixPath); err != nil {
			return fmt.Errorf("Not able to reach the host: %v", err)
		}
		etcdCont, err := host.DClient.ContainerInspect(ctx, EtcdContainerName)
		if err != nil {
			if !client.IsErrNotFound(err) {
				return err
			}
			log.Infof(ctx, "[reconcile] etcd container not found on host [%s], checking for [old-etcd] container", host.Address)
			// etcd container not found check for old etcd container
			if err := renameAndStartEtcd(ctx, host, oldEtcdContainerName); err != nil {
				return err
			}
			continue
		}
		// etcd container found check if stopped
		if etcdCont.State.Status == "exited" || etcdCont.State.Status == "dead" {
			log.Infof(ctx, "[reconcile] stopped [etcd] container found on host [%s], starting container", host.Address)
			if err := docker.StartContainer(ctx, host.DClient, host.Address, EtcdContainerName); err != nil {
				return err
			}
		}
	}
	return nil
}

func renameAndStartEtcd(ctx context.Context, etcdHost *hosts.Host, oldEtcdContainerName string) error {
	_, err := etcdHost.DClient.ContainerInspect(ctx, oldEtcdContainerName)
	if err != nil {
		if !client.IsErrNotFound(err) {
			return err
		}
		// old container not found, return nil
		return nil
	}
	log.Infof(ctx, "[reconcile] stopped old-etcd container found on host [%s], starting container", etcdHost.Address)
	if err := docker.RenameContainer(ctx, etcdHost.DClient, etcdHost.Address, oldEtcdContainerName, EtcdContainerName); err != nil {
		return err
	}
	return docker.StartContainer(ctx, etcdHost.DClient, etcdHost.Address, EtcdContainerName)
}
