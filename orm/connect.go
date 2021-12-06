package orm

import (
	"bytes"
	"fmt"
	"go.uber.org/zap"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
	"moul.io/zapgorm2"
)

func NewMySqlORM(dbOptions *DBOptions, zapLogger *zap.Logger) (*gorm.DB, error) {
	logger := zapgorm2.New(zapLogger)
	logger = logger.LogMode(gormlogger.Info).(zapgorm2.Logger)
	logger.SetAsDefault()

	return gorm.Open(mysql.New(mysql.Config{
		DSN:                       BuildMySqlDSN(dbOptions),
		DefaultStringSize:         dbOptions.DefaultStringSize,
		DisableDatetimePrecision:  dbOptions.DisableDatetimePrecision,
		DefaultDatetimePrecision:  &dbOptions.DefaultDatetimePrecision,
		DontSupportRenameIndex:    dbOptions.DontSupportRenameIndex,
		DontSupportRenameColumn:   dbOptions.DontSupportRenameColumn,
		SkipInitializeWithVersion: dbOptions.SkipInitializeWithVersion,
	}), &gorm.Config{
		SkipDefaultTransaction:                   dbOptions.SkipDefaultTransaction,
		FullSaveAssociations:                     dbOptions.FullSaveAssociations,
		Logger:                                   logger,
		DryRun:                                   dbOptions.DryRun,
		PrepareStmt:                              dbOptions.PrepareStmt,
		DisableAutomaticPing:                     dbOptions.DisableAutomaticPing,
		DisableForeignKeyConstraintWhenMigrating: dbOptions.DisableForeignKeyConstraintWhenMigrating,
		DisableNestedTransaction:                 dbOptions.DisableNestedTransaction,
		AllowGlobalUpdate:                        dbOptions.AllowGlobalUpdate,
		QueryFields:                              dbOptions.QueryFields,
		CreateBatchSize:                          dbOptions.CreateBatchSize,
	})
}

// BuildMySqlDSN https://github.com/go-sql-driver/mysql#dsn-data-source-name
func BuildMySqlDSN(dbOptions *DBOptions) string {
	dsn := ""
	if dbOptions.Password != "" {
		dsn = fmt.Sprintf("%s:%s@tcp(%s)/%s", dbOptions.User, dbOptions.Password, dbOptions.Address, dbOptions.DBName)
	} else {
		dsn = fmt.Sprintf("%s@tcp(%s)/%s", dbOptions.User, dbOptions.Address, dbOptions.DBName)
	}

	buffer := bytes.NewBufferString(dsn)
	buffer.WriteString("?parseTime=True")

	if dbOptions.Charset != "" {
		buffer.WriteString("&charset=" + dbOptions.Charset)
	}

	if dbOptions.TimeZone != "" {
		buffer.WriteString("&loc=" + dbOptions.TimeZone)
	}

	for k, v := range dbOptions.ExtraParams {
		buffer.WriteString(k)
		buffer.WriteString("=")
		buffer.WriteString(v)
	}

	return buffer.String()
}
