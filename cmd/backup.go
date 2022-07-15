package cmd

import (
	"context"
	"fmt"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/spf13/cobra"
)

func init() {
	createVolumeFromArchive.PersistentFlags().String("archive", "", "host path to archive")
	createVolumeFromArchive.PersistentFlags().String("volume", "", "name of the volume to create/populate")
	rootCmd.AddCommand(createVolumeFromArchive)
}

// periodicBackupsCmd represents the add command
var createVolumeFromArchive = &cobra.Command{
	Use:   "create-volume",
	Short: "",
	Long:  ``,
	Run: func(cmd *cobra.Command, args []string) {

		archiveHostPath, err := cmd.PersistentFlags().GetString("archive")
		if err != nil {
			panic(err)
		}

		volumeName, err := cmd.PersistentFlags().GetString("volume")
		if err != nil {
			panic(err)
		}
		if err := cmdCreateVolumeFromArchive(archiveHostPath, volumeName); err != nil {
			panic(err)
		}
	},
}

func cmdCreateVolumeFromArchive(archiveHostPath, volumeName string) error {
	ctx := context.TODO()
	cli, err := client.NewClientWithOpts(client.FromEnv)
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
		Cmd:   []string{"/bin/sh", "-c", "tar -xvzf /archive.tar.gz -C /data --strip-components 1"},
		Image: "ubuntu",
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

	containerName := fmt.Sprintf("backup-%s", RandStringRunes(5))
	body, err := cli.ContainerCreate(ctx, createConfig, hostConfig, networkConfig, platform, containerName)
	if err != nil {
		return err
	}

	return cli.ContainerStart(ctx, body.ID, types.ContainerStartOptions{})
}