package dockerutil

import (
	"context"
	"fmt"

	"docker-volume-backup/cmd/label"
	"docker-volume-backup/cmd/util/randutil"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/docker/errdefs"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
)

// RunCommandInMountedContainer creates a container which mounts the data to be backed up, and
// executes a given command.
func RunCommandInMountedContainer(ctx context.Context, hostPathForBackups string, cli *client.Client, mountPoint types.MountPoint, cmd []string) error {
	createConfig := &container.Config{
		Cmd:    cmd,
		Image:  "busybox:latest",
		Labels: label.Task(),
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
				Source:   hostPathForBackups,
				Target:   "/backups",
				ReadOnly: false,
			},
		},
	}

	networkConfig := &network.NetworkingConfig{}
	platform := &specs.Platform{}

	containerName := fmt.Sprintf("backup-%s-%s", mountPoint.Name, randutil.StringRunes(5))
	body, err := cli.ContainerCreate(ctx, createConfig, hostConfig, networkConfig, platform, containerName)

	if err != nil {
		return err
	}

	err = cli.ContainerStart(ctx, body.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}
	return WaitForContainerToExit(ctx, cli, body)
}

func WaitForContainerToExit(ctx context.Context, cli *client.Client, body container.ContainerCreateCreatedBody) error {
	resultC, errC := cli.ContainerWait(ctx, body.ID, container.WaitConditionNotRunning)
	select {
	case result := <-resultC:
		msg := fmt.Sprintf("container %s exited with code: %d", body.ID, result.StatusCode)
		if result.StatusCode == 0 {
			return cli.ContainerRemove(ctx, body.ID, types.ContainerRemoveOptions{
				RemoveVolumes: false,
			})
		}
		return fmt.Errorf(msg)
	case err := <-errC:
		if !errdefs.IsSystem(err) {
			return fmt.Errorf("expected a Server Error, got %[1]T: %[1]v", err)
		}
	}
	return nil
}
