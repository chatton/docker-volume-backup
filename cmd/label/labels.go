package label

import (
	"fmt"

	"github.com/docker/docker/api/types/filters"
)

func newLabel(s string) string {
	return fmt.Sprintf("%s.%s", DockerLabelPrefix, s)
}

func newFilterValue(k, v string) string {
	return fmt.Sprintf("%s=%s", k, v)
}

const (
	DockerLabelPrefix = "ie.cianhatton.backup"
)

var (
	BackupEnabledLabelKey = newLabel("enabled")
	VolumesLabelKey       = newLabel("volumes")
	TypeLabelKey          = newLabel("type")
	ScheduleLabelKey      = newLabel("key")
	LabelTypeTask         = "task"
)

func Task() map[string]string {
	return map[string]string{
		TypeLabelKey: LabelTypeTask,
	}
}

func BackupEnabledFilters() filters.Args {
	return filters.NewArgs(filters.KeyValuePair{
		Key:   "label",
		Value: newFilterValue(BackupEnabledLabelKey, "true"),
	})
}

// BackupScheduleFilters returns the labels required for a specific schedule.
func BackupScheduleFilters(scheduleKey string) filters.Args {
	backupEnabled := BackupEnabledFilters()
	backupEnabled.Add(ScheduleLabelKey, scheduleKey)
	return backupEnabled
}
