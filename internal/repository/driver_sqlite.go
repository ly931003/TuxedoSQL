package repository

import (
	"fmt"
	"strings"

	_ "modernc.org/sqlite"

	"tuxedosql/internal/model"
)

// SQLiteDriver implements DatabaseDriver for SQLite.
// Uses modernc.org/sqlite, a pure-Go SQLite implementation (no CGO).
type SQLiteDriver struct{}

// DriverName returns the database/sql driver name for SQLite.
func (d *SQLiteDriver) DriverName() string {
	return "sqlite"
}

// BuildDSN constructs a SQLite DSN from connection parameters.
// For SQLite, the DSN is a file path with query parameters.
// Host field is used as the file path (e.g., "/path/to/db.sqlite" or ":memory:").
// If Host is empty, defaults to ":memory:" (in-memory database).
// The database parameter is ignored — SQLite always connects to a single file.
func (d *SQLiteDriver) BuildDSN(conn *model.Connection, database string) string {
	filePath := conn.Host
	if filePath == "" {
		filePath = ":memory:"
	}
	// WAL mode for better concurrent read performance.
	// busy_timeout to wait instead of failing immediately on locked database.
	return fmt.Sprintf("%s?_journal_mode=WAL&_busy_timeout=5000&_cache=shared", filePath)
}

// DefaultDatabase returns the SQLite default database name.
// SQLite always opens a single "main" database per connection.
func (d *SQLiteDriver) DefaultDatabase() string {
	return "main"
}

// SQLiteSchema implements SchemaIntrospector for SQLite.
type SQLiteSchema struct{}

// QuoteIdentifier wraps an identifier in SQLite double-quotes, escaping internal double-quotes.
func (s *SQLiteSchema) QuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// ListDatabasesQuery returns a query that always yields "main".
// SQLite opens a single database file per connection — there is no list of databases
// unless the user explicitly runs ATTACH DATABASE.
func (s *SQLiteSchema) ListDatabasesQuery() string {
	return `SELECT 'main'`
}

// ListTablesQuery returns the SQL to list user tables from sqlite_master,
// excluding internal sqlite_* tables.
func (s *SQLiteSchema) ListTablesQuery() string {
	return `SELECT name FROM sqlite_master WHERE type = 'table' AND name NOT LIKE 'sqlite_%' ORDER BY name`
}

// SystemDatabases returns an empty set — SQLite has no server-level system databases
// that need protection from DROP.
func (s *SQLiteSchema) SystemDatabases() map[string]bool {
	return map[string]bool{}
}

// Compile-time interface compliance checks.
var (
	_ DatabaseDriver     = (*SQLiteDriver)(nil)
	_ SchemaIntrospector = (*SQLiteSchema)(nil)
)
