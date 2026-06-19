# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

TuxedoSQL is a **Wails v3** desktop application with a **Go backend** and a **Vue 3 + TypeScript** frontend. Wails bridges Go services to the frontend via auto-generated TypeScript bindings.

## Directory Structure

```
TuxedoSQL/
├── main.go                      # 应用入口（保持轻量：创建 app、注册服务、启动）
├── go.mod / go.sum              # Go 模块定义
├── Taskfile.yml                 # 根任务编排
├── CLAUDE.md                    # 项目基准文档（本文件）
├── README.md                    # 项目说明
├── .gitignore
│
├── internal/                    # Go 内部包（不可被外部 import）
│   ├── service/                 #   业务服务（Wails 可调用的服务层）
│   │   ├── greet.go             #     示例：GreetService
│   │   └── greet_test.go
│   ├── model/                   #   数据模型/领域对象
│   └── repository/              #   数据访问层（DB 连接、查询执行等）
│
├── pkg/                         # Go 可复用包（可被外部引用）
│   └── sqlparser/               #   （未来）SQL 解析等通用能力
│
├── frontend/                    # Vue 3 + TypeScript + Vite 前端
│   ├── src/
│   │   ├── main.ts              #     Vue 入口：createApp + mount
│   │   ├── App.vue              #     根组件
│   │   ├── components/          #     通用 UI 组件
│   │   ├── features/            #     按功能模块组织的页面/视图
│   │   │   ├── query/           #       查询编辑器
│   │   │   ├── connection/      #       连接管理
│   │   │   └── result/          #       结果展示
│   │   ├── composables/         #     Vue composables（可复用逻辑 hooks）
│   │   ├── stores/              #     Pinia 状态管理
│   │   ├── lib/                 #     纯工具函数
│   │   ├── types/               #     共享 TypeScript 类型定义
│   │   └── styles/              #     全局样式 + 设计 tokens
│   │       ├── tokens.css
│   │       └── global.css
│   ├── bindings/                #   [自动生成] Wails Go→TS 绑定 — 禁止手动编辑
│   ├── public/                  #   静态资源（字体、图片等，直接 URL 引用）
│   │   └── fonts/
│   ├── dist/                    #   [构建产物] Vite 打包输出 — .gitignore 排除
│   ├── index.html               #   SPA 入口
│   ├── package.json
│   ├── vite.config.ts
│   └── tsconfig.json
│
├── build/                       # 构建系统（多平台打包配置）
│   ├── config.yml               #   Wails dev 编排 + 应用元数据（版本号在此）
│   ├── Taskfile.yml             #   通用 Task（绑定生成/前端构建/图标/Docker）
│   ├── docker/                  #   Docker 构建（交叉编译 + server 模式镜像）
│   ├── appicon.icon/            #   macOS 图标源
│   ├── android/                 #   Android 平台
│   ├── darwin/                  #   macOS 平台
│   ├── ios/                     #   iOS 平台
│   ├── linux/                   #   Linux 平台（AppImage/NFPM）
│   └── windows/                 #   Windows 平台（NSIS/MSIX）
│
├── bin/                         # [构建产物] 编译的二进制 — .gitignore 排除
├── .task/                       # [缓存] Task runner checksum — .gitignore 排除
├── docs/                        # 项目文档
└── .claude/                     # Claude Code 本地配置
    └── prds/                    #   PRD 文档
```

### Directory Rules (CRITICAL)

| 目录 | 规则 |
|------|------|
| `frontend/bindings/` | **禁止手动编辑** — 由 `wails3 generate bindings` 自动生成 |
| `frontend/dist/` | **禁止手动编辑** — 由 Vite 构建产生，`.gitignore` 排除 |
| `bin/` | **只放构建产物** — 不要手动放置源文件或配置 |
| `.task/` | **不要手动操作** — Task runner 内部缓存 |
| `internal/` | Go 内部包，外部模块不可 import |
| `pkg/` | Go 可复用包，可被外部 import |
| `build/` | 构建配置，非运行时代码 |

### Go 代码组织规则

1. **所有业务服务**放在 `internal/service/`，每个文件一个 service struct
2. **数据模型**放在 `internal/model/`，纯数据结构，无业务逻辑
3. **数据访问**放在 `internal/repository/`，封装 DB/文件/网络访问
4. **通用工具**放在 `pkg/` 下，具备独立性和可复用性
5. `main.go` 只做应用组装：创建 app → 注册 service → 创建 window → 启动
6. 每个 `.go` 文件对应一个 `_test.go` 文件

### 前端代码组织规则

1. `components/` — 通用 UI 组件（Button, Card, Modal, 等）
2. `features/` — 按功能模块组织的页面级组件和视图
3. `composables/` — Vue 3 composables（`use*` 函数）
4. `stores/` — Pinia store 定义
5. `lib/` — 纯工具函数，不依赖 Vue
6. `types/` — 跨组件共享的 TypeScript 类型/接口
7. `styles/` — CSS 变量 tokens + 全局样式

## Commands

### Development

```bash
wails3 dev                    # Hot-reload dev mode (backend + frontend)
task dev                      # Same as above, via Taskfile
```

`wails3 dev` reads `build/config.yml` to orchestrate: build Go with `DEV=true` → start Vite dev server on port 9245 → launch the native app pointing at the dev server.

### Build & Run

```bash
wails3 build                  # Production build (native desktop binary)
task build                    # Same as above
task run                      # Build and run the native binary

# Server mode (no GUI, pure HTTP):
task build:server             # Build with `-tags server`
task run:server               # Build and run as HTTP server

# Docker (server mode):
task setup:docker             # One-time: build cross-compilation Docker image
task build:docker             # Build distroless Docker image for server mode
task run:docker               # Build and run Docker container (port 8080)
```

### Frontend Only

```bash
cd frontend
npm install                   # Install dependencies
npm run dev                   # Vite dev server on port 9245
npm run build                 # Production build (runs vue-tsc first)
npm run build:dev             # Dev build (no minification)
```

### Go

```bash
go mod tidy
go build -tags server -o bin/tuxedosql-server .
```

## Architecture

### Go → Frontend Bridge

Go service structs registered in `main.go` via `application.NewService()` are auto-exposed to the frontend. Wails generates TypeScript bindings in `frontend/bindings/` by running:

```bash
wails3 generate bindings -f '<build_flags>' -clean=true -ts
```

**Do not edit files in `frontend/bindings/`** — they are regenerated from Go source. To add new frontend-callable APIs:

1. Define a service struct in `internal/service/` (e.g., `internal/service/query.go`)
2. Register the struct in `main.go` via `application.NewService(&service.QueryService{})`
3. Regenerate bindings (this happens automatically as part of `task build:frontend`)

### Key Files

| File | Role |
|------|------|
| `main.go` | App entry point. Creates the Wails app, registers services, creates windows, sets up events. Keep lightweight. |
| `internal/service/*.go` | Go business services. Each file defines one service struct and its methods. |
| `internal/model/*.go` | Data models / domain objects. Pure data, no business logic. |
| `internal/repository/*.go` | Data access layer (DB connections, query execution, file I/O). |
| `build/config.yml` | Wails dev mode orchestration and build metadata. |
| `Taskfile.yml` | Root task definitions (dev, build, run, docker). |
| `build/Taskfile.yml` | Shared build tasks (frontend bundling, binding generation, icon generation, platform-specific tasks). |
| `frontend/src/main.ts` | Vue 3 app mount point. |
| `frontend/src/App.vue` | Root Vue component. |
| `frontend/vite.config.ts` | Vite config with Vue plugin and Wails runtime plugin. |

### Go → Frontend Bridge

Go service structs registered in `main.go` via `application.NewService()` are auto-exposed to the frontend. Wails generates TypeScript bindings in `frontend/bindings/` by running:

```bash
wails3 generate bindings -f '<build_flags>' -clean=true -ts
```

**Do not edit files in `frontend/bindings/`** — they are regenerated from Go source. To add new frontend-callable APIs:

1. Define a service struct in `internal/service/` (e.g., `internal/service/query.go`)
2. Register the struct in `main.go` via `application.NewService(&service.QueryService{})`
3. Regenerate bindings (this happens automatically as part of `task build:frontend`)

### Events

The Go backend emits a `"time"` event every second (see `main.go`). The frontend consumes it via `Events.On("time", callback)` from `@wailsio/runtime`.

### Server Mode

Building with `-tags server` produces a headless HTTP server binary (no native GUI dependencies). The Docker setup (`build/docker/Dockerfile.server`) packages this into a distroless image.

### Cross-Platform Build

The `build/` directory contains platform-specific subdirectories (`darwin/`, `windows/`, `linux/`, `ios/`, `android/`) with platform Taskfiles. Use `task setup:docker` to build the cross-compilation Docker image, then `task build` to cross-compile for any platform.

## Testing & Quality

### Frontend

```bash
npm run typecheck      # TypeScript type checking (vue-tsc)
npm run lint           # ESLint (0 errors, pre-existing warnings only)
npm test               # Vitest (53 tests, 7 suites)
npm run test:watch     # Vitest watch mode
npm run test:coverage  # Vitest with coverage
```

### Go Backend

```bash
go test ./... -count=1          # Run all Go tests (120+ tests, all pass)
go test -race ./...              # Run with race detection
go vet ./...                     # Vet (clean)
golangci-lint run ./...          # Lint (requires golangci-lint installed)
```

### Taskfile Commands

```bash
task test:go         # Run all Go tests
task test:go:race    # Go tests with race detection
task test:go:cover   # Coverage report
task test:frontend   # Vitest frontend tests
task lint:go         # golangci-lint
task lint:frontend   # ESLint
task typecheck       # TypeScript type checking
task check           # All checks (lint + test + typecheck)
```

### CI

GitHub Actions CI at `.github/workflows/ci.yml` runs: lint-go, test-go, typecheck, lint-frontend, test-frontend on push/PR to master/main.
