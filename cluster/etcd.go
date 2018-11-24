package cluster

import (
	"context"
	"fmt"

	"github.com/rancher/rke/log"
	"github.com/rancher/rke/services"
)

func (c *Cluster) SnapshotEtcd(ctx context.Context, snapshotName string) error {
	for _, host := range c.EtcdHosts {
		if err := services.RunEtcdSnapshotSave(ctx, host, c.PrivateRegistriesMap, c.SystemImages.Alpine, c.Services.Etcd.Creation, c.Services.Etcd.Retention, snapshotName, true); err != nil {
			return err
		}
	}
	return nil
}

func (c *Cluster) RestoreEtcdSnapshot(ctx context.Context, snapshotPath string) error {
	if isEqual := c.etcdSnapshotChecksum(ctx, snapshotPath); !isEqual {
		return fmt.Errorf("etcd snapshots are not consistent")
	}
	// Start restore process on all etcd hosts
	initCluster := services.GetEtcdInitialCluster(c.EtcdHosts)
	for _, host := range c.EtcdHosts {
		if err := services.RestoreEtcdSnapshot(ctx, host, c.PrivateRegistriesMap, c.SystemImages.Etcd, snapshotPath, initCluster); err != nil {
			return fmt.Errorf("[etcd] Failed to restore etcd snapshot: %v", err)
		}
	}
	// // Deploy Etcd Plane
	// etcdNodePlanMap := make(map[string]v3.RKEConfigNodePlan)
	// // Build etcd node plan map
	// for _, etcdHost := range c.EtcdHosts {
	// 	etcdNodePlanMap[etcdHost.Address] = BuildRKEConfigNodePlan(ctx, c, etcdHost, etcdHost.DockerInfo)
	// }
	// etcdRollingSnapshots := services.EtcdSnapshot{
	// 	Snapshot:  c.Services.Etcd.Snapshot,
	// 	Creation:  c.Services.Etcd.Creation,
	// 	Retention: c.Services.Etcd.Retention,
	// }
	// if err := services.RunEtcdPlane(ctx, c.EtcdHosts, etcdNodePlanMap, c.LocalConnDialerFactory, c.PrivateRegistriesMap, c.UpdateWorkersOnly, c.SystemImages.Alpine, etcdRollingSnapshots); err != nil {
	// 	return fmt.Errorf("[etcd] Failed to bring up Etcd Plane: %v", err)
	// }
	return nil
}

func (c *Cluster) etcdSnapshotChecksum(ctx context.Context, snapshotPath string) bool {
	log.Infof(ctx, "[etcd] Checking if all snapshots are identical")
	etcdChecksums := []string{}
	for _, etcdHost := range c.EtcdHosts {
		checksum, err := services.GetEtcdSnapshotChecksum(ctx, etcdHost, c.PrivateRegistriesMap, c.SystemImages.Alpine, snapshotPath)
		if err != nil {
			return false
		}
		etcdChecksums = append(etcdChecksums, checksum)
		log.Infof(ctx, "[etcd] Checksum of etcd snapshot on host [%s] is [%s]", etcdHost.Address, checksum)
	}
	hostChecksum := etcdChecksums[0]
	for _, checksum := range etcdChecksums {
		if checksum != hostChecksum {
			return false
		}
	}
	return true
}
