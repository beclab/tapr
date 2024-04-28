package backup

func (w *Watcher) getMiddlewareBackupPath(clusterNamespace string) (string, error) {
	backupPath := "/terminus/rootfs/"
	backupPath += middleware_backup_path

	return backupPath, nil
}
