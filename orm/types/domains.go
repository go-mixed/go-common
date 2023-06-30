package types

import (
	"context"
	"database/sql/driver"
	"fmt"
	"github.com/pkg/errors"
	"gopkg.in/go-mixed/go-common.v1/utils/http"
	"gopkg.in/go-mixed/go-common.v1/utils/text"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
	"gorm.io/gorm/schema"
	"strings"
)

// Domains 重写httpUtils.Domains 为orm.Domains
type Domains httpUtils.Domains

// Value return json value, implement driver.Valuer interface
func (d Domains) Value() (driver.Value, error) {
	if d == nil {
		return nil, nil
	}
	return textUtils.JsonMarshal(d)
}

// Scan scan value into Jsonb, implements sql.Scanner interface
func (d *Domains) Scan(val any) error {
	if val == nil {
		*d = Domains(nil)
		return nil
	}
	var ba []byte
	switch v := val.(type) {
	case []byte:
		ba = v
	case string:
		ba = []byte(v)
	default:
		return errors.New(fmt.Sprint("Failed to unmarshal JSONB value:", val))
	}
	t := Domains{}
	err := textUtils.JsonUnmarshalFromBytes(ba, &t)
	*d = t
	return err
}

// MarshalJSON to output non base64 encoded []byte
func (d Domains) MarshalJSON() ([]byte, error) {
	if d == nil {
		return []byte("null"), nil
	}
	t := ([]string)(d)
	return textUtils.JsonMarshalToBytes(t)
}

// UnmarshalJSON to deserialize []byte
func (d *Domains) UnmarshalJSON(b []byte) error {
	var t []string
	err := textUtils.JsonUnmarshalFromBytes(b, &t)
	*d = t
	return err
}

// GormDataType gorm common data type
func (d Domains) GormDataType() string {
	return "domains"
}

// GormDBDataType gorm db data type
func (Domains) GormDBDataType(db *gorm.DB, field *schema.Field) string {
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

func (d Domains) GormValue(ctx context.Context, db *gorm.DB) clause.Expr {
	data, _ := d.MarshalJSON()
	switch db.Dialector.Name() {
	case "mysql":
		if v, ok := db.Dialector.(*mysql.Dialector); ok && !strings.Contains(v.ServerVersion, "MariaDB") {
			return gorm.Expr("CAST(? AS JSON)", string(data))
		}
	}
	return gorm.Expr("?", string(data))
}
