---
name: phase2-query-editor-complete
description: TuxedoSQL 二期 SQL 编辑器 + 查询执行实现总结，包含架构决策、踩坑记录
metadata: 
  node_type: memory
  type: project
  originSessionId: f6b4941a-9ddd-492e-af2c-81ca89a959e3
---

# TuxedoSQL 二期 SQL 编辑器 + 查询执行

## 实现范围

对标 Navicat 的查询编辑器：多标签页 SQL 编辑器 + 执行查询 + 结果表格展示 + 消息面板。

## 技术架构

延续一期：Go + Wails v3.0.0-alpha.97 + Vue 3 + TypeScript + Element Plus + database/sql + go-sql-driver/mysql。

## 新增文件清单

### Go 后端
| 文件 | 职责 |
|------|------|
| `internal/model/query.go` | ColumnInfo, QueryResult, TabState |
| `internal/repository/connection_pool.go` | ConnectionManager — map[string]*sql.DB 连接池 |
| `internal/repository/tab_repo.go` | 标签页 JSON 持久化 (~/.tuxedosql/tabs.json) |
| `internal/service/query.go` | QueryService — Execute + SaveTabs/LoadTabs |
| `internal/service/query_test.go` | 表驱动单元测试 |

### TypeScript 前端
| 文件 | 职责 |
|------|------|
| `types/query.ts` | ColumnInfo, QueryResult, TabState, QueryTab |
| `stores/query.ts` | Pinia Option API — 标签管理、执行状态、持久化 |
| `components/QueryEditor.vue` | Textarea SQL 编辑器 + 执行/停止按钮 + Ctrl+Enter |
| `components/QueryResult.vue` | el-table 动态列渲染 + NULL 灰色斜体 |
| `components/MessagePanel.vue` | 可折叠消息面板，按内容自动分类颜色 |
| `components/QueryTabs.vue` | 标签栏 + 编辑器 + ResizableSplitter + 结果面板 |
| `components/ResizableSplitter.vue` | 可拖拽分割条，使用 parentElement 计算位置 |

### 修改文件
| 文件 | 改动 |
|------|------|
| `main.go` | 初始化 ConnectionManager，注入 ConnectionService + QueryService |
| `internal/service/connection.go` | 构造函数接受 *ConnectionManager；GetDatabases/GetTables 改用连接池 + USE database |
| `internal/repository/connection_repo.go` | 新增 LoadConnectionByID |
| `frontend/src/App.vue` | ContentArea → QueryTabs |
| `frontend/src/components/Sidebar.vue` | 双击 database/table 打开查询标签 |
| `frontend/src/components/ConnectionTree.vue` | 新增 node-dblclick emit，恢复全 node-click |
| `frontend/src/styles/tokens.css` | 新增编辑器/结果/分割条/tab 栏 CSS 变量 |

### 删除文件
| 文件 | 原因 |
|------|------|
| `frontend/src/components/ContentArea.vue` | 被 QueryTabs 替代 |

## 关键技术决策

1. **连接池 DSN 不绑数据库** — `ConnectionManager.GetDB` 的 DSN 仅用默认 database 做初始连接，实际查询前通过 `USE database` 切换。这样同一个 connectionID 可以跨所有 database 复用同一个 `*sql.DB` 池。
2. **SQL 类型检测** — 前缀匹配 SELECT/SHOW/DESCRIBE/EXPLAIN/WITH → QueryContext，其他 → ExecContext。`DESC` 也作为 DESCRIBE 同义。
3. **结果集 10,000 行硬限制** — 超过时警告截断，防止 OOM。
4. **前端 `.cancel()` 做软停止** — Wails v3 的 CancellablePromise 不会传播到 Go 侧，需要在 JS 侧取消 reject 后 UI 结束等待，Go 继续跑但有 30s 硬超时。
5. **Textarea 而非 CodeMirror** — 第一阶段先打通查询管道，后续可无痛替换。
6. **自定义标签栏** — el-tabs 不适合可关闭/重命名/滚动溢出的编辑器标签。

## 踩坑记录

### 1. 连接池 DSN 绑定数据库导致跨库查询错误（已修复）
- **现象**: 打开连接 A 下的数据库 db1 → 展开表列表正确，再展开同一连接下的 db2 → 展示的表仍是 db1 的。
- **原因**: DSN 中包含了首次访问时的 database 名，连接池按 connectionID 缓存后不会重建，`GetTables` 传入不同 database 参数时直接返回了旧池（仍连在第一个数据库上），`SHOW TABLES` 实际查询的是旧库。
- **修复**: DSN 不指定数据库，改为使用默认数据库或 "mysql"，调用方 (`GetDatabases`/`GetTables`/`Execute`) 显式执行 `USE database` 切换。数据库名用反引号 `` ` `` 包裹，并对输入中的反引号做双重转义防注入。

### 2. ResizableSplitter 拖拽无效（已修复）
- **现象**: 分割条静止不动，鼠标拖动无反应。
- **原因**: `onMouseMove` 中 `e.target.closest('.splitter-container')` 始终返回 `null` — mousemove 的 target 是内部 handle 子元素或 splitter 本身，其父容器是 `query-tabs` 而非 `splitter-container`。
- **修复**: 改用 `splitterRef.value.parentElement` 获取容器，通过 template ref 而非 DOM 遍历定位。

### 3. TestConnection/GetDatabases/GetTables 方法在 connManager 为 nil 时 panic（已修复）
- **原因**: 重构 ConnectionService 接受 connManager 参数后，测试中 `NewConnectionService(nil)` 传入 nil connManager，但 `TestConnection`/`GetDatabases`/`GetTables` 直接调用 `s.connManager.GetDB()` 引发 nil pointer dereference。
- **修复**: 这三个方法开头增加 `if s.connManager == nil` 守卫，返回错误。CRUD/分组方法不依赖 connManager，可以正常测试。

## 验证结果

| 检查 | 结果 |
|------|------|
| `go build ./...` | 通过（仅 Wails CGO GTK4 弃用警告） |
| `go vet ./internal/...` | 通过 |
| `go test ./internal/...` | 34 用例全通过 |
| `vue-tsc --noEmit` | 零类型错误 |
| `vite build` | 构建成功 |
| `wails3 dev` | 应用窗口正常启动 |
