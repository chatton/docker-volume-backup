package cmd

import (
	"testing"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/stretchr/testify/require"
)

const (
	mountName1 = "mount-1"
	mountName2 = "mount-2"
	mountName3 = "mount-3"
)

func newTestContainer(labels map[string]string, mounts ...types.MountPoint) types.Container {
	if labels == nil {
		labels = map[string]string{}
	}
	return types.Container{
		Mounts: mounts,
		Labels: labels,
	}
}

func TestGetVolumeNamesToBackup(t *testing.T) {
	t.Run("none specified", func(t *testing.T) {
		c := newTestContainer(nil,
			types.MountPoint{
				Type: mount.TypeVolume,
				Name: mountName1,
			},
			types.MountPoint{
				Type: mount.TypeVolume,
				Name: mountName2,
			})
		volumes := getVolumeNamesToBackup(c)
		require.Len(t, volumes, 2, "both volumes should be returned")
	})

	t.Run("host volumes", func(t *testing.T) {
		c := newTestContainer(nil,
			types.MountPoint{
				Type: mount.TypeVolume,
				Name: mountName1,
			},
			types.MountPoint{
				Type: mount.TypeVolume,
				Name: mountName2,
			},
			types.MountPoint{
				Type: mount.TypeBind,
				Name: mountName3,
			})
		volumes := getVolumeNamesToBackup(c)
		require.Len(t, volumes, 2, "both volumes should be returned")
		require.Equal(t, mountName1, volumes[0], "typeVolume should be returned")
		require.Equal(t, mountName2, volumes[1], "typeVolume should be returned")
	})

	t.Run("some specified", func(t *testing.T) {
		labels := map[string]string{
			VolumesLabelKey: mountName1,
		}
		c := newTestContainer(labels,
			types.MountPoint{
				Type: mount.TypeVolume,
				Name: mountName1,
			},
			types.MountPoint{
				Type: mount.TypeVolume,
				Name: mountName2,
			})

		volumes := getVolumeNamesToBackup(c)
		require.Len(t, volumes, 1, "volumes not in label should not be backed up")
		require.Equal(t, mountName1, volumes[0], "volume matching label should be returned")
	})
}
