package cmd

func backupMysqlPerformRestore() error {
	var err error
	err = backupPrerequisites()
	if err != nil {
		return err
	}

	// Game plan:

	// - Create volume to house backup
	// - Mount volume to house backup
	// - Download full backup and incremental pieces
	// - Unpack into directory
	// - Prepare backup
	// _, err = pkg.PerformCommand("xtrabackup", "--prepare", "--target-dir", backupDirectory)
	// if err != nil {
	// 	pkg.AlertError("Could not create backup.", err)
	// 	return backupCleanup(volume, mountDirectory, digitalOceanClient)
	// }
	// - Move to MySQL data directory
	// - Set correct permissions

	return nil
}
