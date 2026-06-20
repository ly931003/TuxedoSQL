# 🐈‍⬛ TuxedoSQL

[中文文档](README.zh-CN.md)

<p align="center">
  <strong>Write SQL like you're in a tuxedo — sharp, confident, effortless.</strong>
</p>

**TuxedoSQL** is a sleek, cross-platform desktop database client. Built on **Wails v3**, it combines the raw power of **Go** on the backend with a polished **Vue 3** frontend — delivering a native-speed SQL editing experience without the Electron bloat.

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.25-00ADD8?logo=go" alt="Go 1.25">
  <img src="https://img.shields.io/badge/Vue-3.x-4FC08D?logo=vuedotjs" alt="Vue 3">
  <img src="https://img.shields.io/badge/Wails-v3-DF0000?logo=wails" alt="Wails v3">
  <img src="https://img.shields.io/badge/license-MIT-blue" alt="MIT License">
  <img src="https://img.shields.io/badge/platform-Windows%20%7C%20macOS%20%7C%20Linux-lightgrey" alt="Platforms">
</p>

## ✨ Why TuxedoSQL?

| You want... | You get |
|-------------|---------|
| 🚀 **Fast startup** | Go binary, no Chromium — launches in under a second |
| 🎯 **Rich SQL editing** | CodeMirror 6 with syntax highlighting, multi-cursor, and bracket matching |
| 🔍 **Visual query building** | Point-and-click query builder with nested AND/OR logic groups |
| 📊 **Browse with context** | Table view + inline record editing + DDL preview, all in one screen |
| 🔐 **Credentials you can trust** | OS-native keyring encryption, zero plaintext on disk |
| 🌏 **Timezone-aware** | Per-connection timezone with auto-formatted time columns |
| 🧩 **Works your way** | Desktop app, headless server, or Docker — same codebase |

## 🐱 Why "Tuxedo"?

The name comes from the author's two adorable tuxedo cats — black-and-white felines who look like they're permanently dressed for a formal dinner party. Just like them, this tool aims to be elegant, dependable, and maybe a little mischievous when it comes to handling your data.

## 📸 Sneak Peek

<!-- TODO: add screenshots -->
<!-- ![Query Editor](docs/screenshots/query-editor.png) -->

## 🧰 Features

| Category | Highlights |
|----------|-----------|
| **Connections** | Save, organize, test-connect — MySQL, PostgreSQL, SQLite via SSH tunnels |
| **Query Editor** | Multi-tab sessions, syntax highlighting, auto-completion hints |
| **Query Builder** | Visual WHERE clause builder with nested logic groups — no SQL typing required |
| **Data Browser** | Sortable columns, column-level filters, paginated results |
| **Record Editor** | Click to edit any cell, inline validation, type-aware form controls |
| **Schema Explorer** | Tree-view table list, DDL dump, column/index metadata at a glance |
| **Export** | Dump results to CSV/SQL INSERT/JSON with a single click |
| **Security** | OS keyring (macOS Keychain / Windows Credential Manager / Linux Secret Service) with AES-256 machine-ID fallback |
| **Layout** | Collapsible sidebar, draggable split panes — make the screen yours |
| **Server Mode** | Headless HTTP server mode for remote or containerized deployments |

## 🏗️ Tech Stack

| Layer | Choice | Why |
|-------|--------|-----|
| Desktop Shell | [Wails v3](https://v3.wails.io/) | Native OS bindings, ~10MB binaries |
| Backend | Go 1.25 | Fast, single binary, excellent DB drivers |
| Frontend | Vue 3 + TypeScript + Vite | Reactive, type-safe, HMR in dev |
| UI Kit | [Element Plus](https://element-plus.org/) | Mature Vue 3 component library |
| State | [Pinia](https://pinia.vuejs.org/) | Lightweight, devtools-friendly |
| Editor | CodeMirror 6 | Extensible, mobile-friendly, first-class SQL support |
| DB Driver | [go-sql-driver/mysql](https://github.com/go-sql-driver/mysql), [lib/pq](https://github.com/lib/pq), [modernc.org/sqlite](https://pkg.go.dev/modernc.org/sqlite) | Multi-driver with registry pattern |
| Crypto | OS keyring + `golang.org/x/crypto` | AES-256 with machine-ID derived key as fallback |

## 🚀 Quick Start

### You'll need

- **Go** ≥ 1.25
- **Node.js** ≥ 20
- **Wails v3 CLI**: `go install github.com/wailsapp/wails/v3/cmd/wails3@latest`

### One-minute launch

```bash
git clone https://github.com/your-org/TuxedoSQL.git && cd TuxedoSQL
cd frontend && npm install && cd ..
wails3 dev          # ← hot-reload for both Go and Vue
```

### Build for production

```bash
wails3 build        # native desktop binary in bin/
task run            # build + launch
```

### Server mode (headless)

```bash
task build:server   # pure HTTP server, no GUI deps
task run:server     # starts on :8080
```

### Docker

```bash
task setup:docker   # one-time: build the cross-compile image
task build:docker   # distroless prod image
task run:docker     # run it — port 8080
```

## 📁 Project Layout

```
TuxedoSQL/
├── main.go                   # entry point — creates app, registers services, launches window
├── Taskfile.yml              # task runner: dev / build / run / docker
│
├── internal/
│   ├── service/              # business services exposed to frontend via Wails bridge
│   │   ├── connection.go     #   CRUD + connectivity test + schema browsing
│   │   ├── query.go          #   SQL execution + schema introspection
│   │   ├── query_registry.go #   per-query context cancellation
│   │   └── integration_test.go
│   ├── model/                # domain types (Connection, Query, Tab, ForeignKey, ...)
│   └── repository/           # persistence + multi-driver pools + SSH tunnels
│       ├── connection_pool.go
│       ├── driver_*.go       # DatabaseDriver/SchemaIntrospector per DB
│       ├── ssh_tunnel.go      
│       ├── history_repo.go
│       └── interfaces.go
│
├── frontend/
│   ├── src/
│   │   ├── components/       # 20 Vue components (editor, sidebar, dialogs, panels)
│   │   ├── features/         # feature modules (TableSearch with query builder)
│   │   ├── composables/      # reusable logic hooks
│   │   ├── stores/           # Pinia state stores
│   │   └── types/            # shared TypeScript interfaces
│   ├── bindings/             # [auto-generated] Go→TS bridge — do not edit
│   └── package.json
│
├── build/                    # cross-platform packaging & Docker configs
│   ├── docker/               # distroless server-mode image
│   ├── linux/                # AppImage / NFPM
│   ├── darwin/               # macOS .app bundle
│   └── windows/              # NSIS / MSIX installers
│
└── docs/                     # project docs & PRDs
```

## 🧠 Architecture

TuxedoSQL is three clean layers:

```
┌──────────────────────────────────────┐
│  Vue 3 Frontend                      │
│  Pinia stores → components → Element Plus UI
└──────────────┬───────────────────────┘
               │ Wails Bridge (auto-generated TS bindings)
┌──────────────▼───────────────────────┐
│  Go Service Layer                    │
│  ConnectionService / QueryService    │
│  → repository → model               │
└──────────────┬───────────────────────┘
               │ database/sql (driver registry)
┌──────────────▼───────────────────────┐
│  MySQL  │  PostgreSQL  │  SQLite     │
└──────────────────────────────────────┘
```

Adding a new frontend-callable API is three steps:

1. Define a service struct in `internal/service/`
2. Register it in `main.go` with `application.NewService()`
3. Regenerate bindings: `wails3 generate bindings`

## 🗺️ Roadmap

- [x] Connection management with encrypted credentials
- [x] Multi-tab SQL editor (CodeMirror 6)
- [x] Visual query builder with nested logic groups
- [x] Table data browser with sorting, filtering, pagination
- [x] Inline record editor
- [x] Schema explorer + DDL viewer
- [x] Data export (CSV/SQL INSERT/JSON)
- [x] Server mode + Docker deployment
- [x] PostgreSQL support
- [x] SQLite support
- [x] SSH tunnel connections
- [x] Query history & favorites
- [x] Dark mode
- [x] ER diagram visualizer

## 🤝 Contributing

Pull requests are welcome! Check the [CLAUDE.md](CLAUDE.md) for architecture details, project conventions, and development commands.

## 📄 License

MIT — use it, fork it, ship it.
