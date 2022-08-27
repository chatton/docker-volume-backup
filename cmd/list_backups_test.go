package cmd

import (
	"fmt"
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestGetAllVolumeBackups(t *testing.T) {
	// TODO: fix test
	t.Skip()

	dir := initTestFiles(t)

	vb, err := getAllVolumeBackups(dir, "", false)
	require.NoError(t, err)

	require.Len(t, vb, 3)

	t.Run("newest backups shown first", func(t *testing.T) {
		require.Equal(t, "first-docker-volume-backup_config", vb[0].VolumeName)
		require.Equal(t, "second-docker-volume-backup_metadata", vb[1].VolumeName)
		require.Equal(t, "third-docker-volume-backup_extra", vb[2].VolumeName)
	})
}

func initTestFiles(t *testing.T) string {
	t.Helper()
	dir, err := os.MkdirTemp("", "")
	require.NoError(t, err)

	_, err = os.Create(fmt.Sprintf("%s/first-docker-volume-backup_config-18-7-2022.tar.gz", dir))
	require.NoError(t, err)

	_, err = os.Create(fmt.Sprintf("%s/second-docker-volume-backup_metadata-18-7-2023.tar.gz", dir))
	require.NoError(t, err)

	_, err = os.Create(fmt.Sprintf("%s/third-docker-volume-backup_extra-18-7-2020.tar.gz", dir))
	require.NoError(t, err)

	_, err = os.Create(fmt.Sprintf("%s/invalid-docker-volume-backup_extra-18-7-2020.zip", dir))
	require.NoError(t, err)

	_, err = os.Create(fmt.Sprintf("%s/invalid-docker-volume-backup_extra-187202.tar.gz", dir))
	require.NoError(t, err)

	return dir
}
