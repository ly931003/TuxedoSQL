<script setup lang="ts">
import { ref, watch, onMounted, nextTick, computed } from 'vue'
import { useQueryStore } from '../stores/query'
import { useLayoutStore } from '../stores/layout'
import { QueryService } from '../../bindings/tuxedosql/internal/service'
import QueryEditor from './QueryEditor.vue'
import QueryResult from './QueryResult.vue'
import MessagePanel from './MessagePanel.vue'
import ResizableSplitter from './ResizableSplitter.vue'
import TableView from './TableView.vue'
import TableInfoPanel from './TableInfoPanel.vue'
import TableDDLPanel from './TableDDLPanel.vue'
import type { DBSchemaForCompletion } from '../types/query'
import { parseError } from '../composables/parseError'

const store = useQueryStore()
const layoutStore = useLayoutStore()

const splitPercent = ref(40)
const messagePanelWidth = ref(260)
const editingTabId = ref<string | null>(null)
const editTitle = ref('')
const executePromises = new Map<string, { cancel: () => void }>()
const sidebarTab = ref<'messages' | 'info' | 'ddl'>('messages')

// ── Schema cache for autocomplete ──
const SCHEMA_CACHE_TTL_MS = 30_000

type SchemaCacheEntry = {
  data: DBSchemaForCompletion
  fetchedAt: number
}

const schemaCache = ref<Record<string, SchemaCacheEntry>>({})
const pendingSchemaFetches = new Map<string, Promise<DBSchemaForCompletion | null>>()

function getCachedSchema(cacheKey: string): DBSchemaForCompletion | undefined {
  const cached = schemaCache.value[cacheKey]
  if (!cached) {
    return undefined
  }

  if (Date.now() - cached.fetchedAt > SCHEMA_CACHE_TTL_MS) {
    const { [cacheKey]: _expired, ...rest } = schemaCache.value
    schemaCache.value = rest
    return undefined
  }

  return cached.data
}

async function fetchSchema(
  connectionId: string,
  database: string,
  options: { force?: boolean } = {},
): Promise<DBSchemaForCompletion | null> {
  if (!connectionId || !database) return null

  const cacheKey = `${connectionId}:${database}`
  if (!options.force) {
    const cached = getCachedSchema(cacheKey)
    if (cached !== undefined) {
      return cached
    }
  }

  const pending = pendingSchemaFetches.get(cacheKey)
  if (pending) return pending

  const promise = (async () => {
    try {
      const result = await QueryService.GetDBSchemaForCompletion(connectionId, database)
      if (!result) {
        const { [cacheKey]: _missing, ...rest } = schemaCache.value
        schemaCache.value = rest
        return null
      }

      schemaCache.value = {
        ...schemaCache.value,
        [cacheKey]: {
          data: result,
          fetchedAt: Date.now(),
        },
      }
      return result
    } catch (error: unknown) {
      const { [cacheKey]: _stale, ...rest } = schemaCache.value
      schemaCache.value = rest
      console.warn('Failed to load schema for autocomplete', { connectionId, database, error })
      return null
    } finally {
      pendingSchemaFetches.delete(cacheKey)
    }
  })()

  pendingSchemaFetches.set(cacheKey, promise)
  return promise
}

const activeTabSchema = computed<DBSchemaForCompletion | null>(() => {
  const tab = store.activeTab
  if (!tab?.connectionId || !tab?.database) return null
  return getCachedSchema(`${tab.connectionId}:${tab.database}`) ?? null
})

watch(
  () => store.activeTab,
  (tab) => {
    if (!tab?.connectionId || !tab?.database) {
      return
    }

    void fetchSchema(tab.connectionId, tab.database)
  },
  { immediate: true },
)

function handleRightSidebarResize(width: number) {
  layoutStore.setRightSidebarWidth(width)
}

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

function handleTabDblClick(tab: { id: string; title: string }) {
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
  const tab = store.tabs.find((t) => t.id === tabId)
  if (!tab || tab.isExecuting) return

  store.setExecuting(tabId, true)
  store.updateLastExecutedSQL(tabId, tab.sql)
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

// ── Tab persistence ──

async function saveTabs() {
  try {
    await QueryService.SaveTabs(store.tabStates)
  } catch {
    /* fire and forget */
  }
}

watch(() => store.tabs.length, saveTabs)
watch(
  () => store.tabs.map((t) => t.title + t.sql + t.connectionId + t.database).join('|'),
  saveTabs,
)

onMounted(async () => {
  try {
    const restored = await QueryService.LoadTabs()
    if (restored && restored.length > 0) {
      for (const ts of restored) {
        store.openTab({
          connectionId: ts.connectionId,
          database: ts.database,
          sql: ts.sql ?? '',
          title: ts.title,
        })
      }
    }
  } catch {
    /* no tabs to restore */
  }
})
</script>

<template>
  <div class="query-tabs">
    <!-- No tabs placeholder -->
    <div v-if="store.tabs.length === 0" class="no-tabs">
      <div class="no-tabs-icon">
        <svg
          width="48"
          height="48"
          viewBox="0 0 24 24"
          fill="none"
          stroke="currentColor"
          stroke-width="1.2"
          stroke-linecap="round"
          stroke-linejoin="round"
        >
          <polyline points="16 3 21 3 21 8" />
          <line x1="4" y1="20" x2="21" y2="3" />
          <polyline points="21 16 21 21 16 21" />
          <line x1="15" y1="15" x2="21" y2="21" />
          <line x1="4" y1="4" x2="9" y2="9" />
        </svg>
      </div>
      <h2>查询编辑器</h2>
      <p>双击左侧的数据库或表名来开始查询</p>
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
          <!-- Tab icon -->
          <span class="tab-icon">{{ tab.viewType === 'table' ? '⊞' : '▸' }}</span>

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
            <span class="tab-subtitle" v-if="tab.database">{{ tab.database }}</span>
            <span class="tab-close" @click.stop="handleTabClose(tab.id)">
              <svg
                width="12"
                height="12"
                viewBox="0 0 24 24"
                fill="none"
                stroke="currentColor"
                stroke-width="2.5"
                stroke-linecap="round"
              >
                <line x1="18" y1="6" x2="6" y2="18" />
                <line x1="6" y1="6" x2="18" y2="18" />
              </svg>
            </span>
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
        <ResizableSplitter
          v-show="layoutStore.rightSidebarVisible"
          direction="horizontal"
          :min-width="160"
          :max-width="600"
          @resize-width="handleRightSidebarResize"
        />
        <div
          v-show="layoutStore.rightSidebarVisible"
          class="right-sidebar"
          :style="{ width: layoutStore.rightSidebarWidth + 'px' }"
        >
          <div class="sidebar-tabs">
            <button
              class="sidebar-tab-btn"
              :class="{ active: sidebarTab === 'messages' }"
              @click="sidebarTab = 'messages'"
            >
              消息
            </button>
            <button
              class="sidebar-tab-btn"
              :class="{ active: sidebarTab === 'info' }"
              @click="sidebarTab = 'info'"
            >
              表信息
            </button>
            <button
              class="sidebar-tab-btn"
              :class="{ active: sidebarTab === 'ddl' }"
              @click="sidebarTab = 'ddl'"
            >
              建表语句
            </button>
          </div>
          <div class="sidebar-content">
            <MessagePanel
              v-if="sidebarTab === 'messages'"
              :messages="store.activeTab?.messages ?? []"
              :message-type="store.activeTab?.result?.messageType"
            />
            <TableInfoPanel
              v-if="sidebarTab === 'info'"
              :connection-id="store.activeTab.connectionId"
              :database="store.activeTab.database"
              :table-name="store.activeTab.tableName ?? ''"
            />
            <TableDDLPanel
              v-if="sidebarTab === 'ddl'"
              :connection-id="store.activeTab.connectionId"
              :database="store.activeTab.database"
              :table-name="store.activeTab.tableName ?? ''"
            />
          </div>
        </div>
      </div>

      <!-- Query editor mode -->
      <template v-else>
        <div class="editor-pane" :style="{ flexBasis: `${splitPercent}%` }">
          <QueryEditor
            v-if="store.activeTab"
            :key="store.activeTab.id + '-editor'"
            :model-value="store.activeTab.sql"
            :is-executing="store.activeTab.isExecuting"
            :database="store.activeTab.database"
            :schema="activeTabSchema"
            @update:model-value="store.updateSQL(store.activeTab!.id, $event)"
            @execute="handleExecute(store.activeTab!.id)"
            @stop="handleStop(store.activeTab!.id)"
          />
        </div>
        <ResizableSplitter @resize="(p: number) => (splitPercent = p)" />
        <div class="result-pane" :style="{ flexBasis: `${100 - splitPercent}%` }">
          <div class="result-pane-inner">
            <QueryResult
              :columns="store.activeTab?.result?.columns ?? []"
              :rows="store.activeTab?.result?.rows ?? []"
              :message="store.activeTab?.result?.message"
              :duration="store.activeTab?.result?.duration"
            />
            <MessagePanel
              :messages="store.activeTab?.messages ?? []"
              :message-type="store.activeTab?.result?.messageType"
            />
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
  min-height: 0;
  min-width: 0;
  background: var(--color-bg, #fafbfc);
}

/* ── No tabs placeholder ── */
.no-tabs {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--color-text-muted, #999);
  gap: 12px;
  user-select: none;
}
.no-tabs-icon {
  color: var(--color-text-muted, #bbb);
  opacity: 0.5;
}
.no-tabs h2 {
  font-size: 16px;
  font-weight: 500;
  color: var(--color-text-secondary, #666);
  margin: 0;
}
.no-tabs p {
  font-size: 13px;
  margin: 0;
  color: var(--color-text-muted, #999);
}

/* ── Tab bar ── */
.tab-bar {
  flex-shrink: 0;
  background: var(--color-tab-bar-bg, #f0f1f5);
  border-bottom: 1px solid var(--color-border, #e0e0e3);
  padding: 4px 6px 0;
}
.tab-list {
  display: flex;
  overflow-x: auto;
  scrollbar-width: none;
  gap: 2px;
}
.tab-list::-webkit-scrollbar {
  display: none;
}

.tab-item {
  display: flex;
  align-items: center;
  gap: 5px;
  padding: 6px 10px;
  font-size: 11.5px;
  color: var(--color-tab-text, #888);
  background: transparent;
  border-radius: 6px 6px 0 0;
  cursor: pointer;
  white-space: nowrap;
  user-select: none;
  transition:
    background 0.15s,
    color 0.15s;
  max-width: 200px;
}

.tab-item:hover {
  background: var(--color-tab-hover-bg, rgba(0, 0, 0, 0.05));
  color: var(--color-tab-hover-text, #333);
}

.tab-item.active {
  background: var(--color-tab-active-bg, #fff);
  color: var(--color-tab-active-text, #1a1a2e);
  box-shadow: 0 -1px 3px rgba(0, 0, 0, 0.04);
}

.tab-icon {
  font-size: 9px;
  opacity: 0.5;
  flex-shrink: 0;
}

.tab-title {
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 500;
}

.tab-subtitle {
  font-size: 10px;
  color: var(--color-text-muted, #aaa);
  overflow: hidden;
  text-overflow: ellipsis;
  flex-shrink: 1;
  min-width: 0;
}

.tab-close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  border-radius: 3px;
  flex-shrink: 0;
  opacity: 0;
  transition:
    opacity 0.12s,
    background 0.12s;
  color: inherit;
}
.tab-item:hover .tab-close {
  opacity: 0.6;
}
.tab-close:hover {
  opacity: 1 !important;
  background: var(--color-hover, rgba(0, 0, 0, 0.08));
}

.tab-rename-input {
  width: 100px;
  font-size: 11.5px;
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

.right-sidebar {
  flex-shrink: 0;
  border-left: 1px solid var(--color-border);
  background: var(--color-sidebar);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.sidebar-tabs {
  flex-shrink: 0;
  display: flex;
  border-bottom: 1px solid var(--color-border);
}

.sidebar-tab-btn {
  flex: 1;
  padding: 5px 0;
  font-size: 11px;
  color: var(--color-text-secondary);
  background: transparent;
  border: none;
  cursor: pointer;
  transition:
    background 0.1s,
    color 0.1s;
}

.sidebar-tab-btn:hover {
  background: var(--color-hover);
}

.sidebar-tab-btn.active {
  color: var(--color-text);
  background: var(--color-tab-active-bg, #fff);
  font-weight: 500;
}

.sidebar-content {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

/* ── Editor / Result panes ── */
.editor-pane {
  flex-basis: 0;
  display: flex;
  min-height: 80px;
  overflow: hidden;
}

.result-pane {
  flex-basis: 0;
  display: flex;
  min-height: 80px;
  overflow: hidden;
}

.result-pane-inner {
  flex: 1;
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

/* ── Table view (legacy) ── */
.table-view-pane {
  flex: 1;
  min-height: 0;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}
</style>
