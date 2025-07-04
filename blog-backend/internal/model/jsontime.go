package model

import (
	"database/sql/driver"
	"fmt"
	"time"
)

// JSONTime 是一个自定义类型，它封装了 time.Time，用于处理特定格式（"年-月-日 时:分:秒"）的 JSON 编组和解组。
type JSONTime struct {
	time.Time
}

// MarshalJSON 实现了 json.Marshaler 接口。
// 它将时间格式化为 "年-月-日 时:分:秒" 用于 JSON 输出。
func (t JSONTime) MarshalJSON() ([]byte, error) {
	if t.IsZero() {
		return []byte("null"), nil
	}
	formatted := t.Format("2006-01-02 15:04:05") // "年-月-日 时:分:秒"
	return []byte(fmt.Sprintf(`"%s"`, formatted)), nil
}

// UnmarshalJSON 实现了 json.Unmarshaler 接口。
// 它从 "年-月-日 时:分:秒" 格式的字符串中解析时间。
func (t *JSONTime) UnmarshalJSON(data []byte) error {
	s := string(data)
	// 移除引号
	if len(s) > 1 && s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
	}
	if s == "" || s == "null" {
		*t = JSONTime{Time: time.Time{}}
		return nil
	}
	parsedTime, err := time.ParseInLocation("2006-01-02 15:04:05", s, time.Local)
	if err != nil {
		return err
	}
	*t = JSONTime{Time: parsedTime}
	return nil
}

// Value 实现了 driver.Valuer 接口。
// 它将 JSONTime 转换为 driver.Value 以便存储到数据库。
func (t JSONTime) Value() (driver.Value, error) {
	if t.IsZero() {
		return nil, nil
	}
	return t.Time, nil
}

// Scan 实现了 sql.Scanner 接口。
// 它将数据库值扫描到 JSONTime 中。
func (t *JSONTime) Scan(v interface{}) error {
	value, ok := v.(time.Time)
	if ok {
		*t = JSONTime{Time: value}
		return nil
	}
	return fmt.Errorf("无法将 %v 转换为 JSONTime", v)
}

// GormDataType gorm 通用数据类型
func (JSONTime) GormDataType() string {
	return "datetime"
}
