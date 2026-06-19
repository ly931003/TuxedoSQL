package repository

import (
	"database/sql"
	"strings"
	"testing"

	"tuxedosql/internal/model"
)

func newTestPoolDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("mysql", "user:pass@tcp(127.0.0.1:1)/mysql")
	if err != nil {
		t.Fatalf("sql.Open() error = %v", err)
	}

	return db
}

func TestConnectionManagerNewWithNilRepoInitializesPools(t *testing.T) {
	m := NewConnectionManager(nil)
	if m == nil || m.pools == nil {
		t.Fatal("expected manager and pools map to be initialized")
	}
}

func TestConnectionManagerNewWithRepoInitializesPools(t *testing.T) {
	repo := &ConnectionRepository{}
	m := NewConnectionManager(repo)
	if m == nil || m.pools == nil || m.connRepo != repo {
		t.Fatal("expected manager to keep repo and initialize pools map")
	}
}

func TestConnectionManagerGetDBNilConnectionReturnsError(t *testing.T) {
	m := NewConnectionManager(nil)
	if _, err := m.GetDB(nil, "dbA"); err == nil || err.Error() != "连接不能为空" {
		t.Fatalf("expected nil connection error, got %v", err)
	}
}

func TestConnectionManagerCloseEmptyPoolDoesNotPanic(t *testing.T) {
	m := NewConnectionManager(nil)
	m.Close("missing")
	if len(m.pools) != 0 {
		t.Fatalf("expected empty pool map, got %d", len(m.pools))
	}
}

func TestConnectionManagerCloseAllEmptyPoolDoesNotPanic(t *testing.T) {
	m := NewConnectionManager(nil)
	m.CloseAll()
	if len(m.pools) != 0 {
		t.Fatalf("expected empty pool map, got %d", len(m.pools))
	}
}

func TestConnectionManagerCloseRemovesMatchingPrefixOnly(t *testing.T) {
	m := NewConnectionManager(nil)
	db1 := newTestPoolDB(t)
	db2 := newTestPoolDB(t)
	db3 := newTestPoolDB(t)
	defer db3.Close()

	m.pools["conn1:dbA"] = db1
	m.pools["conn1:dbB"] = db2
	m.pools["conn2:dbA"] = db3

	m.Close("conn1")

	if len(m.pools) != 1 {
		t.Fatalf("expected 1 remaining pool, got %d", len(m.pools))
	}
	if _, ok := m.pools["conn2:dbA"]; !ok {
		t.Fatal("expected conn2:dbA to remain")
	}
}

func TestConnectionManagerCloseAllRemovesAllPools(t *testing.T) {
	m := NewConnectionManager(nil)
	m.pools["conn1:dbA"] = newTestPoolDB(t)
	m.pools["conn1:dbB"] = newTestPoolDB(t)
	m.pools["conn2:dbA"] = newTestPoolDB(t)

	m.CloseAll()

	if len(m.pools) != 0 {
		t.Fatalf("expected all pools removed, got %d", len(m.pools))
	}
}

func TestConnectionManagerGetDBUnreachableHostReturnsError(t *testing.T) {
	m := NewConnectionManager(nil)
	conn := &model.Connection{ID: "conn1", Host: "127.0.0.1", Port: 1, Username: "user", Password: "pass"}

	_, err := m.GetDB(conn, "dbA")
	if err == nil {
		t.Fatal("expected unreachable host error")
	}
	if !strings.Contains(err.Error(), "连接测试失败") {
		t.Fatalf("expected ping failure error, got %v", err)
	}
}

func TestConnectionManagerGetDBUsesConnectionDatabaseWhenNameEmpty(t *testing.T) {
	m := NewConnectionManager(nil)
	conn := &model.Connection{ID: "conn1", Host: "127.0.0.1", Port: 1, Username: "user", Password: "pass", Database: "fallback_db"}

	_, err := m.GetDB(conn, "")
	if err == nil {
		t.Fatal("expected unreachable host error")
	}
	if !strings.Contains(err.Error(), "连接测试失败") {
		t.Fatalf("expected ping failure error, got %v", err)
	}
}
