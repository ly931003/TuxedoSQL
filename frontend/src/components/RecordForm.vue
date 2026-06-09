<script setup lang="ts">
import type { ColumnInfo, DirtyChange } from '../types/query'
import type { TableSchema } from '../../bindings/tuxedosql/internal/model/models'
import { formatCellValue } from '../lib/timeFormat'

const props = defineProps<{
  columns: ColumnInfo[]
  row: Record<string, unknown> | null
  rowIndex: number
  totalInPage: number
  pkColumns: string[]
  schemas: TableSchema[]
  dirtyMap: Record<string, DirtyChange>
  editingField: string | null
  editingValue: string
}>()

const emit = defineEmits<{
  'field-dblclick': [fieldName: string]
  'field-edit-confirm': [fieldName: string, newValue: string]
  'field-edit-cancel': []
  'field-edit-update:value': [value: string]
  'prev-row': []
  'next-row': []
}>()

type FieldState = 'default' | 'readonly' | 'dirty' | 'editing'

function getFieldState(colName: string): FieldState {
  if (props.editingField === colName) return 'editing'
  const key = `${props.rowIndex}:${colName}`
  if (props.dirtyMap[key]) return 'dirty'
  if (props.pkColumns.includes(colName)) return 'readonly'
  return 'default'
}

function getDirtyValue(colName: string): unknown | undefined {
  const key = `${props.rowIndex}:${colName}`
  return props.dirtyMap[key]?.newValue
}

function getDisplayValue(colName: string): string {
  // Prefer dirty value for display
  const dirty = getDirtyValue(colName)
  if (dirty !== undefined) return String(dirty)

  const raw = props.row?.[colName]
  if (raw === null || raw === undefined) return ''
  const col = props.columns.find(c => c.name === colName)
  return formatCellValue(col?.type ?? '', raw)
}

function isNull(colName: string): boolean {
  const dirty = getDirtyValue(colName)
  if (dirty !== undefined) return dirty === null
  return props.row?.[colName] === null || props.row?.[colName] === undefined
}

function getSchema(colName: string): TableSchema | undefined {
  return props.schemas.find(s => s.name === colName)
}

function getTypeLabel(colName: string): string {
  const sc = getSchema(colName)
  return sc?.dataType ?? ''
}

function getConstraintBadge(colName: string): string {
  const sc = getSchema(colName)
  if (!sc) return ''
  const parts: string[] = []
  if (sc.columnKey === 'PRI') parts.push('PK')
  else if (sc.columnKey === 'UNI') parts.push('UNIQUE')
  else if (sc.columnKey === 'MUL') parts.push('INDEX')
  if (!sc.isNullable) parts.push('NOT NULL')
  return parts.join(' · ')
}

function isPK(colName: string): boolean {
  return props.pkColumns.includes(colName)
}

function handleFieldDblClick(colName: string) {
  if (isPK(colName)) return // PK fields are read-only in form view
  emit('field-dblclick', colName)
}
</script>

<template>
  <div class="record-form">
    <!-- Navigation bar (top) -->
    <div class="form-nav">
      <div class="nav-controls">
        <button class="nav-btn" title="上一行" @click="emit('prev-row')">◀</button>
        <span class="nav-info">{{ rowIndex + 1 }} / {{ totalInPage }}</span>
        <button class="nav-btn" title="下一行" @click="emit('next-row')">▶</button>
      </div>
      <span class="nav-label">表单视图</span>
    </div>

    <!-- Fields -->
    <div class="form-body">
      <template v-if="row">
        <div
          v-for="col in columns"
          :key="col.name"
          class="form-field"
          :class="{
            'is-pk': getFieldState(col.name) === 'readonly',
            'is-dirty': getFieldState(col.name) === 'dirty',
            'is-editing': getFieldState(col.name) === 'editing',
          }"
          @dblclick="handleFieldDblClick(col.name)"
        >
          <div class="field-label">
            <span class="field-name">
              <span v-if="isPK(col.name)" class="pk-icon">🔑</span>
              {{ col.name }}
            </span>
            <span class="field-type">{{ getTypeLabel(col.name) }}</span>
            <span v-if="getConstraintBadge(col.name)" class="field-constraint">
              {{ getConstraintBadge(col.name) }}
            </span>
          </div>
          <div class="field-value">
            <!-- Editing mode -->
            <input
              v-if="getFieldState(col.name) === 'editing'"
              class="field-input"
              :value="editingValue"
              @input="emit('field-edit-update:value', ($event.target as HTMLInputElement).value)"
              @keydown.enter.stop="emit('field-edit-confirm', col.name, editingValue)"
              @keydown.escape.stop="emit('field-edit-cancel')"
              @blur="emit('field-edit-confirm', col.name, editingValue)"
            />
            <!-- NULL display -->
            <span v-else-if="isNull(col.name)" class="null-value">(NULL)</span>
            <!-- Normal display -->
            <span v-else class="value-text">{{ getDisplayValue(col.name) }}</span>
          </div>
        </div>
      </template>
      <div v-else class="form-empty">
        <span>无数据</span>
      </div>
    </div>

    <!-- Navigation bar (bottom) -->
    <div class="form-nav form-nav--bottom">
      <div class="nav-controls">
        <button class="nav-btn" title="上一行" @click="emit('prev-row')">◀</button>
        <span class="nav-info">{{ rowIndex + 1 }} / {{ totalInPage }}</span>
        <button class="nav-btn" title="下一行" @click="emit('next-row')">▶</button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.record-form {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  background: var(--color-bg, #fff);
}

/* ── Navigation bar ── */
.form-nav {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  border-bottom: 1px solid var(--color-border, #d9d9dc);
  background: var(--color-surface, #fff);
  flex-shrink: 0;
}

.form-nav--bottom {
  border-bottom: none;
  border-top: 1px solid var(--color-border, #d9d9dc);
}

.nav-controls {
  display: flex;
  align-items: center;
  gap: 8px;
}

.nav-btn {
  width: 28px;
  height: 28px;
  border: 1px solid var(--color-border, #d9d9dc);
  border-radius: var(--radius-sm, 4px);
  background: var(--color-surface, #fff);
  cursor: pointer;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.1s;
  color: var(--color-text, #1a1a2e);
}

.nav-btn:hover {
  background: var(--color-hover, rgba(0, 0, 0, 0.04));
}

.nav-info {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text, #1a1a2e);
  min-width: 60px;
  text-align: center;
}

.nav-label {
  font-size: 11px;
  color: var(--color-text-secondary, #6e6e80);
}

/* ── Form body ── */
.form-body {
  flex: 1;
  overflow-y: auto;
  padding: 8px 0;
}

.form-empty {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px;
  color: var(--color-text-secondary, #6e6e80);
  font-size: 14px;
}

/* ── Field row ── */
.form-field {
  display: flex;
  border-bottom: 1px solid var(--color-border, #d9d9dc);
  min-height: 36px;
  transition: background 0.1s;
}

.form-field:last-child {
  border-bottom: none;
}

.form-field:hover {
  background: var(--color-hover, rgba(0, 0, 0, 0.015));
}

/* PK fields: muted background, non-editable */
.form-field.is-pk {
  background: var(--color-sidebar, #f5f5f7);
}

.form-field.is-pk:hover {
  background: var(--color-hover, rgba(0, 0, 0, 0.03));
}

/* Dirty fields: left border accent */
.form-field.is-dirty {
  border-left: 3px solid var(--color-accent, #6366f1);
}

/* Editing field: stronger border */
.form-field.is-editing {
  border-left: 3px solid var(--color-accent, #6366f1);
  background: var(--color-selected, rgba(99, 102, 241, 0.04));
}

/* ── Field label (left column) ── */
.field-label {
  width: 220px;
  min-width: 160px;
  padding: 8px 12px;
  display: flex;
  flex-direction: column;
  gap: 2px;
  user-select: none;
}

.field-name {
  font-size: 13px;
  font-weight: 500;
  color: var(--color-text, #1a1a2e);
  display: flex;
  align-items: center;
  gap: 4px;
}

.pk-icon {
  font-size: 11px;
}

.field-type {
  font-size: 10px;
  color: var(--color-text-secondary, #6e6e80);
  font-family: 'SF Mono', 'Fira Code', 'Cascadia Code', monospace;
}

.field-constraint {
  font-size: 10px;
  color: var(--color-accent, #6366f1);
  font-weight: 500;
}

/* ── Field value (right column) ── */
.field-value {
  flex: 1;
  padding: 8px 12px;
  display: flex;
  align-items: center;
  min-width: 0;
}

.value-text {
  font-size: 13px;
  color: var(--color-text, #1a1a2e);
  word-break: break-all;
  white-space: pre-wrap;
  line-height: 1.5;
}

.null-value {
  font-size: 13px;
  color: var(--color-result-null, #9ca3af);
  font-style: italic;
}

/* ── Inline edit input ── */
.field-input {
  width: 100%;
  border: 2px solid var(--color-accent, #6366f1);
  border-radius: var(--radius-sm, 4px);
  padding: 4px 8px;
  font-size: 13px;
  font-family: inherit;
  background: var(--color-input-bg, #fff);
  color: var(--color-text, #1a1a2e);
  outline: none;
  box-sizing: border-box;
}

.field-input:focus {
  border-color: var(--color-accent, #6366f1);
}
</style>
