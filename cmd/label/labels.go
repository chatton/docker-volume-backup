package label

import "fmt"

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

	LabelTypeTask = "task"
)

func Task() map[string]string {
	return map[string]string{
		TypeLabelKey: LabelTypeTask,
	}
}
