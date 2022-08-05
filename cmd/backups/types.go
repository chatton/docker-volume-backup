package backups

import (
	"context"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type Config struct {
	// HostPathForBackups is the absolute path that where backups will be stored.
	HostPathForBackups string

	// CronSchedule is the cron schedule that backups will run on.
	CronSchedule string

	// RetainForDays is the number of days that backups should be stored for.
	RetainForDays int

	Mode string
}

type BackupMode interface {
	CrateBackup(ctx context.Context, cli *client.Client, mountPoint types.MountPoint) error
}
