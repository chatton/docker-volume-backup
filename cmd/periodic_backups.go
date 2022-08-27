package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"docker-volume-backup/cmd/backups"
	"docker-volume-backup/cmd/filebackup"
	"docker-volume-backup/cmd/periodic"
	"docker-volume-backup/cmd/s3backup"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/go-co-op/gocron"
	"github.com/spf13/cobra"
)

func init() {
	//periodicBackupsCmd.Flags().String("cron", "", "cron usage")
	//periodicBackupsCmd.Flags().String("host-path", "", "backup host path")
	//periodicBackupsCmd.Flags().String("modes", "filesystem", "specified backup modes")
	//periodicBackupsCmd.Flags().Int("retention-days", 0, "retention days")
	rootCmd.AddCommand(periodicBackupsCmd)
}

// periodicBackupsCmd represents the periodic-backup command.
var periodicBackupsCmd = &cobra.Command{
	Use:   "periodic-backups",
	Short: "periodically backs up containers with volumes",
	Long: `Periodically backs up container volumes based on a provided cron schedule.
An archive is created of the volume contents and is copied to the specified host-path.
Any files in the specified directory older than the specified retention-days will be deleted.

If no volumes are specified under "ie.cianhatton.backup.volumes", all volumes of type
"volume" will be backed up.

This mode is intended to be deployed alongside other containers and left running.

`,
	Run: func(cmd *cobra.Command, args []string) {

		//cron, err := cmd.Flags().GetString("cron")
		//if err != nil {
		//	panic(err)
		//}
		//
		//hostPath, err := cmd.Flags().GetString("host-path")
		//if err != nil {
		//	panic(err)
		//}
		//retainForDays, err := cmd.Flags().GetInt("retention-days")
		//if err != nil {
		//	panic(err)
		//}
		//
		//mode, err := cmd.Flags().GetString("modes")
		//if err != nil {
		//	panic(err)
		//}

		periodicConfig, err := periodic.LoadConfig()
		if err != nil {
			log.Fatalf("failed loading config: %s", err)
		}
		cmdPerformBackups(periodicConfig)
	},
}

func extractBackupModes(bks []periodic.Backup) []backups.BackupMode {
	var backupModes []backups.BackupMode
	for _, item := range bks {
		switch item.Type {
		case "filesystem":
			backupModes = append(backupModes, filebackup.NewMode(item.FilesystemOptions.Hostpath))
		case "s3":
			backupModes = append(backupModes, s3backup.NewMode(item.S3Options.Hostpath, s3backup.Config{
				AwsAccessKeyId:     item.S3Options.AwsAccessKeyID,
				AwsSecretAccessKey: item.S3Options.AwsSecretAccessKey,
				AwsRegion:          item.S3Options.AwsDefaultRegion,
				Bucket:             item.S3Options.AwsBucket,
				Endpoint:           item.S3Options.AwsEndpoint,
			}))
		default:
			panic(fmt.Sprintf("unknown backup modes specified: %s", item))
		}
	}
	return backupModes
}

const (
	DockerLabelPrefix = "ie.cianhatton.backup"
)

func newLabel(s string) string {
	return fmt.Sprintf("%s.%s", DockerLabelPrefix, s)
}

var (
	BackupEnabledLabelKey = newLabel("enabled")
	VolumesLabelKey       = newLabel("volumes")
	TypeLabelKey          = newLabel("type")

	LabelTypeTask = "task"
)

type config struct {
	// hostPathForBackups is the absolute path that where backups will be stored.
	hostPathForBackups string

	// cronSchedule is the cron schedule that backups will run on.
	cronSchedule string

	// retainForDays is the number of days that backups should be stored for.
	retainForDays int

	modes string
}

// getVolumeNamesToBackup extracts a list of volumes to be backed up from
// the container labels.
func getVolumeNamesToBackup(c types.Container) []string {
	volumesStr, ok := c.Labels[VolumesLabelKey]
	if ok {
		return strings.Split(volumesStr, ",")
	}

	// backup all volumes if not are specified
	var volumesToBackup []string
	for _, m := range c.Mounts {
		if m.Type == mount.TypeVolume {
			volumesToBackup = append(volumesToBackup, m.Name)
		}
	}

	return volumesToBackup
}

func cmdPerformBackups(cfg periodic.Config) {
	s := gocron.NewScheduler(time.UTC)
	for _, configuration := range cfg.PeriodicBackups {
		log.Printf("running backups with cron schedule: %q", configuration.Schedule)
		_, err := s.Cron(configuration.Schedule).Do(func() {
			log.Printf("performing %s backups\n", configuration.Name)
			if err := backups.PerformBackups(extractBackupModes(configuration.Backups)...); err != nil {
				log.Printf("failed performing backups: %s", err)
			}
		})
		if err != nil {
			log.Fatalf("error starting schedule: %s", err)
		}
	}
	s.StartBlocking()
}
