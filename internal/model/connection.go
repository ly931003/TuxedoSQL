package model

import "time"

// Connection 表示一个 MySQL 数据库连接配置。
type Connection struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	GroupID   string    `json:"groupId"`
	Host      string    `json:"host"`
	Port      int       `json:"port"`
	Username  string    `json:"username"`
	Password  string    `json:"password"`
	Database  string    `json:"database"`
	Timezone  string    `json:"timezone"` // IANA 时区名（如 "Asia/Shanghai"），空值等价于 "Local"
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// ConnectionGroup 表示连接分组（文件夹式管理）。
type ConnectionGroup struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
}

// TreeNode 表示数据库树中的一个节点，可以是连接、数据库、表或列。
type TreeNode struct {
	Key      string     `json:"key"`
	Label    string     `json:"label"`
	Type     string     `json:"type"` // "group" | "connection" | "database" | "table"
	Children []TreeNode `json:"children,omitempty"`
	Leaf     bool       `json:"leaf"`
}

// CreateConnectionParams 是创建连接时的请求参数。
type CreateConnectionParams struct {
	Name     string `json:"name"`
	GroupID  string `json:"groupId"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Timezone string `json:"timezone"` // IANA 时区名（如 "Asia/Shanghai"），空值等价于 "Local"
}

// UpdateConnectionParams 是更新连接时的请求参数。
type UpdateConnectionParams struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	GroupID  string `json:"groupId"`
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
	Password string `json:"password"`
	Database string `json:"database"`
	Timezone string `json:"timezone"` // IANA 时区名（如 "Asia/Shanghai"），空值等价于 "Local"
}

// TestResult 是测试连接的结果。
type TestResult struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

// CreateGroupParams 是创建分组时的请求参数。
type CreateGroupParams struct {
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
}

// UpdateGroupParams 是更新分组时的请求参数。
type UpdateGroupParams struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	ParentID string `json:"parentId"`
}

// CreateDatabaseParams 是创建新数据库的请求参数。
type CreateDatabaseParams struct {
	ConnectionID string `json:"connectionId"` // 连接ID
	DatabaseName string `json:"databaseName"` // 新数据库名
	Charset      string `json:"charset"`      // 字符集（如 "utf8mb4"），空表示使用 MySQL 默认值
	Collation    string `json:"collation"`    // 排序规则（如 "utf8mb4_unicode_ci"），空表示使用 MySQL 默认值
}

// CreateTableParams 是创建新表的请求参数。
type CreateTableParams struct {
	ConnectionID string       `json:"connectionId"` // 连接ID
	DatabaseName string       `json:"databaseName"` // 目标数据库名
	TableName    string       `json:"tableName"`    // 新表名
	Charset      string       `json:"charset"`      // 字符集
	Collation    string       `json:"collation"`    // 排序规则
	Comment      string       `json:"comment"`      // 表注释
	Columns      []ColumnDef  `json:"columns"`      // 列定义
}

// ColumnDef 是建表时的一列定义。
type ColumnDef struct {
	Name          string `json:"name"`          // 列名
	DataType      string `json:"dataType"`      // 数据类型（如 "INT", "VARCHAR(255)"）
	Nullable      bool   `json:"nullable"`      // 是否可空
	DefaultValue  string `json:"defaultValue"`  // 默认值（空串表示无默认值）
	AutoIncrement bool   `json:"autoIncrement"` // 是否自增
	Unsigned      bool   `json:"unsigned"`      // 是否无符号（数值类型）
	Comment       string `json:"comment"`       // 列注释
	IsPrimaryKey  bool   `json:"isPrimaryKey"`  // 是否主键
}

// DDLResult 是 DDL 操作的通用返回结果。
type DDLResult struct {
	SQL     string `json:"sql"`     // 执行的 SQL
	Message string `json:"message"` // 操作结果消息
}

// CharsetInfo 表示 MySQL 支持的字符集及其默认排序规则。
type CharsetInfo struct {
	Charset         string `json:"charset"`         // 字符集名
	DefaultCollation string `json:"defaultCollation"` // 默认排序规则
	Description     string `json:"description"`     // 描述
}
