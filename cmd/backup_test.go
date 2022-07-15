package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	specs "github.com/opencontainers/image-spec/specs-go/v1"
	"github.com/stretchr/testify/require"
)

const (
	tarFileName = "output.tar.gz"
	volumeName  = "test-volume"
)

var cli *client.Client

func init() {
	var err error
	cli, err = client.NewClientWithOpts(client.FromEnv)
	if err != nil {
		panic(err)
	}

	containersToDelete, err := cli.ContainerList(context.TODO(), types.ContainerListOptions{All: true, Filters: filters.NewArgs(filters.KeyValuePair{
		Key:   "label",
		Value: fmt.Sprintf("%s=%s", TypeLabelKey, LabelTypeTask),
	})})

	for _, c := range containersToDelete {
		log.Printf("deleting existing container: %s\n", c.ID)
		err := cli.ContainerRemove(context.TODO(), c.ID, types.ContainerRemoveOptions{Force: true})
		if err != nil {
			panic(err)
		}
	}

	volumes, err := cli.VolumeList(context.TODO(), filters.Args{})
	for _, v := range volumes.Volumes {
		if v.Name == volumeName {
			err = cli.VolumeRemove(context.TODO(), v.Name, true)
			if err != nil {
				panic(err)
			}
		}
	}
}

func TestCreateVolume(t *testing.T) {
	tarFile := createTarFile(t)
	ctx := context.TODO()

	t.Run("create volume from tar", func(t *testing.T) {
		err := cmdCreateVolumeFromArchive(tarFile, volumeName)
		require.NoError(t, err)

		t.Run("volume created", func(t *testing.T) {
			volumes, err := cli.VolumeList(ctx, filters.Args{})
			require.NoError(t, err)
			found := false
			for _, v := range volumes.Volumes {
				if v.Name == volumeName {
					found = true
					break
				}
			}
			require.True(t, found)
		})

		t.Run("backup container is deleted", func(t *testing.T) {
			f := filters.NewArgs(filters.KeyValuePair{
				Key:   "label",
				Value: fmt.Sprintf("%s=%s", TypeLabelKey, LabelTypeTask),
			})
			containers, err := cli.ContainerList(ctx, types.ContainerListOptions{Filters: f, All: true})
			require.NoError(t, err)
			found := false
			for _, c := range containers {
				for _, n := range c.Names {
					if strings.HasPrefix(n, "/backup-") {
						found = true
					}
				}
			}
			require.False(t, found, fmt.Sprintf("containers: %+v", containers))
		})

		t.Run("volume has correct contents", func(t *testing.T) {
			id := createContainer(t, ctx)
			time.Sleep(10 * time.Second)
			defer func() {
				t.Logf("removing container: %s", id)
				require.NoError(t, cli.ContainerRemove(ctx, id, types.ContainerRemoveOptions{
					Force: true,
				}))
			}()

			resp, err := cli.ContainerExecCreate(ctx, id, types.ExecConfig{
				WorkingDir:   "/data",
				Cmd:          []string{"bash", "-c", "find . -path */T/* | wc -l"},
				AttachStdout: true,
			})
			require.NoError(t, err)

			attach, err := cli.ContainerExecAttach(ctx, resp.ID, types.ExecStartCheck{})
			require.NoError(t, err)

			var (
				waitCh = make(chan struct{})
				resCh  = make(chan string, 1)
			)

			go func() {
				defer close(resCh)
				close(waitCh)
				b, err := ioutil.ReadAll(attach.Reader)
				require.NoError(t, err)
				resCh <- string(b)
			}()
			<-waitCh
			select {
			case <-time.After(10 * time.Second):
				t.Fatal("failed to read the content in time")
			case res := <-resCh:
				// TODO: for now we are just checking that we have 10 files
				// which is what we created in the archive. This can be improved!
				require.True(t, strings.Contains(res, "10"), "Contents: %s", res)
			}
		})
	})

}

func createContainer(t *testing.T, ctx context.Context) string {
	t.Helper()
	createConfig := &container.Config{
		WorkingDir: "/data",
		Cmd:        []string{"sleep", "infinity"},
		Image:      "ubuntu",
	}

	hostConfig := &container.HostConfig{
		Mounts: []mount.Mount{
			// the directory which contains the data to be backed up
			{
				Type:     mount.TypeVolume,
				Source:   volumeName,
				Target:   "/data",
				ReadOnly: false,
			},
		},
	}

	networkConfig := &network.NetworkingConfig{}
	platform := &specs.Platform{}

	containerName := fmt.Sprintf("backup-%s", RandStringRunes(5))
	body, err := cli.ContainerCreate(ctx, createConfig, hostConfig, networkConfig, platform, containerName)
	require.NoError(t, err)
	err = cli.ContainerStart(ctx, body.ID, types.ContainerStartOptions{})
	require.NoError(t, err)
	return body.ID
}

// code adapted from https://gist.github.com/jonmorehouse/9060515
func createTarFile(t *testing.T) string {
	testDir := t.TempDir()
	tarFileFullPathName := fmt.Sprintf("%s/%s", testDir, tarFileName)
	// set up the output file
	file, err := os.Create(tarFileFullPathName)
	require.NoError(t, err)
	defer func() {
		require.NoError(t, file.Close())
	}()
	// set up the gzip writer
	gw := gzip.NewWriter(file)
	defer func() {
		require.NoError(t, gw.Close())
	}()
	tw := tar.NewWriter(gw)
	defer func() {
		require.NoError(t, tw.Close())
	}()

	var tempFiles []string
	for i := 0; i < 10; i++ {
		f, err := ioutil.TempFile("", "")
		require.NoError(t, err)
		tempFiles = append(tempFiles, f.Name())
	}
	for _, f := range tempFiles {
		require.NoError(t, addFile(tw, f))
	}
	return tarFileFullPathName
}

func addFile(tw *tar.Writer, path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()
	if stat, err := file.Stat(); err == nil {
		// now lets create the header as needed for this file within the tarball
		header := new(tar.Header)
		header.Name = path
		header.Size = stat.Size()
		header.Mode = int64(stat.Mode())
		header.ModTime = stat.ModTime()
		// write the header to the tarball archive
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		// copy the file data to the tarball
		if _, err := io.Copy(tw, file); err != nil {
			return err
		}
	}
	return nil
}
