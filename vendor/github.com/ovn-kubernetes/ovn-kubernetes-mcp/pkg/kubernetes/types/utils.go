package types

import (
	"fmt"
	"time"
)

// FormatAge formats the age of a resource in a human readable format.
func FormatAge(age time.Duration) string {
	if age < time.Minute {
		return fmt.Sprintf("%ds", int64(age.Seconds()))
	} else if age < time.Hour {
		return fmt.Sprintf("%dm%ds", int64(age.Minutes()), int64(age.Seconds()-float64(int64(age.Minutes())*60)))
	} else if age < time.Hour*24 {
		return fmt.Sprintf("%dh%dm", int64(age.Hours()), int64(age.Minutes()-float64(int64(age.Hours())*60)))
	} else {
		return fmt.Sprintf("%dd%dh", int64(age.Hours()/24), int64(age.Hours()-float64(int64(age.Hours()/24))*24))
	}
}
