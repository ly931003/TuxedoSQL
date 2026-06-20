# 🐈‍⬛ TuxedoSQL

[English](README.md)

<p align="center">
  <strong>像穿燕尾服一样写 SQL —— 优雅、自信、游刃有余。</strong>
</p>

**TuxedoSQL** 是一款精致的跨平台桌面数据库客户端。基于 **Wails v3** 构建，后端用 **Go** 提供原生级性能，前端用 **Vue 3** 打磨交互体验 —— 没有 Electron 的臃肿，只有飞一般的流畅。

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go" alt="Go 1.25">
  <img src="https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vuedotjs" alt="Vue 3">
  <img src="https://img.shields.io/badge/Wails-v3-DF0000?logo=wails" alt="Wails v3">
  <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT License">
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey" alt="Platforms">
</p>

## ✨ 为什么选择 TuxedoSQL？

| 你想要... | 你得到 |
|-----------|--------|
| 🚀 **秒级启动** | Go 原生二进制，无 Chromium 包袱，启动不到一秒 |
| 🎯 **专业 SQL 编辑** | CodeMirror 6 引擎，语法高亮、多光标、括号匹配 |
| 🔍 **可视化建查询** | 点击式查询构建器，支持嵌套 AND/OR 逻辑组 |
| 📊 **一站式浏览** | 表格视图 + 行内编辑 + DDL 预览，一屏搞定 |
| 🔐 **凭证安全保障** | 操作系统原生密钥环加密，磁盘零明文 |
| 🌏 **时区无痛处理** | 单连接级时区配置，时间列自动格式化 |
| 🧩 **多种部署形态** | 桌面应用、无头服务器、Docker 容器 —— 同一份代码 |

## 🐱 为什么叫 "Tuxedo"？

名字来源于作者家的两只奶牛猫 🐄🐈 —— 黑白相间的小家伙，像永远穿着燕尾服一样神气。就像它们一样，这个工具力求优雅可靠，偶尔在数据处理上也会有点小调皮。

## 📸 界面预览

<!-- TODO: 添加截图 -->
<!-- ![查询编辑器](docs/screenshots/query-editor.png) -->

## 🧰 功能矩阵

| 分类 | 亮点 |
|------|------|
| **连接管理** | 保存、分组、测试连接 —— 已支持 MySQL，更多数据库接入中 |
| **查询编辑器** | 多标签页会话、语法高亮、自动补全提示 |
| **查询构建器** | 可视化 WHERE 条件构建，嵌套逻辑组 —— 不写 SQL 也能查 |
| **数据浏览** | 列排序、列过滤、分页加载 |
| **行编辑器** | 点击即编辑，内联校验，类型感知的表单控件 |
| **Schema 浏览器** | 树形表列表、DDL 导出、列/索引元数据一览 |
| **数据导出** | 一键导出 CSV/SQL INSERT/JSON |
| **安全** | 操作系统密钥环（macOS 钥匙串 / Windows 凭据管理器 / Linux Secret Service）+ AES-256 机器 ID 回退 |
| **布局** | 可折叠侧边栏、拖拽分隔面板 —— 你的屏幕你做主 |
| **服务器模式** | 无头 HTTP 服务器模式，适合远程或容器化部署 |

## 🏗️ 技术栈

| 层 | 选型 | 理由 |
|----|------|------|
| 桌面框架 | [Wails v3](https://v3.wails.io/) | 原生系统绑定，二进制仅 ~10MB |
| 后端 | Go 1.25 | 快、单文件部署、成熟的数据库驱动生态 |
| 前端 | Vue 3 + TypeScript + Vite | 响应式、类型安全、开发时热更新 |
| UI 组件库 | [Element Plus](https://element-plus.org/) | 成熟的 Vue 3 组件库 |
| 状态管理 | [Pinia](https://pinia.vuejs.org/) | 轻量、DevTools 友好 |
| 编辑器 | CodeMirror 6 | 可扩展、移动端友好、原生 SQL 支持 |
| 数据库驱动 | [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql) | 久经考验的 MySQL/MariaDB 驱动 |
| 加密 | 系统密钥环 + `golang.org/x/crypto` | AES-256 + 机器 ID 派生密钥兜底 |

## 🚀 快速开始

### 环境要求

- **Go** ≥ 1.25
- **Node.js** ≥ 20
- **Wails v3 CLI**：`go install github.com/wailsapp/wails/v3/cmd/wails3@latest`

### 一分钟跑起来

```bash
git clone https://github.com/your-org/TuxedoSQL.git && cd TuxedoSQL
cd frontend && npm install && cd ..
wails3 dev          # ← Go + Vue 双端热重载
```

### 生产构建

```bash
wails3 build        # 原生桌面二进制文件输出到 bin/
task run            # 构建 + 启动
```

### 服务器模式（无 GUI）

```bash
task build:server   # 纯 HTTP 服务器，无 GUI 依赖
task run:server     # 启动，监听 :8080
```

### Docker

```bash
task setup:docker   # 一次性：构建交叉编译镜像
task build:docker   # distroless 生产镜像
task run:docker     # 启动容器，端口 8080
```

## 📁 项目结构

```
TuxedoSQL/
├── main.go                   # 入口 —— 创建应用、注册服务、启动窗口
├── Taskfile.yml              # 任务编排：dev / build / run / docker
│
├── internal/
│   ├── service/              # 业务服务层，通过 Wails 桥接暴露给前端
│   │   ├── connection.go     #   连接 CRUD + 连通性测试
│   │   └── query.go          #   SQL 执行 + Schema 内省
│   ├── model/                # 领域类型（Connection, Query, Tab…）
│   └── repository/           # 持久化层（JSON 存储、连接池）
│
├── frontend/
│   ├── src/
│   │   ├── components/       # 19 个 Vue 组件（编辑器、侧边栏、对话框、面板）
│   │   ├── features/         # 功能模块（含查询构建器的 TableSearch）
│   │   ├── composables/      # 可复用逻辑 hooks
│   │   ├── stores/           # Pinia 状态管理
│   │   └── types/            # 共享 TypeScript 接口
│   ├── bindings/             # [自动生成] Go→TS 桥接层 —— 请勿手动编辑
│   └── package.json
│
├── build/                    # 跨平台打包 & Docker 配置
│   ├── docker/               # distroless 服务器模式镜像
│   ├── linux/                # AppImage / NFPM
│   ├── darwin/               # macOS .app 包
│   └── windows/              # NSIS / MSIX 安装器
│
└── docs/                     # 项目文档 & PRD
```

## 🧠 架构

TuxedoSQL 清晰分为三层：

```
┌──────────────────────────────────────┐
│  Vue 3 前端                          │
│  Pinia stores → components → Element Plus UI
└──────────────┬───────────────────────┘
               │ Wails 桥接（自动生成 TS 绑定）
┌──────────────▼───────────────────────┐
│  Go 服务层                            │
│  ConnectionService / QueryService    │
│  → repository → model               │
└──────────────┬───────────────────────┘
               │ database/sql
┌──────────────▼───────────────────────┐
│  MySQL / MariaDB                     │
└──────────────────────────────────────┘
```

添加新的前端可调用 API 只需三步：

1. 在 `internal/service/` 中定义服务结构体
2. 在 `main.go` 中通过 `application.NewService()` 注册
3. 重新生成绑定：`wails3 generate bindings`

## 🗺️ 路线图

- [x] 连接管理（加密凭证存储）
- [x] 多标签页 SQL 编辑器（CodeMirror 6）
- [x] 可视化查询构建器（嵌套逻辑组）
- [x] 表数据浏览（排序、过滤、分页）
- [x] 行内记录编辑器
- [x] Schema 浏览器 + DDL 查看器
- [x] 数据导出（CSV/SQL INSERT/JSON）
- [x] 服务器模式 + Docker 部署
- [ ] PostgreSQL 支持
- [ ] SQLite 支持
- [ ] SSH 隧道连接
- [ ] 查询历史 & 收藏
- [x] 深色模式
- [ ] ER 图可视化

## 🤝 参与贡献

欢迎提交 Pull Request！查看 [CLAUDE.md](CLAUDE.md) 了解架构细节、项目规范和开发命令。

## 📄 许可证

MIT —— 随便用，随便改，随便发。
