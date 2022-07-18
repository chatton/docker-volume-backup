package cmd

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

func init() {
	restoreBackupsCommand.Flags().String("host-path", "", "backup host path")
	restoreBackupsCommand.Flags().String("volumes", "", "comma separated list of volumes to restore, default to all found volumes")
	if err := restoreBackupsCommand.MarkFlagRequired("host-path"); err != nil {
		panic(err)
	}
	rootCmd.AddCommand(restoreBackupsCommand)
}

type backupRestoreArgs struct {
	hostPath string
	volumes  []string
}

// restoreBackupsCommand restores backups.
var restoreBackupsCommand = &cobra.Command{
	Use:   "restore-backups",
	Short: "restore existing backups",
	Long: `Restore backups from a directory.

Specify a directory where backups are located (host-path) and a comma separated
list of volumes (vol1,vol2,vol3) etc. 

Docker volumes will be created from all of the backups. If there are multiple backups
of the same volume, the newest will be chosen.
`,
	Run: func(cmd *cobra.Command, args []string) {
		hostDir, err := cmd.Flags().GetString("host-path")
		if err != nil {
			panic(err)
		}
		volumes, err := cmd.Flags().GetString("volumes")
		if err != nil {
			panic(err)
		}
		backupArgs := backupRestoreArgs{
			hostPath: hostDir,
			volumes:  strings.Split(volumes, ","),
		}
		if err := cmdRestoreBackup(backupArgs); err != nil {
			panic(err)
		}
	},
}

type restoreOutput struct {
	RestoredFrom string    `json:"restoredFrom"`
	VolumeName   string    `json:"volumeName"`
	RestoreTime  time.Time `json:"restoreTime"`
}

func cmdRestoreBackup(args backupRestoreArgs) error {
	allBackups, err := getAllVolumeBackups(args.hostPath)
	if err != nil {
		return err
	}

	volumesBackedUp := map[string]struct{}{}
	// only backup volumes that have been specified if any.
	if len(args.volumes) > 0 && args.volumes[0] != "" {
		var volumesToBackup []backedUpVolume
		for _, b := range allBackups {
			if contains(args.volumes, b.VolumeName) {
				volumesToBackup = append(volumesToBackup, b)
			}
		}
		allBackups = volumesToBackup
	}

	result := []restoreOutput{}
	for _, b := range allBackups {
		_, alreadyRestored := volumesBackedUp[b.VolumeName]
		if alreadyRestored {
			continue
		}
		if err := cmdCreateVolumeFromArchive(b.AbsoluteFilePath, b.VolumeName); err != nil {
			return err
		}
		volumesBackedUp[b.VolumeName] = struct{}{}
		result = append(result, restoreOutput{
			RestoredFrom: b.AbsoluteFilePath,
			VolumeName:   b.VolumeName,
			RestoreTime:  time.Now(),
		})
	}

	bytes, err := json.Marshal(result)
	if err != nil {
		return err
	}
	fmt.Println(string(bytes))
	return nil
}
