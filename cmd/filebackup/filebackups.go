package filebackup

import (
	"context"
	"fmt"
	"log"

	"docker-volume-backup/cmd/util/dateutil"
	"docker-volume-backup/cmd/util/dockerutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
)

type Mode struct {
	// hostPathForBackups is the path on the host where backups should be stored.
	hostPathForBackups string
}

func NewMode(hostPath string) *Mode {
	return &Mode{
		hostPathForBackups: hostPath,
	}
}

func (f *Mode) CreateBackup(ctx context.Context, cli *client.Client, mountPoint types.MountPoint) error {
	log.Println("performing filesystem backup")
	nameOfBackedupArchive := fmt.Sprintf("%s-%s.tar.gz", mountPoint.Name, dateutil.GetDayMonthYear())
	cmd := []string{"tar", "-czvf", fmt.Sprintf("/backups/%s", nameOfBackedupArchive), "/data"}
	return dockerutil.RunCommandInMountedContainer(ctx, f.hostPathForBackups, cli, mountPoint, cmd)
}
