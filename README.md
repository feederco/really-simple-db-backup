# really-simple-db-backup

The goal of this project is to be a really simple backup solution where you can just drop a binary on your database server, setup a cronjob to run at regular intervals, and it will take care of performing full and incremental backups at a regular intervals and uploading them to a cloud bucket.

This project also aims to simplify restoring backups. The goal is for it to be hassle free and require little mental overhead in the greatest time of needs.

Currently this project is built to perform backups of MySQL 8.0 and store them on a [DigitalOcean Space](https://www.digitalocean.com/products/spaces/). The software for actually generating the backup file is the amazing [Percona Xtrabackup](https://www.percona.com/software/mysql-database/percona-xtrabackup).

## Usage

### Download release

Latest release can be found on [releases page](https://github.com/feederco/really-simple-db-backup/releases).

```
ssh your-server
wget https://github.com/feederco/really-simple-db-backup/releases/download/$VERSION/really-simple-db-backup_$VERSION_$PLATFORM_$ARCH.tar.gz -o really-simple-db-backup.tar.gz
tar xvf really-simple-db-backup.tar.gz
sudo mv really-simple-db-backup /usr/bin/really-simple-db-backup
```

### Build & upload binary

```
git clone git@github.com:feederco/really-simple-db-backup.git
cd really-simple-db-backup
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o build/really-simple-db-backup main.go
scp build/really-simple-db-backup my-db-host:/usr/bin/really-simple-db-backup
```

### Setup Cronjob

```shell
crontab -e
```

And add the following:

#### For daily backups

Set to run 05:00 AM every day.

```
SHELL=/bin/sh
PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin
0 5 * * * /usr/bin/really-simple-db-backup perform > /dev/null
```

#### For hourly backups

```
SHELL=/bin/sh
PATH=/usr/local/sbin:/usr/local/bin:/sbin:/bin:/usr/sbin:/usr/bin
1 * * * * /usr/bin/really-simple-db-backup perform > /dev/null
```

### Available commands

```shell
really-simple-db-backup THE_COMMAND
```

- [`perform`](#perform)
- [`perform-full`](#perform-full)
- [`perform-incremental`](#perform-incremental)
- [`restore`](#restore)
- [`upload`](#upload)
- [`download`](#download)
- [`finalize-restore`](#finalize-restore)
- [`test-alert`](#test-alert)
- [`list-backups`](#list-backups)
- [`prune`](#prune)

### Perform backup

Performs either a full or incremental backup by checking for previous runs.

```shell
really-simple-db-backup perform
```

`/etc/really-simple-db-backup.json`

```json
{
  "digitalocean": {
    "key": "digitalocean-app-token",
    "space_endpoint": "fra1.digitaloceanspaces.com",
    "space_name": "my-backups",
    "space_key": "auth-key-for-space",
    "space_secret": "auth-secret-for-space"
  },
  "mysql": {
    "data_path": "(optional)"
  },
  "persistent_storage": "(optional)",
  "alerting": {
    "slack": {
      "webhook_url": "https://hooks.slack.com/services/<your-webhook-url>"
    }
  },
  "retention": {
    "automatically_remove_old": true,
    "retention_in_days": 7,
    "hours_between_full_backups": 24
  }
}
```

#### Perform without config file

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
really-simple-db-backup perform-full
```

To force an incremental backup you can use the `perform-incremental` command.

```shell
really-simple-db-backup perform-incremental
```

### Force upload

If for some reason a backup failed and you were successfully able to retrieve a backup yourself, you can use the `upload` command to upload this to your DigitalOcean Space.

```shell
really-simple-db-backup upload -file /path/to/backup.xbstream
```

### Download and prepare backup without moving it back

`restore` will download a backup to a new volume, extract it, decompress it, run [Xtrabackup's prepare](https://www.percona.com/doc/percona-xtrabackup/8.0/backup_scenarios/full_backup.html#preparing-a-backup) command. When these steps are completed the backup is ready to be moved to the MySQL `datadir` and after that MySQL is ready to start again.

If you just want to download and prepare a backup, you can use the `download` command. This is good if you just want to

```shell
really-simple-db-backup download -hostname my-other-host
```

### Put back after `download`

If you have run the `download` command and have a fully prepared backup that you now wish to use, you can run the `finalize-restore` command which will run the second half of steps that are run by the `restore` command.

```shell
really-simple-db-backup finalize-restore -existing-restore-directory=/mnt/my_restore_volume/really-simple-db-restore
```

**Note**

### Remove old backups

If the config-option `retention.automatically_remove_old` is set to `true`, an automatic prune will be run on each full backup. Backup lineages older than `retention.retention_in_days` (or `retention.retention_in_hours`).

To force a prune the `prune` command can be run:

```
really-simple-db-backup prune
```

To prune on another host the `-hostname` flag can be passed in:

```
really-simple-db-backup prune -hostname other-host
```

### Test alert

To make sure the Slack integration is setup correctly you can use the `test-alert` command to run the same code path that will be executed on a critical error.

```shell
really-simple-db-backup test-alert
```

### Listing existing backups

To list all backups for the current host you can run the `list-backups` command. This is also a good way to test that your access tokens for cloud storage is correct.

```shell
really-simple-db-backup list-backups
```

You can also run this on another host to check the backups of a specific host:

```shell
really-simple-db-backup list-backups -hostname my-other-host
```

To see if there are any backups since a certain timestamp simply pass in the timestamp (as formatted in the backup filenames: `YYYYMMDDHHII`)

```shell
really-simple-db-backup list-backups -timestamp 201901050000
```

## Configuration

By default the script checks for the existence of a config file at `/etc/really-simple-db-backup.json`. If this is found the defaults are loaded from that file and can be overriden by command line options.

If this config file is located somewhere else you can pass that in with the `-config` option.

```shell
really-simple-db-backup perform -config ./my-other-config.json
```

The following format is expected:

```json
{
  "digitalocean": {
    "key": "digitalocean-app-token",
    "space_endpoint": "fra1.digitaloceanspaces.com",
    "space_name": "my-backups",
    "space_key": "auth-key-for-space",
    "space_secret": "auth-secret-for-space"
  },
  "mysql": {
    "data_path": "(optional)"
  },
  "persistent_storage": "(optional)",
  "alerting": {
    "slack": {
      "webhook_url": "https://hooks.slack.com/services/<your-webhook-url>"
    }
  },
  "retention": {
    "automatically_remove_old": true,
    "retention_in_days": 7,
    "hours_between_full_backups": 24
  }
}
```

#### `retention`

If the `retention` option is left empty (or `null`) no pruning is done.

#### `automatically_remove_old`

Set this to `true` for the pruning to be run on each full backup. If it is set to `false` (or not set at all) you need to manually run `really-simple-db-backup prune` to remove old backups.

#### `retention_in_days`

Set to the number of days a backup is kept before being considered for removal.

#### `retention_in_hours`

If you want more fine-grained control of how old backups are kept, use the `retention_in_hours` option instead. If any of the above values are set to `0` (or not included in the config JSON), the other value is used.

#### `hours_between_full_backups`

Set to number of hours between full backups. Note: This does perform the actually scheduling of this command. You need to do that separately in a cronjob or similar. See the section

### Different MySQL data directory

The default directory for MySQL is normally `/var/lib/mysql`. If you have mounted a volume for your data and set different [`datadir`](https://dev.mysql.com/doc/refman/8.0/en/data-directory.html) you can pass in the following option: `-mysql-data-path=/mnt/my_mysql_volume/mysql` or set the `"mysql.data_path"` config property in the JSON config.

### Persistent storage directory

To save state between runs a persistent storage directory is created to store information about the last backup. By default this is: `/var/lib/backup-mysql`. To change this the flag `-persistent-storage=/my/alternate/directory` can be passed in or set the `"persistent_storage"` config property in the JSON config.

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

### Restoring

1. Fetch all backups for the given host on the [DigitalOcean Space](https://www.digitalocean.com/products/spaces/)
2. Find the one that is a best match for the passed in timestamp
4. A [DigitalOcean Block Storage volume](https://www.digitalocean.com/products/block-storage/) is created and mounted. The volume size depends on the file found in the
5. Download & extract all pieces for the backup to this volume
6. Decompress the backup
7. Perform `Xtrabackup`'s prepare command which prepares it for use
8. Move all files back to the MySQL data path

Starting MySQL is up to you when the process is finished.

## Alerting

Backup failures should not be happen silently. Therefor alerting to Slack is built-in to this project.

### Slack

You need to create a `Custom Integration` in your Slack channel with the type `Incoming WebHook`. We recommend creating a separate channel with must-action messages.

![](https://i.imgur.com/tkYCqSW.png?1)

#### Config options

The `slack` entry in the config file can have the following options:

```
{
  "webhook_url": "webhook URL",
  "channel": "Override default channel to share to",
  "username": "Override username (default: BackupsBot)",
  "icon_emoji": "Override avatar of bot (default: :card_file_box: 🗃)",
}
```

## For maintainers

We use [`goreleaser.com`](https://goreleaser.com) for release management. Install `goreleaser` as defined here: [goreleaser.com/install](https://goreleaser.com/install/)

To release a new version you need to get a personal access token with the `repo` scope here: [github.com/settings/tokens/new](https://github.com/settings/tokens/new). Remember that token.

To create a new release tag a version and run `goreleaser`:

```shell
git tag 1.0.0
git push --tags
GITHUB_TOKEN=yourtoken goreleaser
```

### Issues to work on

All issue management is on our [Github issues](https://github.com/feederco/really-simple-db-backup/issues).

Check the issues tagged [`help wanted`](https://github.com/feederco/really-simple-db-backup/issues?q=is%3Aissue+is%3Aopen+label%3A%22help+wanted%22) for good tickets to work on.
