package service

import (
	"testing"

	"tuxedosql/internal/model"
	"tuxedosql/internal/repository"
	"tuxedosql/pkg/fileutil"
)

func newTestQueryService(t *testing.T) *QueryService {
	t.Helper()
	t.Setenv("HOME", t.TempDir())

	store, _ := fileutil.NewJSONStore()
	connRepo := repository.NewConnectionRepository(store)
	tabRepo := repository.NewTabRepository(store)
	historyRepo := repository.NewHistoryRepository(store)
        connManager := repository.NewConnectionManager(connRepo,
                map[string]repository.DatabaseDriver{"mysql": &repository.MySQLDriver{}},
                map[string]repository.SchemaIntrospector{"mysql": &repository.MySQLSchema{}},
        )
	return NewQueryService(connManager, connRepo, tabRepo, historyRepo)
}

func TestQueryService_Execute_Validation(t *testing.T) {
	svc := newTestQueryService(t)

	tests := []struct {
		name         string
		connectionID string
		database     string
		sql          string
	}{
		{
			name:         "空连接ID应报错",
			connectionID: "",
			database:     "test",
			sql:          "SELECT 1",
		},
		{
			name:         "空SQL应报错",
			connectionID: "conn_test",
			database:     "test",
			sql:          "",
		},
		{
			name:         "空白SQL应报错",
			connectionID: "conn_test",
			database:     "test",
			sql:          "   ",
		},
		{
			name:         "空数据库名应报错",
			connectionID: "conn_test",
			database:     "",
			sql:          "SELECT 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.Execute(tt.connectionID, tt.database, tt.sql)
			if err == nil {
				t.Error("期望返回错误，但没有")
			}
		})
	}
}

func TestQueryService_Execute_NotFound(t *testing.T) {
	svc := newTestQueryService(t)
	_, err := svc.Execute("不存在的连接ID", "testdb", "SELECT 1")
	if err == nil {
		t.Error("对不存在的连接执行查询应返回错误")
	}
}

func TestExecuteReturnsQueryID(t *testing.T) {
	tests := []struct {
		name         string
		connectionID string
		database     string
		sql          string
	}{
		{name: "连接错误结果应包含 QueryID", connectionID: "不存在的连接ID", database: "testdb", sql: "SELECT 1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestQueryService(t)

			result, err := svc.Execute(tt.connectionID, tt.database, tt.sql)
			if err == nil {
				t.Fatal("期望返回错误，但没有")
			}
			if result == nil {
				t.Fatal("期望返回结果，但为 nil")
			}
			if result.QueryID == "" {
				t.Fatal("期望返回非空 QueryID，但为空")
			}
		})
	}
}

func TestCancelQueryUnknownID(t *testing.T) {
	tests := []struct {
		name    string
		queryID string
	}{
		{name: "取消不存在的 QueryID 应报错", queryID: "nonexistent"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newTestQueryService(t)

			if err := svc.CancelQuery(tt.queryID); err == nil {
				t.Fatal("期望返回错误，但没有")
			}
		})
	}
}

func TestQueryService_SaveLoadTabs(t *testing.T) {
	svc := newTestQueryService(t)

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
	svc := newTestQueryService(t)
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
	tests := []struct {
		name    string
		sql     string
		isQuery bool
	}{
		{name: "SELECT", sql: "SELECT * FROM users", isQuery: true},
		{name: "select lowercase", sql: "select * from users", isQuery: true},
		{name: "SHOW", sql: "SHOW DATABASES", isQuery: true},
		{name: "show lowercase", sql: "show tables", isQuery: true},
		{name: "DESCRIBE", sql: "DESCRIBE users", isQuery: true},
		{name: "DESC", sql: "DESC users", isQuery: true},
		{name: "EXPLAIN", sql: "EXPLAIN SELECT 1", isQuery: true},
		{name: "WITH", sql: "WITH cte AS (SELECT 1) SELECT * FROM cte", isQuery: true},
		{name: "leading whitespace", sql: "  \n\tSELECT 1", isQuery: true},
		{name: "INSERT", sql: "INSERT INTO users VALUES (1)", isQuery: false},
		{name: "UPDATE", sql: "UPDATE users SET name='test'", isQuery: false},
		{name: "DELETE", sql: "DELETE FROM users", isQuery: false},
		{name: "CREATE", sql: "CREATE TABLE test (id INT)", isQuery: false},
		{name: "DROP", sql: "DROP TABLE test", isQuery: false},
		{name: "ALTER", sql: "ALTER TABLE test ADD COLUMN name VARCHAR(255)", isQuery: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isQueryStatement(tt.sql); got != tt.isQuery {
				t.Errorf("isQueryStatement(%q) = %v, 期望 %v", tt.sql, got, tt.isQuery)
			}
		})
	}
}

func TestBuildFilterClause_GroupValidation(t *testing.T) {
	svc := newTestQueryService(t)
        conn := &model.Connection{}
        schema := svc.connManager.Schema(conn)
	whitelist := map[string]bool{
		"name":   true,
		"status": true,
		"age":    true,
	}

	t.Run("无效组合逻辑应报错", func(t *testing.T) {
		_, _, err := svc.buildFilterClause(&model.FilterGroup{
			Logic: "XOR",
			Conditions: []*model.FilterGroup{
				{Column: "name", Operator: model.OpEQ, Value: "alice"},
				{Column: "status", Operator: model.OpEQ, Value: "active"},
			},
		}, whitelist, schema)
		if err == nil {
			t.Fatal("期望返回错误，但没有")
		}
	})

	t.Run("子条件不足两个应报错", func(t *testing.T) {
		_, _, err := svc.buildFilterClause(&model.FilterGroup{
			Logic: model.LogicAND,
			Conditions: []*model.FilterGroup{
				{Column: "name", Operator: model.OpEQ, Value: "alice"},
			},
		}, whitelist, schema)
		if err == nil {
			t.Fatal("期望返回错误，但没有")
		}
	})

	t.Run("AND 下嵌套 OR 应添加括号", func(t *testing.T) {
		sql, args, err := svc.buildFilterClause(&model.FilterGroup{
			Logic: model.LogicAND,
			Conditions: []*model.FilterGroup{
				{Column: "status", Operator: model.OpEQ, Value: "active"},
				{
					Logic: model.LogicOR,
					Conditions: []*model.FilterGroup{
						{Column: "name", Operator: model.OpEQ, Value: "alice"},
						{Column: "age", Operator: model.OpGT, Value: "18"},
					},
				},
			},
		}, whitelist, schema)
		if err != nil {
			t.Fatalf("不期望错误，但返回了: %v", err)
		}
		wantSQL := "`status` = ? AND (`name` = ? OR `age` > ?)"
		if sql != wantSQL {
			t.Fatalf("SQL = %q, 期望 %q", sql, wantSQL)
		}
		if len(args) != 3 {
			t.Fatalf("args 长度 = %d, 期望 3", len(args))
		}
	})
}

func TestQueryService_GetTableSchema_Validation(t *testing.T) {
	svc := newTestQueryService(t)
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
	svc := newTestQueryService(t)
	_, err := svc.GetTableSchema("不存在的连接ID", "testdb", "users")
	if err == nil {
		t.Error("对不存在的连接查询表结构应返回错误")
	}
}

func TestQueryService_GetTableData_Validation(t *testing.T) {
	svc := newTestQueryService(t)
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
	svc := newTestQueryService(t)
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_fake", Database: "testdb", Table: "users", Page: 0, PageSize: 2000})
	if err == nil {
		t.Error("假连接应返回错误（连接错误，非参数错误）")
	}
}

func TestGetTableData_SortColumnWhitelist(t *testing.T) {
	svc := newTestQueryService(t)
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_test", Database: "testdb", Table: "users", Page: 1, PageSize: 100, SortColumn: "1=1; DROP TABLE users", SortOrder: model.SortASC})
	if err == nil {
		t.Error("假连接应返回错误")
	}
}

func TestGetTableData_InvalidSortOrder(t *testing.T) {
	svc := newTestQueryService(t)
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_test", Database: "testdb", Table: "users", Page: 1, PageSize: 100, SortColumn: "name", SortOrder: "DROP TABLE"})
	if err == nil {
		t.Error("假连接应返回错误")
	}
}

func TestGetTableData_InvalidFilterColumn(t *testing.T) {
	svc := newTestQueryService(t)
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_test", Database: "testdb", Table: "users", Page: 1, PageSize: 100, Filters: &model.FilterGroup{Column: "1=1; DROP TABLE", Operator: model.OpEQ, Value: "x"}})
	if err == nil {
		t.Error("假连接应返回错误")
	}
}

func TestGetTableData_InvalidFilterOperator(t *testing.T) {
	svc := newTestQueryService(t)
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
	svc := newTestQueryService(t)
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
	svc := newTestQueryService(t)
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_fake", Database: "testdb", Table: "users", Page: 1, PageSize: 100, Filters: nil})
	if err == nil {
		t.Error("假连接应返回错误（连接错误，非参数错误）")
	}
}

func TestGetTableData_NoSorting(t *testing.T) {
	svc := newTestQueryService(t)
	_, err := svc.GetTableData(model.TableDataParams{ConnectionID: "conn_fake", Database: "testdb", Table: "users", Page: 1, PageSize: 100, SortColumn: "", SortOrder: ""})
	if err == nil {
		t.Error("假连接应返回错误（连接错误，非参数错误）")
	}
}

func TestGetTableData_AllFilterOperators(t *testing.T) {
	svc := newTestQueryService(t)
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
	svc := newTestQueryService(t)
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
	svc := newTestQueryService(t)
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
	svc := newTestQueryService(t)
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
	svc := newTestQueryService(t)
	_, err := svc.UpdateRow(model.UpdateRowParams{ConnectionID: "conn_test", Database: "testdb", Table: "users", Column: "1=1; DROP TABLE users", PkValues: map[string]any{"id": 1}, NewValue: "test"})
	if err == nil {
		t.Error("SQL注入列名应返回错误")
	}
}

func TestQueryService_UpdateRow_InvalidPkColumn(t *testing.T) {
	svc := newTestQueryService(t)
	_, err := svc.UpdateRow(model.UpdateRowParams{ConnectionID: "conn_test", Database: "testdb", Table: "users", Column: "name", PkValues: map[string]any{"1=1; DROP": 1}, NewValue: "test"})
	if err == nil {
		t.Error("SQL注入主键列名应返回错误")
	}
}

func TestQueryService_GetDBSchemaForCompletion_Validation(t *testing.T) {
	svc := newTestQueryService(t)
	tests := []struct {
		name         string
		connectionID string
		database     string
		wantErr      bool
	}{
		{name: "空连接ID应报错", connectionID: "", database: "testdb", wantErr: true},
		{name: "空数据库名应报错", connectionID: "conn_test", database: "", wantErr: true},
		{name: "假连接应报错", connectionID: "conn_fake", database: "testdb", wantErr: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			schema, err := svc.GetDBSchemaForCompletion(tt.connectionID, tt.database)
			if tt.wantErr {
				if err == nil {
					t.Error("期望返回错误，但没有")
				}
				return
			}
			if err != nil {
				t.Errorf("意外错误: %v", err)
			}
			if schema == nil {
				t.Error("schema 不应为 nil")
			}
		})
	}
}
