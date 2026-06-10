package service

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"tuxedosql/internal/model"
	"tuxedosql/internal/repository"
)

const maxRows = 10000

// QueryService 管理 SQL 查询执行和标签页持久化。
type QueryService struct {
	connManager *repository.ConnectionManager
	connRepo    *repository.ConnectionRepository
	tabRepo     *repository.TabRepository
}

// NewQueryService 创建一个新的 QueryService。
func NewQueryService(connManager *repository.ConnectionManager, connRepo *repository.ConnectionRepository, tabRepo *repository.TabRepository) *QueryService {
	return &QueryService{
		connManager: connManager,
		connRepo:    connRepo,
		tabRepo:     tabRepo,
	}
}

// Execute 在指定连接的指定数据库上执行 SQL 语句并返回结果。
func (s *QueryService) Execute(connectionID, database, sqlStmt string) (*model.QueryResult, error) {
	if connectionID == "" {
		return nil, fmt.Errorf("连接ID不能为空")
	}
	if strings.TrimSpace(sqlStmt) == "" {
		return nil, fmt.Errorf("SQL语句不能为空")
	}
	if database == "" {
		return nil, fmt.Errorf("数据库名不能为空")
	}

	_, db, err := s.connManager.GetDBByID(connectionID, database)
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	start := time.Now()

	trimmed := strings.TrimSpace(sqlStmt)
	upper := strings.ToUpper(trimmed)

	isQuery := strings.HasPrefix(upper, "SELECT") ||
		strings.HasPrefix(upper, "SHOW") ||
		strings.HasPrefix(upper, "DESCRIBE") ||
		strings.HasPrefix(upper, "EXPLAIN") ||
		strings.HasPrefix(upper, "DESC") ||
		strings.HasPrefix(upper, "WITH")

	if isQuery {
		return s.executeQuery(ctx, db, trimmed, start)
	}
	return s.executeExec(ctx, db, trimmed, start)
}

func (s *QueryService) executeQuery(ctx context.Context, db *sql.DB, sqlStmt string, start time.Time) (*model.QueryResult, error) {
	rows, err := db.QueryContext(ctx, sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("执行查询失败: %w", err)
	}
	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("获取列信息失败: %w", err)
	}

	columns := make([]model.ColumnInfo, len(columnTypes))
	for i, ct := range columnTypes {
		columns[i] = model.ColumnInfo{
			Name: ct.Name(),
			Type: ct.DatabaseTypeName(),
		}
	}

	var dataRows []map[string]any
	for rows.Next() {
		if len(dataRows) >= maxRows {
			dataRows = append(dataRows, map[string]any{
				"__warning__": fmt.Sprintf("结果集超过 %d 行，已截断", maxRows),
			})
			break
		}

		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("读取数据行失败: %w", err)
		}

		row := make(map[string]any, len(columns))
		for i, col := range columns {
			val := values[i]
			switch v := val.(type) {
			case []byte:
				row[col.Name] = string(v)
			case nil:
				row[col.Name] = nil
			default:
				row[col.Name] = v
			}
		}
		dataRows = append(dataRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果集失败: %w", err)
	}

	duration := time.Since(start).Milliseconds()
	message := fmt.Sprintf("返回 %d 行", len(dataRows))
	if len(dataRows) >= maxRows {
		message += "（已截断）"
	}
	return &model.QueryResult{
		Columns:     columns,
		Rows:        dataRows,
		Message:     message,
		MessageType: model.ResultSuccess,
		Duration:    duration,
	}, nil
}

func (s *QueryService) executeExec(ctx context.Context, db *sql.DB, sqlStmt string, start time.Time) (*model.QueryResult, error) {
	result, err := db.ExecContext(ctx, sqlStmt)
	if err != nil {
		return nil, fmt.Errorf("执行语句失败: %w", err)
	}

	affected, _ := result.RowsAffected()
	duration := time.Since(start).Milliseconds()

	message := fmt.Sprintf("%d 行受影响", affected)
	return &model.QueryResult{
		AffectedRows: affected,
		Message:      message,
		MessageType:  model.ResultSuccess,
		Duration:     duration,
	}, nil
}

// SaveTabs 持久化所有打开的标签页状态。
func (s *QueryService) SaveTabs(tabs []model.TabState) error {
	return s.tabRepo.SaveTabs(tabs)
}

// LoadTabs 从持久化存储中恢复标签页状态。
func (s *QueryService) LoadTabs() ([]model.TabState, error) {
	return s.tabRepo.LoadTabs()
}

// allowedOperators 是所有合法筛选操作符的白名单集合。
var allowedOperators = map[model.FilterOperator]bool{
	model.OpEQ:       true,
	model.OpNEQ:      true,
	model.OpContains: true,
	model.OpGT:       true,
	model.OpLT:       true,
	model.OpIsNull:   true,
	model.OpNotNull:  true,
}

func isValidSortOrder(order model.SortOrder) bool {
	return order == model.SortASC || order == model.SortDESC
}

// GetTableSchema 返回指定表的列结构元数据。
// 列信息来自 INFORMATION_SCHEMA.COLUMNS，用于前端展示列定义和构建列名白名单。
func (s *QueryService) GetTableSchema(connectionID, database, table string) ([]model.TableSchema, error) {
	if connectionID == "" {
		return nil, fmt.Errorf("连接ID不能为空")
	}
	if database == "" {
		return nil, fmt.Errorf("数据库名不能为空")
	}
	if table == "" {
		return nil, fmt.Errorf("表名不能为空")
	}

	_, db, err := s.connManager.GetDBByID(connectionID, database)
	if err != nil {
		return nil, err
	}

	query := `SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_KEY, COALESCE(COLUMN_DEFAULT, '')
		FROM INFORMATION_SCHEMA.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
		ORDER BY ORDINAL_POSITION`

	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	rows, err := db.QueryContext(ctx2, query, database, table)
	if err != nil {
		return nil, fmt.Errorf("查询表结构失败: %w", err)
	}
	defer rows.Close()

	var schemas []model.TableSchema
	for rows.Next() {
		var sc model.TableSchema
		var isNullable string
		if err := rows.Scan(&sc.Name, &sc.DataType, &isNullable, &sc.ColumnKey, &sc.DefaultValue); err != nil {
			return nil, fmt.Errorf("读取列信息失败: %w", err)
		}
		sc.IsNullable = isNullable == "YES"
		schemas = append(schemas, sc)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历列信息失败: %w", err)
	}

	return schemas, nil
}

// GetCreateTable 返回指定表的 CREATE TABLE 语句（SHOW CREATE TABLE 输出）。
func (s *QueryService) GetCreateTable(connectionID, database, table string) (string, error) {
	if connectionID == "" {
		return "", fmt.Errorf("连接ID不能为空")
	}
	if database == "" {
		return "", fmt.Errorf("数据库名不能为空")
	}
	if table == "" {
		return "", fmt.Errorf("表名不能为空")
	}

	_, db, err := s.connManager.GetDBByID(connectionID, database)
	if err != nil {
		return "", err
	}

	safeTable := "`" + strings.ReplaceAll(table, "`", "``") + "`"
	query := "SHOW CREATE TABLE " + safeTable

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	row := db.QueryRowContext(ctx, query)

	var tableName string
	var ddl string
	if err := row.Scan(&tableName, &ddl); err != nil {
		return "", fmt.Errorf("获取建表语句失败: %w", err)
	}

	return ddl, nil
}

// getColumnWhitelist 返回指定表的合法列名集合，用于排序/筛选列名白名单校验。
func (s *QueryService) getColumnWhitelist(connectionID, database, table string) (map[string]bool, error) {
	schemas, err := s.GetTableSchema(connectionID, database, table)
	if err != nil {
		return nil, err
	}
	wl := make(map[string]bool, len(schemas))
	for _, sc := range schemas {
		wl[sc.Name] = true
	}
	return wl, nil
}

// GetTableData 执行分页查询，支持排序和筛选。
// 列名（SortColumn、Filter.Column）通过 INFORMATION_SCHEMA 白名单校验，防止 SQL 注入。
// 筛选值使用参数化查询（? 占位符），仅操作符通过白名单校验后拼入 SQL。
func (s *QueryService) GetTableData(params model.TableDataParams) (*model.PageResult, error) {
	if params.ConnectionID == "" {
		return nil, fmt.Errorf("连接ID不能为空")
	}
	if params.Database == "" {
		return nil, fmt.Errorf("数据库名不能为空")
	}
	if params.Table == "" {
		return nil, fmt.Errorf("表名不能为空")
	}
	if params.Page < 1 {
		params.Page = 1
	}
	if params.PageSize <= 0 {
		params.PageSize = 100
	}
	if params.PageSize > 1000 {
		params.PageSize = 1000
	}

	_, db, err := s.connManager.GetDBByID(params.ConnectionID, params.Database)
	if err != nil {
		return nil, err
	}

	// getColumnWhitelist 内部调用 GetTableSchema，其 DSN 已指定 database，无需额外 USE
	whitelist, err := s.getColumnWhitelist(params.ConnectionID, params.Database, params.Table)
	if err != nil {
		return nil, fmt.Errorf("获取表结构失败: %w", err)
	}

	// 校验排序列名
	if params.SortColumn != "" {
		if !whitelist[params.SortColumn] {
			return nil, fmt.Errorf("无效的排序列名: %s", params.SortColumn)
		}
		if !isValidSortOrder(params.SortOrder) {
			return nil, fmt.Errorf("无效的排序方向: %s", params.SortOrder)
		}
	}

	// 安全构建表名引用
	safeTable := "`" + strings.ReplaceAll(params.Table, "`", "``") + "`"

	// 递归构建 WHERE 子句（参数化）
	whereSQL, args, err := buildFilterClause(params.Filters, whitelist)
	if err != nil {
		return nil, err
	}
	whereClause := ""
	if whereSQL != "" {
		whereClause = " WHERE " + whereSQL
	}

	// 构建 ORDER BY 子句（列名已白名单校验，排序方向已校验）
	orderClause := ""
	if params.SortColumn != "" && params.SortOrder != "" {
		safeSortCol := "`" + strings.ReplaceAll(params.SortColumn, "`", "``") + "`"
		orderClause = " ORDER BY " + safeSortCol + " " + string(params.SortOrder)
	}

	start := time.Now()

	// 查询总行数
	countQuery := "SELECT COUNT(*) FROM " + safeTable + whereClause
	ctx2, cancel2 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel2()

	var total int64
	countArgs := make([]interface{}, len(args))
	copy(countArgs, args)
	if err := db.QueryRowContext(ctx2, countQuery, countArgs...).Scan(&total); err != nil {
		return nil, fmt.Errorf("查询总数失败: %w", err)
	}

	// 查询当前页数据
	offset := (params.Page - 1) * params.PageSize
	dataQuery := "SELECT * FROM " + safeTable + whereClause + orderClause + " LIMIT ? OFFSET ?"

	ctx3, cancel3 := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel3()

	dataArgs := make([]interface{}, len(args))
	copy(dataArgs, args)
	dataArgs = append(dataArgs, params.PageSize, offset)

	rows, err := db.QueryContext(ctx3, dataQuery, dataArgs...)
	if err != nil {
		return nil, fmt.Errorf("查询表数据失败: %w", err)
	}
	defer rows.Close()

	columnTypes, err := rows.ColumnTypes()
	if err != nil {
		return nil, fmt.Errorf("获取列信息失败: %w", err)
	}

	columns := make([]model.ColumnInfo, len(columnTypes))
	for i, ct := range columnTypes {
		columns[i] = model.ColumnInfo{
			Name: ct.Name(),
			Type: ct.DatabaseTypeName(),
		}
	}

	var dataRows []map[string]any
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("读取数据行失败: %w", err)
		}

		row := make(map[string]any, len(columns))
		for i, col := range columns {
			val := values[i]
			switch v := val.(type) {
			case []byte:
				row[col.Name] = string(v)
			case nil:
				row[col.Name] = nil
			default:
				row[col.Name] = v
			}
		}
		dataRows = append(dataRows, row)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("遍历结果集失败: %w", err)
	}

	totalPages := int(total) / params.PageSize
	if int(total)%params.PageSize > 0 {
		totalPages++
	}

	duration := time.Since(start).Milliseconds()
	return &model.PageResult{
		Columns:     columns,
		Rows:        dataRows,
		Total:       total,
		Page:        params.Page,
		PageSize:    params.PageSize,
		TotalPages:  totalPages,
		Message:     fmt.Sprintf("第 %d/%d 页，共 %d 行", params.Page, totalPages, total),
		MessageType: model.ResultSuccess,
		Duration:    duration,
		SQL:         buildDisplaySQL(params, dataArgs),
	}, nil
}

// buildFilterClause 递归构建参数化 WHERE 子句，支持 AND/OR 嵌套。
// 返回 (SQL 子句, 参数切片, 错误)。根节点为 nil 时返回空串。
func buildFilterClause(group *model.FilterGroup, whitelist map[string]bool) (string, []interface{}, error) {
	if group == nil {
		return "", nil, nil
	}
	if group.IsLeaf() {
		if !whitelist[group.Column] {
			return "", nil, fmt.Errorf("无效的筛选列名: %s", group.Column)
		}
		if !allowedOperators[group.Operator] {
			return "", nil, fmt.Errorf("无效的筛选操作符: %s", group.Operator)
		}
		safeCol := "`" + strings.ReplaceAll(group.Column, "`", "``") + "`"
		switch group.Operator {
		case model.OpEQ:
			return safeCol + " = ?", []interface{}{group.Value}, nil
		case model.OpNEQ:
			return safeCol + " != ?", []interface{}{group.Value}, nil
		case model.OpContains:
			return safeCol + " LIKE ?", []interface{}{"%" + group.Value + "%"}, nil
		case model.OpGT:
			return safeCol + " > ?", []interface{}{group.Value}, nil
		case model.OpLT:
			return safeCol + " < ?", []interface{}{group.Value}, nil
		case model.OpIsNull:
			return safeCol + " IS NULL", nil, nil
		case model.OpNotNull:
			return safeCol + " IS NOT NULL", nil, nil
		default:
			return "", nil, fmt.Errorf("未知操作符: %s", group.Operator)
		}
	}

	// 组合节点：递归构建子句
	if len(group.Conditions) < 2 {
		return "", nil, fmt.Errorf("组合节点至少需要 2 个子条件")
	}

	connector := " " + string(group.Logic) + " "
	var parts []string
	var allArgs []interface{}

	for _, sub := range group.Conditions {
		subSQL, subArgs, err := buildFilterClause(sub, whitelist)
		if err != nil {
			return "", nil, err
		}
		if subSQL == "" {
			continue
		}
		wrap := false
		if !sub.IsLeaf() && sub.Logic == model.LogicOR && group.Logic == model.LogicAND {
			// 当 OR 嵌套在 AND 下时，加括号
			wrap = true
		}
		if wrap {
			parts = append(parts, "("+subSQL+")")
		} else {
			parts = append(parts, subSQL)
		}
		allArgs = append(allArgs, subArgs...)
	}

	if len(parts) == 0 {
		return "", nil, nil
	}
	return strings.Join(parts, connector), allArgs, nil
}

// buildDisplaySQL 构建带实际参数值的可读 SELECT 语句，用于审计展示。
func buildDisplaySQL(params model.TableDataParams, args []interface{}) string {
	safeTable := "`" + strings.ReplaceAll(params.Table, "`", "``") + "`"

	// 构建 WHERE clause — 递归渲染 FilterGroup 为展示 SQL
	var whereClause string
	if params.Filters != nil {
		whereClause = " WHERE " + buildDisplayFilter(params.Filters)
	}

	orderClause := ""
	if params.SortColumn != "" && params.SortOrder != "" {
		safeSortCol := "`" + strings.ReplaceAll(params.SortColumn, "`", "``") + "`"
		orderClause = " ORDER BY " + safeSortCol + " " + string(params.SortOrder)
	}

	_ = args // args contain actual values for parameterized execution, not needed for display
	return fmt.Sprintf("SELECT * FROM %s%s%s LIMIT %d OFFSET %d",
		safeTable, whereClause, orderClause, params.PageSize, offsetForPage(params.Page, params.PageSize))
}

// buildDisplayFilter 递归渲染 FilterGroup 为可读 SQL（值内联）。
func buildDisplayFilter(group *model.FilterGroup) string {
	if group == nil {
		return ""
	}
	if group.IsLeaf() {
		safeCol := "`" + strings.ReplaceAll(group.Column, "`", "``") + "`"
		switch group.Operator {
		case model.OpEQ:
			return fmt.Sprintf("%s = %s", safeCol, displayValue(group.Value))
		case model.OpNEQ:
			return fmt.Sprintf("%s != %s", safeCol, displayValue(group.Value))
		case model.OpContains:
			return fmt.Sprintf("%s LIKE '%%%s%%'", safeCol, strings.ReplaceAll(group.Value, "'", "''"))
		case model.OpGT:
			return fmt.Sprintf("%s > %s", safeCol, displayValue(group.Value))
		case model.OpLT:
			return fmt.Sprintf("%s < %s", safeCol, displayValue(group.Value))
		case model.OpIsNull:
			return safeCol + " IS NULL"
		case model.OpNotNull:
			return safeCol + " IS NOT NULL"
		}
		return ""
	}
	connector := " " + string(group.Logic) + " "
	var parts []string
	for _, sub := range group.Conditions {
		subSQL := buildDisplayFilter(sub)
		if subSQL == "" {
			continue
		}
		if !sub.IsLeaf() && sub.Logic == model.LogicOR && group.Logic == model.LogicAND {
			parts = append(parts, "("+subSQL+")")
		} else {
			parts = append(parts, subSQL)
		}
	}
	return strings.Join(parts, connector)
}

// offsetForPage 根据页码和每页条数计算偏移量。
func offsetForPage(page, pageSize int) int {
	if page < 1 {
		return 0
	}
	return (page - 1) * pageSize
}

// displayValue 将一个值格式化为 SQL 展示字符串（单引号包裹字符串，数字直接展示）。
func displayValue(v interface{}) string {
	if v == nil {
		return "NULL"
	}
	switch val := v.(type) {
	case string:
		return "'" + strings.ReplaceAll(val, "'", "''") + "'"
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%v", val)
	default:
		return fmt.Sprintf("%v", v)
	}
}

// UpdateRow 更新指定表中的单个单元格（一行一列）。
// Column 和 PkValues 的键均通过 INFORMATION_SCHEMA 白名单校验，
// 值使用参数化查询（? 占位符）防止 SQL 注入。
func (s *QueryService) UpdateRow(params model.UpdateRowParams) (*model.UpdateRowResult, error) {
	if params.ConnectionID == "" {
		return nil, fmt.Errorf("连接ID不能为空")
	}
	if params.Database == "" {
		return nil, fmt.Errorf("数据库名不能为空")
	}
	if params.Table == "" {
		return nil, fmt.Errorf("表名不能为空")
	}
	if params.Column == "" {
		return nil, fmt.Errorf("更新列名不能为空")
	}
	if len(params.PkValues) == 0 {
		return nil, fmt.Errorf("主键条件不能为空")
	}

	_, db, err := s.connManager.GetDBByID(params.ConnectionID, params.Database)
	if err != nil {
		return nil, err
	}

	// 列名白名单校验（含 Column 和 PkValues 的键）
	whitelist, err := s.getColumnWhitelist(params.ConnectionID, params.Database, params.Table)
	if err != nil {
		return nil, fmt.Errorf("获取表结构失败: %w", err)
	}
	if !whitelist[params.Column] {
		return nil, fmt.Errorf("无效的列名: %s", params.Column)
	}
	for pkCol := range params.PkValues {
		if !whitelist[pkCol] {
			return nil, fmt.Errorf("无效的主键列名: %s", pkCol)
		}
	}

	// 构建参数化 UPDATE: UPDATE `table` SET `col` = ? WHERE `pk1` = ? AND `pk2` = ?
	safeTable := "`" + strings.ReplaceAll(params.Table, "`", "``") + "`"
	safeCol := "`" + strings.ReplaceAll(params.Column, "`", "``") + "`"

	var whereParts []string
	var args []interface{}
	args = append(args, params.NewValue) // SET 值放第一个

	for pkCol, pkVal := range params.PkValues {
		safePkCol := "`" + strings.ReplaceAll(pkCol, "`", "``") + "`"
		whereParts = append(whereParts, safePkCol+" = ?")
		args = append(args, pkVal)
	}

	sql := fmt.Sprintf("UPDATE %s SET %s = ? WHERE %s",
		safeTable, safeCol, strings.Join(whereParts, " AND "))

	// 构建可读的 SQL 审计字符串（参数替换到 SQL 中）
	auditSQL := buildAuditSQL(params.Table, params.Column, params.NewValue, params.PkValues)

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	start := time.Now()
	result, err := db.ExecContext(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("执行更新失败: %w", err)
	}

	affected, _ := result.RowsAffected()
	duration := time.Since(start).Milliseconds()

	return &model.UpdateRowResult{
		AffectedRows: affected,
		Message:      fmt.Sprintf("更新完成：%d 行受影响，%dms", affected, duration),
		SQL:          auditSQL,
	}, nil
}

// buildAuditSQL 构建带值的审计 SQL 字符串，用于前端 DML 审计展示。
// 注意：此 SQL 仅用于展示/审计，不可执行（值未经重新转义）。
func buildAuditSQL(table, col string, newValue any, pkValues map[string]any) string {
	var setVal string
	switch v := newValue.(type) {
	case nil:
		setVal = "NULL"
	case string:
		setVal = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "\\'"))
	case float64:
		if v == float64(int64(v)) {
			setVal = fmt.Sprintf("%d", int64(v))
		} else {
			setVal = fmt.Sprintf("%v", v)
		}
	default:
		setVal = fmt.Sprintf("%v", v)
	}

	var whereParts []string
	for pkCol, pkVal := range pkValues {
		var valStr string
		switch v := pkVal.(type) {
		case nil:
			valStr = "NULL"
		case string:
			valStr = fmt.Sprintf("'%s'", strings.ReplaceAll(v, "'", "\\'"))
		case float64:
			if v == float64(int64(v)) {
				valStr = fmt.Sprintf("%d", int64(v))
			} else {
				valStr = fmt.Sprintf("%v", v)
			}
		default:
			valStr = fmt.Sprintf("%v", v)
		}
		whereParts = append(whereParts, fmt.Sprintf("`%s` = %s", pkCol, valStr))
	}

	return fmt.Sprintf("UPDATE %s SET %s = %s WHERE %s;", table, col, setVal, strings.Join(whereParts, " AND "))
}
