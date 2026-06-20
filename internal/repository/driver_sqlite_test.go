package repository

import (
	"strings"
	"testing"

	"tuxedosql/internal/model"
)

func TestSQLiteDriverDriverName(t *testing.T) {
	d := &SQLiteDriver{}
	if name := d.DriverName(); name != "sqlite" {
		t.Errorf("DriverName() = %q, want %q", name, "sqlite")
	}
}

func TestSQLiteDriverDefaultDatabase(t *testing.T) {
	d := &SQLiteDriver{}
	if db := d.DefaultDatabase(); db != "main" {
		t.Errorf("DefaultDatabase() = %q, want %q", db, "main")
	}
}

func TestSQLiteDriverBuildDSN(t *testing.T) {
	d := &SQLiteDriver{}

	tests := []struct {
		name     string
		conn     *model.Connection
		database string
		want     string
	}{
		{
			name:     "file path from Host",
			conn:     &model.Connection{ID: "c1", Host: "/path/to/db.sqlite"},
			database: "main",
			want:     "/path/to/db.sqlite?_journal_mode=WAL&_busy_timeout=5000&_cache=shared",
		},
		{
			name:     "empty Host defaults to :memory:",
			conn:     &model.Connection{ID: "c2", Host: ""},
			database: "main",
			want:     ":memory:?_journal_mode=WAL&_busy_timeout=5000&_cache=shared",
		},
		{
			name:     "relative path",
			conn:     &model.Connection{ID: "c3", Host: "./data/mydb.db"},
			database: "main",
			want:     "./data/mydb.db?_journal_mode=WAL&_busy_timeout=5000&_cache=shared",
		},
		{
			name:     "database parameter is ignored",
			conn:     &model.Connection{ID: "c4", Host: "test.db"},
			database: "anything",
			want:     "test.db?_journal_mode=WAL&_busy_timeout=5000&_cache=shared",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := d.BuildDSN(tt.conn, tt.database)
			if got != tt.want {
				t.Errorf("BuildDSN() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestSQLiteSchemaQuoteIdentifier(t *testing.T) {
	s := &SQLiteSchema{}

	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"simple", "users", `"users"`},
		{"with spaces", "user accounts", `"user accounts"`},
		{"with double quote inside", `it"s`, `"it""s"`},
		{"empty", "", `""`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := s.QuoteIdentifier(tt.input)
			if got != tt.want {
				t.Errorf("QuoteIdentifier(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSQLiteSchemaListDatabasesQuery(t *testing.T) {
	s := &SQLiteSchema{}
	query := s.ListDatabasesQuery()
	if query != `SELECT 'main'` {
		t.Errorf("ListDatabasesQuery() = %q, want %q", query, `SELECT 'main'`)
	}
}

func TestSQLiteSchemaListTablesQuery(t *testing.T) {
	s := &SQLiteSchema{}
	query := s.ListTablesQuery()
	if query == "" {
		t.Fatal("ListTablesQuery() returned empty string")
	}
	if !strings.Contains(query, "sqlite_master") {
		t.Errorf("ListTablesQuery() should query sqlite_master, got %q", query)
	}
}

func TestSQLiteSchemaSystemDatabases(t *testing.T) {
	s := &SQLiteSchema{}
	dbs := s.SystemDatabases()
	if len(dbs) != 0 {
		t.Errorf("SystemDatabases() should be empty, got %d entries", len(dbs))
	}
}

// TestSQLiteDriverInMemoryIntegration verifies the driver end-to-end with an in-memory DB.
func TestSQLiteDriverInMemoryIntegration(t *testing.T) {
	m := NewConnectionManager(nil, &SQLiteDriver{}, &SQLiteSchema{})
	defer m.CloseAll()

	conn := &model.Connection{
		ID:   "test-sqlite",
		Host: ":memory:",
	}

	db, err := m.GetDB(conn, "main")
	if err != nil {
		t.Fatalf("GetDB(:memory:) failed: %v", err)
	}

	// Verify connectivity
	if err := db.Ping(); err != nil {
		t.Fatalf("Ping failed: %v", err)
	}

	// Create a test table
	_, err = db.Exec("CREATE TABLE test_users (id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		t.Fatalf("CREATE TABLE failed: %v", err)
	}

	// List tables via SchemaIntrospector
	schema := &SQLiteSchema{}
	rows, err := db.Query(schema.ListTablesQuery())
	if err != nil {
		t.Fatalf("ListTablesQuery failed: %v", err)
	}
	defer func() { _ = rows.Close() }()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			t.Fatalf("Scan table name failed: %v", err)
		}
		tables = append(tables, name)
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("Rows iteration failed: %v", err)
	}

	found := false
	for _, tbl := range tables {
		if tbl == "test_users" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'test_users' in table list, got %v", tables)
	}

	// List databases via SchemaIntrospector
	rows2, err := db.Query(schema.ListDatabasesQuery())
	if err != nil {
		t.Fatalf("ListDatabasesQuery failed: %v", err)
	}
	defer func() { _ = rows2.Close() }()

	var dbs []string
	for rows2.Next() {
		var name string
		if err := rows2.Scan(&name); err != nil {
			t.Fatalf("Scan database name failed: %v", err)
		}
		dbs = append(dbs, name)
	}

	if len(dbs) != 1 || dbs[0] != "main" {
		t.Errorf("expected ['main'], got %v", dbs)
	}

	// Test QuoteIdentifier with actual query
	quoted := schema.QuoteIdentifier("test_users")
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM " + quoted).Scan(&count); err != nil {
		t.Fatalf("SELECT with quoted table failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 rows, got %d", count)
	}
}

// TestSQLiteDriverInterfaceCompliance verifies compile-time interface satisfaction.
func TestSQLiteDriverInterfaceCompliance(t *testing.T) {
	var d DatabaseDriver = &SQLiteDriver{}
	var s SchemaIntrospector = &SQLiteSchema{}
	_ = d
	_ = s
}
