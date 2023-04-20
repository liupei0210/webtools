package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/liupei0210/webtools/timeutil"
	"time"
)

type JsonTime time.Time

func (t JsonTime) MarshalJSON() ([]byte, error) {
	timeString := fmt.Sprintf("\"%s\"", time.Time(t).Format(timeutil.TimeFormat))
	return []byte(timeString), nil
}
func (t *JsonTime) UnmarshalJSON(data []byte) error {
	formatTime, err := time.ParseInLocation(fmt.Sprintf("\"%s\"", timeutil.TimeFormat), string(data), time.Local)
	if err != nil {
		return err
	}
	*t = JsonTime(formatTime)
	return nil
}
func (t *JsonTime) Scan(value interface{}) error {
	times, ok := value.(time.Time)
	if !ok {
		return errors.New(fmt.Sprint("Failed to unmarshal JsonTime value:", value))
	}
	*t = JsonTime(times)
	return nil
}
func (t JsonTime) Value() (driver.Value, error) {
	return marshalJson(t)
}
func marshalJson(t JsonTime) ([]byte, error) {
	timeString := fmt.Sprintf("%s", time.Time(t).Format(timeutil.TimeFormat))
	return []byte(timeString), nil
}

func A2B[B interface{}](a interface{}, handlers ...func(b *B)) (B, error) {
	var b B
	aBytes, err := json.Marshal(a)
	if err != nil {
		return b, err
	}
	err = json.Unmarshal(aBytes, &b)
	if err != nil {
		return b, err
	}
	for _, f := range handlers {
		f(&b)
	}
	return b, nil
}
