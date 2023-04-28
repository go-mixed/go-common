package orm

import (
	"runtime"
	"time"
)

type DBOptions struct {
	Address  string `yaml:"address" validate:"required"`
	User     string `yaml:"user" validate:"required"`
	Password string `yaml:"password"`
	DBName   string `yaml:"db_name" validate:"required"`
	Charset  string `yaml:"charset"`
	TimeZone string `yaml:"time_zone"`
	// 其它DSN参数，使用object的方式传递
	ExtraParams map[string]string `yaml:"extra_params"`

	SkipInitializeWithVersion bool `yaml:"skip_initialize_with_version"`
	DefaultStringSize         uint `yaml:"default_string_size"`
	DefaultDatetimePrecision  int  `yaml:"default_datetime_precision"`
	DisableDatetimePrecision  bool `yaml:"disable_datetime_precision"`
	DontSupportRenameIndex    bool `yaml:"dont_support_rename_index"`
	DontSupportRenameColumn   bool `yaml:"dont_support_rename_column"`
	DontSupportForShareClause bool `yaml:"dont_support_for_share_clause"`

	// GORM perform single create, update, delete operations in transactions by default to ensure database data integrity
	// You can disable it by setting `SkipDefaultTransaction` to true
	SkipDefaultTransaction bool `yaml:"skip_default_transaction"`
	// FullSaveAssociations full save associations
	FullSaveAssociations bool `yaml:"full_save_associations"`
	// DryRun generate sql without execute
	DryRun bool `yaml:"dry_run"`
	// PrepareStmt executes the given query in cached statement
	PrepareStmt bool `yaml:"prepare_stmt"`
	// DisableAutomaticPing
	DisableAutomaticPing bool `yaml:"disable_automatic_ping"`
	// DisableForeignKeyConstraintWhenMigrating
	DisableForeignKeyConstraintWhenMigrating bool `yaml:"disable_foreign_key_constraint_when_migrating"`
	// DisableNestedTransaction disable nested transaction
	DisableNestedTransaction bool `yaml:"disable_nested_transaction"`
	// AllowGlobalUpdate allow global update
	AllowGlobalUpdate bool `yaml:"allow_global_update"`
	// QueryFields executes the SQL query with all fields of the table
	QueryFields bool `yaml:"query_fields"`
	// CreateBatchSize default create batch size
	CreateBatchSize int `yaml:"create_batch_size"`

	// 空闲连接池中连接的最大数量，默认值为2
	// the maximum number of connections in the idle connection pool
	MaxIdleConns int `yaml:"max_idle_conns"`
	// 当前数据库连接的最大数量，可以理解为连接池的大小，默认值为0表示不限制
	// the maximum number of open connections to the database, default value is 0 means unlimited
	MaxOpenConns int `yaml:"max_open_conns"`
	// 连接最大空闲时间，默认值为0表示不限制
	// the maximum idle time for a connection, default value is 0 means unlimited
	MaxIdleTime time.Duration `yaml:"max_idle_time"`
	// 连接生命的最长时间，默认值为0表示不限制
	// the maximum life-time for a connection, default value is 0 means unlimited
	MaxLifeTime time.Duration `yaml:"max_life_time"`
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
		MaxIdleTime:  30 * time.Second,  // 30s
		MaxLifeTime:  300 * time.Second, // 300s
	}
}
