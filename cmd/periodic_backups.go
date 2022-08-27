package cmd

import (
	"fmt"
	"log"
	"strings"
	"time"

	"docker-volume-backup/cmd/backups"
	"docker-volume-backup/cmd/periodic"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/mount"
	"github.com/go-co-op/gocron"
	"github.com/spf13/cobra"
)

func init() {
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
		periodicConfig, err := periodic.LoadConfig()
		if err != nil {
			log.Fatalf("failed loading config: %s", err)
		}
		cmdPerformBackups(periodicConfig)
	},
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
		log.Printf("running %s [%s] backups with cron schedule: %q", configuration.Name, configuration.ScheduleKey, configuration.Schedule)
		_, err := s.Cron(configuration.Schedule).Do(func() {
			log.Printf("performing %s [%s] backups\n", configuration.Name, configuration.ScheduleKey)
			if len(configuration.Backups) == 0 {
				log.Printf("skipping backup as no backups are specified\n")
				return
			}
			if err := backups.PerformBackups(configuration); err != nil {
				log.Printf("failed performing backups: %s", err)
			}
		})
		if err != nil {
			log.Fatalf("error starting schedule: %s", err)
		}
	}
	s.StartBlocking()
}
