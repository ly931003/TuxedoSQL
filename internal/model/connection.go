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
