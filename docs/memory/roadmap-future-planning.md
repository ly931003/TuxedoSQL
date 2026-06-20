---
name: roadmap-future-planning
description: TuxedoSQL 开发路线图，记录已完成功能、待处理技术债务、架构演进方向及后续规划
metadata:
  node_type: memory
  type: project
  originSessionId: f6b4941a-9ddd-492e-af2c-81ca89a959e3
---

# TuxedoSQL 开发路线图

## 当前状态

| 阶段 | 内容 | 状态 |
|------|------|------|
| 第一期 | 连接管理 MVP（CRUD + 测试 + 分组 + 库表树） | ✅ 完成 |
| 第二期 | SQL 编辑器 + 查询执行（多标签页 + 执行 + 结果表格） | ✅ 完成 |
| 第三期 | 数据浏览（表格视图 + 排序筛选 + 行内编辑 + 导出 + 密码加密 + 语法高亮 + 暗色主题） | ✅ 完成 |

---

## 已完成功能清单

### 连接管理
- 连接 CRUD（创建、编辑、删除、测试连接）
- 连接分组管理（树形分组 + 拖拽排序）
- 密码加密存储 — 三层防护：OS keyring (zalando/go-keyring) → AES-256-GCM 机器 ID 回退 → 旧版 .key 迁移
- 库表树浏览（数据库 → 表 → 列的树形结构）

### SQL 编辑器
- 多标签页管理（新建、关闭、重排）
- SQL 执行 + 结果表格展示
- CodeMirror 6 语法高亮（@codemirror/lang-sql，MySQL 方言）
- Schema 感知自动补全（GetDBSchemaForCompletion 提供表和列名提示）
- 查询结果截断保护（Execute() 默认 maxRows=10000，防 OOM）

### 数据浏览
- 表数据分页浏览（LIMIT/OFFSET，前端分页控件：上/下一页、跳转、每页条数）
- 列排序（列白名单校验 + ASC/DESC 切换）
- 列筛选（递归 AND/OR 条件构建器 + 列白名单 + 参数化查询）
- 行内编辑（PK 白名单校验的 UpdateRow()、RecordForm.vue、dirtyMap 脏标记追踪）
- 导出功能（DataExport.vue — 支持 CSV 和 SQL INSERT 两种格式，纯前端实现）
  - 注意：README 中标注为 CSV/JSON 导出，实际仅有 CSV 和 SQL INSERT，无 JSON 导出

### 前端基础设施
- 暗色主题（tokens.css 含完整 53 个 CSS 变量 dark 覆盖、element-overrides.css 暗色覆盖、main.ts getPreferredTheme() 自动检测、App.vue toggleTheme() 切换）
- 可折叠侧边栏 + 可拖拽分割面板
- Element Plus 中文国际化
- Pinia 状态管理（connectionStore、queryStore、layoutStore）

### 后端基础设施
- 共享 JSON 持久化（fileutil.JSONStore，替代重复的 loadJSON/saveJSON）
- 构造函数注入（connRepo 注入 ConnectionManager，避免每次新建）
- 服务端时间戳格式化（timeFormat composable）

---

## 技术债务清单

### ⚠️ HIGH — CancellablePromise 不传播到 Go
- **位置**: QueryTabs.vue 停止按钮
- **现状**: 前端停止按钮仅取消 JS 层的 Promise，不会通知 Go 服务端终止正在执行的查询
- **方案**: 需在 Go 侧维护 goroutine 注册表 + cancelFunc 映射，通过 CancelQuery(queryID) RPC 实现真正的服务端取消
- **优先级**: P1

### 🔄 MEDIUM — 查询结果截断（by-design）
- **位置**: query.go Execute() maxRows=10000
- **现状**: 原始 SQL 编辑器的 Execute() 硬截断 10000 行是主动设计，防 OOM；数据浏览器的 GetTableData() 已有真正的 LIMIT/OFFSET 分页
- **结论**: 设计如此，不是 bug。但原始 SQL 编辑器的结果截断可考虑增加用户提示或"加载更多"按钮

### 🔄 MEDIUM — 测试覆盖率 ~45%
- **位置**: 整体项目
- **现状**: model 100%、repository 73.9%、service 24.2%、pkg 70-83%。service 层低是因需要真实 MySQL 做集成测试
- **方案**: 用 testcontainers-go 启动 Docker MySQL 做集成测试（依赖 Repository 接口化后更易实施）
- **优先级**: P2

### 🔄 LOW — 消息分类依赖中文字符串匹配
- **位置**: MessagePanel.vue
- **现状**: messageType 字段已作为主要分类器；但仍保留了 emoji 回退判断 (msg.startsWith('✅'))
- **方案**: 完全移除 emoji 前缀匹配，严格依赖 messageType 字段
- **优先级**: P2

### ⚠️ LOW — 未使用的 treeCompRef
- **位置**: Sidebar.vue:18 声明、:326 绑定
- **现状**: treeCompRef 声明后绑定到 el-tree，但代码中从未通过 `.value` 访问
- **方案**: 删除声明和模板绑定
- **优先级**: P3

---

## 架构演进方向

### ✅ 已完成

#### SQL 语法高亮
- CodeMirror 6 + @codemirror/lang-sql，MySQL 方言
- 封装为 SqlEditor.vue，支持多光标、括号匹配
- 零 Go 侧改动，纯前端实现

#### 暗色主题
- tokens.css 完整 `[data-theme="dark"]` 选择器（53 个变量）
- element-overrides.css Element Plus 暗色覆盖
- 系统偏好自动检测：`window.matchMedia('(prefers-color-scheme: dark)')`
- 用户可手动切换并在 localStorage 持久化

### 🔜 待实现

#### 1. Repository 接口化
当前 Service 依赖具体 Repository 类型。提取 interface 后可 mock 测试、切换数据源：
```
type ConnectionRepository interface {
    LoadConnections() ([]model.Connection, error)
    SaveConnections([]model.Connection) error
    LoadConnectionByID(id string) (*model.Connection, error)
    // ...
}
```
涉及所有 Service 构造函数改动，建议在功能间歇期做。
**优先级**: P0（基础性重构）| **工时**: 2-3 天

#### 2. 查询取消（服务端）
- Go 侧维护 goroutine 注册表 + context.WithCancel
- 新增 CancelQuery(queryID) RPC 方法
- 前端停止按钮改为调用 RPC 替代 JS Promise.cancel()
**优先级**: P1 | **工时**: 3-5 天

#### 3. 多数据库驱动支持
MySQL 当前在 13+ 位置硬编码（connection_pool.go DSN 构造、SHOW DATABASES/TABLES 语法、反引号引用）。需抽象：
```
type DatabaseDriver interface {
    Open(dsn string) (*sql.DB, error)
    ListDatabases(db *sql.DB) ([]string, error)
    ListTables(db *sql.DB, database string) ([]string, error)
}
type SchemaIntrospector interface {
    GetColumns(db *sql.DB, database, table string) ([]model.ColumnInfo, error)
    GetPrimaryKey(db *sql.DB, database, table string) ([]string, error)
    // ...
}
```
**优先级**: P2（依赖 Repository 接口化）| **工时**: 1-2 周

#### 4. 查询历史
- 复用 tab_repo.go 持久化模式
- model.TabState 已包含所需字段（sql、connectionId、database、createdAt）
- 前端新增历史侧栏或面板
**优先级**: P1 | **工时**: 4 小时

#### 5. JSON 导出
- 补充 DataExport.vue 导出格式（目前仅 CSV + SQL INSERT）
- 纯前端实现
**优先级**: P3 | **工时**: 1 小时

---

## 后续功能规划

### 数据库支持
- **PostgreSQL 支持** — 需要先完成 DatabaseDriver / SchemaIntrospector 接口抽象，然后实现 PG 方言适配（$N 参数占位符、pg_catalog 元数据查询）
- **SQLite 支持** — 接口抽象后相对简单（file 协议 DSN，sqlite 元数据 SQL）

### 连接增强
- **SSH 隧道** — golang.org/x/crypto 已在 go.mod（被 AES 引用），但 crypto/ssh 尚未导入。需在连接模型中增加 SSH 配置字段，连接池层包一层 ssh.Dial

### 编辑器增强
- **查询收藏** — 类似查询历史的文件结构，增加用户侧收藏标记和快速访问面板
- **查询历史** — 按时间线展示已执行 SQL，支持回放和搜索

### 可视化
- **ER 图可视化** — 从 schema introspection 获取外键关系，前端用 Canvas/SVG 渲染。目前无相关代码

### 主题完善
- **暗色主题完善** — SVG 图标适配暗色 + 主题切换过渡动画

---

## 功能优先级矩阵

| 功能 | 优先级 | 工时估计 | 状态 |
|------|--------|---------|------|
| Repository 接口化 | P0（基础） | 2-3 天 | 🔜 下一步 |
| 查询取消（服务端） | P1 | 3-5 天 | 🔜 |
| 查询历史 | P1 | 4 小时 | 🔜 |
| 暗色主题完善 | P2 | 1 小时 | 🔜（SVG 图标 + 过渡动画） |
| PostgreSQL 支持 | P2 | 1-2 周 | 🔜（依赖接口化） |
| 集成测试（testcontainers） | P2 | 1 周 | 🔜（依赖接口化） |
| SSH 隧道 | P3 | 1 周 | 🔜 |
| SQLite 支持 | P3 | 6 小时 | 🔜 |
| ER 图可视化 | P3 | 8 小时+ | 🔜 |
| JSON 导出 | P3 | 1 小时 | 🔜（补全导出格式） |
