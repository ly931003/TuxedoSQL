<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useQueryStore } from '../stores/query'
import { SortOrder } from '../types/query'
import { QueryService } from '../../bindings/tuxedosql/internal/service'
import * as models from '../../bindings/tuxedosql/internal/model/models'
import QueryResult from './QueryResult.vue'
import DataExport from './DataExport.vue'
import TableSearch from '../features/table/TableSearch.vue'
import type { QueryTab, PageResult, FilterCondition, DirtyChange, EditingCell } from '../types/query'
import type { TableSchema } from '../../bindings/tuxedosql/internal/model/models'

const store = useQueryStore()

const props = defineProps<{
  tab: QueryTab
}>()

const exportVisible = ref(false)
const loading = ref(false)

// ── Inline editing state ──
const editingCell = ref<EditingCell | null>(null)
const editingValue = ref('')
const dirtyMap = reactive<Record<string, DirtyChange>>({})
const dirtyCount = computed(() => Object.keys(dirtyMap).length)
const applying = ref(false)
const pkColumns = ref<string[]>([])
const schemaLoaded = ref(false)
const searchFilter = ref<{ column: string; keyword: string } | null>(null)

function dirtyKey(rowIndex: number, columnName: string): string {
  return `${rowIndex}:${columnName}`
}

function parseError(err: unknown): string {
  if (err instanceof Error) {
    try { const p = JSON.parse(err.message); if (p?.message) return String(p.message) } catch {}
    return err.message
  }
  if (err && typeof err === 'object') {
    const msg = (err as Record<string, unknown>).message
    if (typeof msg === 'string') return msg
  }
  const raw = String(err)
  try { const p = JSON.parse(raw); if (p?.message) return String(p.message) } catch {}
  return raw
}

function buildResultFromPage(p: PageResult): Parameters<typeof store.setResult>[1] {
  return {
    columns: p.columns,
    rows: p.rows,
    affectedRows: 0,
    message: p.message,
    messageType: p.messageType,
    duration: p.duration,
  }
}

// ── Schema loading (for primary key detection) ──

async function loadSchema() {
  const tab = props.tab
  if (!tab.tableName || !tab.connectionId || !tab.database) return

  try {
    const schemas: TableSchema[] = await QueryService.GetTableSchema(
      tab.connectionId, tab.database, tab.tableName
    )
    pkColumns.value = schemas
      .filter(s => s.columnKey === 'PRI')
      .map(s => s.name)
    schemaLoaded.value = true
  } catch (err: unknown) {
    store.addMessage(tab.id, `加载表结构失败: ${parseError(err)}`)
  }
}

// loadData 接受可选覆盖参数，翻页/排序/筛选时直接传入新值，绕过 Pinia 响应式传播延迟
async function loadData(overrides?: {
  page?: number
  pageSize?: number
  sortColumn?: string
  sortOrder?: SortOrder
  filters?: FilterCondition[]
}) {
  const tab = props.tab
  if (!tab.tableName || !tab.connectionId || !tab.database) return
  if (loading.value) return

  if (!schemaLoaded.value) {
    await loadSchema()
  }

  loading.value = true
  store.setExecuting(tab.id, true)

  const page = overrides?.page ?? tab.page ?? 1
  const pageSize = overrides?.pageSize ?? tab.pageSize ?? 100
  const sortColumn = overrides?.sortColumn ?? tab.sortColumn ?? ''
  const sortOrder = overrides?.sortOrder ?? tab.sortOrder ?? SortOrder.SortASC
  const filters = overrides?.filters ?? tab.filters ?? []

  // Attach search as LIKE filter
  const allFilters = [...filters]
  if (searchFilter.value) {
    allFilters.push({
      column: searchFilter.value.column,
      operator: 'contains' as FilterCondition['operator'],
      value: searchFilter.value.keyword,
    } as FilterCondition)
  }

  try {
    const params = new models.TableDataParams({
      connectionId: tab.connectionId,
      database: tab.database,
      table: tab.tableName,
      page,
      pageSize,
      sortColumn,
      sortOrder: sortOrder as models.SortOrder,
      filters: allFilters.map(f => new models.FilterCondition({
        column: f.column,
        operator: f.operator as models.FilterOperator,
        value: f.value,
      })),
    })
    const result = await QueryService.GetTableData(params)
    if (result) {
      store.setResult(tab.id, buildResultFromPage(result))
      store.setTablePageResult(tab.id, result.total, result.totalPages)
      // 审计：仅在翻页/排序/筛选变化时记录 SQL（首次加载也一样）
      if (result.sql) {
        store.addMessage(tab.id, `📋 ${result.sql}`)
      }
    }
  } catch (err: unknown) {
    store.addMessage(tab.id, parseError(err))
  } finally {
    store.setExecuting(tab.id, false)
    loading.value = false
  }
}

// ── Table search handlers ──

function handleTableSearch(params: { column: string; keyword: string }) {
  searchFilter.value = { column: params.column, keyword: params.keyword }
  loadData({ page: 1 })
}

function handleTableSearchReset() {
  searchFilter.value = null
  loadData({ page: 1 })
}

// ── Navigation handlers (auto-apply dirty) ──

async function discardIfDirty() {
  if (dirtyCount.value > 0) {
    await handleApply()
  }
}

async function handleSortChange(column: string, order: SortOrder) {
  store.setTableSorting(props.tab.id, column, order)
  await discardIfDirty()
  loadData({ page: 1, sortColumn: column, sortOrder: order as SortOrder })
}

async function handleFilterChange(filterList: FilterCondition[]) {
  store.setTableFilters(props.tab.id, filterList)
  await discardIfDirty()
  loadData({ page: 1, filters: filterList })
}

async function handlePageChange(page: number) {
  store.setTablePage(props.tab.id, page)
  await discardIfDirty()
  loadData({ page })
}

async function handlePageSizeChange(pageSize: number) {
  store.setTablePageSize(props.tab.id, pageSize)
  await discardIfDirty()
  loadData({ page: 1, pageSize })
}

// ── Cell editing handlers ──

function handleCellDblClick(rowIndex: number, columnName: string) {
  const rows = props.tab.result?.rows ?? []
  const row = rows[rowIndex]
  if (!row) return

  const existingKey = dirtyKey(rowIndex, columnName)
  const existingDirty = dirtyMap[existingKey]
  const currentValue = existingDirty ? existingDirty.newValue : row[columnName]

  editingValue.value = currentValue === null || currentValue === undefined ? '' : String(currentValue)
  editingCell.value = { rowIndex, columnName }
}

function handleCellEditConfirm(rowIndex: number, columnName: string, newValue: string) {
  if (!editingCell.value) return

  const rows = props.tab.result?.rows ?? []
  const row = rows[rowIndex]
  if (!row) { editingCell.value = null; return }

  const oldValue = row[columnName]
  const oldStr = oldValue === null || oldValue === undefined ? '' : String(oldValue)
  const key = dirtyKey(rowIndex, columnName)

  if (newValue === oldStr) {
    delete dirtyMap[key]
    editingCell.value = null
    return
  }

  const pks: Record<string, unknown> = {}
  for (const pkCol of pkColumns.value) {
    pks[pkCol] = row[pkCol]
  }

  dirtyMap[key] = {
    rowIndex,
    columnName,
    oldValue,
    newValue,
    pkValues: pks,
  }

  editingCell.value = null
}

function handleCellEditCancel() {
  editingCell.value = null
}

// ── Apply / Discard ──

async function handleApply() {
  if (applying.value) return
  applying.value = true

  const tab = props.tab
  const entries = Object.entries(dirtyMap)
  let successCount = 0
  let failCount = 0

  for (const [key, change] of entries) {
    try {
      const params = new models.UpdateRowParams({
        connectionId: tab.connectionId!,
        database: tab.database!,
        table: tab.tableName!,
        pkValues: change.pkValues,
        column: change.columnName,
        newValue: change.newValue,
      })
      const result = await QueryService.UpdateRow(params)
      if (result && result.affectedRows > 0) {
        delete dirtyMap[key]
        successCount++
        // 审计 SQL — 同时显示 toast 和写入消息面板
        const auditSQL = result.sql || `UPDATE ${tab.tableName} SET ${change.columnName} = '${change.newValue}' WHERE ...`
        ElMessage({ message: auditSQL, type: 'success', duration: 3000 })
        store.addMessage(tab.id, `✅ ${auditSQL}`)
      } else {
        failCount++
      }
    } catch (err: unknown) {
      failCount++
      ElMessage({ message: `更新 ${change.columnName} 失败: ${parseError(err)}`, type: 'error' })
      store.addMessage(tab.id, `更新 ${change.columnName} 失败: ${parseError(err)}`)
    }
  }

  applying.value = false

  if (successCount > 0) {
    const summary = `成功更新 ${successCount} 项${failCount > 0 ? `，${failCount} 项失败` : ''}`
    ElMessage({ message: summary, type: 'success' })
    store.addMessage(tab.id, summary)
    await loadData()
  }
}

onMounted(() => loadData())
</script>

<template>
  <div class="table-view">
    <div class="table-toolbar">
      <div class="table-info">
        <span class="table-name">{{ tab.tableName }}</span>
        <span class="table-meta">{{ tab.database }} / {{ tab.tableName }}</span>
      </div>
      <div class="table-actions">
        <button class="tb-btn" title="导出数据" @click="exportVisible = true">⬇ 导出</button>
        <button class="tb-btn" title="刷新" :disabled="loading" @click="loadData()">↻ 刷新</button>
      </div>
    </div>
    <TableSearch
      :columns="(tab.result?.columns ?? []).map(c => c.name)"
      :loading="loading"
      @search="handleTableSearch"
      @reset="handleTableSearchReset"
    />
    <div class="table-content">
      <Transition name="fade">
        <div v-if="loading" class="loading-overlay">
          <div class="loading-spinner" />
          <span class="loading-text">加载中...</span>
        </div>
      </Transition>
      <QueryResult
        :columns="tab.result?.columns ?? []"
        :rows="tab.result?.rows ?? []"
        :message="tab.result?.message"
        :duration="tab.result?.duration"
        :paginated="true"
        :page="tab.page ?? 1"
        :page-size="tab.pageSize ?? 100"
        :total="tab.totalRows ?? 0"
        :total-pages="tab.totalPages ?? 1"
        :sort-column="tab.sortColumn ?? ''"
        :sort-order="tab.sortOrder ?? SortOrder.SortASC"
        :filters="tab.filters ?? []"
        :loading="loading"
        :editable="true"
        :editing-cell="editingCell"
        :editing-value="editingValue"
        @sort-change="handleSortChange"
        @filter-change="handleFilterChange"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
        @cell-dblclick="handleCellDblClick"
        @cell-edit-confirm="handleCellEditConfirm"
        @cell-edit-cancel="handleCellEditCancel"
        @cell-edit-update:value="editingValue = $event"
      />
    </div>

    <!-- Dirty changes bar -->
    <Transition name="slide-up">
      <div v-if="dirtyCount > 0" class="dirty-bar">
        <span class="dirty-info">{{ dirtyCount }} 项修改待提交</span>
        <div class="dirty-actions">
          <button class="bar-btn bar-btn--discard" :disabled="applying" @click="handleApply">放弃</button>
          <button class="bar-btn bar-btn--apply" :disabled="applying" @click="handleApply">
            {{ applying ? '提交中...' : '应用' }}
          </button>
        </div>
      </div>
    </Transition>

    <DataExport
      :visible="exportVisible"
      :table-name="tab.tableName ?? ''"
      :columns="tab.result?.columns ?? []"
      :rows="tab.result?.rows ?? []"
      :all-row-count="tab.totalRows ?? 0"
      :current-page="tab.page ?? 1"
      :page-size="tab.pageSize ?? 100"
      @close="exportVisible = false"
    />
  </div>
</template>

<style scoped>
.table-view {
  display: flex;
  flex-direction: column;
  flex: 1;
  height: 100%;
  min-height: 0;
  background: var(--color-bg, #fff);
}

.table-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  border-bottom: 1px solid var(--color-border, #d9d9dc);
  background: var(--color-surface, #fff);
  flex-shrink: 0;
}

.table-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.table-name {
  font-size: 13px;
  font-weight: 600;
  color: var(--color-text, #1a1a2e);
}

.table-meta {
  font-size: 11px;
  color: var(--color-text-secondary, #6e6e80);
}

.table-actions {
  display: flex;
  gap: 6px;
}

.tb-btn {
  font-size: 12px;
  padding: 4px 10px;
  border: 1px solid var(--color-border, #d9d9dc);
  border-radius: var(--radius-sm, 4px);
  background: var(--color-surface, #fff);
  cursor: pointer;
  color: var(--color-text, #1a1a2e);
  transition: background 0.1s;
}

.tb-btn:hover:not(:disabled) {
  background: var(--color-hover, rgba(0,0,0,0.04));
}

.tb-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.table-content {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
  position: relative;
}

/* ── Loading overlay ── */
.loading-overlay {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 12px;
  background: var(--color-overlay);
}

.loading-spinner {
  width: 32px;
  height: 32px;
  border: 3px solid var(--color-border, #d9d9dc);
  border-top-color: var(--color-accent, #6366f1);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.loading-text {
  font-size: 13px;
  color: var(--color-text-secondary, #6e6e80);
}

.fade-enter-active,
.fade-leave-active {
  transition: opacity 0.2s ease;
}
.fade-enter-from,
.fade-leave-to {
  opacity: 0;
}

/* Ensure el-table inside table-content gets a scrolling body */
.table-content :deep(.el-table) {
  height: 100%;
}

.table-content :deep(.el-table__body-wrapper) {
  overflow-y: auto;
  flex: 1;
}

/* ── Dirty changes bar ── */
.dirty-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  background: var(--color-accent, #6366f1);
  color: #fff;
  font-size: 12px;
  flex-shrink: 0;
}

.dirty-info {
  font-weight: 500;
}

.dirty-actions {
  display: flex;
  gap: 8px;
}

.bar-btn {
  font-size: 12px;
  padding: 3px 14px;
  border: 1px solid rgba(255,255,255,0.3);
  border-radius: var(--radius-sm, 4px);
  cursor: pointer;
  transition: background 0.15s;
  color: #fff;
  background: transparent;
}

.bar-btn:hover:not(:disabled) {
  background: rgba(255,255,255,0.15);
}

.bar-btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.bar-btn--apply {
  background: #fff;
  color: var(--color-accent, #6366f1);
  border-color: #fff;
}

.bar-btn--apply:hover:not(:disabled) {
  background: rgba(255,255,255,0.9);
}

/* ── Slide-up transition ── */
.slide-up-enter-active,
.slide-up-leave-active {
  transition: all 0.2s ease;
}
.slide-up-enter-from,
.slide-up-leave-to {
  opacity: 0;
  transform: translateY(100%);
}
</style>
