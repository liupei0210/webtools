package timeutil

import (
	"time"
)

func ParseTime(t string) time.Time {
	formatTime, err := time.ParseInLocation(time.DateTime, t, time.Local)
	if err != nil {
		formatTime = time.Time{}
	}
	return formatTime
}
func TimeToString(t time.Time) string {
	return t.Format(time.DateTime)
}
