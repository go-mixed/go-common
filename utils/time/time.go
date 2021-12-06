package time_utils

import (
	"database/sql"
	"database/sql/driver"
	"encoding/json"
	"github.com/araddon/dateparse"
	"time"
)

func TimeStampToTime(timestamp int64) {
	time.Unix(timestamp, 0)
}

func TimeToStandard(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func TimeToIso8601(t time.Time) string {
	// .Format("2006-01-02T15:04:05-0700")
	return t.Format(time.RFC3339)
}

func Iso8601ToTime(str string) time.Time {
	t, err := time.Parse(time.RFC3339Nano, str)
	if err != nil {
		t, err = time.Parse(time.RFC3339, str)
	}

	if err == nil {
		return t
	}
	return time.Time{}
}

type AnyTime time.Time

func NewAnyTime(value interface{}) (AnyTime, error) {
	var n time.Time
	var err error
	switch value.(type) {
	case string:
		n, err = dateparse.ParseLocal(value.(string))
	case []byte:
		n, err = dateparse.ParseLocal(string(value.([]byte)))
	case time.Time:
		n = value.(time.Time)
	case sql.NullTime:
		n = value.(sql.NullTime).Time
	case AnyTime:
		n = time.Time(value.(AnyTime))
	}
	return AnyTime(n), err
}

func (t *AnyTime) UnmarshalJSON(data []byte) error {
	var s string
	if err := json.Unmarshal(data, &s); err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	n, err := NewAnyTime(s)
	if err != nil {
		return err
	}
	*t = n
	return nil
}

func (t AnyTime) ToTime() time.Time {
	return time.Time(t)
}

// GobEncode implements the gob.GobEncoder interface.
func (t AnyTime) GobEncode() ([]byte, error) {
	return t.ToTime().GobEncode()
}

// GobDecode implements the gob.GobDecoder interface.
func (t *AnyTime) GobDecode(data []byte) error {
	return (*time.Time)(t).GobDecode(data)
}

// Scan for sql decode
func (t *AnyTime) Scan(value interface{}) error {
	nullTime := &sql.NullTime{}
	err := nullTime.Scan(value)
	*t = AnyTime(nullTime.Time)
	return err
}

// Value for sql encode
func (t AnyTime) Value() (driver.Value, error) {
	return t.ToTime(), nil
}

// GormDataType gorm common data type
func (t AnyTime) GormDataType() string {
	return "datetime"
}
