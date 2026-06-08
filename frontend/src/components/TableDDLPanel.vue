<script setup lang="ts">
import { ref, onMounted, watch } from 'vue'
import { QueryService } from '../../bindings/tuxedosql/internal/service'

const props = defineProps<{
  connectionId: string
  database: string
  tableName: string
}>()

const ddl = ref('')
const loading = ref(false)
const error = ref('')
const copied = ref(false)

async function loadDDL() {
  if (!props.connectionId || !props.database || !props.tableName) return
  loading.value = true
  error.value = ''
  try {
    ddl.value = await QueryService.GetCreateTable(props.connectionId, props.database, props.tableName)
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

async function copyDDL() {
  try {
    await navigator.clipboard.writeText(ddl.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  } catch { /* fallback not available */ }
}

onMounted(loadDDL)
watch(() => [props.connectionId, props.database, props.tableName], loadDDL)
</script>

<template>
  <div class="table-ddl-panel">
    <div v-if="loading" class="ddl-loading">加载中...</div>
    <div v-else-if="error" class="ddl-error">{{ error }}</div>
    <div v-else-if="!ddl" class="ddl-empty">暂无建表语句</div>
    <template v-else>
      <div class="ddl-toolbar">
        <button class="ddl-copy-btn" @click="copyDDL">
          {{ copied ? '✓ 已复制' : '复制' }}
        </button>
      </div>
      <pre class="ddl-content">{{ ddl }}</pre>
    </template>
  </div>
</template>

<style scoped>
.table-ddl-panel {
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.ddl-toolbar {
  flex-shrink: 0;
  padding: 4px 8px;
  display: flex;
  justify-content: flex-end;
}

.ddl-copy-btn {
  font-size: 11px;
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  border-radius: 3px;
  background: var(--color-hover);
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: background 0.15s;
}

.ddl-copy-btn:hover {
  background: var(--color-accent);
  color: #fff;
}

.ddl-loading,
.ddl-empty {
  font-size: 12px;
  color: var(--color-text-secondary);
  text-align: center;
  padding: 16px;
}

.ddl-error {
  font-size: 12px;
  color: #e74c3c;
  text-align: center;
  padding: 16px;
}

.ddl-content {
  flex: 1;
  overflow-y: auto;
  font-family: var(--font-mono, monospace);
  font-size: 11px;
  line-height: 1.6;
  padding: 8px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  color: var(--color-text);
}
</style>