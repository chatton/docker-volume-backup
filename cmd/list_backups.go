package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"github.com/spf13/cobra"
)

var rxp *regexp.Regexp

func init() {
	rxp = regexp.MustCompile("(.*)-\\d+-\\d+-\\d{4}.*.tar.gz$")
	listBackupsCommand.Flags().String("host-path", "", "backup host path")
	listBackupsCommand.Flags().String("volume-name-filter", "", "string volume name must contain")
	listBackupsCommand.Flags().Bool("newest-only", false, "return only 1 backup per volume")
	if err := listBackupsCommand.MarkFlagRequired("host-path"); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(listBackupsCommand)
}

// restoreOrCreateVolumeFromArchive creates a docker volume and pre-populates it with
// data from a specified archive.
var listBackupsCommand = &cobra.Command{
	Use:   "list-backups",
	Short: "list existing backups",
	Long:  "List backups that exist in the specified host directory",
	Run: func(cmd *cobra.Command, args []string) {
		hostDir, err := cmd.Flags().GetString("host-path")
		if err != nil {
			panic(err)
		}
		volumeNameFilter, err := cmd.Flags().GetString("volume-name-filter")
		if err != nil {
			panic(err)
		}
		newestOnly, err := cmd.Flags().GetBool("newest-only")
		if err != nil {
			panic(err)
		}

		if err := cmdListBackups(hostDir, volumeNameFilter, newestOnly); err != nil {
			panic(err)
		}
	},
}

// backedUpVolume holds information about a volume backup.
type backedUpVolume struct {
	VolumeName       string    `json:"volumeName"`
	AbsoluteFilePath string    `json:"absoluteFilePath"`
	FileName         string    `json:"fileName"`
	LastModTime      time.Time `json:"lastModTime"`
}

func getAllVolumeBackups(hostDir string, volumeNameFilter string, newestOnly bool) ([]backedUpVolume, error) {
	var result []backedUpVolume
	err := filepath.Walk(hostDir, func(filePath string, info os.FileInfo, err error) error {
		if info.IsDir() {
			return nil
		}
		fileName := path.Base(filePath)
		if !rxp.Match([]byte(fileName)) {
			return nil
		}

		match := rxp.FindAllStringSubmatch(fileName, -1)
		dockerVolumeName := match[0][1]

		result = append(result, backedUpVolume{
			VolumeName:       dockerVolumeName,
			AbsoluteFilePath: filePath,
			LastModTime:      info.ModTime(),
			FileName:         info.Name(),
		})

		return nil
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].LastModTime.After(result[j].LastModTime)
	})

	// we only want to see one of each volume. They have already
	// been sorted by time, so we just take the first of each name.
	if newestOnly {
		seenVolumes := map[string]struct{}{}
		var filteredResult []backedUpVolume
		for _, b := range result {
			if _, seenAlready := seenVolumes[b.VolumeName]; seenAlready {
				continue
			}
			filteredResult = append(filteredResult, b)
			seenVolumes[b.VolumeName] = struct{}{}
		}
		return filteredResult, nil
	}

	return result, nil
}

// cmdListBackups outputs a list of backups in the given directory.
// e.g.
/*
[
  {
    "volumeName": "docker-volume-backup_config",
    "absoluteFilePath": "/var/folders/k1/z859m4qx7ld12gdjfv_79tv80000gn/T/tmp.9a5CNbHp/docker-volume-backup_config-18-7-2022.tar.gz",
    "fileName": "docker-volume-backup_config-18-7-2022.tar.gz",
    "lastModTime": "2022-07-18T19:00:07.553274229+01:00"
  },
  {
    "volumeName": "docker-volume-backup_metadata",
    "absoluteFilePath": "/var/folders/k1/z859m4qx7ld12gdjfv_79tv80000gn/T/tmp.9a5CNbHp/docker-volume-backup_metadata-18-7-2022.tar.gz",
    "fileName": "docker-volume-backup_metadata-18-7-2022.tar.gz",
    "lastModTime": "2022-07-18T19:00:08.607626133+01:00"
  }
]
*/
func cmdListBackups(hostDir string, filter string, newestOnly bool) error {
	result, err := getAllVolumeBackups(hostDir, filter, newestOnly)
	if err != nil {
		return err
	}
	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))
	return nil
}
