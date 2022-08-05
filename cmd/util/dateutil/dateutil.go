package dateutil

import (
	"fmt"
	"time"
)

func GetDayMonthYear() string {
	t := time.Now()
	return fmt.Sprintf("%d-%d-%d", t.Day(), t.Month(), t.Year())
}
