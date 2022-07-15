package cmd

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"

	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
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
			// TODO
		})
	})

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
