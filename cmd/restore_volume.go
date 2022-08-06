package cmd

import (
	"context"
	"fmt"
	"os"

	"docker-volume-backup/cmd/s3backup"
	"docker-volume-backup/cmd/util/dockerutil"
	"docker-volume-backup/cmd/util/randutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
)

const (
	volumeFlag  = "volume"
	s3Mode      = "s3"
	s3KeyFlag   = "s3key"
	archiveFlag = "archive"
)

func init() {
	restoreOrCreateVolume.Flags().String(archiveFlag, "", "host path to archive")
	restoreOrCreateVolume.Flags().String(s3KeyFlag, "", "specific s3Key to restore")
	restoreOrCreateVolume.Flags().Bool(s3Mode, false, "look in s3 for backup")
	restoreOrCreateVolume.Flags().String(volumeFlag, "", "name of the volume to create/populate")

	if err := restoreOrCreateVolume.MarkFlagRequired(volumeFlag); err != nil {
		panic(err)
	}
	restoreOrCreateVolume.MarkFlagsMutuallyExclusive(s3KeyFlag, archiveFlag)
	rootCmd.AddCommand(restoreOrCreateVolume)
}

// restoreOrCreateVolume creates a docker volume and pre-populates it with
// data from a specified archive.
var restoreOrCreateVolume = &cobra.Command{
	Use:   "restore-volume",
	Short: "create a pre-populated volume or restore an existing one.",
	Long:  "Creates a docker volume and extracts the contents of the specified archive into it or restores an existing volume",
	Run: func(cmd *cobra.Command, args []string) {

		archiveHostPath, err := cmd.Flags().GetString(archiveFlag)
		if err != nil {
			panic(err)
		}

		s3Key, err := cmd.Flags().GetString(s3KeyFlag)
		if err != nil {
			panic(err)
		}

		volumeName, err := cmd.Flags().GetString(volumeFlag)
		if err != nil {
			panic(err)
		}

		useS3, err := cmd.Flags().GetBool(s3Mode)
		if err != nil {
			panic(err)
		}

		if useS3 || s3Key != "" {
			// no s3key specified, so we must try and find the newest backup.
			if s3Key == "" {
				obj, err := s3backup.FindMostRecentBackupForVolume(volumeName)
				if err != nil {
					panic(err)
				}
				s3Key = *obj.Key
			}

			fileName := fmt.Sprintf("/tmp/%s", s3Key)
			f, err := os.Create(fileName)
			if err != nil {
				panic(err)
			}
			defer func() {
				_ = os.Remove(f.Name())
			}()

			if err := s3backup.DownloadFromS3(s3Key, f); err != nil {
				panic(err)
			}
			archiveHostPath = f.Name()
		}

		if err := cmdRestoreVolumeFromArchive(archiveHostPath, volumeName); err != nil {
			panic(err)
		}
	},
}

func cmdRestoreVolumeFromArchive(archiveHostPath, volumeName string) error {
	ctx := context.TODO()
	cli, err := client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		return err
	}

	_, err = cli.ImagePull(ctx, "ubuntu:latest", types.ImagePullOptions{})
	if err != nil {
		return err
	}

	vol, err := cli.VolumeCreate(ctx, volume.VolumeCreateBody{
		Name: volumeName,
	})
	if err != nil {
		return err
	}

	createConfig := &container.Config{
		WorkingDir: "/data",
		// --strip-components 1 to remove the directory, so that the files of the archive are at the root.
		Cmd:   []string{"/bin/sh", "-c", "rm -rf /data/* && tar -xvzf /archive.tar.gz -C /data --strip-components 1"},
		Image: "ubuntu",
		Labels: map[string]string{
			TypeLabelKey: LabelTypeTask,
		},
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			// the directory which contains the data to be backed up
			{
				Type:     mount.TypeVolume,
				Source:   vol.Name,
				Target:   "/data",
				ReadOnly: false,
			},
			{
				Type:     mount.TypeBind,
				Source:   archiveHostPath,
				Target:   "/archive.tar.gz",
				ReadOnly: false,
			},
		},
	}

	networkConfig := &network.NetworkingConfig{}
	platform := &specs.Platform{}

	containerName := fmt.Sprintf("backup-%s", randutil.StringRunes(5))
	body, err := cli.ContainerCreate(ctx, createConfig, hostConfig, networkConfig, platform, containerName)
	if err != nil {
		return err
	}

	err = cli.ContainerStart(ctx, body.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	// once the container has completed, it should be removed.
	return dockerutil.WaitForContainerToExit(ctx, cli, body)
}
