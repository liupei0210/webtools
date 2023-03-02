package timeutil

import (
	"time"
)

const TimeFormat = "2006-01-02 15:04:05"

func ParseTime(t string) time.Time {
	formatTime, err := time.ParseInLocation(TimeFormat, t, time.Local)
	if err != nil {
		formatTime = time.Time{}
	}
	return formatTime
}
func TimeToString(t time.Time) string {
	return t.Format(TimeFormat)
}
