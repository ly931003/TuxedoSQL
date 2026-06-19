# TuxedoSQL — Service Layer (`internal/service/`)

## OVERVIEW
Business logic services. Each service is a struct registered in `main.go`, auto-exposed to the Vue frontend via Wails bridge.

## STRUCTURE
```
internal/service/
├── connection.go        # ConnectionService — CRUD, groups, DDL, schema browsing (730 lines)
├── connection_test.go   # Table-driven tests (12 functions)
├── query.go             # QueryService — SQL exec, pagination, filtering, row edit (819 lines)
├── query_test.go        # Table-driven tests (27 functions)
├── greet.go             # GreetService — unused boilerplate (NOT registered in main.go)
└── greet_test.go        # Example test
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Add a new service | Create new file, register in `main.go:32-34` | Constructor injection, return pointer |
| SQL security | `query.go:buildFilterClause()` | Whitelist + backtick + `?` — 3 defense layers |
| Execute SQL | `query.go:Execute()` | Auto-detect SELECT vs DML (`isQueryStatement`) |
| Browse table data | `query.go:GetTableData()` | Pagination, sort, nested filters |
| Inline row edit | `query.go:UpdateRow()` | Whitelist-validated PK + column |
| Create connection | `connection.go:Create()` | Validates empty fields, calls repo.SaveConnections |
| DDL operations | `connection.go:CreateTable/DropTable` | String-based SQL building |

## CONVENTIONS
- **1 struct = 1 file** — never two service structs in one file
- **Constructor: `New<Service>(deps...)`** — takes `*repository.Xxx` pointers, returns `*Service`
- **Chinese errors** for user-facing strings (`fmt.Errorf("连接名称不能为空")`)
- **Every public method starts with validation** — empty name/host/ID guards first
- **context.WithTimeout** for every DB operation: 30s queries, 10s DDL, 5s ping
- **Audit SQL is separate** — `buildDisplaySQL()` / `buildAuditSQL()` produce human-readable SQL for display only
- **maxRows = 10000** cap, PageSize clamped to 1000

## ANTI-PATTERNS
- **NEVER** import `pkg/` directly — services go through repository
- **NEVER** call another service — services are independent silos
- **NEVER** use raw SQL string interpolation — parameterize with `?` + whitelist

## UNIQUE STYLES
- `findConnection()` / `isDescendant()` as unexported helpers within same file
- `isQueryStatement()` discriminates SELECT/SHOW/DESCRIBE/WITH vs INSERT/UPDATE/DELETE
- `getColumnWhitelist()` fetches `INFORMATION_SCHEMA.COLUMNS` dynamically per-table
