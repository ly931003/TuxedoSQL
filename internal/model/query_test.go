package model

import (
	"encoding/json"
	"testing"
)

func mustRoundTrip[T any](t *testing.T, in T) T {
	t.Helper()
	data, err := json.Marshal(in)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}
	var out T
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	return out
}

func TestFilterGroupIsLeafNilConditions(t *testing.T) {
	if !((&FilterGroup{}).IsLeaf()) {
		t.Fatal("expected nil Conditions to be leaf")
	}
}

func TestFilterGroupIsLeafEmptyConditions(t *testing.T) {
	if !((&FilterGroup{Conditions: []*FilterGroup{}}).IsLeaf()) {
		t.Fatal("expected empty Conditions to be leaf")
	}
}

func TestFilterGroupIsLeafNonEmptyConditions(t *testing.T) {
	if (&FilterGroup{Conditions: []*FilterGroup{{Column: "id", Operator: OpEQ, Value: "1"}}}).IsLeaf() {
		t.Fatal("expected non-empty Conditions to be non-leaf")
	}
}

func TestResultTypeConstants(t *testing.T) {
	for _, tt := range []struct {
		name      string
		got, want ResultType
	}{{"success", ResultSuccess, "success"}, {"error", ResultError, "error"}, {"info", ResultInfo, "info"}} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q want %q", tt.got, tt.want)
			}
		})
	}
}

func TestSortOrderConstants(t *testing.T) {
	for _, tt := range []struct {
		name      string
		got, want SortOrder
	}{{"asc", SortASC, "ASC"}, {"desc", SortDESC, "DESC"}} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q want %q", tt.got, tt.want)
			}
		})
	}
}

func TestFilterOperatorConstants(t *testing.T) {
	for _, tt := range []struct {
		name      string
		got, want FilterOperator
	}{{"eq", OpEQ, "eq"}, {"neq", OpNEQ, "neq"}, {"contains", OpContains, "contains"}, {"gt", OpGT, "gt"}, {"lt", OpLT, "lt"}, {"isnull", OpIsNull, "isnull"}, {"notnull", OpNotNull, "notnull"}} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q want %q", tt.got, tt.want)
			}
		})
	}
}

func TestLogicOpConstants(t *testing.T) {
	for _, tt := range []struct {
		name      string
		got, want LogicOp
	}{{"and", LogicAND, "AND"}, {"or", LogicOR, "OR"}} {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.want {
				t.Fatalf("got %q want %q", tt.got, tt.want)
			}
		})
	}
}

func TestColumnInfoJSONRoundTrip(t *testing.T) {
	out := mustRoundTrip(t, ColumnInfo{Name: "email", Type: "VARCHAR(255)"})
	if out.Name != "email" || out.Type != "VARCHAR(255)" {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestTableSchemaJSONRoundTrip(t *testing.T) {
	out := mustRoundTrip(t, TableSchema{Name: "id", DataType: "int", IsNullable: false, ColumnKey: "PRI", DefaultValue: ""})
	if out.Name != "id" || out.DataType != "int" || out.IsNullable || out.ColumnKey != "PRI" || out.DefaultValue != "" {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestQueryResultJSONRoundTrip(t *testing.T) {
	out := mustRoundTrip(t, QueryResult{Columns: []ColumnInfo{{Name: "id", Type: "INT"}}, Rows: []map[string]any{{"id": "1", "name": "Ada"}}, AffectedRows: 2, Message: "ok", MessageType: ResultSuccess, Duration: 15})
	t.Run("metadata", func(t *testing.T) {
		if len(out.Columns) != 1 || out.Columns[0].Name != "id" || out.Columns[0].Type != "INT" {
			t.Fatalf("unexpected columns: %+v", out.Columns)
		}
	})
	t.Run("rows", func(t *testing.T) {
		if len(out.Rows) != 1 || out.Rows[0]["id"] != "1" || out.Rows[0]["name"] != "Ada" {
			t.Fatalf("unexpected rows: %+v", out.Rows)
		}
	})
	t.Run("summary", func(t *testing.T) {
		if out.AffectedRows != 2 || out.Message != "ok" || out.MessageType != ResultSuccess || out.Duration != 15 {
			t.Fatalf("unexpected summary: %+v", out)
		}
	})
}

func TestTabStateJSONRoundTrip(t *testing.T) {
	out := mustRoundTrip(t, TabState{ID: "tab-1", Title: "Users", ConnectionID: "conn-1", Database: "app", SQL: "select * from users"})
	if out.ID != "tab-1" || out.Title != "Users" || out.ConnectionID != "conn-1" || out.Database != "app" || out.SQL != "select * from users" {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestFilterConditionJSONRoundTrip(t *testing.T) {
	out := mustRoundTrip(t, FilterCondition{Column: "name", Operator: OpContains, Value: "ada"})
	if out.Column != "name" || out.Operator != OpContains || out.Value != "ada" {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestFilterGroupLeafJSONRoundTrip(t *testing.T) {
	out := mustRoundTrip(t, FilterGroup{Column: "status", Operator: OpEQ, Value: "active"})
	if !out.IsLeaf() || out.Column != "status" || out.Operator != OpEQ || out.Value != "active" {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestFilterGroupGroupJSONRoundTrip(t *testing.T) {
	out := mustRoundTrip(t, FilterGroup{Logic: LogicAND, Conditions: []*FilterGroup{{Column: "status", Operator: OpEQ, Value: "active"}, {Column: "role", Operator: OpNEQ, Value: "guest"}}})
	if out.IsLeaf() || out.Logic != LogicAND || len(out.Conditions) != 2 || out.Conditions[1].Column != "role" || out.Conditions[1].Operator != OpNEQ {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestFilterGroupNestedJSONRoundTrip(t *testing.T) {
	out := mustRoundTrip(t, FilterGroup{Logic: LogicOR, Conditions: []*FilterGroup{{Logic: LogicAND, Conditions: []*FilterGroup{{Column: "age", Operator: OpGT, Value: "18"}, {Column: "age", Operator: OpLT, Value: "65"}}}, {Column: "deleted_at", Operator: OpIsNull, Value: ""}}})
	if out.IsLeaf() || len(out.Conditions) != 2 || out.Conditions[0].IsLeaf() || len(out.Conditions[0].Conditions) != 2 || out.Conditions[0].Conditions[1].Value != "65" || out.Conditions[1].Operator != OpIsNull {
		t.Fatalf("unexpected nested round-trip result: %+v", out)
	}
}

func TestTableDataParamsJSONRoundTripNilFilters(t *testing.T) {
	out := mustRoundTrip(t, TableDataParams{ConnectionID: "conn", Database: "app", Table: "users", Page: 1, PageSize: 20, SortColumn: "id", SortOrder: SortASC, Filters: nil})
	if out.Filters != nil || out.SortOrder != SortASC || out.Page != 1 || out.PageSize != 20 {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestTableDataParamsJSONRoundTripEmptyLeafFilters(t *testing.T) {
	out := mustRoundTrip(t, TableDataParams{ConnectionID: "conn", Database: "app", Table: "users", Page: 2, PageSize: 10, SortColumn: "created_at", SortOrder: SortDESC, Filters: &FilterGroup{}})
	if out.Filters == nil || !out.Filters.IsLeaf() || out.SortOrder != SortDESC || out.SortColumn != "created_at" {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestTableDataParamsJSONRoundTripNestedFilters(t *testing.T) {
	out := mustRoundTrip(t, TableDataParams{ConnectionID: "conn", Database: "app", Table: "users", Page: 3, PageSize: 5, Filters: &FilterGroup{Logic: LogicAND, Conditions: []*FilterGroup{{Column: "status", Operator: OpEQ, Value: "active"}, {Logic: LogicOR, Conditions: []*FilterGroup{{Column: "city", Operator: OpEQ, Value: "Paris"}, {Column: "city", Operator: OpEQ, Value: "Berlin"}}}}}})
	if out.Filters == nil || out.Filters.IsLeaf() || len(out.Filters.Conditions) != 2 || out.Filters.Conditions[1].Logic != LogicOR || len(out.Filters.Conditions[1].Conditions) != 2 {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestPageResultJSONRoundTrip(t *testing.T) {
	out := mustRoundTrip(t, PageResult{Columns: []ColumnInfo{{Name: "id", Type: "INT"}}, Rows: []map[string]any{{"id": "1"}}, Total: 42, Page: 2, PageSize: 10, TotalPages: 5, Message: "loaded", MessageType: ResultInfo, Duration: 8, SQL: "select * from users limit 10 offset 10"})
	if len(out.Columns) != 1 || len(out.Rows) != 1 || out.Total != 42 || out.TotalPages != 5 || out.MessageType != ResultInfo || out.SQL == "" {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestUpdateRowParamsJSONRoundTrip(t *testing.T) {
	out := mustRoundTrip(t, UpdateRowParams{ConnectionID: "conn", Database: "app", Table: "users", PkValues: map[string]any{"id": "1"}, Column: "name", NewValue: "Grace"})
	if out.ConnectionID != "conn" || out.PkValues["id"] != "1" || out.Column != "name" || out.NewValue != "Grace" {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestUpdateRowParamsJSONRoundTripNilNewValue(t *testing.T) {
	out := mustRoundTrip(t, UpdateRowParams{ConnectionID: "conn", Database: "app", Table: "users", PkValues: map[string]any{"id": "1"}, Column: "deleted_at", NewValue: nil})
	if out.NewValue != nil || out.Column != "deleted_at" || out.PkValues["id"] != "1" {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestUpdateRowResultJSONRoundTrip(t *testing.T) {
	out := mustRoundTrip(t, UpdateRowResult{AffectedRows: 1, Message: "updated", SQL: "update users set name = ? where id = ?"})
	if out.AffectedRows != 1 || out.Message != "updated" || out.SQL == "" {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}

func TestDBSchemaForCompletionJSONRoundTrip(t *testing.T) {
	out := mustRoundTrip(t, DBSchemaForCompletion{Tables: map[string][]string{"users": {"id", "email"}, "orders": {"id", "user_id"}}, Views: []string{"active_users"}})
	if len(out.Tables) != 2 || len(out.Tables["users"]) != 2 || out.Tables["orders"][1] != "user_id" || len(out.Views) != 1 || out.Views[0] != "active_users" {
		t.Fatalf("unexpected round-trip result: %+v", out)
	}
}
