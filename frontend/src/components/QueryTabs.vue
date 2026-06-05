<script setup lang="ts">
import { ref, watch, onMounted, nextTick } from 'vue'
import { useQueryStore } from '../stores/query'
import { QueryService } from '../../bindings/tuxedosql/internal/service'
import QueryEditor from './QueryEditor.vue'
import QueryResult from './QueryResult.vue'
import MessagePanel from './MessagePanel.vue'
import ResizableSplitter from './ResizableSplitter.vue'
import TableView from './TableView.vue'
import type { QueryTab } from '../types/query'

const store = useQueryStore()

const splitPercent = ref(40)
const messagePanelWidth = ref(260)
const editingTabId = ref<string | null>(null)
const editTitle = ref('')
const executePromises = new Map<string, { cancel: () => void }>()

// ── Tab helpers ──

function handleTabClick(id: string) {
  store.setActiveTab(id)
}

function handleTabClose(id: string) {
  const p = executePromises.get(id)
  if (p) {
    p.cancel()
    executePromises.delete(id)
  }
  store.closeTab(id)
}

function handleTabDblClick(tab: QueryTab) {
  editingTabId.value = tab.id
  editTitle.value = tab.title
  nextTick(() => {
    const input = document.querySelector('.tab-rename-input') as HTMLInputElement | null
    input?.focus()
    input?.select()
  })
}

function finishRename(tabId: string) {
  if (editTitle.value.trim()) {
    store.renameTab(tabId, editTitle.value.trim())
  }
  editingTabId.value = null
}

function handleRenameKeydown(e: KeyboardEvent, tabId: string) {
  if (e.key === 'Enter') finishRename(tabId)
  if (e.key === 'Escape') editingTabId.value = null
}

// ── Query execution ──

async function handleExecute(tabId: string) {
  const tab = store.tabs.find(t => t.id === tabId)
  if (!tab || tab.isExecuting) return

  store.setExecuting(tabId, true)
  try {
    const promise = QueryService.Execute(tab.connectionId, tab.database, tab.sql)
    executePromises.set(tabId, promise)
    const result = await promise
    if (result) {
      store.setResult(tabId, result)
    }
  } catch (err: unknown) {
    const msg = parseError(err)
    store.addMessage(tabId, msg)
  } finally {
    executePromises.delete(tabId)
    store.setExecuting(tabId, false)
  }
}

function handleStop(tabId: string) {
  const p = executePromises.get(tabId)
  if (p) {
    p.cancel()
    executePromises.delete(tabId)
  }
  store.addMessage(tabId, '查询已取消')
  store.setExecuting(tabId, false)
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

// ── Tab persistence ──

async function saveTabs() {
  try {
    await QueryService.SaveTabs(store.tabStates)
  } catch { /* fire and forget */ }
}

watch(() => store.tabs.length, saveTabs)
watch(() => store.tabs.map(t => t.title + t.sql + t.connectionId + t.database).join('|'), saveTabs)

onMounted(async () => {
  try {
    const restored = await QueryService.LoadTabs()
    if (restored && restored.length > 0) {
      for (const ts of restored) {
        store.openTab({
          connectionId: ts.connectionId,
          database: ts.database,
          sql: ts.sql,
          title: ts.title,
        })
      }
    }
  } catch { /* no tabs to restore */ }
})
</script>

<template>
  <div class="query-tabs">
    <!-- No tabs placeholder -->
    <div v-if="store.tabs.length === 0" class="no-tabs">
      <div class="no-tabs-icon">📝</div>
      <h2>查询编辑器</h2>
      <p>双击左侧的数据库或表名打开查询标签</p>
    </div>

    <!-- Tab bar -->
    <div v-else class="tab-bar">
      <div class="tab-list">
        <div
          v-for="tab in store.tabs"
          :key="tab.id"
          class="tab-item"
          :class="{ active: tab.id === store.activeTabId }"
          @click="handleTabClick(tab.id)"
          @dblclick="handleTabDblClick(tab)"
        >
          <template v-if="editingTabId === tab.id">
            <input
              class="tab-rename-input"
              v-model="editTitle"
              @blur="finishRename(tab.id)"
              @keydown="handleRenameKeydown($event, tab.id)"
            />
          </template>
          <template v-else>
            <span class="tab-title">{{ tab.title }}</span>
            <span class="tab-close" @click.stop="handleTabClose(tab.id)">×</span>
          </template>
        </div>
      </div>
    </div>

    <!-- Active tab content -->
    <template v-if="store.activeTab">
      <!-- Table view mode -->
      <div v-if="store.activeTab.viewType === 'table'" class="table-view-wrapper">
        <div class="table-main">
          <TableView :key="store.activeTab.id" :tab="store.activeTab" />
        </div>
        <div class="message-sidebar">
          <MessagePanel :messages="store.activeTab?.messages ?? []" :message-type="store.activeTab?.result?.messageType" />
        </div>
      </div>

      <!-- Query editor mode -->
      <template v-else>
        <div class="editor-pane" :style="{ flex: `0 0 ${splitPercent}%` }">
          <QueryEditor
            v-if="store.activeTab"
            :key="store.activeTab.id + '-editor'"
            :model-value="store.activeTab.sql"
            :is-executing="store.activeTab.isExecuting"
            :database="store.activeTab.database"
            @update:model-value="store.updateSQL(store.activeTab!.id, $event)"
            @execute="handleExecute(store.activeTab!.id)"
            @stop="handleStop(store.activeTab!.id)"
          />
        </div>
        <ResizableSplitter @resize="(p: number) => splitPercent = p" />
        <div class="result-pane" :style="{ flex: `0 0 ${100 - splitPercent}%` }">
          <div class="result-pane-inner">
            <QueryResult
              :columns="store.activeTab?.result?.columns ?? []"
              :rows="store.activeTab?.result?.rows ?? []"
              :message="store.activeTab?.result?.message"
              :duration="store.activeTab?.result?.duration"
            />
            <MessagePanel :messages="store.activeTab?.messages ?? []" :message-type="store.activeTab?.result?.messageType" />
          </div>
        </div>
      </template>
    </template>
  </div>
</template>

<style scoped>
.query-tabs {
  display: flex;
  flex-direction: column;
  flex: 1;
  height: 100%;
  min-width: 0;
  background: var(--color-bg, #fff);
}

/* ── No tabs placeholder ── */
.no-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--color-text-secondary, #6e6e80);
  gap: 8px;
  user-select: none;
}
.no-tabs-icon { font-size: 48px; }
.no-tabs h2 {
  font-size: 18px;
  font-weight: 600;
  color: var(--color-text, #1a1a2e);
  margin: 0;
}
.no-tabs p { font-size: 14px; margin: 0; }

/* ── Tab bar ── */
.tab-bar {
  flex-shrink: 0;
  background: var(--color-tab-bg, #f0f0f3);
  border-bottom: 1px solid var(--color-border, #d9d9dc);
}
.tab-list {
  display: flex;
  overflow-x: auto;
  scrollbar-width: none;
}
.tab-list::-webkit-scrollbar { display: none; }

.tab-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  font-size: 12px;
  color: var(--color-tab-text, #6e6e80);
  background: transparent;
  border-right: 1px solid var(--color-border, #d9d9dc);
  cursor: pointer;
  white-space: nowrap;
  user-select: none;
  transition: background 0.1s;
  position: relative;
}
.tab-item:hover { background: var(--color-tab-hover-bg, #e8e8ec); }
.tab-item.active {
  background: var(--color-tab-active-bg, #fff);
  color: var(--color-tab-active-text, #1a1a2e);
}

.tab-title {
  max-width: 140px;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tab-close {
  font-size: 14px;
  line-height: 1;
  padding: 0 2px;
  border-radius: 3px;
  opacity: 0;
  transition: opacity 0.1s;
}
.tab-item:hover .tab-close { opacity: 1; }
.tab-close:hover { background: var(--color-hover, rgba(0,0,0,0.08)); }

.tab-rename-input {
  width: 120px;
  font-size: 12px;
  padding: 1px 4px;
  border: 1px solid var(--color-accent, #6366f1);
  border-radius: 3px;
  outline: none;
  font-family: var(--font-sans);
  background: #fff;
}

/* ── Table view + right sidebar ── */
.table-view-wrapper {
  flex: 1;
  min-height: 0;
  display: flex;
  overflow: hidden;
}

.table-main {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.message-sidebar {
  width: 260px;
  flex-shrink: 0;
  border-left: 1px solid var(--color-border);
  background: var(--color-sidebar);
  overflow-y: auto;
}

/* ── Editor / Result panes ── */
.editor-pane {
  display: flex;
  min-height: 60px;
  overflow: hidden;
}

.result-pane {
  display: flex;
  min-height: 60px;
  overflow: hidden;
}

.result-pane-inner {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

/* ── Table view (legacy, kept for compatibility) ── */
.table-view-pane {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
