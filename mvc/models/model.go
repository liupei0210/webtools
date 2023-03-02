package models

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/liupei0210/webtools/timeutil"
	log "github.com/sirupsen/logrus"
	"go.mongodb.org/mongo-driver/mongo"
	"reflect"
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

func Model2DTO[DTO interface{}](model interface{}, handlers ...func(d *DTO)) (*DTO, error) {
	var dto DTO
	modelJ, err := json.Marshal(model)
	if err != nil {
		return nil, err
	}
	err = json.Unmarshal(modelJ, &dto)
	if err != nil {
		return nil, err
	}
	for _, f := range handlers {
		f(&dto)
	}
	return &dto, nil
}
func AssembleMongoCursor(cur *mongo.Cursor, slicePtr interface{}) {
	src := reflect.ValueOf(slicePtr).Elem()
	elementType := reflect.TypeOf(slicePtr).Elem().Elem()
	arr := make([]reflect.Value, 0)
	for cur.Next(context.Background()) {
		e := reflect.New(elementType)
		err := cur.Decode(e.Interface())
		if err != nil {
			log.Error(err)
		}
		arr = append(arr, e.Elem())
	}
	dest := reflect.Append(src, arr...)
	src.Set(dest)
}
