# really-simple-db-backup

The goal of this project is to be a really simple backup solution where you can just drop a binary on your database server, setup a cronjob to run at regular intervals, and it will take care of performing full and incremental backups at a regular intervals and uploading them to a cloud bucket.

This project also aims to simplify restoring backups. The goal is for it to be hassle free and require little mental overhead in the greatest time of needs.

Currently this project is built to perform backups of MySQL 8.0 and store them on a [DigitalOcean Space](https://www.digitalocean.com/products/spaces/). The software for actually generating the backup file is the amazing [Percona Xtrabackup](https://www.percona.com/software/mysql-database/percona-xtrabackup).

## Usage

### Build & upload binary

```
git clone git@github.com:feederco/really-simple-db-backup.git
cd really-simple-db-backup
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/really-simple-db-backup main.go
scp build/really-simple-db-backup my-db-host:/usr/bin/really-simple-db-backup
```

### Available commands

```shell
really-simple-db-backup perform|perform-full|perform-incremental|upload
```

### Perform backup

Performs either a full or incremental backup by checking for previous runs.

```shell
really-simple-db-backup perform \
  -do-key=digitalocean-app-token \
  -do-space-endpoint=fra1.digitaloceanspaces.com \
  -do-space-name=my-backups \
  -do-space-key=auth-key-for-space \
  -do-space-secret=auth-secret-for-space
```

### Force a full backup (or incremental backup)

To force a full backup you can use the `perform-full` command.

```shell
really-simple-db-backup perform-full -options...
```

To force an incremental backup you can use the `perform-incremental` command.

```shell
really-simple-db-backup perform-full -options...
```

### Force upload

If for some reason a backup failed and you were successfully able to retrieve a backup yourself, you can use the `upload` command to upload this to your DigitalOcean Space.

```shell
really-simple-db-backup upload -file /path/to/backup.xbstream -options...
```

### Config file

By default the script checks for the existence of a config file at `/etc/really-simple-db-backup.json`. If this is found the defaults are loaded from that file and can be overriden by command line options.

If this config file is located somewhere else you can pass that in with the `-config` option.

```shell
really-simple-db-backup perform -config ./my-other-config.json
```

The following format is expected:

```json
{
  "do_key": "digitalocean-app-token",
  "do_space_endpoint": "fra1.digitaloceanspaces.com",
  "do_space_name": "my-backups",
  "do_space_key": "auth-key-for-space",
  "do_space_secret": "auth-secret-for-space",
  "mysql_data_path": "(optional)",
  "persistent_storage": "(optional)"
}
```

### Different MySQL data directory

The default directory for MySQL is normally `/var/lib/mysql`. If you have mounted a volume for your data and set different [`datadir`](https://dev.mysql.com/doc/refman/8.0/en/data-directory.html) you can pass in the following option: `-mysql-data-dir=/mnt/my_mysql_volume/mysql`.

### Persistent storage directory

To save state between runs a persistent storage directory is created to store information about the last backup. By default this is: `/var/lib/backup-mysql`. To change this the flag `-persistent-storage=/my/alternate/directory` can be passed in.

## Process

Below is a short run-through of what this script does.

The process and code is based on the excellent guide from DigitalOcean Docs: [How To Back Up MySQL Databases to Object Storage with Percona](https://www.digitalocean.com/community/tutorials/how-to-back-up-mysql-databases-to-object-storage-with-percona-on-ubuntu-16-04#creating-the-remote-backup-scripts)

### Backups

When running the following things happen:

1. Check to see if MySQL is installed with the expected version (8.0)
2. Check to see that all necessary software installed (Percona Xtrabackup 8.0)
3. Decide wether a full or an incremental backup is needed by checking for previous runs of this software
4. A [DigitalOcean Block Storage volume](https://www.digitalocean.com/products/block-storage/) is created and mounted. The volume size depends on the MySQL data directory
5. Percona Xtrabackup is run and a compressed backup file is created onto the volume
6. The backup file is uploaded to a DigitalOcean Space for safe storage

### _WIP_ Restoring

## _WIP_ Alerting

Backup failures should not be happen silently. Therefor alerting to Slack is built-in to this project.

# WIP

The following functionality has yet been implemented.

1. Restore functionality
2. Pluggable alert functionality: Slack alert on failure

## For maintainers

We use [`goreleaser.com`](https://goreleaser.com) for release management. Install `goreleaser` as defined here: [goreleaser.com/install](https://goreleaser.com/install/)

To release a new version you need to get a personal access token with the `repo` scope here: [github.com/settings/tokens/new](https://github.com/settings/tokens/new). Remember that token.

To create a new release tag a version and run `goreleaser`:

```shell
git tag 1.0.0
git push --tags
GITHUB_TOKEN=yourtoken goreleaser
```
