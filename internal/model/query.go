package model

// ColumnInfo 描述查询结果中的一列元数据。
type ColumnInfo struct {
	Name string `json:"name"` // 列名
	Type string `json:"type"` // MySQL 类型字符串，如 "VARCHAR(255)", "INT"
}

// ResultType 表示查询结果消息的类型。
type ResultType string

const (
	ResultSuccess ResultType = "success"
	ResultError   ResultType = "error"
	ResultInfo    ResultType = "info"
)

// QueryResult 是一次 SQL 查询的执行结果。
type QueryResult struct {
	Columns      []ColumnInfo     `json:"columns"`      // 列元数据（仅 SELECT 类查询有值）
	Rows         []map[string]any `json:"rows"`         // 数据行（仅 SELECT 类查询有值）
	AffectedRows int64            `json:"affectedRows"` // 影响行数（INSERT/UPDATE/DELETE 等）
	Message      string           `json:"message"`      // 执行状态消息，如 "3 rows affected"
	MessageType  ResultType       `json:"messageType"`  // 消息类型：success / error / info
	Duration     int64            `json:"duration"`     // 执行耗时（毫秒）
}

// TabState 表示一个持久化的查询标签页状态，用于应用重启后恢复标签。
type TabState struct {
	ID           string `json:"id"`           // 标签唯一标识
	Title        string `json:"title"`        // 标签标题
	ConnectionID string `json:"connectionId"` // 关联的连接ID
	Database     string `json:"database"`     // 当前选中的数据库
	SQL          string `json:"sql"`          // 编辑器中的 SQL 文本
}

// TableSchema 表示表的一列结构元数据，来自 INFORMATION_SCHEMA.COLUMNS。
type TableSchema struct {
	Name         string `json:"name"`         // 列名
	DataType     string `json:"dataType"`     // MySQL 数据类型，如 "varchar", "int"
	IsNullable   bool   `json:"isNullable"`   // 是否可空
	ColumnKey    string `json:"columnKey"`    // 键类型：PRI / UNI / MUL / ""
	DefaultValue string `json:"defaultValue"` // 默认值（字符串表示），可为空
}

// SortOrder 表示排序方向。
type SortOrder string

const (
	SortASC  SortOrder = "ASC"
	SortDESC SortOrder = "DESC"
)

// FilterOperator 表示筛选操作符。
type FilterOperator string

const (
	OpEQ       FilterOperator = "eq"
	OpNEQ      FilterOperator = "neq"
	OpContains FilterOperator = "contains"
	OpGT       FilterOperator = "gt"
	OpLT       FilterOperator = "lt"
	OpIsNull   FilterOperator = "isnull"
	OpNotNull  FilterOperator = "notnull"
)

// LogicOp 表示筛选组内的布尔连接词。
type LogicOp string

const (
	LogicAND LogicOp = "AND"
	LogicOR  LogicOp = "OR"
)

// FilterCondition 表示一个叶子筛选条件（列 op 值）。
type FilterCondition struct {
	Column   string         `json:"column"`   // 列名
	Operator FilterOperator `json:"operator"` // 操作符
	Value    string         `json:"value"`    // 筛选值（isnull/notnull 时忽略）
}

// FilterGroup 表示一个可嵌套的布尔筛选表达式。
// 若 Conditions 非空，则为 AND/OR 组合节点；否则为叶子节点（使用 Column/Operator/Value）。
type FilterGroup struct {
	Logic      LogicOp          `json:"logic"`      // 组合逻辑：AND 或 OR
	Conditions []*FilterGroup   `json:"conditions"`  // 嵌套子组（至少 2 个）
	Column     string           `json:"column"`      // 叶子节点：列名
	Operator   FilterOperator   `json:"operator"`    // 叶子节点：操作符
	Value      string           `json:"value"`       // 叶子节点：筛选值
}

// IsLeaf reports whether this group is a leaf condition (not a logic group).
func (g *FilterGroup) IsLeaf() bool {
	return len(g.Conditions) == 0
}

// TableDataParams 是 GetTableData 的入参，包含分页、排序和筛选条件。
type TableDataParams struct {
	ConnectionID string       `json:"connectionId"` // 连接ID
	Database     string       `json:"database"`     // 数据库名
	Table        string       `json:"table"`        // 表名
	Page         int          `json:"page"`         // 页码，从1开始
	PageSize     int          `json:"pageSize"`     // 每页条数
	SortColumn   string       `json:"sortColumn"`   // 排序列名，空表示不排序
	SortOrder    SortOrder    `json:"sortOrder"`    // 排序方向
	Filters      *FilterGroup `json:"filters"`      // 筛选条件（nil 表示无筛选）
}

// PageResult 是一次分页查询的返回结果。
type PageResult struct {
	Columns     []ColumnInfo     `json:"columns"`     // 列元数据
	Rows        []map[string]any `json:"rows"`        // 当前页数据行
	Total       int64            `json:"total"`       // 符合条件的总行数
	Page        int              `json:"page"`        // 当前页码
	PageSize    int              `json:"pageSize"`    // 每页条数
	TotalPages  int              `json:"totalPages"`  // 总页数
	Message     string           `json:"message"`     // 状态消息
	MessageType ResultType       `json:"messageType"` // 消息类型
	Duration    int64            `json:"duration"`    // 执行耗时（毫秒）
	SQL         string           `json:"sql"`         // 执行的 SELECT 语句（用于审计）
}

// UpdateRowParams 包含更新表中单行单个单元格所需的全部参数。
// PkValues 是从主键列名到其值的映射，用于构建 WHERE 子句。
type UpdateRowParams struct {
	ConnectionID string         `json:"connectionId"` // 连接ID
	Database     string         `json:"database"`     // 数据库名
	Table        string         `json:"table"`        // 表名
	PkValues     map[string]any `json:"pkValues"`     // 主键列名 → 值
	Column       string         `json:"column"`       // 要更新的目标列名
	NewValue     any            `json:"newValue"`     // 新值（nil 表示 SQL NULL）
}

// UpdateRowResult 报告单行更新操作的结果。
type UpdateRowResult struct {
	AffectedRows int64  `json:"affectedRows"` // 受影响行数
	Message      string `json:"message"`      // 操作结果消息
	SQL          string `json:"sql"`          // 执行的 SQL（用于审计）
}
