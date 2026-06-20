<script setup lang="ts">
import { ref, computed } from 'vue'
import { ElMessage } from 'element-plus'
import { QueryService } from '../../bindings/tuxedosql/internal/service'
import * as models from '../../bindings/tuxedosql/internal/model/models'
import type { ColumnInfo, FilterGroup } from '../types/query'

const props = defineProps<{
  visible: boolean
  tableName: string
  columns: ColumnInfo[]
  rows: Record<string, unknown>[]
  allRowCount: number
  currentPage: number
  pageSize: number
  connectionId?: string
  database?: string
  filters?: FilterGroup
}>()

const emit = defineEmits<{
  close: []
}>()

type ExportFormat = 'csv' | 'sql' | 'json'
type ExportRange = 'current' | 'all'

const format = ref<ExportFormat>('csv')
const range = ref<ExportRange>('current')
const fetching = ref(false)

const columnNames = computed(() => props.columns.map((c) => c.name))

// ── CSV generation ──

function escapeCSV(value: unknown): string {
  if (value === null || value === undefined) return ''
  let str = String(value)
  // If contains comma, quote, or newline — wrap in quotes and escape inner quotes
  if (str.includes(',') || str.includes('"') || str.includes('\n') || str.includes('\r')) {
    str = '"' + str.replace(/"/g, '""') + '"'
  }
  return str
}

function generateCSV(rows: Record<string, unknown>[]): string {
  // BOM for Excel Chinese compatibility
  const BOM = '﻿'
  const header = columnNames.value.map((c) => escapeCSV(c)).join(',')
  const lines = rows.map((row) => columnNames.value.map((c) => escapeCSV(row[c])).join(','))
  return BOM + [header, ...lines].join('\n')
}

// ── SQL INSERT generation ──

function sqlValue(val: unknown): string {
  if (val === null || val === undefined) return 'NULL'
  if (typeof val === 'number') return String(val)
  // Escape single quotes and backslashes
  return "'" + String(val).replace(/\\/g, '\\\\').replace(/'/g, "\\'") + "'"
}

function generateSQL(rows: Record<string, unknown>[]): string {
  const cols = columnNames.value.map((c) => '`' + c + '`').join(', ')
  const lines = rows.map((row) => {
    const vals = columnNames.value.map((c) => sqlValue(row[c])).join(', ')
    return `INSERT INTO \`${props.tableName}\` (${cols}) VALUES (${vals});`
  })
  return lines.join('\n') + '\n'
}
// ── JSON export ──

function generateJSON(rows: Record<string, unknown>[]): string {
  return JSON.stringify(rows, null, 2)
}


// ── Fetch all rows via paginated backend calls ──

async function fetchAllRows(): Promise<Record<string, unknown>[]> {
  if (!props.connectionId || !props.database) {
    // Fallback: can't reach backend, use current page
    return [...props.rows]
  }

  const FETCH_SIZE = 2000
  const MAX_TOTAL = 10000
  const allRows: Record<string, unknown>[] = []
  let page = 1

  while (allRows.length < MAX_TOTAL) {
    try {
      const params = new models.TableDataParams({
        connectionId: props.connectionId,
        database: props.database,
        table: props.tableName,
        page,
        pageSize: FETCH_SIZE,
        sortColumn: '',
        sortOrder: models.SortOrder.SortASC,
        filters: props.filters ? new models.FilterGroup(props.filters) : null,
      })
      const result = await QueryService.GetTableData(params)
      if (!result || result.rows.length === 0) break

      allRows.push(...result.rows)
      if (result.rows.length < FETCH_SIZE || allRows.length >= result.total) break
      page++
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : String(err)
      ElMessage({ message: `获取全部数据失败: ${msg}`, type: 'error' })
      return [...props.rows]
    }
  }

  if (allRows.length >= MAX_TOTAL) {
    ElMessage({ message: `已达导出上限 ${MAX_TOTAL} 行`, type: 'warning', duration: 3000 })
  }
  return allRows
}

// ── Download ──

async function handleExport() {
  let rows: Record<string, unknown>[]

  if (range.value === 'all') {
    fetching.value = true
    rows = await fetchAllRows()
    fetching.value = false
  } else {
    rows = props.rows
  }

  let content: string
  let mime: string
  let ext: string

  if (format.value === 'csv') {
    content = generateCSV(rows)
    mime = 'text/csv;charset=utf-8'
    ext = 'csv'
  } else if (format.value === 'sql') {
    content = generateSQL(rows)
    mime = 'text/plain;charset=utf-8'
    ext = 'sql'
  } else {
    content = generateJSON(rows)
    mime = 'application/json;charset=utf-8'
    ext = 'json'
  }

  const blob = new Blob([content], { type: mime })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${props.tableName}_${range.value}.${ext}`
  document.body.appendChild(a)
  a.click()
  document.body.removeChild(a)
  URL.revokeObjectURL(url)

  emit('close')
}

const rangeInfo = computed(() => {
  if (range.value === 'current') {
    return `当前页 ${props.rows.length} 行`
  }
  return `全部 ${props.allRowCount} 行`
})
</script>

<template>
  <el-dialog
    :model-value="visible"
    title="导出数据"
    width="420px"
    :close-on-click-modal="true"
    @close="emit('close')"
  >
    <div class="export-body">
      <div class="export-field">
        <label>导出格式</label>
        <div class="radio-group">
          <label class="radio-item">
            <input v-model="format" type="radio" value="csv" />
            <span>CSV</span>
          </label>
          <label class="radio-item">
            <input v-model="format" type="radio" value="sql" />
            <span>SQL INSERT</span>
          </label>
          <label class="radio-item">
            <input v-model="format" type="radio" value="json" />
            <span>JSON</span>
          </label>
        </div>
      </div>

      <div class="export-field">
        <label>导出范围</label>
        <div class="radio-group">
          <label class="radio-item">
            <input v-model="range" type="radio" value="current" />
            <span>当前页</span>
          </label>
          <label class="radio-item">
            <input v-model="range" type="radio" value="all" :disabled="allRowCount <= 0" />
            <span>全部</span>
          </label>
        </div>
        <p class="field-hint">{{ rangeInfo }}</p>
      </div>
    </div>

    <template #footer>
      <button class="export-btn export-btn--cancel" @click="emit('close')">取消</button>
      <button class="export-btn export-btn--confirm" :disabled="fetching" @click="handleExport">
        {{ fetching ? '获取中...' : '导出' }}
      </button>
    </template>
  </el-dialog>
</template>

<style scoped>
.export-body {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.export-field label {
  display: block;
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text, #1a1a2e);
  margin-bottom: 6px;
}

.radio-group {
  display: flex;
  gap: 16px;
}

.radio-item {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  cursor: pointer;
}

.radio-item input[disabled] + span {
  opacity: 0.4;
  cursor: not-allowed;
}

.field-hint {
  font-size: 11px;
  color: var(--color-text-secondary, #6e6e80);
  margin: 4px 0 0;
}

.export-btn {
  font-size: 13px;
  padding: 6px 16px;
  border: none;
  border-radius: var(--radius-sm, 4px);
  cursor: pointer;
}

.export-btn--cancel {
  background: var(--color-hover, rgba(0, 0, 0, 0.04));
  color: var(--color-text, #1a1a2e);
  margin-right: 8px;
}

.export-btn--confirm {
  background: var(--color-accent, #6366f1);
  color: #fff;
}
</style>
