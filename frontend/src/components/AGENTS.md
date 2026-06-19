# TuxedoSQL — Frontend Components (`frontend/src/components/`)

## OVERVIEW
19 Vue 3 SFCs using `<script setup lang="ts">` + Element Plus. Component tree is 4 levels deep with App.vue as root.

## STRUCTURE
```
components/
├── App.vue (in ../)                    # Root layout — sidebar + tabs + dialogs + bottom bar
├── Sidebar.vue                         # Connection/schema tree + CRUD dialogs (377 lines)
│   ├── ConnectionTree.vue              # el-tree: drag-drop, context menu (316 lines)
│   ├── GroupDialog.vue                 # Group create/edit modal
│   ├── CreateDatabaseDialog.vue        # DB creation modal
│   └── CreateTableDialog.vue           # Table creation wizard (402 lines)
├── QueryTabs.vue                       # Central orchestrator — tabs, editor/table toggle (626 lines)
│   ├── QueryEditor.vue                 # Toolbar + SqlEditor wrapper (130 lines)
│   │   └── SqlEditor.vue               # CodeMirror 6 — schema-aware autocomplete (202 lines)
│   ├── QueryResult.vue                 # el-table: sort, filter, inline edit, pagination (581 lines)
│   ├── TableView.vue                   # Data browser — table/form toggle, export (713 lines)
│   │   ├── RecordForm.vue              # Single-row editing form (350 lines)
│   │   ├── DataExport.vue              # CSV/SQL export modal
│   │   └── TableSearch.vue (features/) # Visual query builder (AND/OR groups)
│   ├── TableInfoPanel.vue              # Column metadata display
│   ├── TableDDLPanel.vue               # DDL viewer with copy
│   ├── MessagePanel.vue                # Collapsible message/SQL log
│   └── ResizableSplitter.vue           # Drag-to-resize pane divider
├── ConnectionDialog.vue                # Connection CRUD form with test-connect
└── BottomBar.vue                       # Status bar — SQL display, sidebar toggles
```

## WHERE TO LOOK
| Task | Location | Notes |
|------|----------|-------|
| Add new component | New `.vue` file here | `<script setup lang="ts">`, CSS variables |
| Add dialog | Follow existing pattern: `:visible` prop + `close`/`saved` emits |
| Call Go service | Import from `../../bindings/tuxedosql/internal/service` | Use `parseError()` for error handling |
| Add Pinia action | `frontend/src/stores/` | Components read stores, never mutate directly |
| Add composable | `frontend/src/composables/` | Currently: `parseError()` (5 consumers) |
| Add utility | `frontend/src/lib/` | Currently: `formatCellValue()` (3 consumers) |

## CONVENTIONS
- **`<script setup lang="ts">`** on every component — no Options API
- **`defineProps<>()` + `defineEmits<>()`** for component interface — never `PropType<>`
- **CSS variables** for theming — `var(--color-accent)`, `var(--color-border)`, etc. No hardcoded hex
- **`<style scoped>`** universally; unscoped `<style>` only for Teleported context menus
- **Emit-based communication** — `emit('event', payload)`, never `provide/inject`
- **`withDefaults()`** only for optional props with defaults
- **Dialogs**: props `visible` + emits `close`/`saved` — consistent across all 6 dialog components
- **v-model**: `modelValue` + `update:modelValue` for two-way binding (SqlEditor, QueryEditor)

## ANTI-PATTERNS
- **NEVER** import from `frontend/bindings/` directly — use `frontend/src/types/` re-exports
- **NEVER** mutate Pinia state directly in components — use store actions
- **NEVER** use Options API — `<script setup>` is mandatory
- **NEVER** hardcode colors — use CSS custom properties
