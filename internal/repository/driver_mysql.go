package repository

import (
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"tuxedosql/internal/model"
)

// MySQLDriver implements DatabaseDriver for MySQL/MariaDB.
type MySQLDriver struct{}

// DriverName returns the database/sql driver name.
func (d *MySQLDriver) DriverName() string {
	return "mysql"
}

// BuildDSN constructs a MySQL DSN from connection parameters.
// Format: user:pass@tcp(host:port)/db?timeout=5s&parseTime=true&loc=IANA
func (d *MySQLDriver) BuildDSN(conn *model.Connection, database string) string {
	tz := conn.Timezone
	if tz == "" {
		tz = "Local"
	}
	if tz != "Local" {
		if loc, err := time.LoadLocation(tz); err != nil {
			tz = "Local"
		} else {
			tz = loc.String()
		}
	}
	encodedLoc := strings.ReplaceAll(tz, "/", "%2F")

	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?timeout=5s&parseTime=true&loc=%s",
		conn.Username, conn.Password, conn.Host, conn.Port, database, encodedLoc)
}

// DefaultDatabase returns the MySQL default administrative database.
func (d *MySQLDriver) DefaultDatabase() string {
	return "mysql"
}

// MySQLSchema implements SchemaIntrospector for MySQL/MariaDB.
type MySQLSchema struct{}

// QuoteIdentifier wraps an identifier in MySQL backtick quotes, escaping internal backticks.
func (s *MySQLSchema) QuoteIdentifier(name string) string {
	return "`" + strings.ReplaceAll(name, "`", "``") + "`"
}

// ListDatabasesQuery returns SHOW DATABASES.
func (s *MySQLSchema) ListDatabasesQuery() string {
	return "SHOW DATABASES"
}

// ListTablesQuery returns SHOW TABLES (requires prior USE database).
func (s *MySQLSchema) ListTablesQuery() string {
	return "SHOW TABLES"
}

// SystemDatabases returns the set of MySQL system databases that should not be dropped.
func (s *MySQLSchema) SystemDatabases() map[string]bool {
	return map[string]bool{
		"INFORMATION_SCHEMA": true,
		"MYSQL":              true,
		"PERFORMANCE_SCHEMA": true,
		"SYS":                true,
	}
}

// Compile-time interface compliance checks.
var (
	_ DatabaseDriver    = (*MySQLDriver)(nil)
	_ SchemaIntrospector = (*MySQLSchema)(nil)
)
