<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { ElMessageBox } from 'element-plus'
import { QueryService } from '../../bindings/tuxedosql/internal/service'
import type { QueryHistoryEntry } from '../types/query'

const props = defineProps<{ visible: boolean }>()
const emit = defineEmits<{ close: []; pickSql: [sql: string] }>()

const entries = ref<QueryHistoryEntry[]>([])
const loading = ref(false)
const searchQuery = ref('')

const filteredEntries = computed(() => {
  if (!searchQuery.value.trim()) return entries.value
  const q = searchQuery.value.toLowerCase()
  return entries.value.filter(
    (e) =>
      e.sql.toLowerCase().includes(q) ||
      e.database.toLowerCase().includes(q),
  )
})

// Load history when panel becomes visible
watch(
  () => props.visible,
  (v) => {
    if (v) loadHistory()
  },
)

async function loadHistory() {
  loading.value = true
  try {
    entries.value = await QueryService.LoadHistory()
  } catch {
    entries.value = []
  } finally {
    loading.value = false
  }
}

async function handleClear() {
  try {
    await ElMessageBox.confirm('确定要清空所有查询历史吗？', '清空历史', {
      confirmButtonText: '确定',
      cancelButtonText: '取消',
      type: 'warning',
    })
    await QueryService.ClearHistory()
    entries.value = []
  } catch {
    /* user cancelled or error — ignore */
  }
}

function handlePick(entry: QueryHistoryEntry) {
  emit('pickSql', entry.sql)
  emit('close')
}

function formatTime(ts: number): string {
  if (!ts) return ''
  return new Date(ts).toLocaleString('zh-CN')
}

function formatDuration(ms: number): string {
  if (ms < 1000) return `${ms}ms`
  return `${(ms / 1000).toFixed(1)}s`
}

function truncateSQL(sql: string, maxLen = 80): string {
  const singleLine = sql.replace(/\s+/g, ' ').trim()
  if (singleLine.length <= maxLen) return singleLine
  return singleLine.slice(0, maxLen) + '…'
}
</script>

<template>
  <Teleport to="body">
    <div v-if="visible" class="history-overlay" @click.self="emit('close')">
      <div class="history-panel">
        <!-- Header -->
        <div class="panel-header">
          <h3 class="panel-title">查询历史</h3>
          <div class="header-actions">
            <button
              class="btn-clear"
              :disabled="entries.length === 0"
              title="清空历史"
              @click="handleClear"
            >
              清空
            </button>
            <button class="btn-close" title="关闭" @click="emit('close')">
              <svg
                width="14"
                height="14"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2"
                stroke-linecap="round"
              >
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </button>
          </div>
        </div>

        <!-- Search -->
        <div class="search-bar">
          <svg
            class="search-icon"
            width="14"
            height="14"
            viewBox="0 0 24 24"
            fill="none"
            stroke="currentColor"
            stroke-width="2"
            stroke-linecap="round"
          >
            <circle cx="11" cy="11" r="8" />
            <line x1="21" y1="21" x2="16.65" y2="16.65" />
          </svg>
          <input
            v-model="searchQuery"
            class="search-input"
            type="text"
            placeholder="搜索历史..."
          />
        </div>

        <!-- Content -->
        <div class="panel-body">
          <!-- Loading -->
          <div v-if="loading" class="state-message">加载中…</div>

          <!-- Empty -->
          <div v-else-if="filteredEntries.length === 0" class="state-message">
            {{ searchQuery ? '无匹配结果' : '暂无历史记录' }}
          </div>

          <!-- History list -->
          <div v-else class="history-list">
            <div
              v-for="entry in filteredEntries"
              :key="entry.id"
              class="history-item"
              @click="handlePick(entry)"
            >
              <div class="item-status">
                <svg
                  v-if="entry.success"
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                  class="status-icon success"
                >
                  <polyline points="20 6 9 17 4 12" />
                </svg>
                <svg
                  v-else
                  width="14"
                  height="14"
                  viewBox="0 0 24 24"
                  fill="none"
                  stroke="currentColor"
                  stroke-width="2.5"
                  stroke-linecap="round"
                  class="status-icon error"
                >
                  <line x1="18" y1="6" x2="6" y2="18" />
                  <line x1="6" y1="6" x2="18" y2="18" />
                </svg>
              </div>
              <div class="item-content">
                <div class="item-sql">{{ truncateSQL(entry.sql) }}</div>
                <div class="item-meta">
                  <span class="meta-db">{{ entry.database || '—' }}</span>
                  <span class="meta-time">{{ formatTime(entry.timestamp) }}</span>
                  <span v-if="entry.success" class="meta-duration">
                    {{ formatDuration(entry.duration) }}
                  </span>
                  <span v-if="entry.success && entry.rowCount > 0" class="meta-rows">
                    {{ entry.rowCount }} 行
                  </span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </Teleport>
</template>


<style scoped>
/* ── Overlay ── */
.history-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.3);
}

/* ── Panel ── */
.history-panel {
  width: 640px;
  max-width: 90vw;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  background: var(--color-surface, #fff);
  border-radius: 10px;
  box-shadow:
    0 8px 40px rgba(0, 0, 0, 0.15),
    0 2px 8px rgba(0, 0, 0, 0.08);
  overflow: hidden;
}

/* ── Header ── */
.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px 18px;
  border-bottom: 1px solid var(--color-border, #e0e0e3);
  flex-shrink: 0;
}

.panel-title {
  font-size: 14px;
  font-weight: 600;
  color: var(--color-text, #1a1a2e);
  margin: 0;
}

.header-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.btn-clear {
  font-size: 12px;
  padding: 4px 10px;
  border: 1px solid var(--color-border, #e0e0e3);
  border-radius: var(--radius-sm, 4px);
  background: transparent;
  color: var(--color-text-secondary, #666);
  cursor: pointer;
  transition:
    background 0.15s,
    color 0.15s;
  font-family: var(--font-sans);
}

.btn-clear:hover:not(:disabled) {
  background: var(--color-hover, rgba(0, 0, 0, 0.05));
  color: #c0392b;
}

.btn-clear:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}

.btn-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--radius-sm, 4px);
  background: transparent;
  color: var(--color-text-muted, #999);
  cursor: pointer;
  transition: background 0.15s;
}

.btn-close:hover {
  background: var(--color-hover, rgba(0, 0, 0, 0.05));
}

/* ── Search ── */
.search-bar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 18px;
  border-bottom: 1px solid var(--color-border, #e0e0e3);
  flex-shrink: 0;
}

.search-icon {
  color: var(--color-text-muted, #999);
  flex-shrink: 0;
}

.search-input {
  flex: 1;
  border: none;
  outline: none;
  font-size: 13px;
  color: var(--color-text, #333);
  background: transparent;
  font-family: var(--font-sans);
}

.search-input::placeholder {
  color: var(--color-text-muted, #bbb);
}

/* ── Body ── */
.panel-body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.state-message {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 48px 0;
  font-size: 13px;
  color: var(--color-text-muted, #999);
}

/* ── History list ── */
.history-list {
  padding: 4px 0;
}

.history-item {
  display: flex;
  align-items: flex-start;
  gap: 10px;
  padding: 10px 18px;
  cursor: pointer;
  transition: background 0.1s;
}

.history-item:hover {
  background: var(--color-hover, rgba(0, 0, 0, 0.03));
}

.item-status {
  padding-top: 2px;
  flex-shrink: 0;
}

.status-icon.success {
  color: #27ae60;
}

.status-icon.error {
  color: #e74c3c;
}

.item-content {
  flex: 1;
  min-width: 0;
}

.item-sql {
  font-size: 12.5px;
  font-family: var(--font-mono, 'Cascadia Code', 'Fira Code', 'JetBrains Mono', monospace);
  color: var(--color-text, #333);
  line-height: 1.5;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.item-meta {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-top: 4px;
  font-size: 11px;
  color: var(--color-text-muted, #999);
}

.meta-db {
  font-weight: 500;
  color: var(--color-accent, #6366f1);
}

.meta-duration {
  color: var(--color-text-secondary, #666);
}

.meta-rows {
  color: var(--color-text-secondary, #666);
}
</style>
