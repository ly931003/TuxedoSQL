<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { QueryService } from '../../bindings/tuxedosql/internal/service'
import type { TableSchema } from '../../bindings/tuxedosql/internal/model/models'

const props = defineProps<{
  connectionId: string
  database: string
  tableName: string
}>()

const schemas = ref<TableSchema[]>([])
const loading = ref(false)
const error = ref('')

async function loadSchema() {
  if (!props.connectionId || !props.database || !props.tableName) return
  loading.value = true
  error.value = ''
  try {
    schemas.value = await QueryService.GetTableSchema(props.connectionId, props.database, props.tableName)
  } catch (err: unknown) {
    error.value = parseError(err)
  } finally {
    loading.value = false
  }
}

function parseError(err: unknown): string {
  if (err instanceof Error) {
    try { const p = JSON.parse(err.message); if (p?.message) return String(p.message) } catch {}
    return err.message
  }
  return String(err)
}

function keyLabel(key: string): string {
  if (key === 'PRI') return '🔑 主键'
  if (key === 'UNI') return '唯一'
  if (key === 'MUL') return '索引'
  return ''
}

onMounted(loadSchema)
watch(() => [props.connectionId, props.database, props.tableName], loadSchema)
</script>

<template>
  <div class="table-info-panel">
    <div v-if="loading" class="info-loading">加载中...</div>
    <div v-else-if="error" class="info-error">{{ error }}</div>
    <div v-else-if="schemas.length === 0" class="info-empty">暂无列信息</div>
    <table v-else class="schema-table">
      <thead>
        <tr>
          <th>列名</th>
          <th>类型</th>
          <th>可空</th>
          <th>键</th>
          <th>默认值</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="col in schemas" :key="col.name">
          <td class="col-name">{{ col.name }}</td>
          <td class="col-type">{{ col.dataType }}</td>
          <td class="col-nullable">{{ col.isNullable ? '✓' : '✗' }}</td>
          <td class="col-key">{{ keyLabel(col.columnKey) }}</td>
          <td class="col-default">{{ col.defaultValue || '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<style scoped>
.table-info-panel {
  height: 100%;
  overflow-y: auto;
  padding: 0 8px 8px;
}

.info-loading,
.info-empty {
  font-size: 12px;
  color: var(--color-text-secondary);
  text-align: center;
  padding: 16px;
}

.info-error {
  font-size: 12px;
  color: #e74c3c;
  text-align: center;
  padding: 16px;
}

.schema-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 11px;
}

.schema-table th {
  font-weight: 500;
  color: var(--color-text-secondary);
  padding: 4px 6px;
  border-bottom: 1px solid var(--color-border);
  text-align: left;
  white-space: nowrap;
}

.schema-table td {
  padding: 3px 6px;
  border-bottom: 1px solid var(--color-border);
  vertical-align: top;
}

.col-name {
  font-weight: 500;
  color: var(--color-text);
}

.col-type {
  color: var(--color-text-secondary);
  font-family: var(--font-mono, monospace);
  font-size: 10px;
}

.col-nullable {
  text-align: center;
}

.col-key {
  font-size: 10px;
}

.col-default {
  font-family: var(--font-mono, monospace);
  font-size: 10px;
  color: var(--color-text-secondary);
}
</style>