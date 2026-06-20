package repository

import (
	"database/sql"

	"tuxedosql/internal/model"
)

type ConnectionStore interface {
	LoadConnections() ([]model.Connection, error)
	SaveConnections(connections []model.Connection) error
	LoadGroups() ([]model.ConnectionGroup, error)
	SaveGroups(groups []model.ConnectionGroup) error
	LoadConnectionByID(id string) (*model.Connection, error)
	DeleteCredential(connectionID string)
}

type PoolManager interface {
	GetDB(conn *model.Connection, database string) (*sql.DB, error)
	GetDBByID(connectionID, database string) (*model.Connection, *sql.DB, error)
	Close(connectionID string)
	CloseAll()
	Schema(conn *model.Connection) SchemaIntrospector
}

type TabStore interface {
	LoadTabs() ([]model.TabState, error)
	SaveTabs(tabs []model.TabState) error
}

type HistoryStore interface {
	LoadHistory() ([]model.QueryHistoryEntry, error)
	SaveHistory(entries []model.QueryHistoryEntry) error
}

var (
	_ ConnectionStore = (*ConnectionRepository)(nil)
	_ PoolManager     = (*ConnectionManager)(nil)
	_ TabStore        = (*TabRepository)(nil)
	_ HistoryStore    = (*HistoryRepository)(nil)
)

// DatabaseDriver defines the behavior of a specific database driver (e.g., MySQL, PostgreSQL).
// It handles DSN construction, driver registration, and driver-specific defaults.
type DatabaseDriver interface {
	// DriverName returns the database/sql driver name used with sql.Open.
	DriverName() string
	// BuildDSN constructs a database/sql DSN string from connection parameters.
	// The returned DSN must be valid for the specific driver.
	BuildDSN(conn *model.Connection, database string) string
	// DefaultDatabase returns the fallback database name when none is specified.
	DefaultDatabase() string
}

// SchemaIntrospector provides database-specific schema metadata queries and identifier quoting.
// Each driver implements its own quoting convention (backtick for MySQL, double-quote for PG),
// metadata queries (SHOW commands vs pg_catalog), and system database protection.
type SchemaIntrospector interface {
	// QuoteIdentifier wraps an identifier (table/column name) in the driver's quoting convention.
	// Must handle escaping of the quote character within the identifier.
	QuoteIdentifier(name string) string
	// ListDatabasesQuery returns the SQL to list all accessible databases.
	ListDatabasesQuery() string
	// ListTablesQuery returns the SQL to list tables. Called within the context of a specific database.
	// For MySQL this is SHOW TABLES (after USE), for PG this queries pg_catalog.
	ListTablesQuery() string
	// SystemDatabases returns the set of system database names that should be protected from DROP.
	SystemDatabases() map[string]bool
}
