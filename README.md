## Docker Volume Backup

`docker-volume-backup` is a tool which can be deployed alongside other containers.

By specifying certain labels on containers, the `docker-volume-backup` container
will periodically take backups of the specified volumes as `.tar.gz` archives and
store them on in a specified directory on the docker host.


## Labels

The possible labels which can be applied to containers to configure backups.

| Label                          | Description                                             | Example                    |
|--------------------------------|---------------------------------------------------------|----------------------------|
| `ie.cianhatton.backup.enabled` | Marks the container for volume backups.                 | true                       |
| `ie.cianhatton.backup.volumes` | Comma separated string of volume names to be backed up. | `data_volume,config_volume` |

Note: depending on how your containers are created, the volumes might be named differently. You must ensure that `ie.cianhatton.backup.volumes`
matches the names of the **created** volumes.

## Cobra commands

### periodic-backups

Periodically backs up container volumes based on a provided cron schedule.
An archive is created of the volume contents and is copied to the specified host-path.
Any files in the specified directory older than the specified retention-days will be deleted.

This mode is intended to be deployed alongside other containers and left running.

Usage:
    docker-volume-backup periodic-backups [flags]

Flags:
    --cron string          cron usage
    -h, --help             help for periodic-backups
    --host-path string     backup host path
    --retention-days int   retention days
### create-volume

Creates a docker volume and extracts the contents of the specified archive into it

Usage:
    docker-volume-backup create-volume [flags]

Flags:
    --archive string   host path to archive
    -h, --help             help for create-volume
    --volume string    name of the volume to create/populate

## Requirements

* The `docker-volume-backup` must have access to the host docker socket.

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
