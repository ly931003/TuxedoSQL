//go:build integration

package service

import (
	"context"
	"fmt"
	"strconv"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"tuxedosql/internal/model"
	"tuxedosql/internal/repository"
	"tuxedosql/pkg/fileutil"
)

// portToInt extracts the integer port number from a testcontainers Port value.
func portToInt(t *testing.T, port interface{ Port() string }) int {
	t.Helper()
	portInt, err := strconv.Atoi(port.Port())
	if err != nil {
		t.Fatalf("端口解析失败: %v", err)
	}
	return portInt
}

// toStringVal converts a MySQL query result value to string, handling both
// []byte (MySQL 8.0) and string (MySQL 5.7) types returned by go-sql-driver.
func toStringVal(v any) string {
	switch val := v.(type) {
	case []byte:
		return string(val)
	case string:
		return val
	default:
		return fmt.Sprintf("%v", val)
	}
}

// ============================================================================
// MySQL integration test suite
// ============================================================================

type mysqlSuite struct {
	container testcontainers.Container
	connRepo  repository.ConnectionStore
	connMgr   repository.PoolManager
	connSvc   *ConnectionService
	querySvc  *QueryService
	connID    string
	dbName    string
}

// setupMySQL spins up a MySQL container via testcontainers and creates
// a ConnectionService + QueryService wired to it. The returned suite
// MUST be cleaned up via suite.cleanup(t).
func setupMySQL(t *testing.T) *mysqlSuite {
	t.Helper()
	ctx := context.Background()

	// -- 1. Start MySQL container (uses locally cached docker.io/library/mysql:5.7.44) --
	req := testcontainers.ContainerRequest{
		Image:        "docker.io/library/mysql:5.7.44",
		ExposedPorts: []string{"3306/tcp"},
		Env: map[string]string{
			"MYSQL_ROOT_PASSWORD": "test",
			"MYSQL_DATABASE":      "testdb",
			"MYSQL_USER":          "testuser",
			"MYSQL_PASSWORD":      "testpass",
		},
		WaitingFor: wait.ForLog("ready for connections").WithStartupTimeout(120 * time.Second),
	}
	mysqlContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatalf("启动 MySQL 容器失败: %v", err)
	}

	host, err := mysqlContainer.Host(ctx)
	if err != nil {
		mysqlContainer.Terminate(ctx)
		t.Fatalf("获取容器 host 失败: %v", err)
	}
	port, err := mysqlContainer.MappedPort(ctx, "3306/tcp")
	if err != nil {
		mysqlContainer.Terminate(ctx)
		t.Fatalf("获取容器 port 失败: %v", err)
	}

	// MySQL 5.7 reports "ready" before fully accepting TCP — give it a few seconds
	time.Sleep(5 * time.Second)

	// -- 2. Build repo + service stack --
	store, err := fileutil.NewJSONStore()
	if err != nil {
		mysqlContainer.Terminate(ctx)
		t.Fatalf("创建 JSONStore 失败: %v", err)
	}
	connRepo := repository.NewConnectionRepository(store)
	tabRepo := repository.NewTabRepository(store)
	historyRepo := repository.NewHistoryRepository(store)
	connMgr := repository.NewConnectionManager(connRepo, &repository.MySQLDriver{}, &repository.MySQLSchema{})
	connSvc := NewConnectionService(connMgr, connRepo)
	querySvc := NewQueryService(connMgr, connRepo, tabRepo, historyRepo)

	// -- 3. Persist a Connection so GetDBByID / findConnection work --
	conn, err := connSvc.Create(model.CreateConnectionParams{
		Name:     "integration-test-mysql",
		Host:     host,
		Port:     portToInt(t, port),
		Username: "testuser",
		Password: "testpass",
		Database: "testdb",
		Timezone: "Local",
	})
	if err != nil {
		connMgr.CloseAll()
		mysqlContainer.Terminate(ctx)
		t.Fatalf("创建连接失败: %v", err)
	}

	return &mysqlSuite{
		container: mysqlContainer,
		connRepo:  connRepo,
		connMgr:   connMgr,
		connSvc:   connSvc,
		querySvc:  querySvc,
		connID:    conn.ID,
		dbName:    "testdb",
	}
}

func (s *mysqlSuite) cleanup(t *testing.T) {
	t.Helper()
	s.connMgr.CloseAll()
	if err := s.container.Terminate(context.Background()); err != nil {
		t.Logf("清理 MySQL 容器失败: %v", err)
	}
}

// -- MySQL: GetDatabases --

func TestIntegration_MySQL_GetDatabases(t *testing.T) {
	suite := setupMySQL(t)
	defer suite.cleanup(t)

	dbs, err := suite.connSvc.GetDatabases(suite.connID)
	if err != nil {
		t.Fatalf("GetDatabases 失败: %v", err)
	}

	if len(dbs) == 0 {
		t.Fatal("期望返回至少一个数据库，但列表为空")
	}

	foundTestDB := false
	for _, db := range dbs {
		if db == suite.dbName {
			foundTestDB = true
			break
		}
	}
	if !foundTestDB {
		t.Errorf("数据库列表中未找到 %q，实际列表: %v", suite.dbName, dbs)
	}
}

// -- MySQL: GetTables --

func TestIntegration_MySQL_GetTables(t *testing.T) {
	suite := setupMySQL(t)
	defer suite.cleanup(t)

	// testdb created by the container is empty — should return empty slice, not error
	tables, err := suite.connSvc.GetTables(suite.connID, suite.dbName)
	if err != nil {
		t.Fatalf("GetTables 失败: %v", err)
	}

	if len(tables) != 0 {
		t.Errorf("空数据库应返回空列表, 实际=%d 个表", len(tables))
	}
}

// -- MySQL: Execute (SELECT) --

func TestIntegration_MySQL_Execute_Select(t *testing.T) {
	suite := setupMySQL(t)
	defer suite.cleanup(t)

	result, err := suite.querySvc.Execute(suite.connID, suite.dbName, "SELECT 1 AS one, 'hello' AS greeting")
	if err != nil {
		t.Fatalf("Execute SELECT 失败: %v", err)
	}

	if result.MessageType != model.ResultSuccess {
		t.Errorf("期望 messageType=success, 实际=%s", result.MessageType)
	}
	if len(result.Rows) != 1 {
		t.Fatalf("期望 1 行结果, 实际=%d", len(result.Rows))
	}

	row := result.Rows[0]
	if v, ok := row["one"]; !ok {
		t.Error("结果中缺少列 'one'")
	} else {
		if v.(int64) != 1 {
			t.Errorf("one = %v, 期望 1", v)
		}
	}
	if v, ok := row["greeting"]; !ok {
		t.Error("结果中缺少列 'greeting'")
	} else {
		if toStringVal(v) != "hello" {
			t.Errorf("greeting = %v, 期望 'hello'", v)
		}
	}

	if len(result.Columns) != 2 {
		t.Errorf("期望 2 列, 实际=%d", len(result.Columns))
	}
	if result.Duration < 0 {
		t.Error("Duration 不应为负")
	}
}

// -- MySQL: Execute (DDL + DML) --

func TestIntegration_MySQL_Execute_DDL_DML(t *testing.T) {
	suite := setupMySQL(t)
	defer suite.cleanup(t)

	// DDL: create table
	createResult, err := suite.querySvc.Execute(suite.connID, suite.dbName,
		"CREATE TABLE test_people (id INT AUTO_INCREMENT PRIMARY KEY, name VARCHAR(100) NOT NULL, age INT)")
	if err != nil {
		t.Fatalf("CREATE TABLE 失败: %v", err)
	}
	if createResult.AffectedRows != 0 {
		t.Logf("CREATE TABLE affectedRows=%d (DDL 通常返回 0)", createResult.AffectedRows)
	}

	// DML: insert
	insertResult, err := suite.querySvc.Execute(suite.connID, suite.dbName,
		"INSERT INTO test_people (name, age) VALUES ('Alice', 30), ('Bob', 25)")
	if err != nil {
		t.Fatalf("INSERT 失败: %v", err)
	}
	if insertResult.AffectedRows != 2 {
		t.Errorf("INSERT affectedRows=%d, 期望 2", insertResult.AffectedRows)
	}

	// DML: select
	selectResult, err := suite.querySvc.Execute(suite.connID, suite.dbName,
		"SELECT name, age FROM test_people ORDER BY age")
	if err != nil {
		t.Fatalf("SELECT 失败: %v", err)
	}
	if len(selectResult.Rows) != 2 {
		t.Fatalf("SELECT 期望 2 行, 实际=%d", len(selectResult.Rows))
	}

	// Verify row order by age
	row0 := selectResult.Rows[0]
	row1 := selectResult.Rows[1]
	if toStringVal(row0["name"]) != "Bob" {
		t.Errorf("第一行 name = %v, 期望 'Bob'", row0["name"])
	}
	if toStringVal(row1["name"]) != "Alice" {
		t.Errorf("第二行 name = %v, 期望 'Alice'", row1["name"])
	}

	// DDL: drop
	dropResult, err := suite.querySvc.Execute(suite.connID, suite.dbName, "DROP TABLE test_people")
	if err != nil {
		t.Fatalf("DROP TABLE 失败: %v", err)
	}
	if dropResult.MessageType != model.ResultSuccess {
		t.Errorf("DROP TABLE messageType=%s, 期望 success", dropResult.MessageType)
	}
}

// -- MySQL: GetTableSchema --

func TestIntegration_MySQL_GetTableSchema(t *testing.T) {
	suite := setupMySQL(t)
	defer suite.cleanup(t)

	// Create a table with known structure
	_, err := suite.querySvc.Execute(suite.connID, suite.dbName,
		"CREATE TABLE test_schema_table ("+
			"id INT AUTO_INCREMENT PRIMARY KEY, "+
			"username VARCHAR(50) NOT NULL, "+
			"email VARCHAR(100), "+
			"score INT DEFAULT 0)")
	if err != nil {
		t.Fatalf("CREATE TABLE 失败: %v", err)
	}

	schemas, err := suite.querySvc.GetTableSchema(suite.connID, suite.dbName, "test_schema_table")
	if err != nil {
		t.Fatalf("GetTableSchema 失败: %v", err)
	}

	if len(schemas) != 4 {
		t.Fatalf("期望 4 列, 实际=%d", len(schemas))
	}

	// Build lookup by column name
	colMap := make(map[string]model.TableSchema, len(schemas))
	for _, sc := range schemas {
		colMap[sc.Name] = sc
	}

	// id: PRI, NOT NULL, int
	idCol, ok := colMap["id"]
	if !ok {
		t.Fatal("缺少列 'id'")
	}
	if idCol.ColumnKey != "PRI" {
		t.Errorf("id.ColumnKey = %q, 期望 PRI", idCol.ColumnKey)
	}
	if idCol.IsNullable {
		t.Error("id.IsNullable = true, 期望 false")
	}

	// username: NOT NULL, varchar
	userCol, ok := colMap["username"]
	if !ok {
		t.Fatal("缺少列 'username'")
	}
	if userCol.IsNullable {
		t.Error("username.IsNullable = true, 期望 false")
	}

	// email: nullable
	emailCol, ok := colMap["email"]
	if !ok {
		t.Fatal("缺少列 'email'")
	}
	if !emailCol.IsNullable {
		t.Error("email.IsNullable = false, 期望 true")
	}

	// score: has default '0'
	scoreCol, ok := colMap["score"]
	if !ok {
		t.Fatal("缺少列 'score'")
	}
	if scoreCol.DefaultValue != "0" {
		t.Errorf("score.DefaultValue = %q, 期望 '0'", scoreCol.DefaultValue)
	}

	// Cleanup
	_, _ = suite.querySvc.Execute(suite.connID, suite.dbName, "DROP TABLE test_schema_table")
}

// -- MySQL: Execute with non-existent connection --

func TestIntegration_MySQL_Execute_InvalidConnection(t *testing.T) {
	suite := setupMySQL(t)
	defer suite.cleanup(t)

	_, err := suite.querySvc.Execute("nonexistent_conn_id", suite.dbName, "SELECT 1")
	if err == nil {
		t.Error("不存在的连接应返回错误")
	}
}

// -- MySQL: GetTableSchema with non-existent table --

func TestIntegration_MySQL_GetTableSchema_InvalidTable(t *testing.T) {
	suite := setupMySQL(t)
	defer suite.cleanup(t)

	schemas, err := suite.querySvc.GetTableSchema(suite.connID, suite.dbName, "nosuchtable")
	if err != nil {
		t.Fatalf("GetTableSchema 不应返回错误: %v", err)
	}
	if len(schemas) != 0 {
		t.Errorf("不存在的表应返回空 schema 列表, 实际=%d 列", len(schemas))
	}
}

// -- MySQL: GetDatabases with non-existent connection --

func TestIntegration_MySQL_GetDatabases_InvalidConnection(t *testing.T) {
	suite := setupMySQL(t)
	defer suite.cleanup(t)

	_, err := suite.connSvc.GetDatabases("nonexistent_conn_id")
	if err == nil {
		t.Error("不存在的连接应返回错误")
	}
}

// ============================================================================
// PostgreSQL integration tests
// (only methods that use SchemaIntrospector — placeholder $N syntax not yet abstracted)
// Note: requires docker.io/library/postgres:16-alpine image available locally.
// ============================================================================

type pgSuite struct {
	container testcontainers.Container
	connRepo  repository.ConnectionStore
	connMgr   repository.PoolManager
	connSvc   *ConnectionService
	connID    string
	dbName    string
}

func setupPostgres(t *testing.T) *pgSuite {
	t.Helper()
	ctx := context.Background()

	req := testcontainers.ContainerRequest{
		Image:        "docker.io/library/postgres:16-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
		},
		WaitingFor: wait.ForLog("database system is ready to accept connections").WithStartupTimeout(60 * time.Second),
	}
	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Skipf("跳过 PostgreSQL 测试（无法启动容器，可能是镜像未缓存）: %v", err)
		return nil
	}

	host, err := pgContainer.Host(ctx)
	if err != nil {
		pgContainer.Terminate(ctx)
		t.Fatalf("获取容器 host 失败: %v", err)
	}
	port, err := pgContainer.MappedPort(ctx, "5432/tcp")
	if err != nil {
		pgContainer.Terminate(ctx)
		t.Fatalf("获取容器 port 失败: %v", err)
	}

	store, err := fileutil.NewJSONStore()
	if err != nil {
		pgContainer.Terminate(ctx)
		t.Fatalf("创建 JSONStore 失败: %v", err)
	}
	connRepo := repository.NewConnectionRepository(store)
	connMgr := repository.NewConnectionManager(connRepo, &repository.PostgresDriver{}, &repository.PostgresSchema{})
	connSvc := NewConnectionService(connMgr, connRepo)

	conn, err := connSvc.Create(model.CreateConnectionParams{
		Name:     "integration-test-pg",
		Host:     host,
		Port:     portToInt(t, port),
		Username: "testuser",
		Password: "testpass",
		Database: "testdb",
		Timezone: "UTC",
	})
	if err != nil {
		connMgr.CloseAll()
		pgContainer.Terminate(ctx)
		t.Fatalf("创建连接失败: %v", err)
	}

	return &pgSuite{
		container: pgContainer,
		connRepo:  connRepo,
		connMgr:   connMgr,
		connSvc:   connSvc,
		connID:    conn.ID,
		dbName:    "testdb",
	}
}

func (s *pgSuite) cleanup(t *testing.T) {
	t.Helper()
	s.connMgr.CloseAll()
	if err := s.container.Terminate(context.Background()); err != nil {
		t.Logf("清理 PostgreSQL 容器失败: %v", err)
	}
}

func TestIntegration_Postgres_GetDatabases(t *testing.T) {
	suite := setupPostgres(t)
	if suite == nil {
		return // skipped
	}
	defer suite.cleanup(t)

	dbs, err := suite.connSvc.GetDatabases(suite.connID)
	if err != nil {
		t.Fatalf("GetDatabases 失败: %v", err)
	}

	if len(dbs) == 0 {
		t.Fatal("期望返回至少一个数据库，但列表为空")
	}

	foundTestDB := false
	for _, db := range dbs {
		if db == suite.dbName {
			foundTestDB = true
			break
		}
	}
	if !foundTestDB {
		t.Errorf("数据库列表中未找到 %q，实际列表: %v", suite.dbName, dbs)
	}

	// PostgreSQL should NOT return template databases (template0/template1)
	for _, db := range dbs {
		if db == "template0" || db == "template1" {
			t.Errorf("PostgreSQL 不应返回模板数据库 %q", db)
		}
	}
}

func TestIntegration_Postgres_GetTables(t *testing.T) {
	suite := setupPostgres(t)
	if suite == nil {
		return
	}
	defer suite.cleanup(t)

	// Empty testdb — should return empty slice
	tables, err := suite.connSvc.GetTables(suite.connID, suite.dbName)
	if err != nil {
		t.Fatalf("GetTables 失败: %v", err)
	}

	if tables == nil {
		t.Error("空数据库应返回空切片，不是 nil")
	}
}

func TestIntegration_Postgres_GetDatabases_InvalidConnection(t *testing.T) {
	suite := setupPostgres(t)
	if suite == nil {
		return
	}
	defer suite.cleanup(t)

	_, err := suite.connSvc.GetDatabases("nonexistent_conn_id")
	if err == nil {
		t.Error("不存在的连接应返回错误")
	}
}

func TestIntegration_Postgres_GetTables_InvalidConnection(t *testing.T) {
	suite := setupPostgres(t)
	if suite == nil {
		return
	}
	defer suite.cleanup(t)

	_, err := suite.connSvc.GetTables("nonexistent_conn_id", suite.dbName)
	if err == nil {
		t.Error("不存在的连接应返回错误")
	}
}

// -- Edge case: CloseAll is safe to call and pools are recreated on demand --

func TestIntegration_MySQL_CloseAll_RecreatesPools(t *testing.T) {
	suite := setupMySQL(t)
	// CloseAll should not panic
	suite.connMgr.CloseAll()

	// After CloseAll, pools are recreated on demand — GetDatabases should still work
	dbs, err := suite.connSvc.GetDatabases(suite.connID)
	if err != nil {
		t.Fatalf("CloseAll 后 GetDatabases 应仍可正常工作: %v", err)
	}
	if len(dbs) == 0 {
		t.Error("关闭池后应仍能获取数据库列表")
	}

	suite.container.Terminate(context.Background())
}

// -- Edge case: concurrent Execute calls --

func TestIntegration_MySQL_Execute_Concurrent(t *testing.T) {
	suite := setupMySQL(t)
	defer suite.cleanup(t)

	// Insert some data first
	_, err := suite.querySvc.Execute(suite.connID, suite.dbName,
		"CREATE TABLE IF NOT EXISTS test_concurrent (id INT AUTO_INCREMENT PRIMARY KEY, val INT)")
	if err != nil {
		t.Fatalf("CREATE TABLE 失败: %v", err)
	}

	// Insert 100 rows
	batch := "INSERT INTO test_concurrent (val) VALUES "
	for i := 0; i < 100; i++ {
		if i > 0 {
			batch += ", "
		}
		batch += fmt.Sprintf("(%d)", i)
	}
	_, err = suite.querySvc.Execute(suite.connID, suite.dbName, batch)
	if err != nil {
		t.Fatalf("批量 INSERT 失败: %v", err)
	}

	// Execute concurrent SELECTs
	results := make(chan *model.QueryResult, 3)
	errs := make(chan error, 3)

	for i := 0; i < 3; i++ {
		go func(idx int) {
			r, e := suite.querySvc.Execute(suite.connID, suite.dbName, "SELECT COUNT(*) AS cnt FROM test_concurrent")
			if e != nil {
				errs <- e
			} else {
				results <- r
			}
		}(i)
	}

	for i := 0; i < 3; i++ {
		select {
		case err := <-errs:
			t.Errorf("并发查询 %d 失败: %v", i, err)
		case result := <-results:
			if result.MessageType != model.ResultSuccess {
				t.Errorf("并发查询 %d messageType=%s, 期望 success", i, result.MessageType)
			}
			if len(result.Rows) != 1 {
				t.Errorf("并发查询 %d 期望 1 行, 实际=%d", i, len(result.Rows))
			}
		}
	}

	// Cleanup
	_, _ = suite.querySvc.Execute(suite.connID, suite.dbName, "DROP TABLE test_concurrent")
}
