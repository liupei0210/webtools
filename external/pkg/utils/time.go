package utils

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// LocalTime 是一个自定义的时间类型，用于处理时间的序列化和反序列化。
type LocalTime time.Time

// MarshalJSON 实现了 json.Marshaler 接口，用于将 LocalTime 格式化为 JSON 字符串。
func (t LocalTime) MarshalJSON() ([]byte, error) {
	formatted := fmt.Sprintf("\"%s\"", time.Time(t).Format(time.DateTime))
	return []byte(formatted), nil
}

// Value 实现了 driver.Valuer 接口，用于将 LocalTime 转换为数据库支持的类型。
func (t LocalTime) Value() (driver.Value, error) {
	if time.Time(t).IsZero() {
		return nil, nil
	}
	return time.Time(t), nil
}

// Scan 实现了 sql.Scanner 接口，用于从数据库读取数据并转换为 LocalTime。
func (t *LocalTime) Scan(v interface{}) error {
	switch value := v.(type) {
	case []byte:
		parsedTime, err := time.ParseInLocation(time.DateTime, string(value), time.Local)
		if err != nil {
			return fmt.Errorf("failed to parse time from bytes: %w", err)
		}
		*t = LocalTime(parsedTime)
	case time.Time:
		*t = LocalTime(value)
	default:
		return fmt.Errorf("unsupported type for LocalTime: %T", v)
	}
	return nil
}
