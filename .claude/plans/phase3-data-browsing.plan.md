# Plan: 第三期 — 数据浏览（表格视图）+ 多项增强

**Source PRD**: `.claude/prds/database-connection-management.prd.md`
**Selected Milestone**: 3 — 数据浏览（表格视图）
**Complexity**: Large

## Summary

第三期核心目标是实现 Navicat 的 "双击表 → 打开表格视图" 体验：分页浏览表数据、列头排序/筛选、CSV 导出。同步解决两个关键技术债务：密码加密存储（安全基线）和 CodeMirror 语法高亮（UX 提升）。Go 后端新增 `GetTableData`/`GetTableSchema` 两个服务方法，前端在 `QueryResult` 基础上增加分页控件和列交互，新增 `DataExport` 组件和 `SqlEditor` (CodeMirror) 组件。

## Patterns to Mirror

| Category | Source | Pattern |
|---|---|---|
| Naming | `internal/service/query.go:33` | Go 方法 `ServiceName.MethodName(params) (result, error)`，中文错误消息 |
| Naming | `internal/service/query_test.go:15` | 表驱动测试 `func TestServiceName_MethodName_Scenario(t *testing.T)` |
| Errors | `internal/service/query.go:34-42` | 先校验空值 → `fmt.Errorf("中文描述")` → 返回 `nil, err` |
| Errors (frontend) | `QueryTabs.vue:89-101` | `parseError` 三层解析：instanceof Error → object.message → JSON.parse |
| Immutability | `stores/query.ts:77-84` | `this.tabs = [...slice(0, idx), { ...tab, field }, ...slice(idx+1)]` |
| Data flow | `main.go:23-26` | Go struct → `application.NewService()` → Wails bindings → Pinia store → Vue |
| CSS | `tokens.css:1-40` | `:root { --color-* }` CSS 自定义属性，组件内 `var(--color-*, fallback)` |
| Validation | `connection.go:32-38` | Go 侧参数校验：空值检查 + 范围检查 |
| Context | `connection_pool.go:79` | `context.WithTimeout(context.Background(), N*time.Second)` |
| Service reg | `main.go:24-25` | `application.NewService(service.NewXxxService(deps))` |

## Files to Change

### Go Backend

| File | Action | Why |
|---|---|---|
| `internal/model/query.go` | UPDATE | 新增 `TableSchema`, `TableDataParams`, `FilterCondition`, `PageResult` 类型 |
| `internal/service/query.go` | UPDATE | 新增 `GetTableSchema` (获取列定义白名单), `GetTableData` (分页+排序+筛选查询) |
| `internal/service/query_test.go` | UPDATE | 新增表数据查询/列名校验/排序白名单/筛选参数化测试 |
| `main.go` | UPDATE | 新增 `QueryService` 注册（已存在，无需改动） |

### Frontend

| File | Action | Why |
|---|---|---|
| `frontend/src/types/query.ts` | UPDATE | 新增 `TableSchema`, `TableDataParams`, `FilterCondition`, `PageResult`, `SortOrder` 类型 |
| `frontend/src/stores/query.ts` | UPDATE | 新增 `openTableView` action，表数据分页/排序/筛选状态管理 |
| `frontend/src/components/QueryResult.vue` | REWRITE | 增加分页控件（上一页/下一页/跳转/每页条数）、列头排序（单击升序→降序→取消）、列头右键筛选菜单 |
| `frontend/src/components/DataExport.vue` | CREATE | 导出对话框：格式选择（CSV/SQL INSERT）、导出范围（当前页/全部），纯前端实现 |
| `frontend/src/components/SqlEditor.vue` | CREATE | CodeMirror 6 SQL 编辑器封装，替换 textarea |
| `frontend/src/components/QueryEditor.vue` | UPDATE | 将 textarea 替换为 `SqlEditor` 组件 |
| `frontend/src/components/Sidebar.vue` | UPDATE | 双击 table 节点改为打开表格视图（而非查询标签），新增"查询表"右键菜单项 |
| `frontend/src/components/TableView.vue` | CREATE | 表数据浏览页：组合 QueryResult + 分页 + 导出按钮 |
| `frontend/src/components/QueryTabs.vue` | UPDATE | Tab 内容区支持渲染 TableView（当 tab 类型为 table-view 时） |
| `frontend/src/styles/tokens.css` | UPDATE | 新增分页控件、筛选菜单、导出对话框、CodeMirror 相关 CSS 变量 |
| `frontend/package.json` | UPDATE | 新增 `codemirror`, `@codemirror/lang-sql`, `@codemirror/view`, `@codemirror/state` 等依赖 |

## Tasks

### Task 1: Go 后端 — 数据模型扩展
- **Action**: 在 `internal/model/query.go` 中新增 `TableSchema`, `FilterCondition`, `TableDataParams`, `PageResult` 类型。`FilterCondition` 包含 `Column`, `Operator` (eq/neq/contains/gt/lt/isnull), `Value`。`PageResult` 包含分页元数据 (Total, Page, PageSize, TotalPages)。
- **Mirror**: 现有 `ColumnInfo`/`QueryResult` 的 json tag 风格
- **Validate**: `go build ./internal/...`

### Task 2: Go 后端 — GetTableSchema 方法
- **Action**: 在 `QueryService` 中新增 `GetTableSchema(connectionID, database, table string) ([]model.TableSchema, error)`。执行 `SELECT COLUMN_NAME, DATA_TYPE, IS_NULLABLE, COLUMN_KEY, COLUMN_DEFAULT FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION`。
- **Mirror**: `GetDatabases`/`GetTables` 的参数校验 + `connManager.GetDBByID` + `USE database` 模式
- **Validate**: `go test ./internal/service/ -run TestGetTableSchema -v`

### Task 3: Go 后端 — GetTableData 方法（分页+排序+筛选）
- **Action**: 在 `QueryService` 中新增 `GetTableData(params model.TableDataParams) (*model.PageResult, error)`。**安全关键**：先调用内部 `getTableSchema` 获取列名白名单，校验 `SortColumn` 和所有 `Filters[*].Column` 在白名单内；`SortOrder` 仅允许 `ASC`/`DESC`；`Operator` 白名单校验。SQL 组装：`SELECT * FROM table WHERE filters ORDER BY col LIMIT ? OFFSET ?`，值使用 `?` 参数化。同时执行 `SELECT COUNT(*) FROM table WHERE filters` 获取总数。
- **Mirror**: `Execute` 方法的 connManager + USE database + context timeout 模式
- **Validate**: `go test ./internal/service/ -run TestGetTableData -v`

### Task 4: Go 后端 — 单元测试
- **Action**: 为 `GetTableSchema` 和 `GetTableData` 编写表驱动测试。验证：空参数校验、不存在的连接/数据库/表报错、列名白名单校验（非法列名被拒绝）、排序方向校验、筛选操作符校验、分页边界（page=0, pageSize=0, pageSize>max）。
- **Mirror**: 现有 `TestQueryService_Execute_Validation` 表驱动模式
- **Validate**: `go test ./internal/service/ -cover`

### Task 5: 前端 — 类型定义 + Store 扩展
- **Action**: 在 `types/query.ts` 中新增 TS 接口（与 Go 模型对齐）。在 `query.ts` Pinia store 中新增：
  - `openTableView` action（tab 类型标记 + 初始加载首頁数据）
  - `loadTableData` action（根据分页/排序/筛选参数调用 `QueryService.GetTableData`）
  - `setSorting`/`setFilters`/`setPage` mutations（不可变更新）
  - 扩展 `QueryTab` 类型支持 `viewType: 'query' | 'table'` 和相关 table 状态
- **Mirror**: 现有 `openTab`/`setResult` 的不可变更新模式（`...slice` + `{...obj, field}`）
- **Validate**: `cd frontend && npx vue-tsc --noEmit`

### Task 6: 前端 — QueryResult 增强（分页+排序+筛选）
- **Action**: 重写 `QueryResult.vue`，增加：
  - 列头点击排序：单击升序 ↑ → 降序 ↓ → 取消。emit `sort-change` 事件。
  - 列头右键筛选菜单（Teleport to body，参考 ConnectionTree 右键菜单模式）：操作符选择（等于/不等于/包含/大于/小于/为空/不为空）+ 值输入
  - 底部分页栏：上一页/下一页/页码跳转/每页条数选择（20/50/100/200）
  - 仅当 `paginated` prop 为 true 时显示分页控件；当为 false 时保持现有查询结果展示
- **Mirror**: 现有 `ConnectionTree.vue` 的 Teleport 右键菜单模式、`el-table` 的 `show-overflow-tooltip` 列渲染
- **Validate**: `cd frontend && npx vue-tsc --noEmit && npm run build`

### Task 7: 前端 — TableView 组件
- **Action**: 新建 `TableView.vue`，组合 `QueryResult`(paginated mode) + 顶栏（表名 breadcrumb、导出按钮、刷新按钮）。双击 sidebar 表名时打开此视图替代选中行的查询标签。
- **Mirror**: `QueryTabs.vue` 的 tab 管理 + editor/result 分栏模式
- **Validate**: `cd frontend && npx vue-tsc --noEmit`

### Task 8: 前端 — DataExport 组件
- **Action**: 新建 `DataExport.vue` 对话框组件。支持：
  - 格式选择：CSV（含 BOM for Excel 中文兼容）、SQL INSERT 语句
  - 导出范围：当前页 / 全部（当前筛选条件下）
  - CSV 生成：BOM + header 行 + 数据行，NULL 转空字符串，含逗号/引号的字段用 `"..."` 包裹并转义内部 `"`
  - SQL 生成：每行一条 `INSERT INTO table (cols) VALUES (...);`
  - 触发浏览器下载（`URL.createObjectURL` + `<a>` click）
  - 纯前端实现，无需 Go 调用
- **Mirror**: `ConnectionDialog.vue` 的 `el-dialog` + 表单模式
- **Validate**: `cd frontend && npx vue-tsc --noEmit`

### Task 9: 前端 — CodeMirror SQL 编辑器
- **Action**: 新建 `SqlEditor.vue` 封装 CodeMirror 6：
  - 安装 `codemirror`, `@codemirror/lang-sql`, `@codemirror/view`, `@codemirror/state`, `@codemirror/commands`, `@codemirror/language`
  - 支持 MySQL SQL 语法高亮 + 自动补全（keywords, tables from schema）
  - Ctrl+Enter 执行（emit `execute`）
  - v-model 双向绑定（通过 `EditorView.updateListener`）
  - 最小配置：行号、高亮当前行、括号匹配
- **Action**: 修改 `QueryEditor.vue`，将 `<textarea>` 替换为 `<SqlEditor>` 组件
- **Mirror**: 保持现有 `QueryEditor` 的 props/emits 接口不变（`modelValue`, `isExecuting`, `database`, `execute`, `stop`）
- **Validate**: `cd frontend && npm run build`

### Task 10: 前端 — Sidebar 双击行为调整
- **Action**: 修改 `Sidebar.vue` 的 `handleNodeDblClick`：
  - 双击 database → 保持现有行为（打开查询标签，`SELECT * FROM table LIMIT 100`）
  - 双击 table → 改为打开 `TableView`（`viewType: 'table'`），预设 tab title 为表名
  - 右键 table 节点 → 新增"查询表"菜单项（打开带 `SELECT * FROM table` 的查询标签）
- **Mirror**: 现有 `ConnectionTree.vue` 的右键菜单 Teleport 模式
- **Validate**: `cd frontend && npx vue-tsc --noEmit`

### Task 11: 密码加密存储（技术债务 CRITICAL）
- **Action**: 
  - Go 侧使用 `crypto/aes` + `crypto/cipher` 实现 AES-256-GCM 加密
  - 加密密钥存储在 `~/.tuxedosql/.key`（首次运行时随机生成 32 字节）
  - `ConnectionRepository` 保存时自动加密 Password 字段，加载时自动解密
  - 新增 `internal/pkg/crypto/aes.go` — 通用 AES-GCM 加密/解密工具
  - 新增 `internal/pkg/crypto/aes_test.go`
  - 兼容旧数据：首次加载时检测明文密码 → 自动加密迁移
- **Mirror**: `connection_repo.go` 的 JSON 持久化模式 + Go 标准库 `crypto/*` 使用惯例
- **Validate**: `go test ./internal/pkg/crypto/ -v`

### Task 12: 集成验证 + PRD 更新
- **Action**: 
  - 运行 `go build ./... && go vet ./internal/... && go test ./internal/...`
  - 运行 `cd frontend && npx vue-tsc --noEmit && npm run build`
  - 更新 `.claude/prds/database-connection-management.prd.md`，新增 Phase 3 milestone 并标记 complete
  - 更新 roadmap memory
- **Validate**: 全量构建通过

## Validation

```bash
# Go 后端
go build ./...
go vet ./internal/...
go test ./internal/... -cover

# 前端
cd frontend && npx vue-tsc --noEmit
cd frontend && npm run build

# 全量集成
task build
```

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| SQL 注入（列名无法参数化） | 中 | INFORMATION_SCHEMA 白名单校验排序/筛选列名，非法列名直接拒绝 |
| CodeMirror 6 打包体积增加 | 低 | 按需导入模块，约 +150KB gzipped 前 |
| LIMIT offset 大偏移量性能差 | 低 | 第三期先不分页深度优化；后续可用 keyset pagination |
| AES 密钥文件丢失 → 密码无法解密 | 低 | 保存前备份密钥到安全位置，丢失后用户需重新输入密码 |
| 旧版明文密码自动迁移失败 | 低 | 迁移逻辑先检测是否已是加密格式（GCM 前缀），幂等安全 |
| Wails v3 API 变动 | 中 | 锁定当前 alpha.97 版本 |

## Acceptance

- [ ] 双击表名打开表格视图，显示分页数据（100 行/页）
- [ ] 列头单击排序（升序→降序→取消），图标正确切换
- [ ] 列头右键筛选（等于/不等于/包含/大于/小于/为空/不为空）
- [ ] 分页控件完整可用（首页/上一页/下一页/末页/跳转/每页条数）
- [ ] 导出 CSV（含 BOM，Excel 正常打开）和 SQL INSERT 语句
- [ ] SQL 编辑器有语法高亮，Ctrl+Enter 执行
- [ ] 密码以加密形式存储，明文不出现在 connections.json
- [ ] 旧版明文密码首次加载自动迁移为加密格式
- [ ] 所有现有功能（连接管理、查询执行）不受影响
- [ ] `go test ./internal/...` 全部通过
- [ ] `npm run build` 成功
- [ ] 安全 review 通过（排序/筛选列名白名单校验确认）
