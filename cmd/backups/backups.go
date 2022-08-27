package backups

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"docker-volume-backup/cmd/label"
	"docker-volume-backup/cmd/util/collectionutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

func PerformBackups(backupModes ...BackupMode) error {
	ctx := context.TODO()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}
	// list all containers with backup enabled
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: label.BackupEnabledFilters(),
		All:     true,
	})
	if err != nil {
		return err
	}

	log.Printf("found %d containers to backup", len(containers))

	_, err = cli.ImagePull(ctx, "busybox:latest", types.ImagePullOptions{})
	if err != nil {
		return err
	}
	log.Printf("successfully pulled busybox image\n")
	time.Sleep(time.Second * 5) // TODO: remove this, wait until the image exists instead.

	for _, c := range containers {
		log.Printf("Stopping container: %s (%s)\n", c.Image, c.ID)
		err := cli.ContainerStop(ctx, c.ID, nil)
		if err != nil {
			return fmt.Errorf("failed sto stop container: %s", err)
		}

		if err := backupContainerMount(ctx, cli, c, backupModes); err != nil {
			return fmt.Errorf("failed processing container: %s", err)
		}

		log.Printf("Starting container: %s (%s)\n", c.Image, c.ID)
		err = cli.ContainerStart(ctx, c.ID, types.ContainerStartOptions{})
		if err != nil {
			return fmt.Errorf("failed to start container: %s", err)
		}
	}
	return nil
}

// backupContainerMount backs up the given mounts for the specified container.
func backupContainerMount(ctx context.Context, cli *client.Client, c types.Container, backupModes []BackupMode) error {
	volumesToBackup := getVolumeNamesToBackup(c)

	for _, m := range c.Mounts {
		if !collectionutil.Contains(volumesToBackup, m.Name) {
			continue
		}

		log.Printf("backing up volume: %s (%s)", m.Name, c.ID)
		for _, bm := range backupModes {
			if err := bm.CreateBackup(ctx, cli, m); err != nil {
				return fmt.Errorf("failed creating backup: %s", err)
			}
		}
	}
	return nil
}

// getVolumeNamesToBackup extracts a list of volumes to be backed up from
// the container labels.
func getVolumeNamesToBackup(c types.Container) []string {
	volumesStr, ok := c.Labels[label.VolumesLabelKey]
	if ok {
		return strings.Split(volumesStr, ",")
	}

	// backup all volumes if not are specified
	var volumesToBackup []string
	for _, m := range c.Mounts {
		if m.Type == mount.TypeVolume {
			volumesToBackup = append(volumesToBackup, m.Name)
		}
	}

	return volumesToBackup
}
