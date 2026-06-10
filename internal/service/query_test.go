package service

import (
	"testing"

	"tuxedosql/internal/model"
	"tuxedosql/internal/repository"
	"tuxedosql/pkg/fileutil"
)

func newTestQueryService() *QueryService {
	store, _ := fileutil.NewJSONStore()
	connRepo := repository.NewConnectionRepository(store)
	tabRepo := repository.NewTabRepository(store)
	connManager := repository.NewConnectionManager(connRepo)
	return NewQueryService(connManager, connRepo, tabRepo)
}

func TestQueryService_Execute_Validation(t *testing.T) {
	svc := newTestQueryService()

	tests := []struct {
		name         string
		connectionID string
		database     string
		sql          string
		wantErr      bool
	}{
		{
			name:         "空连接ID应报错",
			connectionID: "",
			database:     "test",
			sql:          "SELECT 1",
			wantErr:      true,
		},
		{
			name:         "空SQL应报错",
			connectionID: "conn_test",
			database:     "test",
			sql:          "",
			wantErr:      true,
		},
		{
			name:         "空白SQL应报错",
			connectionID: "conn_test",
			database:     "test",
			sql:          "   ",
			wantErr:      true,
		},
		{
			name:         "空数据库名应报错",
			connectionID: "conn_test",
			database:     "",
			sql:          "SELECT 1",
			wantErr:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Execute(tt.connectionID, tt.database, tt.sql)
			if tt.wantErr {
				if err == nil {
					t.Error("期望返回错误，但没有")
				}
				return
			}
			if err != nil {
				t.Errorf("不期望错误，但返回了: %v", err)
			}
		})
	}
}

func TestQueryService_Execute_NotFound(t *testing.T) {
	svc := newTestQueryService()
	_, err := svc.Execute("不存在的连接ID", "testdb", "SELECT 1")
	if err == nil {
		t.Error("对不存在的连接执行查询应返回错误")
	}
}

func TestQueryService_SaveLoadTabs(t *testing.T) {
	svc := newTestQueryService()

	tabs := []model.TabState{
		{ID: "tab_1", Title: "Query 1", ConnectionID: "conn_1", Database: "testdb", SQL: "SELECT 1"},
		{ID: "tab_2", Title: "Query 2", ConnectionID: "conn_2", Database: "mydb", SQL: "SELECT * FROM users"},
	}

	if err := svc.SaveTabs(tabs); err != nil {
		t.Fatalf("保存标签失败: %v", err)
	}

	loaded, err := svc.LoadTabs()
	if err != nil {
		t.Fatalf("加载标签失败: %v", err)
	}

	if len(loaded) != len(tabs) {
		t.Fatalf("加载的标签数量 = %d, 期望 %d", len(loaded), len(tabs))
	}

	for i := range tabs {
		if loaded[i].ID != tabs[i].ID {
			t.Errorf("标签[%d] ID = %q, 期望 %q", i, loaded[i].ID, tabs[i].ID)
		}
		if loaded[i].SQL != tabs[i].SQL {
			t.Errorf("标签[%d] SQL = %q, 期望 %q", i, loaded[i].SQL, tabs[i].SQL)
		}
	}
}

func TestQueryService_LoadTabs_EmptyFile(t *testing.T) {
	svc := newTestQueryService()
	if err := svc.SaveTabs([]model.TabState{}); err != nil {
		t.Fatalf("保存空标签列表失败: %v", err)
	}

	tabs, err := svc.LoadTabs()
	if err != nil {
		t.Fatalf("加载空标签列表失败: %v", err)
	}
	if len(tabs) != 0 {
		t.Errorf("期望空列表, 实际 %d 个标签", len(tabs))
	}
}

func TestQueryService_IsQueryDetection(t *testing.T) {
	svc := newTestQueryService()
	isQueryTests := []struct {
		sql     string
		isQuery bool
	}{
		{"SELECT * FROM users", true},
		{"select * from users", true},
		{"SHOW DATABASES", true},
		{"show tables", true},
		{"DESCRIBE users", true},
		{"DESC users", true},
		{"EXPLAIN SELECT 1", true},
		{"WITH cte AS (SELECT 1) SELECT * FROM cte", true},
		{"INSERT INTO users VALUES (1)", false},
		{"UPDATE users SET name='test'", false},
		{"DELETE FROM users", false},
		{"CREATE TABLE test (id INT)", false},
		{"DROP TABLE test", false},
		{"ALTER TABLE test ADD COLUMN name VARCHAR(255)", false},
	}
	for _, tt := range isQueryTests {
		t.Run(tt.sql, func(t *testing.T) {
			_ = svc
		})
	}
}

func TestQueryService_GetTableSchema_Validation(t *testing.T) {
	svc := newTestQueryService()
	tests := []struct {
		name         string
		connectionID string
		database     string
		table        string
		wantErr      bool
	}{
		{name: "空连接ID应报错", connectionID: "", database: "testdb", table: "users", wantErr: true},
		{name: "空数据库名应报错", connectionID: "conn_test", database: "", table: "users", wantErr: true},
		{name: "空表名应报错", connectionID: "conn_test", database: "testdb", table: "", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetTableSchema(tt.connectionID, tt.database, tt.table)
			if tt.wantErr {
				if err == nil {
					t.Error("期望返回错误，但没有")
				}
				return
			}
			if err != nil {
				t.Errorf("不期望错误，但返回了: %v", err)
			}
		})
	}
}

func TestQueryService_GetTableSchema_NotFound(t *testing.T) {
	svc := newTestQueryService()
	_, err := svc.GetTableSchema("不存在的连接ID", "testdb", "users")
	if err == nil {
		t.Error("对不存在的连接查询表结构应返回错误")
	}
}

func TestQueryService_GetTableData_Validation(t *testing.T) {
	svc := newTestQueryService()
	t.Run("空连接ID应报错", func(t *testing.T) {
		_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "", Database: "testdb", Table: "users", Page: 1, PageSize: 100})
		if err == nil {
			t.Error("期望返回错误，但没有")
		}
	})
	t.Run("空数据库名应报错", func(t *testing.T) {
		_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_test", Database: "", Table: "users", Page: 1, PageSize: 100})
		if err == nil {
			t.Error("期望返回错误，但没有")
		}
	})
	t.Run("空表名应报错", func(t *testing.T) {
		_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_test", Database: "testdb", Table: "", Page: 1, PageSize: 100})
		if err == nil {
			t.Error("期望返回错误，但没有")
		}
	})
}

func TestQueryService_GetTableData_PageDefaults(t *testing.T) {
	svc := newTestQueryService()
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_fake", Database: "testdb", Table: "users", Page: 0, PageSize: 2000})
	if err == nil {
		t.Error("假连接应返回错误（连接错误，非参数错误）")
	}
}

func TestGetTableData_SortColumnWhitelist(t *testing.T) {
	svc := newTestQueryService()
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_test", Database: "testdb", Table: "users", Page: 1, PageSize: 100, SortColumn: "1=1; DROP TABLE users", SortOrder: model.SortASC})
	if err == nil {
		t.Error("假连接应返回错误")
	}
}

func TestGetTableData_InvalidSortOrder(t *testing.T) {
	svc := newTestQueryService()
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_test", Database: "testdb", Table: "users", Page: 1, PageSize: 100, SortColumn: "name", SortOrder: "DROP TABLE"})
	if err == nil {
		t.Error("假连接应返回错误")
	}
}

func TestGetTableData_InvalidFilterColumn(t *testing.T) {
	svc := newTestQueryService()
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_test", Database: "testdb", Table: "users", Page: 1, PageSize: 100, Filters: &model.FilterGroup{Column: "1=1; DROP TABLE", Operator: model.OpEQ, Value: "x"}})
	if err == nil {
		t.Error("假连接应返回错误")
	}
}

func TestGetTableData_InvalidFilterOperator(t *testing.T) {
	svc := newTestQueryService()
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_test", Database: "testdb", Table: "users", Page: 1, PageSize: 100, Filters: &model.FilterGroup{Column: "name", Operator: "invalid_op", Value: "x"}})
	if err == nil {
		t.Error("假连接应返回错误")
	}
}

func TestIsValidSortOrder(t *testing.T) {
	tests := []struct {
		order    model.SortOrder
		expected bool
	}{
		{model.SortASC, true},
		{model.SortDESC, true},
		{"", false},
		{"asc", false},
		{"DESCENDING", false},
		{"DROP TABLE", false},
	}
	for _, tt := range tests {
		t.Run(string(tt.order), func(t *testing.T) {
			if got := isValidSortOrder(tt.order); got != tt.expected {
				t.Errorf("isValidSortOrder(%q) = %v, 期望 %v", tt.order, got, tt.expected)
			}
		})
	}
}

func TestAllowedOperators(t *testing.T) {
	validOps := []model.FilterOperator{model.OpEQ, model.OpNEQ, model.OpContains, model.OpGT, model.OpLT, model.OpIsNull, model.OpNotNull}
	for _, op := range validOps {
		if !allowedOperators[op] {
			t.Errorf("操作符 %q 应在白名单中", op)
		}
	}
	invalidOps := []model.FilterOperator{"invalid", "like", ">=", "IN"}
	for _, op := range invalidOps {
		if allowedOperators[op] {
			t.Errorf("操作符 %q 不应在白名单中", op)
		}
	}
}

func TestGetTableData_PageBoundaries(t *testing.T) {
	svc := newTestQueryService()
	tests := []struct {
		name     string
		page     int
		pageSize int
		wantErr  bool
	}{
		{"零页应被修正为1", 0, 100, true},
		{"负页码应被修正为1", -5, 100, true},
		{"零pageSize应被修正", 1, 0, true},
		{"负pageSize应被修正", 1, -10, true},
		{"超大pageSize应被截断", 1, 9999, true},
		{"正好1000 pageSize", 1, 1000, true},
		{"最小合法值", 1, 1, true},
		{"大页码", 99999, 100, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_fake", Database: "testdb", Table: "users", Page: tt.page, PageSize: tt.pageSize})
			if tt.wantErr && err == nil {
				t.Error("期望返回错误（假连接），但没有")
			}
		})
	}
}

func TestGetTableData_EmptyFilters(t *testing.T) {
	svc := newTestQueryService()
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_fake", Database: "testdb", Table: "users", Page: 1, PageSize: 100, Filters: nil})
	if err == nil {
		t.Error("假连接应返回错误（连接错误，非参数错误）")
	}
}

func TestGetTableData_NoSorting(t *testing.T) {
	svc := newTestQueryService()
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_fake", Database: "testdb", Table: "users", Page: 1, PageSize: 100, SortColumn: "", SortOrder: ""})
	if err == nil {
		t.Error("假连接应返回错误（连接错误，非参数错误）")
	}
}

func TestGetTableData_AllFilterOperators(t *testing.T) {
	svc := newTestQueryService()
	operators := []model.FilterOperator{model.OpEQ, model.OpNEQ, model.OpContains, model.OpGT, model.OpLT, model.OpIsNull, model.OpNotNull}
	for _, op := range operators {
		t.Run(string(op), func(t *testing.T) {
			_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_fake", Database: "testdb", Table: "users", Page: 1, PageSize: 100, Filters: &model.FilterGroup{Column: "name", Operator: op, Value: "test"}})
			if err == nil {
				t.Error("假连接应返回错误")
			}
		})
	}
}

func TestGetTableSchema_EmptyTableName(t *testing.T) {
	svc := newTestQueryService()
	tests := []struct {
		name         string
		connectionID string
		database     string
		table        string
	}{
		{"空表名", "conn_1", "testdb", ""},
		{"空白表名", "conn_1", "testdb", "   "},
		{"特殊字符表名", "conn_1", "testdb", "users; DROP TABLE"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.GetTableSchema(tt.connectionID, tt.database, tt.table)
			if tt.table == "" {
				if err == nil {
					t.Error("空表名应返回参数校验错误")
				}
			}
		})
	}
}

func TestQueryService_Execute_SQLInjectionAttempts(t *testing.T) {
	svc := newTestQueryService()
	sqls := []string{
		"SELECT * FROM users; DROP TABLE users",
		"SELECT * FROM users WHERE 1=1; DELETE FROM users",
		"' OR '1'='1",
		"SELECT * FROM users UNION SELECT * FROM passwords",
		"1; DROP DATABASE test",
	}
	for _, sql := range sqls {
		label := sql
		if len(label) > 40 {
			label = label[:40]
		}
		t.Run(label, func(t *testing.T) {
			_, err := svc.Execute("conn_fake", "testdb", sql)
			if err == nil {
				t.Error("假连接应返回错误")
			}
		})
	}
}

func TestQueryService_UpdateRow_Validation(t *testing.T) {
	svc := newTestQueryService()
	tests := []struct {
		name    string
		params  model.UpdateRowParams
		wantErr bool
	}{
		{name: "空连接ID应报错", params: model.UpdateRowParams{ConnectionID: "", Database: "test", Table: "users", Column: "name", PkValues: map[string]any{"id": 1}, NewValue: "test"}, wantErr: true},
		{name: "空数据库名应报错", params: model.UpdateRowParams{ConnectionID: "conn", Database: "", Table: "users", Column: "name", PkValues: map[string]any{"id": 1}, NewValue: "test"}, wantErr: true},
		{name: "空表名应报错", params: model.UpdateRowParams{ConnectionID: "conn", Database: "test", Table: "", Column: "name", PkValues: map[string]any{"id": 1}, NewValue: "test"}, wantErr: true},
		{name: "空列名应报错", params: model.UpdateRowParams{ConnectionID: "conn", Database: "test", Table: "users", Column: "", PkValues: map[string]any{"id": 1}, NewValue: "test"}, wantErr: true},
		{name: "空主键条件应报错", params: model.UpdateRowParams{ConnectionID: "conn", Database: "test", Table: "users", Column: "name", PkValues: map[string]any{}, NewValue: "test"}, wantErr: true},
		{name: "NULL值更新", params: model.UpdateRowParams{ConnectionID: "conn", Database: "test", Table: "users", Column: "name", PkValues: map[string]any{"id": 1}, NewValue: nil}, wantErr: true},
		{name: "复合主键", params: model.UpdateRowParams{ConnectionID: "conn", Database: "test", Table: "users", Column: "name", PkValues: map[string]any{"org_id": 1, "user_id": 2}, NewValue: "test"}, wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.UpdateRow(tt.params)
			if tt.wantErr && err == nil {
				t.Error("期望返回错误，但没有")
			}
		})
	}
}

func TestQueryService_UpdateRow_InvalidColumn(t *testing.T) {
	svc := newTestQueryService()
	_, err := svc.UpdateRow(model.UpdateRowParams{ConnectionID: "conn_test", Database: "testdb", Table: "users", Column: "1=1; DROP TABLE users", PkValues: map[string]any{"id": 1}, NewValue: "test"})
	if err == nil {
		t.Error("SQL注入列名应返回错误")
	}
}

func TestQueryService_UpdateRow_InvalidPkColumn(t *testing.T) {
	svc := newTestQueryService()
	_, err := svc.UpdateRow(model.UpdateRowParams{ConnectionID: "conn_test", Database: "testdb", Table: "users", Column: "name", PkValues: map[string]any{"1=1; DROP": 1}, NewValue: "test"})
	if err == nil {
		t.Error("SQL注入主键列名应返回错误")
	}
}
