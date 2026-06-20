package service

import (
	"context"
	"fmt"
	"strings"
	"time"


	"tuxedosql/internal/model"
	"tuxedosql/internal/repository"
)

// ConnectionService 管理数据库连接的增删改查、测试和元数据浏览。
type ConnectionService struct {
	repo        repository.ConnectionStore
	connManager repository.PoolManager
}

// NewConnectionService 创建一个新的 ConnectionService。
// connManager 可以为 nil（仅测试场景不需要数据库连接时）。
func NewConnectionService(connManager repository.PoolManager, connRepo repository.ConnectionStore) *ConnectionService {
	return &ConnectionService{
		repo:        connRepo,
		connManager: connManager,
	}
}

// Create 创建一条新的数据库连接配置。
func (s *ConnectionService) Create(params model.CreateConnectionParams) (*model.Connection, error) {
	if params.Name == "" {
		return nil, fmt.Errorf("连接名称不能为空")
	}
	if params.Host == "" {
		return nil, fmt.Errorf("主机地址不能为空")
	}
	if params.Port <= 0 {
		params.Port = 3306
	}
	if params.Username == "" {
		return nil, fmt.Errorf("用户名不能为空")
	}

	connections, err := s.repo.LoadConnections()
	if err != nil {
		return nil, fmt.Errorf("加载连接列表失败: %w", err)
	}

	now := time.Now()
	tz := params.Timezone
	if tz == "" {
		tz = "Local"
	}
	conn := model.Connection{
		ID:        fmt.Sprintf("conn_%d", now.UnixMilli()),
		Name:      params.Name,
		GroupID:   params.GroupID,
		Host:      params.Host,
		Port:      params.Port,
		Username:  params.Username,
		Password:  params.Password,
		Database:  params.Database,
		Timezone:  tz,
		SSH:       params.SSH,
		CreatedAt: now,
		UpdatedAt: now,
	}

	connections = append(connections, conn)
	if err := s.repo.SaveConnections(connections); err != nil {
		return nil, fmt.Errorf("保存连接失败: %w", err)
	}

	return &conn, nil
}

// Update 更新一条已有的数据库连接配置。
func (s *ConnectionService) Update(params model.UpdateConnectionParams) (*model.Connection, error) {
	if params.ID == "" {
		return nil, fmt.Errorf("连接ID不能为空")
	}

	connections, err := s.repo.LoadConnections()
	if err != nil {
		return nil, fmt.Errorf("加载连接列表失败: %w", err)
	}

	idx := -1
	for i, c := range connections {
		if c.ID == params.ID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, fmt.Errorf("连接不存在: %s", params.ID)
	}

	conn := &connections[idx]
	conn.Name = params.Name
	conn.GroupID = params.GroupID
	conn.Host = params.Host
	conn.Port = params.Port
	conn.Username = params.Username
	conn.Password = params.Password
	conn.Database = params.Database
	conn.SSH = params.SSH
	conn.Name = params.Name
	conn.GroupID = params.GroupID
	conn.Host = params.Host
	conn.Port = params.Port
	conn.Username = params.Username
	conn.Password = params.Password
	conn.Database = params.Database
	if params.Timezone == "" {
		conn.Timezone = "Local"
	} else {
		conn.Timezone = params.Timezone
	}
	conn.UpdatedAt = time.Now()

	if err := s.repo.SaveConnections(connections); err != nil {
		return nil, fmt.Errorf("保存连接失败: %w", err)
	}

	// 关闭旧连接池：配置变更后旧池的 DSN 已过期，必须回收
	if s.connManager != nil {
		s.connManager.Close(params.ID)
	}

	return conn, nil
}

// Delete 删除一条数据库连接配置。
func (s *ConnectionService) Delete(id string) error {
	if id == "" {
		return fmt.Errorf("连接ID不能为空")
	}

	connections, err := s.repo.LoadConnections()
	if err != nil {
		return fmt.Errorf("加载连接列表失败: %w", err)
	}

	filtered := make([]model.Connection, 0, len(connections))
	found := false
	for _, c := range connections {
		if c.ID == id {
			found = true
			continue
		}
		filtered = append(filtered, c)
	}
	if !found {
		return fmt.Errorf("连接不存在: %s", id)
	}

	if err := s.repo.SaveConnections(filtered); err != nil {
		return err
	}
	// 清理 OS 密钥环中已删除连接的凭证条目
	s.repo.DeleteCredential(id)
	// 关闭已删除连接的连接池
	if s.connManager != nil {
		s.connManager.Close(id)
	}
	return nil
}

// List 返回所有连接配置。
func (s *ConnectionService) List() ([]model.Connection, error) {
	return s.repo.LoadConnections()
}

// TestConnection 测试数据库连接是否有效。
func (s *ConnectionService) TestConnection(id string) model.TestResult {
	if s.connManager == nil {
		return model.TestResult{Success: false, Message: "连接管理器未初始化"}
	}

	conn, err := s.findConnection(id)
	if err != nil {
		return model.TestResult{Success: false, Message: err.Error()}
	}

	db, err := s.connManager.GetDB(conn, conn.Database)
	if err != nil {
		return model.TestResult{Success: false, Message: fmt.Sprintf("连接测试失败: %v", err)}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return model.TestResult{Success: false, Message: fmt.Sprintf("连接测试失败: %v", err)}
	}

	return model.TestResult{Success: true, Message: "连接成功"}
}

// GetDatabases 获取指定连接下的所有数据库列表。
func (s *ConnectionService) GetDatabases(connectionID string) ([]string, error) {
	if s.connManager == nil {
		return nil, fmt.Errorf("连接管理器未初始化")
	}
	conn, err := s.findConnection(connectionID)
	if err != nil {
		return nil, err
	}

	db, err := s.connManager.GetDB(conn, "")
	if err != nil {
		return nil, err
	}

	schema := s.connManager.Schema()
	rows, err := db.Query(schema.ListDatabasesQuery())
	if err != nil {
		return nil, fmt.Errorf("查询数据库列表失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var databases []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("读取数据库名失败: %w", err)
		}
		databases = append(databases, name)
	}

	return databases, rows.Err()
}

// GetTables 获取指定数据库中所有表的列表。
func (s *ConnectionService) GetTables(connectionID, databaseName string) ([]string, error) {
	if s.connManager == nil {
		return nil, fmt.Errorf("连接管理器未初始化")
	}
	conn, err := s.findConnection(connectionID)
	if err != nil {
		return nil, err
	}

	// 直接连接到目标数据库，避免 USE 切换（MySQL）或跨库查询（PostgreSQL）
	db, err := s.connManager.GetDB(conn, databaseName)
	if err != nil {
		return nil, err
	}

	schema := s.connManager.Schema()
	rows, err := db.Query(schema.ListTablesQuery())
	if err != nil {
		return nil, fmt.Errorf("查询表列表失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("读取表名失败: %w", err)
		}
		tables = append(tables, name)
	}

	return tables, rows.Err()
}

// CreateGroup 创建一个新的连接分组。
func (s *ConnectionService) CreateGroup(params model.CreateGroupParams) (*model.ConnectionGroup, error) {
	if params.Name == "" {
		return nil, fmt.Errorf("分组名称不能为空")
	}

	groups, err := s.repo.LoadGroups()
	if err != nil {
		return nil, fmt.Errorf("加载分组列表失败: %w", err)
	}

	group := model.ConnectionGroup{
		ID:       fmt.Sprintf("group_%d", time.Now().UnixMilli()),
		Name:     params.Name,
		ParentID: params.ParentID,
	}

	groups = append(groups, group)
	if err := s.repo.SaveGroups(groups); err != nil {
		return nil, fmt.Errorf("保存分组失败: %w", err)
	}

	return &group, nil
}

// ListGroups 返回所有连接分组。
func (s *ConnectionService) ListGroups() ([]model.ConnectionGroup, error) {
	return s.repo.LoadGroups()
}

// DeleteGroup 删除一个分组及其下所有连接（连接移至未分组）、子分组移至上级。
func (s *ConnectionService) DeleteGroup(id string) error {
	if id == "" {
		return fmt.Errorf("分组ID不能为空")
	}

	groups, err := s.repo.LoadGroups()
	if err != nil {
		return fmt.Errorf("加载分组列表失败: %w", err)
	}

	var targetParentID string
	filtered := make([]model.ConnectionGroup, 0, len(groups))
	found := false
	for _, g := range groups {
		if g.ID == id {
			found = true
			targetParentID = g.ParentID
			continue
		}
		filtered = append(filtered, g)
	}
	if !found {
		return fmt.Errorf("分组不存在: %s", id)
	}

	// Re-parent child groups to the deleted group's parent
	for i := range filtered {
		if filtered[i].ParentID == id {
			filtered[i].ParentID = targetParentID
		}
	}

	if err := s.repo.SaveGroups(filtered); err != nil {
		return fmt.Errorf("保存分组失败: %w", err)
	}

	connections, err := s.repo.LoadConnections()
	if err != nil {
		return fmt.Errorf("加载连接列表失败: %w", err)
	}
	for i := range connections {
		if connections[i].GroupID == id {
			connections[i].GroupID = ""
		}
	}
	return s.repo.SaveConnections(connections)
}

// UpdateGroup 更新一个分组（名称或父分组）。
func (s *ConnectionService) UpdateGroup(params model.UpdateGroupParams) (*model.ConnectionGroup, error) {
	if params.ID == "" {
		return nil, fmt.Errorf("分组ID不能为空")
	}
	if params.Name == "" {
		return nil, fmt.Errorf("分组名称不能为空")
	}
	// Prevent circular reference: cannot set parent to itself
	if params.ParentID == params.ID {
		return nil, fmt.Errorf("分组不能将自己作为父分组")
	}

	groups, err := s.repo.LoadGroups()
	if err != nil {
		return nil, fmt.Errorf("加载分组列表失败: %w", err)
	}

	idx := -1
	for i, g := range groups {
		if g.ID == params.ID {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, fmt.Errorf("分组不存在: %s", params.ID)
	}

	// Prevent circular reference: check if new parent is a descendant
	if params.ParentID != "" {
		if s.isDescendant(groups, params.ID, params.ParentID) {
			return nil, fmt.Errorf("不能将分组移动到自己的子分组下")
		}
	}

	groups[idx].Name = params.Name
	groups[idx].ParentID = params.ParentID
	if err := s.repo.SaveGroups(groups); err != nil {
		return nil, fmt.Errorf("保存分组失败: %w", err)
	}

	return &groups[idx], nil
}

// isDescendant checks whether targetID is a descendant of ancestorID in the group tree.
func (s *ConnectionService) isDescendant(groups []model.ConnectionGroup, ancestorID, targetID string) bool {
	// Build parent->children map
	children := make(map[string][]string)
	for _, g := range groups {
		children[g.ParentID] = append(children[g.ParentID], g.ID)
	}
	// BFS from ancestorID — if we reach targetID, then target is a descendant
	queue := []string{ancestorID}
	visited := make(map[string]bool)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if visited[current] {
			continue
		}
		visited[current] = true
		if current == targetID {
			return true
		}
		queue = append(queue, children[current]...)
	}
	return false
}

// isSystemDatabase performs case-insensitive lookup in the system database set.
func isSystemDatabase(systemDBs map[string]bool, name string) bool {
	for db := range systemDBs {
		if strings.EqualFold(db, name) {
			return true
		}
	}
	return false
}

func (s *ConnectionService) findConnection(id string) (*model.Connection, error) {
	connections, err := s.repo.LoadConnections()
	if err != nil {
		return nil, fmt.Errorf("加载连接列表失败: %w", err)
	}
	for i := range connections {
		if connections[i].ID == id {
			return &connections[i], nil
		}
	}
	return nil, fmt.Errorf("连接不存在: %s", id)
}

// CreateDatabase 在指定连接上创建一个新数据库。
func (s *ConnectionService) CreateDatabase(params model.CreateDatabaseParams) (*model.DDLResult, error) {
	if params.ConnectionID == "" {
		return nil, fmt.Errorf("连接ID不能为空")
	}
	if params.DatabaseName == "" {
		return nil, fmt.Errorf("数据库名不能为空")
	}

	conn, err := s.findConnection(params.ConnectionID)
	if err != nil {
		return nil, err
	}

	db, err := s.connManager.GetDB(conn, "")
	if err != nil {
		return nil, err
	}

	schema := s.connManager.Schema()
	safeName := schema.QuoteIdentifier(params.DatabaseName)
	createSQL := "CREATE DATABASE " + safeName
	if params.Charset != "" {
		createSQL += " CHARACTER SET " + strings.ReplaceAll(params.Charset, "'", "")
	}
	if params.Collation != "" {
		createSQL += " COLLATE " + strings.ReplaceAll(params.Collation, "'", "")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, createSQL); err != nil {
		return nil, fmt.Errorf("创建数据库失败: %w", err)
	}

	return &model.DDLResult{
		SQL:     createSQL,
		Message: fmt.Sprintf("数据库 \"%s\" 创建成功", params.DatabaseName),
	}, nil
}

// DropDatabase 删除指定连接上的一个数据库。
func (s *ConnectionService) DropDatabase(connectionID, databaseName string) (*model.DDLResult, error) {
	if connectionID == "" {
		return nil, fmt.Errorf("连接ID不能为空")
	}
	if databaseName == "" {
		return nil, fmt.Errorf("数据库名不能为空")
	}

	// 安全检查：禁止删除系统数据库（通过 SchemaIntrospector 获取系统库列表）
	schema := s.connManager.Schema()
	systemDBs := schema.SystemDatabases()
	if isSystemDatabase(systemDBs, databaseName) {
		return nil, fmt.Errorf("禁止删除系统数据库: %s", databaseName)
	}

	conn, err := s.findConnection(connectionID)
	if err != nil {
		return nil, err
	}

	db, err := s.connManager.GetDB(conn, "")
	if err != nil {
		return nil, err
	}

	safeName := schema.QuoteIdentifier(databaseName)
	dropSQL := "DROP DATABASE " + safeName

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, dropSQL); err != nil {
		return nil, fmt.Errorf("删除数据库失败: %w", err)
	}

	// 清理连接池：删除数据库后，该 database 的连接池已无效
	s.connManager.Close(connectionID)

	return &model.DDLResult{
		SQL:     dropSQL,
		Message: fmt.Sprintf("数据库 \"%s\" 已删除", databaseName),
	}, nil
}

// CreateTable 在指定数据库中创建一个新表。
func (s *ConnectionService) CreateTable(params model.CreateTableParams) (*model.DDLResult, error) {
	if params.ConnectionID == "" {
		return nil, fmt.Errorf("连接ID不能为空")
	}
	if params.DatabaseName == "" {
		return nil, fmt.Errorf("数据库名不能为空")
	}
	if params.TableName == "" {
		return nil, fmt.Errorf("表名不能为空")
	}
	if len(params.Columns) == 0 {
		return nil, fmt.Errorf("至少需要定义一个列")
	}

	conn, err := s.findConnection(params.ConnectionID)
	if err != nil {
		return nil, err
	}

	db, err := s.connManager.GetDB(conn, params.DatabaseName)
	if err != nil {
		return nil, err
	}

	schema := s.connManager.Schema()
	safeTable := schema.QuoteIdentifier(params.TableName)

	var colDefs []string
	var pkCols []string
	for _, col := range params.Columns {
		safeCol := schema.QuoteIdentifier(col.Name)
		def := safeCol + " " + col.DataType
		if col.Unsigned {
			def += " UNSIGNED"
		}
		if !col.Nullable {
			def += " NOT NULL"
		} else {
			def += " NULL"
		}
		if col.AutoIncrement {
			def += " AUTO_INCREMENT"
		}
		if col.DefaultValue != "" {
			def += " DEFAULT " + col.DefaultValue
		}
		if col.Comment != "" {
			def += " COMMENT '" + strings.ReplaceAll(col.Comment, "'", "\\'") + "'"
		}
		colDefs = append(colDefs, def)
		if col.IsPrimaryKey {
			pkCols = append(pkCols, safeCol)
		}
	}

	if len(pkCols) > 0 {
		colDefs = append(colDefs, "PRIMARY KEY ("+strings.Join(pkCols, ", ")+")")
	}

	createSQL := "CREATE TABLE " + safeTable + " (\n  " + strings.Join(colDefs, ",\n  ") + "\n)"
	if params.Charset != "" {
		createSQL += " CHARACTER SET " + strings.ReplaceAll(params.Charset, "'", "")
	}
	if params.Collation != "" {
		createSQL += " COLLATE " + strings.ReplaceAll(params.Collation, "'", "")
	}
	if params.Comment != "" {
		createSQL += " COMMENT='" + strings.ReplaceAll(params.Comment, "'", "\\'") + "'"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, createSQL); err != nil {
		return nil, fmt.Errorf("创建表失败: %w", err)
	}

	return &model.DDLResult{
		SQL:     createSQL,
		Message: fmt.Sprintf("表 \"%s\" 创建成功", params.TableName),
	}, nil
}

// DropTable 删除指定数据库中的表。
func (s *ConnectionService) DropTable(connectionID, databaseName, tableName string) (*model.DDLResult, error) {
	if connectionID == "" {
		return nil, fmt.Errorf("连接ID不能为空")
	}
	if databaseName == "" {
		return nil, fmt.Errorf("数据库名不能为空")
	}
	if tableName == "" {
		return nil, fmt.Errorf("表名不能为空")
	}

	conn, err := s.findConnection(connectionID)
	if err != nil {
		return nil, err
	}

	db, err := s.connManager.GetDB(conn, databaseName)
	if err != nil {
		return nil, err
	}

	schema := s.connManager.Schema()
	safeTable := schema.QuoteIdentifier(tableName)
	dropSQL := "DROP TABLE " + safeTable

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if _, err := db.ExecContext(ctx, dropSQL); err != nil {
		return nil, fmt.Errorf("删除表失败: %w", err)
	}

	return &model.DDLResult{
		SQL:     dropSQL,
		Message: fmt.Sprintf("表 \"%s\" 已删除", tableName),
	}, nil
}

// GetCharsets 返回 MySQL 支持的字符集列表（含默认排序规则）。
func (s *ConnectionService) GetCharsets(connectionID string) ([]model.CharsetInfo, error) {
	if connectionID == "" {
		return nil, fmt.Errorf("连接ID不能为空")
	}

	conn, err := s.findConnection(connectionID)
	if err != nil {
		return nil, err
	}

	db, err := s.connManager.GetDB(conn, "")
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SHOW CHARACTER SET")
	if err != nil {
		return nil, fmt.Errorf("查询字符集失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var charsets []model.CharsetInfo
	for rows.Next() {
		var charset, defaultCollation, desc string
		var maxLen int
		if err := rows.Scan(&charset, &desc, &defaultCollation, &maxLen); err != nil {
			return nil, fmt.Errorf("读取字符集信息失败: %w", err)
		}
		if strings.HasPrefix(charset, "utf8") || strings.HasPrefix(charset, "gb") || strings.HasPrefix(charset, "latin") || charset == "ascii" {
			charsets = append(charsets, model.CharsetInfo{
				Charset:          charset,
				DefaultCollation: defaultCollation,
				Description:      desc,
			})
		}
	}

	// 对于剩余字符集，全部追加以确保完整性
	// 但迭代器已经消耗，需要重新查询
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return charsets, nil
}

// GetCollations 返回指定字符集对应的排序规则列表。
func (s *ConnectionService) GetCollations(connectionID, charset string) ([]string, error) {
	if connectionID == "" {
		return nil, fmt.Errorf("连接ID不能为空")
	}
	if charset == "" {
		return nil, fmt.Errorf("字符集名不能为空")
	}

	conn, err := s.findConnection(connectionID)
	if err != nil {
		return nil, err
	}

	db, err := s.connManager.GetDB(conn, "")
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rows, err := db.QueryContext(ctx, "SHOW COLLATION WHERE Charset = ?", charset)
	if err != nil {
		return nil, fmt.Errorf("查询排序规则失败: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var collations []string
	for rows.Next() {
		var collation string
		var isDefault string
		var compiled string
		var sortlen int
		if err := rows.Scan(&collation, &charset, &isDefault, &compiled, &sortlen); err != nil {
			return nil, fmt.Errorf("读取排序规则失败: %w", err)
		}
		collations = append(collations, collation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return collations, nil
}
