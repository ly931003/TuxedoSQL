---
name: mvp-implementation-complete
description: 数据库连接管理 MVP 一期实现完成，包含架构、关键技术决策和踩坑记录
metadata:
  type: project
---

# TuxedoSQL 一期 MVP 实现总结

## 实现范围

类似 Navicat 的数据库可视化管理工具第一期：MySQL 连接管理 + 库表树形浏览。

## 技术架构

- 后端: Go + Wails v3.0.0-alpha.97
- 前端: Vue 3 + TypeScript + Vite + Pinia
- 数据库驱动: `database/sql` + `go-sql-driver/mysql` v1.10.0
- 持久化: JSON 文件 (`~/.tuxedosql/connections.json` + `groups.json`)

## 文件清单

### Go 后端
| 文件 | 职责 |
|------|------|
| `internal/model/connection.go` | Connection, ConnectionGroup, TreeNode, Create/Update params, TestResult |
| `internal/repository/connection_repo.go` | JSON 持久化, sync.RWMutex, os.UserHomeDir |
| `internal/service/connection.go` | 10 方法: CRUD + Test + GetDatabases/GetTables + Group CRUD |
| `internal/service/connection_test.go` | 10 个表驱动单元测试 |

### TypeScript 前端
| 文件 | 职责 |
|------|------|
| `types/connection.ts` | 与 Go model 对应 |
| `stores/connection.ts` | Pinia 状态管理 |
| `components/Sidebar.vue` | 左侧面板 + 事件协调 + toast |
| `components/ConnectionTree.vue` | 递归树 + 右键菜单 + Teleport |
| `components/ConnectionDialog.vue` | 连接表单, 可拖拽, 分组选择, 测试连接 |
| `components/GroupDialog.vue` | 分组新建/重命名 |
| `components/ToastMessage.vue` | 错误/成功, 4s 自动消失 |
| `components/ContentArea.vue` | 右侧占位 |
| `styles/tokens.css` + `global.css` | 白色主题 CSS 变量 |
| `main.ts` + `App.vue` | 入口注册 Pinia, 左右分栏布局 |

## 关键技术决策

1. `database/sql` 而非 ORM — MVP 只需 SHOW DATABASES/SHOW TABLES/Ping
2. JSON 文件存储 — 单用户桌面应用, 密码明文 (第一期不做加密)
3. Teleport to body — 解决 WebKitGTK 层叠上下文裁剪
4. parseError 三层解析 — instanceof Error → object.message → JSON.parse
5. 无遮罩弹窗 — WebKitGTK 不支持 backdrop-filter

## WebKitGTK 踩坑

- `backdrop-filter: blur()` 渲染为纯黑
- `position: fixed` + 祖先 `overflow: hidden` 裁剪弹窗
- GTK4 弃用警告不影响功能

## 下一步

- 第二期: SQL 编辑器 + 查询执行
- 第三期: 数据浏览 (表格视图)
- 连接加密 (可选)
