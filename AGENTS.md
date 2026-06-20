# TuxedoSQL — Project Knowledge Base

**Updated:** 2026-06-20
**Branch:** master

## OVERVIEW
TuxedoSQL is a Wails v3 desktop SQL client — Go 1.25 backend, Vue 3 + TypeScript + Element Plus frontend. Multi-driver support: **MySQL, PostgreSQL, SQLite** via driver registry pattern. SSH tunnel support for remote connections. ~270 files, ~25k LoC. Two registered services (ConnectionService, QueryService). Version 0.1.0.

## STRUCTURE
```
.
├── main.go                   # App assembly — creates store → repos → services → window
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
| Task | Location | Notes |
|------|----------|-------|
| Register new Go service | `main.go:47-48` | `application.NewService()` — also regenerate bindings |
| Add DB driver | `internal/repository/driver_*.go` | Implement `DatabaseDriver` + `SchemaIntrospector` interfaces |
| Add DB type | `internal/model/` | Pure structs with `json:"..."` tags |
| Add business logic | `internal/service/` | Constructor injection, returns `model.*` types |
| Add persistence | `internal/repository/` | JSONStore + sync.RWMutex pattern |
| Add Vue component | `frontend/src/components/` | `<script setup lang="ts">`, CSS variables |
| Add Pinia store | `frontend/src/stores/` | Options API (`state + getters + actions`) |
| Add frontend type | `frontend/src/types/` | Re-export from bindings + app-specific interfaces |
| Call Go from frontend | `frontend/bindings/` | Auto-generated — use types/ not bindings/ directly |
| SQL security | `internal/service/query.go` | Whitelist + quote escape + `?` parameterization |
| Multi-driver routing | `internal/repository/connection_pool.go:48` | `resolveDriverAndSchema(conn)` by `conn.Driver` field |
| SSH tunnel | `internal/repository/ssh_tunnel.go` | Port forwarding via `crypto/ssh` |
| Query cancellation | `internal/service/query_registry.go` | Per-query context with `Stop()` |
| ER diagram | `frontend/src/components/TableERDiagram.vue` | Pure SVG layout, `INFORMATION_SCHEMA.KEY_COLUMN_USAGE` |
| Query history | `frontend/src/components/QueryHistoryPanel.vue` | Persisted via `HistoryRepository` |

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

## ANTI-PATTERNS (THIS PROJECT)
- **NEVER** manually edit `frontend/bindings/` — it's regenerated by `wails3 generate bindings`
- **NEVER** call other services from a service — services are independent silos
- **NEVER** import `internal/` from `pkg/` — enforced by Go toolchain
- **NEVER** commit `frontend/dist/` or `bin/` — excluded in `.gitignore`
- **NEVER** use raw SQL string interpolation — parameterize with `?` + whitelist columns
- **NEVER** hardcode a single driver — use `conn.Driver` to resolve at connection time
- **NEVER** call `Schema()` without passing `conn` — the signature is `Schema(conn *model.Connection)`

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

## NOTES
- `greet.go` in `internal/service/` is unused boilerplate — not registered in `main.go`
- Theme logic split between `main.ts` (init) and `App.vue` (toggle) — both use `data-theme` attribute
- Root `Taskfile.yml` delegates quality commands via `common:` prefix — `task check` won't work from root; use `task common:check` or add forwarding tasks
- `build/docker/Dockerfile.cross` uses Go 1.26 but `go.mod` specifies 1.25 — cross-compile for platforms
- `build/config.yml` ignores `frontend/` in dev watch mode (served by Vite dev server separately)
- PostgreSQL driver uses `github.com/lib/pq`; SQLite driver uses `modernc.org/sqlite` (pure Go, no CGO)
- SQLite `Connection.Host` doubles as file path (or `:memory:`); port/username/password optional
