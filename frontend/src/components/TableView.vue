<script setup lang="ts">
import { ref, reactive, computed, onMounted } from 'vue'
import { ElMessage } from 'element-plus'
import { useQueryStore } from '../stores/query'
import { SortOrder } from '../types/query'
import { QueryService } from '../../bindings/tuxedosql/internal/service'
import * as models from '../../bindings/tuxedosql/internal/model/models'
import QueryResult from './QueryResult.vue'
import RecordForm from './RecordForm.vue'
import DataExport from './DataExport.vue'
import TableSearch from '../features/table/TableSearch.vue'
import { formatCellValue } from '../lib/timeFormat'
import { parseError } from '../composables/parseError'
import type { QueryTab, PageResult, FilterGroup, DirtyChange, EditingCell } from '../types/query'
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
const currentRowDirtyValues = computed<Record<string, unknown>>(() => {
  const prefix = `${formRowIndex.value}:`
  const result: Record<string, unknown> = {}
  for (const [key, change] of Object.entries(dirtyMap)) {
    if (key.startsWith(prefix)) {
      result[change.columnName] = change.newValue
    }
  }
  return result
})
const applying = ref(false)
const pkColumns = ref<string[]>([])
const schemaLoaded = ref(false)
const searchFilterGroup = ref<FilterGroup | null>(null)
const viewMode = ref<'table' | 'form'>('table')
const formRowIndex = ref(0)
const selectedRowIndex = ref(0)
const schemas = ref<TableSchema[]>([])

// effectiveFilters: either the search filter group, or the tab's persistent group
const effectiveFilters = computed<FilterGroup | undefined | null>(() => {
  return searchFilterGroup.value ?? props.tab.filters ?? undefined
})

function dirtyKey(rowIndex: number, columnName: string): string {
  return `${rowIndex}:${columnName}`
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
    const schemasRaw: TableSchema[] = await QueryService.GetTableSchema(
      tab.connectionId,
      tab.database,
      tab.tableName,
    )
    schemas.value = schemasRaw
    pkColumns.value = schemasRaw.filter((s) => s.columnKey === 'PRI').map((s) => s.name)
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
  filters?: FilterGroup | null | undefined
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
  const filters = overrides?.filters ?? effectiveFilters.value

  try {
    const params = new models.TableDataParams({
      connectionId: tab.connectionId,
      database: tab.database,
      table: tab.tableName,
      page,
      pageSize,
      sortColumn,
      sortOrder: sortOrder as models.SortOrder,
      filters: filters ? new models.FilterGroup(filters) : null,
    })
    const result = await QueryService.GetTableData(params)
    if (result) {
      store.setResult(tab.id, buildResultFromPage(result))
      store.setTablePageResult(tab.id, result.total, result.totalPages)
      // Update bottom bar with the actual executed SQL
      if (result.sql) {
        store.updateLastExecutedSQL(tab.id, result.sql)
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

function handleTableSearch(group: FilterGroup | null) {
  searchFilterGroup.value = group
  loadData({ page: 1 })
}

function handleTableSearchReset() {
  searchFilterGroup.value = null
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

async function handleFilterChange(filterList: FilterGroup | null | undefined) {
  store.setTableFilters(props.tab.id, filterList)
  await discardIfDirty()
  loadData({ page: 1, filters: filterList })
}

async function handlePageChange(page: number) {
  store.setTablePage(props.tab.id, page)
  await discardIfDirty()
  formRowIndex.value = 0
  selectedRowIndex.value = 0
  loadData({ page })
}

async function handlePageSizeChange(pageSize: number) {
  store.setTablePageSize(props.tab.id, pageSize)
  await discardIfDirty()
  formRowIndex.value = 0
  selectedRowIndex.value = 0
  loadData({ page: 1, pageSize })
}

// ── Form row navigation ──

function handleFormPrevRow() {
  if (formRowIndex.value > 0) {
    formRowIndex.value--
    selectedRowIndex.value = formRowIndex.value
  }
}

function handleFormNextRow() {
  const total = props.tab.result?.rows?.length ?? 0
  if (formRowIndex.value < total - 1) {
    formRowIndex.value++
    selectedRowIndex.value = formRowIndex.value
  }
}

// ── Row selection ──

function handleRowSelect(index: number) {
  selectedRowIndex.value = index
}

function handleViewModeSwitch(mode: 'table' | 'form') {
  if (mode === 'form') {
    // Sync selected row → form row when switching to form view
    formRowIndex.value = selectedRowIndex.value
  } else {
    // Sync form row → selected row when switching back to table view
    selectedRowIndex.value = formRowIndex.value
  }
  viewMode.value = mode
}

// ── Cell editing handlers ──

function handleCellDblClick(rowIndex: number, columnName: string) {
  const rows = props.tab.result?.rows ?? []
  const row = rows[rowIndex]
  if (!row) return

  const existingKey = dirtyKey(rowIndex, columnName)
  const existingDirty = dirtyMap[existingKey]
  const currentValue = existingDirty ? existingDirty.newValue : row[columnName]
  const col = props.tab.result?.columns?.find((c) => c.name === columnName)
  const colType = col?.type ?? ''

  editingValue.value =
    currentValue === null || currentValue === undefined
      ? ''
      : formatCellValue(colType, currentValue)
  editingCell.value = { rowIndex, columnName }
}

function handleCellEditConfirm(rowIndex: number, columnName: string, newValue: string) {
  if (!editingCell.value) return

  const rows = props.tab.result?.rows ?? []
  const row = rows[rowIndex]
  if (!row) {
    editingCell.value = null
    return
  }

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

// ── Form field editing ──

function handleFormFieldDblClick(fieldName: string) {
  const row = props.tab.result?.rows?.[formRowIndex.value]
  if (!row) return

  const existingKey = dirtyKey(formRowIndex.value, fieldName)
  const existingDirty = dirtyMap[existingKey]
  const currentValue = existingDirty ? existingDirty.newValue : row[fieldName]
  const col = props.tab.result?.columns?.find((c) => c.name === fieldName)
  const colType = col?.type ?? ''

  editingValue.value =
    currentValue === null || currentValue === undefined
      ? ''
      : formatCellValue(colType, currentValue)
  editingCell.value = { rowIndex: formRowIndex.value, columnName: fieldName }
}

// ── Apply / Discard ──

function handleDiscard() {
  if (dirtyCount.value === 0) return
  for (const key of Object.keys(dirtyMap)) {
    delete dirtyMap[key]
  }
}

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
        const auditSQL =
          result.sql ||
          `UPDATE ${tab.tableName} SET ${change.columnName} = '${change.newValue}' WHERE ...`
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
        <div class="view-toggle">
          <button
            class="toggle-btn"
            :class="{ active: viewMode === 'table' }"
            @click="handleViewModeSwitch('table')"
          >
            表格
          </button>
          <button
            class="toggle-btn"
            :class="{ active: viewMode === 'form' }"
            @click="handleViewModeSwitch('form')"
          >
            表单
          </button>
        </div>
        <button class="tb-btn" title="导出数据" @click="exportVisible = true">⬇ 导出</button>
        <button class="tb-btn" title="刷新" :disabled="loading" @click="loadData()">↻ 刷新</button>
      </div>
    </div>
    <TableSearch
      :columns="(tab.result?.columns ?? []).map((c) => c.name)"
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

      <!-- Table view -->
      <QueryResult
        v-if="viewMode === 'table'"
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
        :filters="effectiveFilters ?? undefined"
        :loading="loading"
        :editable="true"
        :editing-cell="editingCell"
        :editing-value="editingValue"
        :selected-row-index="selectedRowIndex"
        @sort-change="handleSortChange"
        @filter-change="handleFilterChange"
        @update:page="handlePageChange"
        @update:page-size="handlePageSizeChange"
        @cell-dblclick="handleCellDblClick"
        @cell-edit-confirm="handleCellEditConfirm"
        @cell-edit-cancel="handleCellEditCancel"
        @cell-edit-update:value="editingValue = $event"
        @row-select="handleRowSelect"
      />

      <!-- Form view -->
      <RecordForm
        v-else
        :columns="tab.result?.columns ?? []"
        :row="(tab.result?.rows ?? [])[formRowIndex] ?? null"
        :row-index="formRowIndex"
        :total-in-page="(tab.result?.rows ?? []).length"
        :pk-columns="pkColumns"
        :schemas="schemas"
        :dirty-map="dirtyMap"
        :editing-field="editingCell?.columnName ?? null"
        :editing-value="editingValue"
        @field-dblclick="handleFormFieldDblClick"
        @field-edit-confirm="
          (fieldName: string, newValue: string) =>
            handleCellEditConfirm(formRowIndex, fieldName, newValue)
        "
        @field-edit-cancel="handleCellEditCancel"
        @field-edit-update:value="editingValue = $event"
        @prev-row="handleFormPrevRow"
        @next-row="handleFormNextRow"
      />
    </div>

    <!-- Dirty changes bar -->
    <Transition name="slide-up">
      <div v-if="dirtyCount > 0" class="dirty-bar">
        <span class="dirty-info">{{ dirtyCount }} 项修改待提交</span>
        <div class="dirty-actions">
          <button class="bar-btn bar-btn--discard" :disabled="applying" @click="handleDiscard">
            放弃
          </button>
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
      :connection-id="tab.connectionId ?? ''"
      :database="tab.database ?? ''"
      :filters="effectiveFilters ?? undefined"
      @close="exportVisible = false"
    />
  </div>
</template>

<style scoped>
.table-view {
  display: flex;
  flex-direction: column;
  flex: 1;
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
  align-items: center;
  gap: 6px;
}

/* ── View toggle button group ── */

.view-toggle {
  display: flex;
  border: 1px solid var(--color-border, #d9d9dc);
  border-radius: var(--radius-sm, 4px);
  overflow: hidden;
}

.toggle-btn {
  font-size: 12px;
  padding: 4px 10px;
  border: none;
  background: var(--color-surface, #fff);
  cursor: pointer;
  color: var(--color-text-secondary, #6e6e80);
  transition:
    background 0.15s,
    color 0.15s;
  border-right: 1px solid var(--color-border, #d9d9dc);
}

.toggle-btn:last-child {
  border-right: none;
}

.toggle-btn.active {
  background: var(--color-accent, #6366f1);
  color: #fff;
}

.toggle-btn:hover:not(.active) {
  background: var(--color-hover, rgba(0, 0, 0, 0.04));
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
  background: var(--color-hover, rgba(0, 0, 0, 0.04));
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
  to {
    transform: rotate(360deg);
  }
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
  border: 1px solid rgba(255, 255, 255, 0.3);
  border-radius: var(--radius-sm, 4px);
  cursor: pointer;
  transition: background 0.15s;
  color: #fff;
  background: transparent;
}

.bar-btn:hover:not(:disabled) {
  background: rgba(255, 255, 255, 0.15);
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
  background: rgba(255, 255, 255, 0.9);
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
