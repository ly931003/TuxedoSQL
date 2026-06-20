package repository

import (
	"fmt"
	"net/url"
	"strings"

	_ "github.com/lib/pq"

	"tuxedosql/internal/model"
)

// PostgresDriver implements DatabaseDriver for PostgreSQL.
type PostgresDriver struct{}

// DriverName returns the database/sql driver name.
func (d *PostgresDriver) DriverName() string {
	return "postgres"
}

// BuildDSN constructs a PostgreSQL DSN from connection parameters.
// Uses URL format: postgres://user:pass@host:port/dbname?sslmode=disable&connect_timeout=5
func (d *PostgresDriver) BuildDSN(conn *model.Connection, database string) string {
	host := conn.Host
	port := conn.Port
	if port <= 0 {
		port = 5432
	}

	u := url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(conn.Username, conn.Password),
		Host:   fmt.Sprintf("%s:%d", host, port),
		Path:   database,
	}

	// Timezone: PG doesn't support timezone in DSN — it will be set via SET command
	// if the service layer chooses to handle it per-session.
	u.RawQuery = "sslmode=disable&connect_timeout=5"

	return u.String()
}

// DefaultDatabase returns the PostgreSQL default administrative database.
func (d *PostgresDriver) DefaultDatabase() string {
	return "postgres"
}

// PostgresSchema implements SchemaIntrospector for PostgreSQL.
type PostgresSchema struct{}

// QuoteIdentifier wraps an identifier in PostgreSQL double-quotes, escaping internal double-quotes.
func (s *PostgresSchema) QuoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// ListDatabasesQuery returns the SQL to list non-template databases via pg_catalog.
func (s *PostgresSchema) ListDatabasesQuery() string {
	return `SELECT datname
		FROM pg_catalog.pg_database
		WHERE datistemplate = false
		  AND datallowconn = true
		ORDER BY datname`
}

// ListTablesQuery returns the SQL to list tables in the current database via pg_catalog.
func (s *PostgresSchema) ListTablesQuery() string {
	return `SELECT tablename
		FROM pg_catalog.pg_tables
		WHERE schemaname NOT IN ('pg_catalog', 'information_schema')
		ORDER BY tablename`
}

// SystemDatabases returns the set of PostgreSQL system databases that should not be dropped.
func (s *PostgresSchema) SystemDatabases() map[string]bool {
	return map[string]bool{
		"postgres":  true,
		"template0": true,
		"template1": true,
	}
}

// Compile-time interface compliance checks.
var (
	_ DatabaseDriver     = (*PostgresDriver)(nil)
	_ SchemaIntrospector = (*PostgresSchema)(nil)
)
