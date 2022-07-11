package cmd

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	"github.com/go-co-op/gocron"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
)

func init() {
	periodicBackupsCmd.PersistentFlags().String("cron", "", "cron usage")
	periodicBackupsCmd.PersistentFlags().String("host-path", "", "backup host path")
	periodicBackupsCmd.PersistentFlags().Int("retention-days", 0, "retention days")
	rootCmd.AddCommand(periodicBackupsCmd)
}

// periodicBackupsCmd represents the add command
var periodicBackupsCmd = &cobra.Command{
	Use:   "periodic-backups",
	Short: "add values passed to function",
	Long:  `Demo application to demonstrate cobra featues`,
	Run: func(cmd *cobra.Command, args []string) {

		cron, err := cmd.PersistentFlags().GetString("cron")
		if err != nil {
			panic(err)
		}

		hostPath, err := cmd.PersistentFlags().GetString("host-path")
		if err != nil {
			panic(err)
		}
		retainForDays, err := cmd.PersistentFlags().GetInt("retention-days")
		if err != nil {
			panic(err)
		}
		cmdPerformBackups(config{
			hostPathForBackups: hostPath,
			cronSchedule:       cron,
			retainForDays:      retainForDays,
		})
	},
}

const (
	DockerLabelPrefix = "ie.cianhatton.backup"
)

func newLabel(s string) string {
	return fmt.Sprintf("%s.%s", DockerLabelPrefix, s)
}

func newFilterValue(k, v string) string {
	return fmt.Sprintf("%s=%s", k, v)
}

var (
	BackupEnabledLabelKey = newLabel("enabled")
	VolumesLabelKey       = newLabel("volumes")
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func backupEnabledFilters() filters.Args {
	return filters.NewArgs(filters.KeyValuePair{
		Key:   "label",
		Value: newFilterValue(BackupEnabledLabelKey, "true"),
	})
}

func contains[T comparable](elems []T, v T) bool {
	for _, e := range elems {
		if v == e {
			return true
		}
	}
	return false
}

type config struct {
	// hostPathForBackups is the absolute path that where backups will be stored.
	hostPathForBackups string

	// cronSchedule is the cron schedule that backups will run on.
	cronSchedule string

	// retainForDays is the number of days that backups should be stored for.
	retainForDays int
}

// runCommandInMountedContainer creates a container which mounts the data to be backed up, and
// executes a given command.
func runCommandInMountedContainer(ctx context.Context, cfg config, cli *client.Client, mountPoint types.MountPoint, cmd []string) error {
	createConfig := &container.Config{
		Cmd:   cmd,
		Image: "busybox:latest",
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			// the directory which contains the data to be backed up
			{
				Type:     mount.TypeVolume,
				Source:   mountPoint.Name,
				Target:   "/data",
				ReadOnly: false,
			},
			// hostpath where the backups will be stored
			{
				Type:     mount.TypeBind,
				Source:   cfg.hostPathForBackups,
				Target:   "/backups",
				ReadOnly: false,
			},
		},
	}

	networkConfig := &network.NetworkingConfig{}
	platform := &specs.Platform{}

	containerName := fmt.Sprintf("backup-%s-%s", mountPoint.Name, RandStringRunes(5))
	body, err := cli.ContainerCreate(ctx, createConfig, hostConfig, networkConfig, platform, containerName)

	if err != nil {
		return err
	}

	err = cli.ContainerStart(ctx, body.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	resultC, errC := cli.ContainerWait(ctx, body.ID, container.WaitConditionNotRunning)
	select {
	case result := <-resultC:
		msg := fmt.Sprintf("container %s existed with code: %d", body.ID, result.StatusCode)
		if result.StatusCode == 0 {
			return cli.ContainerRemove(ctx, body.ID, types.ContainerRemoveOptions{
				RemoveVolumes: false,
			})
		}
		return fmt.Errorf(msg)
	case err := <-errC:
		if !errdefs.IsSystem(err) {
			log.Fatalf("expected a Server Error, got %[1]T: %[1]v", err)
		}
	}
	return nil
}

// deleteOldBackups deletes backups that are older than a certain age.
func deleteOldBackups(ctx context.Context, cfg config, cli *client.Client, mountPoint types.MountPoint) error {
	cmd := []string{"find", "backups", "-mtime", fmt.Sprintf("+%d", cfg.retainForDays), "-delete"}
	return runCommandInMountedContainer(ctx, cfg, cli, mountPoint, cmd)
}

// performBackup creates a container which mounts the data to be backed up, and creates an archive
// of the data in the specified hostpath.
func performBackup(ctx context.Context, cfg config, cli *client.Client, mountPoint types.MountPoint) error {
	now := strings.ReplaceAll(time.Now().Format(time.RFC3339), ":", "-")
	nameOfBackedupArchive := fmt.Sprintf("%s-%s.tar.gz", mountPoint.Name, now)
	cmd := []string{"tar", "-czvf", fmt.Sprintf("/backups/%s", nameOfBackedupArchive), "/data"}
	return runCommandInMountedContainer(ctx, cfg, cli, mountPoint, cmd)
}

// handleContainerMount backs up the given mounts for the specified container.
func handleContainerMount(ctx context.Context, cfg config, cli *client.Client, c types.Container) error {
	oldBackupDeleted := false
	volumesToBackup := extractVolumeNamesFromLabels(c.Labels)
	for _, m := range c.Mounts {
		if contains(volumesToBackup, m.Name) {
			log.Printf("backing up volume: %s (%s)", m.Name, c.ID)
			err := performBackup(ctx, cfg, cli, m)
			if err != nil {
				return fmt.Errorf("failed backup: %s", err)
			}

			if cfg.retainForDays > 0 && !oldBackupDeleted {
				log.Printf("removing backups older than %d days\n", cfg.retainForDays)
				if err := deleteOldBackups(ctx, cfg, cli, m); err != nil {
					return fmt.Errorf("failed removing old backups: %s", err)
				}
				oldBackupDeleted = true
			}
		}
	}
	return nil
}

// extractVolumeNamesFromLabels extracts a list of volumes to be backed up from
// the container labels.
func extractVolumeNamesFromLabels(labels map[string]string) []string {
	volumesStr, ok := labels[VolumesLabelKey]
	if !ok {
		log.Printf("key %s not specifed, no volumes will be backed up.", VolumesLabelKey)
	}
	return strings.Split(volumesStr, ",")
}

func performBackups(cfg config) error {
	ctx := context.TODO()

	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}
	// list all containers with backup enabled
	containers, err := cli.ContainerList(context.Background(), types.ContainerListOptions{
		Filters: backupEnabledFilters(),
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
		if len(extractVolumeNamesFromLabels(c.Labels)) == 0 {
			log.Printf("container: %s (%s) does not have any volumes specified, skipping backup", c.Image, c.ID)
			continue
		}

		log.Printf("Stopping container: %s (%s)\n", c.Image, c.ID)
		err := cli.ContainerStop(ctx, c.ID, nil)
		if err != nil {
			return fmt.Errorf("failed sto stop container: %s", err)
		}

		if err := handleContainerMount(ctx, cfg, cli, c); err != nil {
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

func cmdPerformBackups(cfg config) {
	s := gocron.NewScheduler(time.UTC)
	log.Printf("running backups with cron schedule: %q", cfg.cronSchedule)
	_, err := s.Cron(cfg.cronSchedule).Do(func() {
		log.Println("performing backups")
		if err := performBackups(cfg); err != nil {
			log.Printf("error performing backup: %s\n", err)
		}
	})
	if err != nil {
		log.Fatalf("error starting schedule: %s", err)
	}
	s.StartBlocking()
}
