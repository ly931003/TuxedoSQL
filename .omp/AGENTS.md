# TuxedoSQL — Project Knowledge Base

**Updated:** 2026-08-01
**Branch:** master

## OVERVIEW
TuxedoSQL is a Wails v3 desktop SQL client — Go 1.25 backend, Vue 3 + TypeScript + Element Plus frontend. Multi-driver support: **MySQL, PostgreSQL, SQLite** via driver registry pattern. SSH tunnel support for remote connections. ~270 files, ~25k LoC. Two registered services (ConnectionService, QueryService). Version 0.1.0.

## STRUCTURE
```
.
├── main.go                   # App assembly — creates store → repos → services → window
├── .omp/                     # omp agent config (this file, RULES.md, branch-strategy.md, config.yml)
├── Taskfile.yml              # Root task delegation (→ build/Taskfile.yml for quality commands)
├── build/
│   ├── config.yml            # App metadata (v0.1.0), Wails dev orchestration
│   ├── Taskfile.yml          # Shared build tasks: frontend, bindings, lint, test, docker
│   └── docker/               # Dockerfile.server (distroless), Dockerfile.cross (cross-compile)
├── internal/
│   ├── model/                # Pure data structs (zero project deps)
│   ├── service/              # Business services — registered in main.go, exposed to frontend
│   └── repository/           # Data access — JSON persistence, multi-driver pools, SSH, credentials
├── pkg/
│   ├── crypto/               # AES-256-GCM (internal to credential)
│   ├── credential/           # 3-tier password storage (OS keyring → AES fallback → legacy .key)
│   └── fileutil/             # JSON file persistence (~/.tuxedosql/)
├── frontend/
│   ├── src/
│   │   ├── main.ts           # Vue entry — Pinia + Element Plus (zh-CN) + theme init
│   │   ├── App.vue           # Root layout (sidebar + tabs + dialogs + bottom bar)
│   │   ├── components/       # 20 Vue SFCs (script setup), see components/AGENTS.md
│   │   ├── features/         # Self-contained modules (TableSearch: visual query builder)
│   │   ├── stores/           # Pinia stores (connection, query, layout) — Options API
│   │   ├── composables/      # Shared logic (parseError)
│   │   ├── lib/              # Pure utilities (timeFormat, messageClassify)
│   │   ├── types/            # TS type re-exports from Go bindings + app-specific types
│   │   └── styles/           # CSS tokens (60+ vars), Element Plus overrides
│   ├── bindings/             # AUTO-GENERATED Wails bridge — DO NOT EDIT
│   └── vite.config.ts        # Vite + Vitest + Wails plugin
├── .github/workflows/ci.yml  # CI: lint-go, test-go, typecheck, lint-frontend, test-frontend
├── .golangci.yml             # Go linter config (10 linters)
├── .prettierrc               # Frontend formatting (semi=false, singleQuote, trailingComma=all)
└── frontend/eslint.config.js # ESLint 9 flat config (Vue + TS)
```

## WHERE TO LOOK
|Task|Location|Notes|
|---|---|---|
|Register new Go service|`main.go:47-48`|`application.NewService()` — also regenerate bindings|
|Add DB driver|`internal/repository/driver_*.go`|Implement `DatabaseDriver` + `SchemaIntrospector` interfaces|
|Add DB type|`internal/model/`|Pure structs with `json:"..."` tags|
|Add business logic|`internal/service/`|Constructor injection, returns `model.*` types|
|Add persistence|`internal/repository/`|JSONStore + sync.RWMutex pattern|
|Add Vue component|`frontend/src/components/`|`<script setup lang="ts">`, CSS variables|
|Add Pinia store|`frontend/src/stores/`|Options API (`state + getters + actions`)|
|Add frontend type|`frontend/src/types/`|Re-export from bindings + app-specific interfaces|
|Call Go from frontend|`frontend/bindings/`|Auto-generated — use types/ not bindings/ directly|
|SQL security|`internal/service/query.go`|Whitelist + quote escape + `?` parameterization|
|Multi-driver routing|`internal/repository/connection_pool.go:48`|`resolveDriverAndSchema(conn)` by `conn.Driver` field|
|SSH tunnel|`internal/repository/ssh_tunnel.go`|Port forwarding via `crypto/ssh`|
|Query cancellation|`internal/service/query_registry.go`|Per-query context with `Stop()`|
|ER diagram|`frontend/src/components/TableERDiagram.vue`|Pure SVG layout, `INFORMATION_SCHEMA.KEY_COLUMN_USAGE`|
|Query history|`frontend/src/components/QueryHistoryPanel.vue`|Persisted via `HistoryRepository`|

## MULTI-DRIVER ARCHITECTURE
```
main.go: 注册 drivers map ─→ ConnectionManager ─→ GetDB(conn) ─→ resolveDriverName(conn)
                              drivers["mysql"]                    ├─ "" → "mysql" (默认)
                              drivers["postgres"]                 └─ "postgres" → PostgresDriver
                              drivers["sqlite"]

Connection.Driver ──┐
                    ├── "mysql"    → MySQLDriver / MySQLSchema    (`` ` `` quoting, SHOW DATABASES)
                    ├── "postgres" → PostgresDriver / PostgresSchema (`` " `` quoting, pg_catalog)
                    └── "sqlite"   → SQLiteDriver / SQLiteSchema  (`` " `` quoting, sqlite_master)
```
- `Connection.Driver` 字段控制使用哪个驱动（空值默认 `"mysql"`）
- `main.go` 将所有驱动注册进 `map[string]DatabaseDriver` / `map[string]SchemaIntrospector`
- `ConnectionManager.resolveDriverAndSchema()` 根据 `conn.Driver` 在连接时查找驱动
- `SchemaIntrospector` 处理每种数据库的标识符引用：MySQL `` ` ``，PG/SQLite `"`
- 旧连接（无 `driver` 字段）默认回退到 `"mysql"` — 向后兼容
- Schema 解析为惰性：`GetDB` → `resolveDriverAndSchema` 首次调用时才查找，不在构造时固定

## CONVENTIONS
- **Chinese error messages** for user-facing strings; English for code comments
- **All Go test files** use table-driven tests with `t.Run()` — no testify
- **All Vue components** use `<script setup lang="ts">` — no Options API
- **Go imports**: stdlib → third-party → `tuxedosql/...` (3 groups)
- **No path aliases** in frontend — all imports are relative (`../`, `../../`)
- **Constructor injection everywhere** — no global state, no `init()`, no `sync.Once` (except credential lazy key)
- **Driver registry pattern** — new drivers registered in `main.go` maps only
- **Connection.Driver** empty → defaults to `"mysql"` for backward compat

## COMMANDS
```bash
# Development
wails3 dev                        # Hot-reload (backend + frontend)

# Quality (all pass)
task check                        # All gates: lint+test+typecheck
go test ./... -count=1            # Go tests (160+)
go vet ./...                      # Go vet
npm test                          # Vitest (60 tests) — from frontend/
npx vue-tsc --noEmit              # TypeScript check — from frontend/
npx eslint src/                   # ESLint (0 errors) — from frontend/
golangci-lint run ./...           # Go linter

# Build
wails3 build                      # Production desktop binary
task build:server                 # HTTP server mode (-tags server)
task build:docker                 # Distroless Docker image

# Bindings (after Go model changes)
wails3 generate bindings          # Regenerate TS bridge
```

## BRANCH STRATEGY
- **轻量主干开发（GitHub Flow 变体）** — 完整策略见 `.omp/branch-strategy.md`
- `master` 始终可发布；功能走短命分支 `feat/`/`fix/`/`chore/`，squash 合并后即删
- 单文件小修复可直推 master（先过 `task common:check`）；多文件改动/依赖升级走分支 + PR
- 版本：语义化预发布 `v0.1.0-alpha.N`，里程碑完成从 master 直接打 tag
- wails alpha 线：每 2 周一次 `chore/deps/wails` 批量升级（标准流程见 branch-strategy.md），平台相关崩溃修复可提前

## NOTES
- Theme logic split between `main.ts` (init) and `App.vue` (toggle) — both use `data-theme` attribute
- Root `Taskfile.yml` delegates quality commands via `common:` prefix — `task check` won't work from root; use `task common:check` or add forwarding tasks
- `build/docker/Dockerfile.cross` uses Go 1.26 but `go.mod` specifies 1.25 — cross-compile for platforms
- `build/config.yml` ignores `frontend/` in dev watch mode (served by Vite dev server separately)
- PostgreSQL driver uses `github.com/lib/pq`; SQLite driver uses `modernc.org/sqlite` (pure Go, no CGO)
- SQLite `Connection.Host` doubles as file path (or `:memory:`); port/username/password optional
- Hard NEVER-rules (bindings, service silos, SQL interpolation, driver hardcoding…) live in `.omp/RULES.md` — always-apply sticky rules loaded by omp
- Project omp settings (`disabledProviders`, …) live in `.omp/config.yml`
- Roadmap docs: `.omp/prds/`（产品 PRD + 里程碑状态）、`.omp/plans/`（里程碑实施计划）— 自 Claude Code `.claude/` 继承迁移
