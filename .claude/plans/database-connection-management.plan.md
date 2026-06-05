# Plan: 数据库连接管理 MVP

**Source PRD**: `.claude/prds/database-connection-management.prd.md`
**Selected Milestone**: 1 — 连接管理 MVP
**Complexity**: Medium

## Summary

实现 TuxedoSQL 第一期：MySQL 数据库连接管理 + 库表树形浏览。后端通过 Go service 暴露连接 CRUD、测试连接、获取库表列表的 API；前端使用 Vue 3 实现 Navicat 风格的左侧连接树 + 右侧内容区布局。连接信息以 JSON 文件持久化存储。

## Patterns to Mirror

| Category | Source | Pattern |
|---|---|---|
| Go Service | `internal/service/greet.go:6-11` | 无状态 struct，方法签名 `func (s *S) Method(args) result`，在 `main.go` 注册 |
| Go Tests | `internal/service/greet_test.go:5-38` | 表驱动测试，`t.Run(name, func)`，中文用例名 |
| Vue Component | `frontend/src/components/HelloWorld.vue:1-47` | `<script setup lang="ts">`，从 `bindings/` 导入服务调用 |
| Frontend API Call | `HelloWorld.vue:17-21` | 直接调用 `ServiceName.Method(args).then().catch()` |
| Events | `HelloWorld.vue:24-28` | `Events.On(eventName, callback)` 监听后端事件 |

## Files to Change

| File | Action | Why |
|---|---|---|
| `internal/model/connection.go` | CREATE | 连接/分组/库/表数据模型 |
| `internal/repository/connection_repo.go` | CREATE | JSON 文件持久化读写 |
| `internal/service/connection.go` | CREATE | 连接管理业务逻辑，暴露给前端 |
| `internal/service/connection_test.go` | CREATE | 连接服务单元测试 |
| `main.go` | UPDATE | 注册 ConnectionService，移除 demo 事件 |
| `frontend/src/types/connection.ts` | CREATE | 前端类型定义 |
| `frontend/src/stores/connection.ts` | CREATE | Pinia 连接状态管理 |
| `frontend/src/components/Sidebar.vue` | CREATE | 左侧连接树面板 |
| `frontend/src/components/ConnectionTree.vue` | CREATE | 连接/库/表树形组件 |
| `frontend/src/components/ConnectionDialog.vue` | CREATE | 新建/编辑连接对话框 |
| `frontend/src/components/ContentArea.vue` | CREATE | 右侧内容占位区 |
| `frontend/src/App.vue` | UPDATE | 替换为 Navicat 风格布局 |
| `frontend/src/components/HelloWorld.vue` | DELETE | 移除示例组件 |
| `frontend/src/styles/tokens.css` | UPDATE | 设计 tokens |
| `frontend/src/styles/global.css` | UPDATE | 全局样式，左侧面板 + 右侧主区域布局 |

## Tasks

### Task 1: Go 数据模型定义
- **Action**: 在 `internal/model/connection.go` 中定义 `Connection`（id, name, groupId, host, port, username, password, database, createdAt, updatedAt）、`ConnectionGroup`（id, name, parentId）、`DatabaseTreeNode`（name, type, children）等结构体
- **Mirror**: 纯数据结构，无业务逻辑
- **Validate**: `go build ./internal/model/`

### Task 2: JSON 文件持久化 Repository
- **Action**: 在 `internal/repository/connection_repo.go` 中实现 `LoadConnections()`, `SaveConnections()`, `LoadGroups()`, `SaveGroups()` 方法。存储路径使用 `~/.tuxedosql/connections.json` 和 `~/.tuxedosql/groups.json`
- **Mirror**: Go 标准 `encoding/json` + `os.UserHomeDir`
- **Validate**: `go build ./internal/repository/`

### Task 3: ConnectionService 实现
- **Action**: 在 `internal/service/connection.go` 中实现：`Create`, `Update`, `Delete`, `List`, `TestConnection`, `GetDatabases(connectionID)`, `GetTables(connectionID, database)`。TestConnection 用 `database/sql` + `go-sql-driver/mysql` 尝试连接并返回成功/失败消息。GetDatabases/GetTables 执行 `SHOW DATABASES` / `SHOW TABLES`。
- **Mirror**: GreetService 的无状态 struct 模式
- **Validate**: `go build ./internal/service/`

### Task 4: ConnectionService 单元测试
- **Action**: 在 `internal/service/connection_test.go` 中编写表驱动测试，覆盖 Create/Update/Delete/List 的 CRUD 闭环
- **Mirror**: `greet_test.go` 的表驱动 + `t.Run` 模式
- **Validate**: `go test ./internal/service/`

### Task 5: 前端类型定义
- **Action**: 在 `frontend/src/types/connection.ts` 中定义与 Go model 对应的 TypeScript interface
- **Mirror**: TypeScript interface 优先于 type
- **Validate**: `cd frontend && npx vue-tsc --noEmit`

### Task 6: Pinia 连接状态管理
- **Action**: 在 `frontend/src/stores/connection.ts` 中创建 Pinia store，管理 connections/groups 列表、选中状态、展开/折叠状态、对话框可见性
- **Mirror**: Composition API 风格 Pinia store
- **Validate**: `cd frontend && npx vue-tsc --noEmit`

### Task 7: 前端 UI 组件
- **Action**: 创建 `Sidebar.vue`（左侧面板容器，含连接分组列表+右键菜单）、`ConnectionTree.vue`（递归树组件，展示分组→连接→database→table 层级）、`ConnectionDialog.vue`（新建/编辑连接表单，含主机/端口/用户名/密码/默认数据库字段+测试连接按钮）、`ContentArea.vue`（右侧内容占位区，后续留给 SQL 编辑器/数据浏览）
- **Mirror**: Vue 3 `<script setup lang="ts">`，Wails bindings 调用模式
- **Validate**: `cd frontend && npx vue-tsc --noEmit && npx vite build:dev`

### Task 8: 整合注册与布局
- **Action**: 更新 `main.go` 注册 ConnectionService、移除 demo 定时事件；更新 `App.vue` 实现左右分栏布局（左侧 Sidebar ~280px，右侧 ContentArea flex-grow）；清理 HelloWorld.vue
- **Mirror**: main.go 的 `application.NewService()` 注册模式；App.vue 单根元素模板
- **Validate**: `go build ./... && cd frontend && npx vue-tsc --noEmit`

## Validation

```bash
# Go 编译 + 测试
go build ./...
go test ./...

# 前端类型检查 + 构建
cd frontend
npx vue-tsc --noEmit
npx vite build:dev

# 全栈 Wails 构建
wails3 build DEV=true
```

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| go-sql-driver/mysql 测试连接需真实 MySQL 实例 | 高 | TestConnection 方法做好超时控制（5s context），单元测试 mock；手动测试时用本地 MySQL |
| Wails v3 alpha 绑定生成不稳定 | 低 | 锁定 v3.0.0-alpha.97，绑定生成失败时手动检查 |
| JSON 文件并发写入 | 低 | 单用户桌面应用，读-改-写回模式可行；后续加文件锁 |

## Acceptance

- [ ] 用户可创建/编辑/删除 MySQL 连接
- [ ] 用户可以测试连接有效性
- [ ] 连接可按分组管理（新建分组、拖拽/移动连接）
- [ ] 展开连接可看到 databases → tables 树形层级
- [ ] Navicat 风格布局：左侧连接树 + 右侧内容区
- [ ] 连接信息持久化到 JSON 文件，重启不丢失
- [ ] Go 单元测试通过
