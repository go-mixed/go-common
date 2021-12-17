package orm

import (
	"go-common/utils/time"
	"runtime"
)

type DBOptions struct {
	Address     string            `json:"address" yaml:"address" validate:"required"`
	User        string            `json:"user" yaml:"user" validate:"required"`
	Password    string            `json:"password" yaml:"password"`
	DBName      string            `json:"db_name" yaml:"db_name" validate:"required"`
	Charset     string            `json:"charset" yaml:"charset"`
	TimeZone    string            `json:"time_zone" yaml:"time_zone"`
	ExtraParams map[string]string `json:"extra_params" yaml:"extra_params"`

	SkipInitializeWithVersion bool `json:"skip_initialize_with_version" yaml:"skip_initialize_with_version"`
	DefaultStringSize         uint `json:"default_string_size" yaml:"default_string_size"`
	DefaultDatetimePrecision  int  `json:"default_datetime_precision" yaml:"default_datetime_precision"`
	DisableDatetimePrecision  bool `json:"disable_datetime_precision" yaml:"disable_datetime_precision"`
	DontSupportRenameIndex    bool `json:"dont_support_rename_index" yaml:"dont_support_rename_index"`
	DontSupportRenameColumn   bool `json:"dont_support_rename_column" yaml:"dont_support_rename_column"`
	DontSupportForShareClause bool `json:"dont_support_for_share_clause" yaml:"dont_support_for_share_clause"`

	// GORM perform single create, update, delete operations in transactions by default to ensure database data integrity
	// You can disable it by setting `SkipDefaultTransaction` to true
	SkipDefaultTransaction bool `json:"skip_default_transaction" yaml:"skip_default_transaction"`
	// FullSaveAssociations full save associations
	FullSaveAssociations bool `json:"full_save_associations" yaml:"full_save_associations"`
	// DryRun generate sql without execute
	DryRun bool `json:"dry_run" yaml:"dry_run"`
	// PrepareStmt executes the given query in cached statement
	PrepareStmt bool `json:"prepare_stmt" yaml:"prepare_stmt"`
	// DisableAutomaticPing
	DisableAutomaticPing bool `json:"disable_automatic_ping" yaml:"disable_automatic_ping"`
	// DisableForeignKeyConstraintWhenMigrating
	DisableForeignKeyConstraintWhenMigrating bool `json:"disable_foreign_key_constraint_when_migrating" yaml:"disable_foreign_key_constraint_when_migrating"`
	// DisableNestedTransaction disable nested transaction
	DisableNestedTransaction bool `json:"disable_nested_transaction" yaml:"disable_nested_transaction"`
	// AllowGlobalUpdate allow global update
	AllowGlobalUpdate bool `json:"allow_global_update" yaml:"allow_global_update"`
	// QueryFields executes the SQL query with all fields of the table
	QueryFields bool `json:"query_fields" yaml:"query_fields"`
	// CreateBatchSize default create batch size
	CreateBatchSize int `json:"create_batch_size" yaml:"create_batch_size"`

	// 空闲连接池中连接的最大数量
	MaxIdleConns int `json:"max_idle_conns" yaml:"max_idle_conns"`
	// 打开数据库连接的最大数量
	MaxOpenConns int `json:"max_open_conns" yaml:"max_open_conns"`
	// 最大空闲时间
	MaxIdleTime time_utils.MillisecondDuration `json:"max_idle_time" yaml:"max_idle_time"`
	// 连接可复用的最大时间
	MaxLifeTime time_utils.MillisecondDuration `json:"max_life_time" yaml:"max_life_time"`
}

func DefaultDBOptions() *DBOptions {
	return &DBOptions{
		User:                     "root",
		Charset:                  "utf-8",
		TimeZone:                 "Local",
		DefaultStringSize:        256,
		DefaultDatetimePrecision: 2,

		MaxIdleConns: runtime.NumCPU() * 2,
		MaxOpenConns: runtime.NumCPU() * 2,
		MaxIdleTime:  30_000,  // 30s
		MaxLifeTime:  300_000, // 300s
	}
}
