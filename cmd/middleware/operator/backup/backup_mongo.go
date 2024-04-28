package backup

import (
	"fmt"

	"bytetrade.io/web3os/tapr/pkg/workload/percona"
	"k8s.io/klog/v2"
)

func (w *Watcher) backupMongo() error {
	clusters, err := percona.ListPerconaMongoCluster(w.ctx, *w.dynamicClient, "")
	if err != nil {
		klog.Error("list mongo clusters error, ", err)
		return err
	}

	klog.Info("start to backup all users' mongo clusters, of ", len(clusters))
	for _, cluster := range clusters {
		klog.Info("create crd to backup cluster, ", cluster.Name, ", ", cluster.Namespace)
		backup := percona.ClusterBackup.DeepCopy()
		backup.Namespace = cluster.Namespace
		backup.ObjectMeta.Labels = make(map[string]string)
		backup.ObjectMeta.Labels["managered-by"] = fmt.Sprintf("mongo-backup-%s", cluster.Name)

		klog.Info("find userspace hostpath, ", cluster.Namespace)
		backupPath, err := w.getMiddlewareBackupPath(cluster.Namespace)
		if err != nil {
			klog.Info("find userspace hostpath error, ", err)
			return err
		}
		backupPath += "/mongo-backup"
		err = percona.ForceCreateNewMongoClusterBackup(w.ctx, w.dynamicClient, backup, backupPath)
		if err != nil {
			return err
		}
	}

	klog.Info("wait for all mongo cluster backup complete")
	err = percona.WaitForAllBackupComplete(w.ctx, w.dynamicClient)
	if err != nil {
		klog.Error("wait for backup complete error, ", err)
	}

	return err
}

func (w *Watcher) restoreMongo() error {
	clusters, err := percona.ListPerconaMongoCluster(w.ctx, *w.dynamicClient, "")
	if err != nil {
		klog.Error("list mongo clusters error, ", err)
		return err
	}

	if err = percona.WaitForInitializeComplete(w.ctx, *&w.dynamicClient, *&w.k8sClientSet); err != nil {
		klog.Error("mongo cluster initialize error, ", err)
		return err
	}

	// It is possible for the MongoDB restoration process to occur repeatedly,
	// leading to the MongoDB cluster service becoming abnormally unavailable.
	ok, err := percona.CheckMongoRestoreStatus(w.ctx, *&w.dynamicClient)
	if !ok && err != nil {
		klog.Error("mongo restore status check, ", err)
	}

	if ok {
		klog.Info("mongo restore in progress or already finished")
		return nil
	}

	klog.Info("start to restore all users' mongo clusters, of ", len(clusters))
	for _, cluster := range clusters {
		klog.Info("create crd to restore cluster, ", cluster.Name, ", ", cluster.Namespace)

		restore := percona.ClusterRestore.DeepCopy()
		restore.Namespace = cluster.Namespace

		err = percona.ForceCreateNewMongoClusterRestore(w.ctx, w.dynamicClient, restore)
		if err != nil {
			return err
		}

	}

	klog.Info("wait for all mongo cluster restore complete")
	err = percona.WaitForAllRestoreComplete(w.ctx, w.dynamicClient)
	if err != nil {
		klog.Error("wait for mongo restore complete error, ", err)
	}

	return err
}
