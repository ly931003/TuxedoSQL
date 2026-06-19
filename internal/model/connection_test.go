package model

import (
	"encoding/json"
	"testing"
	"time"
)

func roundTripJSON[T any](t *testing.T, input T) T {
	t.Helper()
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var output T
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	return output
}

func TestConnectionJSONRoundTrip(t *testing.T) {
	t.Run("all fields including timezone survive", func(t *testing.T) {
		createdAt := time.Date(2024, time.March, 14, 15, 9, 26, 0, time.UTC)
		updatedAt := time.Date(2024, time.April, 1, 8, 30, 0, 0, time.FixedZone("CST", 8*3600))
		got := roundTripJSON(t, Connection{ID: "conn-1", Name: "Primary", GroupID: "grp-1", Host: "db.local", Port: 3306, Username: "root", Password: "secret", Database: "app", Timezone: "Asia/Shanghai", CreatedAt: createdAt, UpdatedAt: updatedAt})
		if got.ID != "conn-1" || got.Name != "Primary" || got.GroupID != "grp-1" || got.Host != "db.local" || got.Port != 3306 || got.Username != "root" || got.Password != "secret" || got.Database != "app" || got.Timezone != "Asia/Shanghai" || !got.CreatedAt.Equal(createdAt) || !got.UpdatedAt.Equal(updatedAt) {
			t.Fatalf("unexpected round-trip result: %#v", got)
		}
	})
}

func TestConnectionGroupJSONRoundTrip(t *testing.T) {
	t.Run("group fields survive", func(t *testing.T) {
		got := roundTripJSON(t, ConnectionGroup{ID: "grp-1", Name: "Favorites", ParentID: "root"})
		if got.ID != "grp-1" || got.Name != "Favorites" || got.ParentID != "root" {
			t.Fatalf("unexpected group: %#v", got)
		}
	})
}

func TestTreeNodeGroupType(t *testing.T) {
	t.Run("group node keeps type and non-leaf", func(t *testing.T) {
		got := roundTripJSON(t, TreeNode{Key: "g1", Label: "Group", Type: "group", Leaf: false})
		if got.Type != "group" || got.Leaf || len(got.Children) != 0 { t.Fatalf("unexpected node: %#v", got) }
	})
}

func TestTreeNodeConnectionType(t *testing.T) {
	t.Run("connection node keeps type", func(t *testing.T) {
		got := roundTripJSON(t, TreeNode{Key: "c1", Label: "Conn", Type: "connection", Leaf: false})
		if got.Type != "connection" || got.Leaf { t.Fatalf("unexpected node: %#v", got) }
	})
}

func TestTreeNodeDatabaseType(t *testing.T) {
	t.Run("database node keeps type", func(t *testing.T) {
		got := roundTripJSON(t, TreeNode{Key: "d1", Label: "app", Type: "database", Leaf: false})
		if got.Type != "database" || got.Leaf { t.Fatalf("unexpected node: %#v", got) }
	})
}

func TestTreeNodeTableType(t *testing.T) {
	t.Run("table node keeps type and leaf", func(t *testing.T) {
		got := roundTripJSON(t, TreeNode{Key: "t1", Label: "users", Type: "table", Leaf: true})
		if got.Type != "table" || !got.Leaf { t.Fatalf("unexpected node: %#v", got) }
	})
}

func TestTreeNodeChildrenNesting(t *testing.T) {
	t.Run("nested children survive", func(t *testing.T) {
		root := TreeNode{Key: "g1", Label: "Group", Type: "group", Children: []TreeNode{{Key: "c1", Label: "Conn", Type: "connection", Children: []TreeNode{{Key: "d1", Label: "app", Type: "database", Children: []TreeNode{{Key: "t1", Label: "users", Type: "table", Leaf: true}}}}}}}
		got := roundTripJSON(t, root)
		if len(got.Children) != 1 || got.Children[0].Type != "connection" || len(got.Children[0].Children) != 1 || got.Children[0].Children[0].Type != "database" || len(got.Children[0].Children[0].Children) != 1 || got.Children[0].Children[0].Children[0].Type != "table" || !got.Children[0].Children[0].Children[0].Leaf {
			t.Fatalf("unexpected nested node: %#v", got)
		}
	})
}

func TestCreateConnectionParamsJSONRoundTrip(t *testing.T) {
	t.Run("create params keep all fields", func(t *testing.T) {
		got := roundTripJSON(t, CreateConnectionParams{Name: "Primary", GroupID: "grp-1", Host: "127.0.0.1", Port: 3306, Username: "user", Password: "pw", Database: "app", Timezone: "UTC"})
		if got.Name != "Primary" || got.GroupID != "grp-1" || got.Host != "127.0.0.1" || got.Port != 3306 || got.Username != "user" || got.Password != "pw" || got.Database != "app" || got.Timezone != "UTC" {
			t.Fatalf("unexpected create params: %#v", got)
		}
	})
}

func TestUpdateConnectionParamsJSONRoundTrip(t *testing.T) {
	t.Run("update params add id to create fields", func(t *testing.T) {
		got := roundTripJSON(t, UpdateConnectionParams{ID: "conn-1", Name: "Primary", GroupID: "grp-1", Host: "127.0.0.1", Port: 3306, Username: "user", Password: "pw", Database: "app", Timezone: "UTC"})
		if got.ID != "conn-1" || got.Name != "Primary" || got.GroupID != "grp-1" || got.Host != "127.0.0.1" || got.Port != 3306 || got.Username != "user" || got.Password != "pw" || got.Database != "app" || got.Timezone != "UTC" {
			t.Fatalf("unexpected update params: %#v", got)
		}
	})
}

func TestCreateGroupParamsJSONRoundTrip(t *testing.T) {
	t.Run("create group excludes id", func(t *testing.T) {
		got := roundTripJSON(t, CreateGroupParams{Name: "Reporting", ParentID: "root"})
		if got.Name != "Reporting" || got.ParentID != "root" { t.Fatalf("unexpected create group: %#v", got) }
	})
}

func TestUpdateGroupParamsJSONRoundTrip(t *testing.T) {
	t.Run("update group includes id", func(t *testing.T) {
		got := roundTripJSON(t, UpdateGroupParams{ID: "grp-2", Name: "Reporting", ParentID: "root"})
		if got.ID != "grp-2" || got.Name != "Reporting" || got.ParentID != "root" { t.Fatalf("unexpected update group: %#v", got) }
	})
}

func TestCreateDatabaseParamsWithCharsetAndCollation(t *testing.T) {
	t.Run("database params keep charset and collation", func(t *testing.T) {
		got := roundTripJSON(t, CreateDatabaseParams{ConnectionID: "conn-1", DatabaseName: "analytics", Charset: "utf8mb4", Collation: "utf8mb4_unicode_ci"})
		if got.ConnectionID != "conn-1" || got.DatabaseName != "analytics" || got.Charset != "utf8mb4" || got.Collation != "utf8mb4_unicode_ci" { t.Fatalf("unexpected database params: %#v", got) }
	})
}

func TestCreateDatabaseParamsWithoutCharsetAndCollation(t *testing.T) {
	t.Run("database params allow empty charset and collation", func(t *testing.T) {
		got := roundTripJSON(t, CreateDatabaseParams{ConnectionID: "conn-1", DatabaseName: "analytics"})
		if got.ConnectionID != "conn-1" || got.DatabaseName != "analytics" || got.Charset != "" || got.Collation != "" { t.Fatalf("unexpected database params: %#v", got) }
	})
}

func TestCreateTableParamsWithColumns(t *testing.T) {
	t.Run("table params keep column definitions", func(t *testing.T) {
		got := roundTripJSON(t, CreateTableParams{ConnectionID: "conn-1", DatabaseName: "app", TableName: "users", Charset: "utf8mb4", Collation: "utf8mb4_general_ci", Comment: "user table", Columns: []ColumnDef{{Name: "id", DataType: "INT", Nullable: false, AutoIncrement: true, Unsigned: true, Comment: "pk", IsPrimaryKey: true}, {Name: "email", DataType: "VARCHAR(255)", Nullable: false, DefaultValue: "", Comment: "login"}}})
		if got.TableName != "users" || len(got.Columns) != 2 || got.Columns[0].Name != "id" || !got.Columns[0].AutoIncrement || !got.Columns[0].Unsigned || !got.Columns[0].IsPrimaryKey || got.Columns[1].DataType != "VARCHAR(255)" || got.Columns[1].Nullable {
			t.Fatalf("unexpected table params: %#v", got)
		}
	})
}

func TestColumnDefBooleanFieldsTrue(t *testing.T) {
	t.Run("boolean true flags survive", func(t *testing.T) {
		got := roundTripJSON(t, ColumnDef{Name: "id", DataType: "INT", Nullable: true, DefaultValue: "0", AutoIncrement: true, Unsigned: true, Comment: "identifier", IsPrimaryKey: true})
		if !got.Nullable || !got.AutoIncrement || !got.Unsigned || !got.IsPrimaryKey { t.Fatalf("unexpected booleans: %#v", got) }
	})
}

func TestColumnDefBooleanFieldsFalse(t *testing.T) {
	t.Run("boolean false flags survive", func(t *testing.T) {
		got := roundTripJSON(t, ColumnDef{Name: "name", DataType: "VARCHAR(64)", Nullable: false, AutoIncrement: false, Unsigned: false, IsPrimaryKey: false})
		if got.Nullable || got.AutoIncrement || got.Unsigned || got.IsPrimaryKey { t.Fatalf("unexpected booleans: %#v", got) }
	})
}

func TestTestResultJSONRoundTrip(t *testing.T) {
	t.Run("success and message survive", func(t *testing.T) {
		got := roundTripJSON(t, TestResult{Success: true, Message: "connected"})
		if !got.Success || got.Message != "connected" { t.Fatalf("unexpected test result: %#v", got) }
	})
}

func TestDDLResultJSONRoundTrip(t *testing.T) {
	t.Run("sql and message survive", func(t *testing.T) {
		got := roundTripJSON(t, DDLResult{SQL: "CREATE TABLE users(id INT PRIMARY KEY)", Message: "created"})
		if got.SQL == "" || got.Message != "created" { t.Fatalf("unexpected ddl result: %#v", got) }
	})
}

func TestCharsetInfoJSONRoundTrip(t *testing.T) {
	t.Run("charset metadata survives", func(t *testing.T) {
		got := roundTripJSON(t, CharsetInfo{Charset: "utf8mb4", DefaultCollation: "utf8mb4_0900_ai_ci", Description: "UTF-8 Unicode"})
		if got.Charset != "utf8mb4" || got.DefaultCollation != "utf8mb4_0900_ai_ci" || got.Description != "UTF-8 Unicode" { t.Fatalf("unexpected charset info: %#v", got) }
	})
}
