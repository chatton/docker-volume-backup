## Docker Volume Backup

`docker-volume-backup` is a tool which can be deployed alongside other containers.

By specifying certain labels on containers, the `docker-volume-backup` container
will periodically take backups of the specified volumes as `.tar.gz` archives and
store them on in a specified directory on the docker host.


## Labels

The possible labels which can be applied to containers to configure backups.

| Label                         | Description                                             | Example                    |
|-------------------------------|---------------------------------------------------------|----------------------------|
| `ie.cianhatton.backup.enabled` | Marks the container for volume backups.                 | true                       |
| `ie.cianhatton.backup.volumes`                     | Comma separated string of volume names to be backed up. | `data_volume,config_volume` |


Note: depending on how your containers are created, the volumes might be named differently. You must ensure that ``ie.cianhatton.backup.volumes``
matches the names of the **created** volumes.

## Environment Variables

Environment variables that must be configured for the `docker-volume-backup` container.

| Environment Variable           | Description                                                              | Example                |
|--------------------------------|--------------------------------------------------------------------------|------------------------|
| `CRON_SCHEDULE` | The cron schedule for when backups should run.                           | `* * * * *`              |
| `BACKUP_HOST_PATH` | The absolute path on the docker host for where backups should be stored. | `/Users/chatton/backups` |


## Requirements

* The `docker-volume-backup` must have access to the host docker socket. 
* The specified backup directory should be mounted into the `docker-volume-backup` container. 

See [this example](./docker-compose.yml)


## Example



```bash
# make some temporary directories
export BACKUP_HOST_PATH="$(mktemp -d)"
export AUDIO_BOOKS_DIRECTORY="$(mktemp -d)"
export PODCASTS_DIRECTORY="$(mktemp -d)"

# start the containers
docker compose up -d 
```

After this, you will notice that every minute (as specified by the `CRON_SCHEDULE` ) the `audiobookshelf` is stopped, backups are created of the two specified
volumes, and then it is restarted.

we can see the backups with.

```bash
ls ${BACKUP_HOST_PATH}
```
