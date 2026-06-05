---
name: roadmap-future-planning
description: TuxedoSQL 后续开发路线图，包含第三期规划、技术债务、架构演进
metadata: 
  node_type: memory
  type: project
  originSessionId: f6b4941a-9ddd-492e-af2c-81ca89a959e3
---

# TuxedoSQL 后续开发路线图

## 当前状态

| 阶段 | 内容 | 状态 |
|------|------|------|
| 第一期 | 连接管理 MVP（CRUD + 测试 + 分组 + 库表树） | ✅ 完成 |
| 第二期 | SQL 编辑器 + 查询执行（多标签页 + 执行 + 结果表格） | ✅ 完成 |
| 第三期 | 数据浏览（表格视图） | 🔜 下一步 |

---

## 第三期：数据浏览（表格视图）规划

对标 Navicat 的"双击表名 → 打开表格视图"功能，在查询编辑器基础上增加数据的可视化浏览和编辑能力。

### 核心功能

#### 3.1 表数据分页浏览
- **Go 后端**: `QueryService.GetTableData(connectionID, database, table, page, pageSize, sortColumn, sortOrder)` → 分页查询结果
- **Go 后端**: `QueryService.GetTableSchema(connectionID, database, table)` → 获取列定义（名称、类型、是否可空、键类型、默认值）
- **前端**: 在 QueryResult 基础上增加分页控件（上一页/下一页/跳转/每页条数）
- 默认 `SELECT * FROM table LIMIT 0, 100`

#### 3.2 排序 + 筛选
- **排序**: 点击列头切换升序/降序/取消，拼入 `ORDER BY`
- **筛选**: 列头右键 → 自定义条件（等于/包含/大于/小于/NULL），拼入 `WHERE`
- SQL 在 Go 侧用参数化查询组装，防注入

#### 3.3 行内编辑（可选，放 3.3）
- 双击单元格进入编辑模式
- 修改后高亮标记，底部显示"应用/丢弃"按钮
- 生成 UPDATE/INSERT/DELETE 语句
- **风险**: 直接 UPDATE 无 UNDO，需要先做数据快照或确认弹窗

#### 3.4 导出功能（可选，放 3.3）
- 导出当前页/全部为 CSV
- 导出为 SQL INSERT 语句
- 纯前端实现（在已有查询结果上序列化），不增加 Go 方法

### 技术设计要点

| 决策 | 选择 | 理由 |
|------|------|------|
| 分页 | `LIMIT offset, count`（Go 侧参数化） | 简单可靠，百万行以下足够 |
| 排序 | 列名白名单校验，仅允许 `ASC`/`DESC` | 防 SQL 注入 |
| 筛选 | 列名白名单 + 参数化值 | 防 SQL 注入 |
| 编辑确认 | 前端 Diff → 显示变更摘要 → 确认后批量执行 | 减少数据库往返 |
| 大表虚拟滚动 | 第三期先不做 | 100 行/页足够，后续加 |

### 预计文件变更

| 文件 | 操作 | 说明 |
|------|------|------|
| `internal/model/query.go` | 修改 | 新增 TableDataParams, TableSchema, PageResult |
| `internal/service/query.go` | 修改 | 新增 GetTableData, GetTableSchema |
| `internal/service/query_test.go` | 修改 | 新增分页/排序/筛选测试 |
| `frontend/src/types/query.ts` | 修改 | 新增对应 TS 接口 |
| `frontend/src/components/QueryResult.vue` | 修改 | 增加分页控件 + 列排序 + 列头右键筛选 |
| `frontend/src/components/DataExport.vue` | 新建 | 导出对话框（CSV/SQL 格式选择） |

---

## 技术债务清单

### CRITICAL — 密码明文存储
- **位置**: `~/.tuxedosql/connections.json`
- **方案**: macOS Keychain / Linux libsecret / Windows Credential Manager，通过平台 API 加密存储密码
- **优先级**: 上线前必须修

### HIGH — 查询结果无分页，仅硬截断
- **位置**: `query.go` maxRows=10000
- **方案**: 第三期实现分页浏览时一并解决

### HIGH — CancellablePromise 不传播到 Go
- **位置**: `QueryTabs.vue` 停止按钮
- **方案**: Wails v3 框架限制；如需真正的服务端取消，需要 Go 侧维护 goroutine 注册表 + cancel func 映射，通过 `CancelQuery(queryID)` RPC 方法实现

### MEDIUM — 重复的 loadJSON/saveJSON
- **位置**: `connection_repo.go` 和 `tab_repo.go` 各一份
- **方案**: 抽取 `pkg/fileutil/json_store.go` 公共 helper

### MEDIUM — ConnectionManager.GetDBByID 每次创建新 ConnectionRepository
- **位置**: `connection_pool.go:87`
- **方案**: 将 ConnectionRepository 注入 ConnectionManager 构造函数

### MEDIUM — 测试覆盖率 ~40%
- **位置**: `query_test.go`
- **方案**: 集成测试需要真实 MySQL 实例；可考虑用 `testcontainers-go` 启动 Docker MySQL

### LOW — 消息分类依赖中文字符串匹配
- **位置**: `MessagePanel.vue`
- **方案**: QueryResult 中增加 `type: "success" | "error" | "info"` 字段

### LOW — 未使用的 treeCompRef
- **位置**: `Sidebar.vue:13`
- **方案**: 直接删除

---

## 架构演进方向

### 1. Repository 接口化
当前 Service 依赖具体 Repository 类型，改为 interface 后可 Mock 测试：
```go
type ConnectionRepository interface {
    LoadConnections() ([]model.Connection, error)
    SaveConnections([]model.Connection) error
    LoadConnectionByID(id string) (*model.Connection, error)
    // ...
}
```
这个改动较大（涉及所有 Service 构造函数），适合在功能稳定的间歇期做。

### 2. 多数据库驱动支持
当前硬编码 MySQL。引入 PostgreSQL 需要：
- `internal/repository/connection_pool.go` 中根据连接类型选择驱动
- `internal/service/query.go` 中 `SHOW DATABASES` / `SHOW TABLES` 语法适配（MySQL vs pg_catalog）
- 前端连接对话框增加"数据库类型"下拉
- 抽象 `DatabaseDriver` interface：
```go
type DatabaseDriver interface {
    Open(dsn string) (*sql.DB, error)
    ListDatabases(db *sql.DB) ([]string, error)
    ListTables(db *sql.DB, database string) ([]string, error)
}
```

### 3. SQL 语法高亮
当前 textarea 无高亮。引入 CodeMirror 6：
- 安装 `@codemirror/lang-sql` + `codemirror`
- 封装为 `SqlEditor.vue` 替换 `QueryEditor.vue` 中的 textarea
- 零 Go 侧改动，纯前端替换

### 4. 暗色主题
当前仅有 light 主题。方案：
- `tokens.css` 增加 `[data-theme="dark"]` 选择器定义暗色变量
- 在 `App.vue` 或 setting store 中持久化主题选择
- 系统偏好自动检测：`window.matchMedia('(prefers-color-scheme: dark)')`

---

## 功能优先级矩阵

| 功能 | 优先级 | 工时估计 |
|------|--------|---------|
| 表数据分页浏览 | P0（第三期核心） | 2-3h |
| 列排序 | P0 | 1h |
| 列筛选 | P1 | 2h |
| 行内编辑 | P2 | 4h+ |
| CSV 导出 | P1 | 1h |
| 密码加密存储 | P1（安全基线） | 2h |
| CodeMirror 语法高亮 | P2 | 2h |
| 暗色主题 | P3 | 2h |
| PostgreSQL 支持 | P2 | 8h+ |
| Repository 接口化 | P3（重构） | 4h |
| testcontainers 集成测试 | P2 | 3h |
| 服务端查询取消 | P3 | 3h |
