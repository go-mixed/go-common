package orm

import (
	"context"
	"database/sql/driver"
	"errors"
	"fmt"
	"go-common/utils/text"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"strings"
)

type Int64Slice []int64

func (m *Int64Slice) Scan(value interface{}) error {
	if value == nil {
		*m = Int64Slice(nil)
		return nil
	}
	var ba []byte
	switch v := value.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", value))
	}

	var metadata Int64Slice
	if err := text_utils.JsonUnmarshalFromBytes(ba, &metadata); err != nil {
		return err
	}

	*m = metadata
	return nil
}

func (m Int64Slice) Value() (driver.Value, error) {
	if len(m) == 0 {
		return nil, nil
	}
	return text_utils.JsonMarshalToBytes(m)
}

// GormDataType gorm common data type
func (Int64Slice) GormDataType() string {
	return "int64_slice"
}

// MarshalJSON to output non base64 encoded []byte
func (m Int64Slice) MarshalJSON() ([]byte, error) {
	if m == nil {
		return []byte("null"), nil
	}
	t := ([]int64)(m)
	return text_utils.JsonMarshalToBytes(t)
}

// UnmarshalJSON to deserialize []byte
func (m *Int64Slice) UnmarshalJSON(b []byte) error {
	var t []int64
	err := text_utils.JsonUnmarshalFromBytes(b, &t)
	*m = t
	return err
}

// GormDBDataType gorm db data type
func (Int64Slice) GormDBDataType(db *gorm.DB, field *schema.Field) string {
	switch db.Dialector.Name() {
	case "sqlite":
		return "JSON"
	case "mysql":
		return "JSON"
	case "postgres":
		return "JSONB"
	case "sqlserver":
		return "NVARCHAR(MAX)"
	}
	return ""
}

func (m Int64Slice) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	data, _ := m.MarshalJSON()
	switch db.Dialector.Name() {
	case "mysql":
		if v, ok := db.Dialector.(*mysql.Dialector); ok && !strings.Contains(v.ServerVersion, "MariaDB") {
			return gorm.Expr("CAST(? AS JSON)", string(data))
		}
	}
	return gorm.Expr("?", string(data))
}
