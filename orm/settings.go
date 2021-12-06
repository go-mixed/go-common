package orm

type DBOptions struct {
	Address     string            `json:"address" validate:"required"`
	User        string            `json:"user" validate:"required"`
	Password    string            `json:"password"`
	DBName      string            `json:"db_name" validate:"required"`
	Charset     string            `json:"charset"`
	TimeZone    string            `json:"time_zone"`
	ExtraParams map[string]string `json:"extra_params"`

	SkipInitializeWithVersion bool `json:"skip_initialize_with_version"`
	DefaultStringSize         uint `json:"default_string_size"`
	DefaultDatetimePrecision  int  `json:"default_datetime_precision"`
	DisableDatetimePrecision  bool `json:"disable_datetime_precision"`
	DontSupportRenameIndex    bool `json:"dont_support_rename_index"`
	DontSupportRenameColumn   bool `json:"dont_support_rename_column"`
	DontSupportForShareClause bool `json:"dont_support_for_share_clause"`

	// GORM perform single create, update, delete operations in transactions by default to ensure database data integrity
	// You can disable it by setting `SkipDefaultTransaction` to true
	SkipDefaultTransaction bool `json:"skip_default_transaction"`
	// FullSaveAssociations full save associations
	FullSaveAssociations bool
	// DryRun generate sql without execute
	DryRun bool `json:"dry_run"`
	// PrepareStmt executes the given query in cached statement
	PrepareStmt bool `json:"prepare_stmt"`
	// DisableAutomaticPing
	DisableAutomaticPing bool `json:"disable_automatic_ping"`
	// DisableForeignKeyConstraintWhenMigrating
	DisableForeignKeyConstraintWhenMigrating bool `json:"disable_foreign_key_constraint_when_migrating"`
	// DisableNestedTransaction disable nested transaction
	DisableNestedTransaction bool `json:"disable_nested_transaction"`
	// AllowGlobalUpdate allow global update
	AllowGlobalUpdate bool `json:"allow_global_update"`
	// QueryFields executes the SQL query with all fields of the table
	QueryFields bool `json:"query_fields"`
	// CreateBatchSize default create batch size
	CreateBatchSize int `json:"create_batch_size"`
}

func DefaultDBOptions() *DBOptions {
	return &DBOptions{
		User:                     "root",
		Charset:                  "utf-8",
		TimeZone:                 "Local",
		DefaultStringSize:        256,
		DefaultDatetimePrecision: 2,
	}
}
