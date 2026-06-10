<script setup lang="ts">
import { ref, nextTick, watch } from 'vue'
import { SortOrder, FilterOperator, LogicOp } from '../types/query'
import type { ColumnInfo, FilterCondition, FilterGroup } from '../types/query'
import { FILTER_OPERATOR_LABELS } from '../types/query'
import { formatCellValue } from '../lib/timeFormat'

const props = defineProps<{
  columns: ColumnInfo[]
  rows: Record<string, unknown>[]
  message?: string
  duration?: number
  // ── Pagination (optional, for table view) ──
  paginated?: boolean
  page?: number
  pageSize?: number
  total?: number
  totalPages?: number
  // ── Sorting (optional) ──
  sortColumn?: string
  sortOrder?: SortOrder
  // ── Filters (optional) ──
  filters?: FilterGroup | null
  // ── Loading (optional) ──
  loading?: boolean
  // ── Inline editing (optional) ──
  editable?: boolean
  editingCell?: { rowIndex: number; columnName: string } | null
  editingValue?: string
  // ── Row selection (optional) ──
  selectedRowIndex?: number
}>()

const emit = defineEmits<{
  'update:page': [page: number]
  'update:pageSize': [pageSize: number]
  'sort-change': [column: string, order: SortOrder]
  'filter-change': [filters: FilterGroup | null]
  'cell-dblclick': [rowIndex: number, columnName: string]
  'cell-edit-confirm': [rowIndex: number, columnName: string, newValue: string]
  'cell-edit-cancel': []
  'cell-edit-update:value': [value: string]
  'row-select': [rowIndex: number]
}>()

// ── Sort helpers ──

function getSortIcon(colName: string): string {
  if (props.sortColumn !== colName) return '↕'
  return props.sortOrder === SortOrder.SortASC ? '↑' : '↓'
}

function handleSortClick(colName: string) {
  if (props.sortColumn !== colName) {
    emit('sort-change', colName, SortOrder.SortASC)
  } else if (props.sortOrder === SortOrder.SortASC) {
    emit('sort-change', colName, SortOrder.SortDESC)
  } else {
    emit('sort-change', '', SortOrder.SortASC)
  }
}

// ── Filter context menu ──

const filterVisible = ref(false)
const filterColumn = ref('')
const filterPos = ref({ x: 0, y: 0 })
const filterOperator = ref<FilterOperator>(FilterOperator.OpEQ)
const filterValue = ref('')

function handleHeaderContextMenu(event: MouseEvent, colName: string) {
  event.preventDefault()
  event.stopPropagation()
  filterColumn.value = colName
  filterPos.value = { x: event.clientX, y: event.clientY }
  filterOperator.value = FilterOperator.OpEQ
  filterValue.value = ''
  filterVisible.value = true
}

function applyFilter() {
  if (filterColumn.value) {
    const existing = (props.filters?.conditions ?? []).filter((f): f is FilterGroup => f != null && 'column' in f && (f as FilterCondition).column !== filterColumn.value)
    const newChildren: FilterGroup[] = [...existing]
    if (filterOperator.value === FilterOperator.OpIsNull || filterOperator.value === FilterOperator.OpNotNull || filterValue.value) {
      newChildren.push({
        logic: LogicOp.LogicAND, conditions: [],
        column: filterColumn.value,
        operator: filterOperator.value,
        value: filterValue.value,
      })
    }
    if (newChildren.length > 0) {
      emit('filter-change', { logic: LogicOp.LogicAND, conditions: newChildren, column: '', operator: FilterOperator.OpEQ, value: '' })
    } else {
      emit('filter-change', null)
    }
  }
  filterVisible.value = false
}

function removeFilter(colName: string) {
  const remaining = (props.filters?.conditions ?? []).filter((f): f is FilterGroup => f != null && 'column' in f && (f as FilterCondition).column !== colName)
  if (remaining.length > 0) {
    emit('filter-change', { logic: LogicOp.LogicAND, conditions: remaining, column: '', operator: FilterOperator.OpEQ, value: '' })
  } else {
    emit('filter-change', null)
  }
}

// ── Pagination ──

const jumpPage = ref('')

function goToPage(p: number) {
  if (p < 1 || p > (props.totalPages ?? 1)) return
  emit('update:page', p)
}

function handleJump() {
  const p = parseInt(jumpPage.value, 10)
  if (isNaN(p)) return
  goToPage(p)
  jumpPage.value = ''
}

// ── Row selection ──

function handleRowClick(row: Record<string, unknown>) {
  const index = props.rows.indexOf(row)
  if (index !== -1) {
    emit('row-select', index)
  }
}

// ── Inline editing auto-focus ──

watch(() => props.editingCell, (newVal) => {
  if (newVal) {
    nextTick(() => {
      const input = document.querySelector('.cell-input') as HTMLInputElement | null
      input?.focus()
      input?.select()
    })
  }
})
</script>

<template>
  <div class="query-result">
    <div class="result-table-wrapper">
      <el-table
        :data="rows"
        border
        stripe
        size="small"
        height="100%"
        table-layout="auto"
        highlight-current-row
        :class="{ 'is-loading': loading }"
        v-loading="loading"
        @row-click="handleRowClick"
      >
        <el-table-column
          v-for="col in columns"
          :key="col.name"
          :min-width="120"
          show-overflow-tooltip
        >
          <template #header>
            <div
              class="col-header"
              @click="paginated ? handleSortClick(col.name) : undefined"
              @contextmenu="paginated ? handleHeaderContextMenu($event, col.name) : undefined"
            >
              <span class="col-name">{{ col.name }}</span>
              <span v-if="paginated" class="col-type">{{ col.type }}</span>
              <span v-if="paginated" class="sort-icon">{{ getSortIcon(col.name) }}</span>
            </div>
          </template>
          <template #default="{ row, $index }">
            <span
              v-if="!editable || !editingCell || editingCell.rowIndex !== $index || editingCell.columnName !== col.name"
              class="cell-display"
              :class="{ 'cell-editable': editable }"
              @dblclick="editable && emit('cell-dblclick', $index, col.name)"
            >
              <span v-if="row[col.name] === null || row[col.name] === undefined" class="null-value">(NULL)</span>
              <span v-else>{{ formatCellValue(col.type, row[col.name]) }}</span>
            </span>
            <input
              v-else
              class="cell-input"
              :value="editingValue"
              @input="emit('cell-edit-update:value', ($event.target as HTMLInputElement).value)"
              @keydown.enter.stop="emit('cell-edit-confirm', $index, col.name, editingValue ?? '')"
              @keydown.escape.stop="emit('cell-edit-cancel')"
              @blur="emit('cell-edit-confirm', $index, col.name, editingValue ?? '')"
            />
          </template>
        </el-table-column>
      </el-table>
    </div>

    <!-- Pagination footer -->
    <div v-if="paginated" class="result-footer">
      <div class="footer-left">
        <span v-if="message">{{ message }}</span>
        <span v-if="duration && duration > 0" class="result-duration">
          用时 {{ (duration / 1000).toFixed(3) }}s
        </span>
      </div>
      <div class="footer-right pagination-bar">
        <span class="page-info">共 {{ total ?? 0 }} 行</span>
        <select
          class="page-size-select"
          :value="pageSize ?? 100"
          @change="emit('update:pageSize', parseInt(($event.target as HTMLSelectElement).value, 10))"
        >
          <option :value="20">20</option>
          <option :value="50">50</option>
          <option :value="100">100</option>
          <option :value="200">200</option>
        </select>
        <span class="page-info">行/页</span>
        <button class="page-btn" :disabled="(page ?? 1) <= 1" @click="goToPage(1)" title="首页">«</button>
        <button class="page-btn" :disabled="(page ?? 1) <= 1" @click="goToPage((page ?? 1) - 1)" title="上一页">‹</button>
        <span class="page-current">{{ page ?? 1 }} / {{ totalPages ?? 1 }}</span>
        <button class="page-btn" :disabled="(page ?? 1) >= (totalPages ?? 1)" @click="goToPage((page ?? 1) + 1)" title="下一页">›</button>
        <button class="page-btn" :disabled="(page ?? 1) >= (totalPages ?? 1)" @click="goToPage(totalPages ?? 1)" title="末页">»</button>
        <input
          v-model="jumpPage"
          class="jump-input"
          type="number"
          :min="1"
          :max="totalPages ?? 1"
          placeholder="页"
          @keydown.enter="handleJump"
        />
      </div>
    </div>

    <!-- Non-paginated footer -->
    <div v-else class="result-footer">
      <span>{{ message }}</span>
      <span v-if="duration && duration > 0" class="result-duration">
        用时 {{ (duration / 1000).toFixed(3) }}s
      </span>
    </div>

    <!-- Active filter badges -->
    <div v-if="paginated && filters && filters.conditions && filters.conditions.length > 0" class="filter-badges">
      <span
        v-for="f in filters.conditions.filter((c): c is FilterGroup => c != null && 'column' in c && typeof (c as FilterCondition).column === 'string')"
        :key="(f as FilterCondition).column"
        class="filter-badge"
      >
        {{ (f as FilterCondition).column }} {{ (FILTER_OPERATOR_LABELS as Record<string, string>)[(f as FilterCondition).operator] }} {{ (f as FilterCondition).operator === 'isnull' || (f as FilterCondition).operator === 'notnull' ? '' : (f as FilterCondition).value }}
        <button class="filter-remove" @click="removeFilter((f as FilterCondition).column)">×</button>
      </span>
    </div>

    <!-- Filter context menu (Teleport to body) -->
    <Teleport to="body">
      <div v-if="filterVisible" class="ctx-fixed" @click="filterVisible = false" />
      <div v-if="filterVisible" class="ctx-menu filter-menu" :style="{ left: filterPos.x + 'px', top: filterPos.y + 'px' }">
        <div class="filter-menu-title">{{ filterColumn }}</div>
        <div class="filter-menu-body">
          <select v-model="filterOperator" class="filter-op-select">
            <option v-for="(label, op) in FILTER_OPERATOR_LABELS" :key="op" :value="op">{{ label }}</option>
          </select>
          <input
            v-if="filterOperator !== 'isnull' && filterOperator !== 'notnull'"
            v-model="filterValue"
            class="filter-val-input"
            type="text"
            placeholder="输入值..."
            @keydown.enter="applyFilter"
          />
        </div>
        <div class="filter-menu-actions">
          <button class="filter-btn filter-btn--apply" @click="applyFilter">应用</button>
          <button class="filter-btn filter-btn--cancel" @click="filterVisible = false">取消</button>
        </div>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.query-result {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  background: var(--color-surface, #fff);
  overflow: hidden;
}

.result-table-wrapper {
  flex: 1;
  min-height: 0;
  overflow: auto;
  position: relative;
}

/* ── Column header ── */
.col-header {
  display: flex;
  align-items: center;
  gap: 4px;
  cursor: default;
  user-select: none;
}

.col-name {
  font-weight: 600;
}

.col-type {
  font-size: 10px;
  color: var(--color-text-secondary, #6e6e80);
  font-weight: 400;
  margin-left: auto;
}

.sort-icon {
  font-size: 11px;
  color: var(--color-accent, #6366f1);
  width: 14px;
  text-align: center;
}

.null-value {
  color: var(--color-result-null, #9ca3af);
  font-style: italic;
}

/* ── Footer ── */
.result-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 4px 12px;
  font-size: 12px;
  color: var(--color-text-secondary, #6e6e80);
  border-top: 1px solid var(--color-border, #d9d9dc);
  background: var(--color-sidebar, #f5f5f7);
  flex-shrink: 0;
  min-height: 28px;
}

.footer-left {
  display: flex;
  align-items: center;
  gap: 8px;
}

.footer-right {
  display: flex;
  align-items: center;
  gap: 4px;
}

.result-duration {
  color: var(--color-text-secondary, #6e6e80);
  font-size: 11px;
}

/* ── Pagination ── */
.pagination-bar {
  display: flex;
  align-items: center;
  gap: 4px;
}

.page-info {
  font-size: 11px;
  color: var(--color-text-secondary, #6e6e80);
}

.page-size-select {
  font-size: 11px;
  padding: 1px 4px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm, 4px);
  background: var(--color-input-bg);
  color: var(--color-text);
  outline: none;
}

.page-btn {
  width: 22px;
  height: 22px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm, 4px);
  background: var(--color-input-bg);
  cursor: pointer;
  font-size: 12px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.1s;
  color: var(--color-text);
}

.page-btn:hover:not(:disabled) {
  background: var(--color-hover, rgba(0,0,0,0.04));
}

.page-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.page-current {
  font-size: 11px;
  color: var(--color-text);
  min-width: 50px;
  text-align: center;
}

.jump-input {
  width: 40px;
  font-size: 11px;
  padding: 1px 4px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm, 4px);
  outline: none;
  text-align: center;
  background: var(--color-input-bg);
  color: var(--color-text);
}

/* ── Filter badges ── */
.filter-badges {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
  padding: 4px 12px;
  background: var(--color-sidebar, #f5f5f7);
  border-top: 1px solid var(--color-border, #d9d9dc);
  flex-shrink: 0;
}

.filter-badge {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 11px;
  padding: 1px 6px;
  background: var(--color-selected, rgba(99, 102, 241, 0.10));
  color: var(--color-accent, #6366f1);
  border-radius: 10px;
}

.filter-remove {
  border: none;
  background: none;
  cursor: pointer;
  font-size: 13px;
  line-height: 1;
  padding: 0;
  color: var(--color-text-secondary, #6e6e80);
}

.filter-remove:hover {
  color: #e74c3c;
}

/* ── Inline cell editing ── */
.cell-editable {
  cursor: pointer;
  display: inline-block;
  min-width: 100%;
}

.cell-editable:hover {
  background: var(--color-selected, rgba(99, 102, 241, 0.08));
  border-radius: 2px;
}

.cell-input {
  width: 100%;
  border: 2px solid var(--color-accent, #6366f1);
  border-radius: var(--radius-sm, 4px);
  padding: 2px 6px;
  font-size: 12px;
  font-family: inherit;
  background: var(--color-input-bg, #fff);
  color: var(--color-text, #1a1a2e);
  outline: none;
  box-sizing: border-box;
}

/* ── Current row highlight (el-table highlight-current-row) ── */
:deep(.el-table__body tr.current-row > td) {
  background: var(--color-selected, rgba(99, 102, 241, 0.10)) !important;
}
</style>

<style>
/* Context menu — unscoped for Teleport */
.ctx-fixed { position: fixed; inset: 0; z-index: 9998; }
.ctx-menu {
  position: fixed; z-index: 9999;
  background: var(--color-dropdown-bg);
  border: 1px solid var(--color-border);
  border-radius: 6px; box-shadow: 0 4px 16px var(--color-dropdown-shadow);
  min-width: 200px;
}

.filter-menu {
  padding: 0;
}

.filter-menu-title {
  padding: 6px 12px;
  font-size: 12px;
  font-weight: 600;
  color: var(--color-text);
  border-bottom: 1px solid var(--color-border);
}

.filter-menu-body {
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.filter-op-select {
  font-size: 12px;
  padding: 4px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm, 4px);
  outline: none;
  background: var(--color-input-bg);
  color: var(--color-text);
}

.filter-val-input {
  font-size: 12px;
  padding: 4px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm, 4px);
  outline: none;
  background: var(--color-input-bg);
  color: var(--color-text);
}

.filter-val-input:focus {
  border-color: var(--color-accent);
}

.filter-menu-actions {
  display: flex;
  justify-content: flex-end;
  gap: 6px;
  padding: 6px 12px;
  border-top: 1px solid var(--color-border);
}

.filter-btn {
  font-size: 12px;
  padding: 3px 12px;
  border: none;
  border-radius: var(--radius-sm, 4px);
  cursor: pointer;
}

.filter-btn--apply {
  background: var(--color-accent);
  color: #fff;
}

.filter-btn--cancel {
  background: var(--color-hover);
  color: var(--color-text);
}
</style>
VUEOOF
echo "QueryResult.vue written"