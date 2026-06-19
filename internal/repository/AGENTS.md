# TuxedoSQL — Repository Layer (`internal/repository/`)

## OVERVIEW
Data access layer. Manages JSON file persistence, MySQL connection pools, and credential storage. Uses `sync.RWMutex` for thread safety.

## STRUCTURE
```
internal/repository/
├── connection_repo.go        # ConnectionRepository — JSON store + credential manager (170 lines)
├── connection_repo_test.go   # Tests: isLegacyAES, load/save, credential markers
├── connection_pool.go        # ConnectionManager — sql.DB pool cache per connID:database (152 lines)
├── connection_pool_test.go   # Tests: init, pool ops, prefix matching
├── tab_repo.go               # TabRepository — tab state persistence (45 lines)
└── tab_repo_test.go          # Tests: save/load round-trip, concurrency
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Load/save connections | `connection_repo.go:LoadConnections/SaveConnections` | Handles 3 password formats |
| Connection pooling | `connection_pool.go:GetDB()` | Double-checked locking, DSN with timezone |
| Pool cleanup | `connection_pool.go:Close/CloseAll` | Prefix-match on `connID:` |
| Tab persistence | `tab_repo.go:LoadTabs/SaveTabs` | JSON round-trip |
| Credential storage | `connection_repo.go` → `pkg/credential` | OS keyring → AES fallback → legacy .key |

## CONVENTIONS
- **sync.RWMutex** on every repository struct — `RLock()` for reads, `Lock()` for writes
- **JSONStore as sole persistence** — all repos take `*fileutil.JSONStore` in constructor
- **File name constants** — `connectionsFile`, `groupsFile`, `tabsFile` all in `~/.tuxedosql/`
- **Connection pool key: `connectionID:database`** — independent pools per (conn, db) pair
- **Pool limits**: max 5 open, 2 idle, 30min idle timeout, 1hr lifetime
- **Double-checked locking** in `GetDB()` — RLock check → Lock → double-check
- **Backward-compatible password migration** — 3 formats: `keyring:`, `aes256gcm$`, plaintext

## ANTI-PATTERNS
- **NEVER** import `internal/service/` — dependency flows service → repository only
- **NEVER** expose `*sql.DB` directly beyond pool management — service uses `GetDBByID()`
- **NEVER** use `os.File` directly — go through `fileutil.JSONStore`
