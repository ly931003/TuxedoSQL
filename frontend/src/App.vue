<script setup lang="ts">
import { ref } from 'vue'
import Sidebar from './components/Sidebar.vue'
import QueryTabs from './components/QueryTabs.vue'
import ConnectionDialog from './components/ConnectionDialog.vue'
import BottomBar from './components/BottomBar.vue'
import ResizableSplitter from './components/ResizableSplitter.vue'
import QueryHistoryPanel from './components/QueryHistoryPanel.vue'
import { useLayoutStore } from './stores/layout'
import { useQueryStore } from './stores/query'

const layoutStore = useLayoutStore()
const sidebarRef = ref<InstanceType<typeof Sidebar> | null>(null)
const isDark = ref(document.documentElement.getAttribute('data-theme') === 'dark')
const queryStore = useQueryStore()
const showHistory = ref(false)

function handlePickSql(sql: string) {
  queryStore.addTab({
    connectionId: '',
    database: '',
    sql,
  })
}

function refreshTree() {
  sidebarRef.value?.loadData()
}

function toggleTheme() {
  isDark.value = !isDark.value
  const theme = isDark.value ? 'dark' : 'light'
  document.documentElement.setAttribute('data-theme', theme)
  localStorage.setItem('tuxedosql-theme', theme)
  document.body.setAttribute('data-theme', theme)
}

function handleLeftSidebarResize(width: number) {
  layoutStore.setLeftSidebarWidth(width)
}
</script>

<template>
  <div class="app-layout">
    <div class="content-row">
      <Sidebar ref="sidebarRef" v-show="layoutStore.leftSidebarVisible" />
      <ResizableSplitter
        v-show="layoutStore.leftSidebarVisible"
        direction="horizontal"
        :min-width="180"
        :max-width="500"
        @resize-width="handleLeftSidebarResize"
      />
      <div class="main-area">
        <div class="top-bar">
          <span class="app-brand">TuxedoSQL</span>
          <button
            class="history-btn"
            title="查询历史"
            @click="showHistory = true"
          >
            历史
          </button>
          <button
            class="theme-toggle"
            :title="isDark ? '切换到浅色主题' : '切换到暗色主题'"
            @click="toggleTheme"
          >
            <svg v-if="isDark" xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <circle cx="12" cy="12" r="4" />
              <path d="M12 2v2M12 20v2M4.93 4.93l1.41 1.41M17.66 17.66l1.41 1.41M2 12h2M20 12h2M6.34 17.66l-1.41 1.41M19.07 4.93l-1.41 1.41" />
            </svg>
            <svg v-else xmlns="http://www.w3.org/2000/svg" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
              <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
            </svg>
          </button>
        </div>
        <QueryTabs />
      </div>
    </div>
    <BottomBar />
    <ConnectionDialog @saved="refreshTree" />
    <QueryHistoryPanel
      :visible="showHistory"
      @close="showHistory = false"
      @pick-sql="handlePickSql"
    />
  </div>
</template>

<style scoped>
.app-layout {
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.content-row {
  display: flex;
  flex: 1;
  min-height: 0;
  overflow: hidden;
}

.main-area {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-width: 0;
  overflow: hidden;
}

.top-bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  height: 32px;
  padding: 0 12px;
  background: var(--color-sidebar);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.app-brand {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-accent);
  letter-spacing: 0.3px;
}

/* ── Top bar buttons ── */

.top-bar-right {
  display: flex;
  align-items: center;
  gap: 4px;
}

.theme-toggle {
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  font-size: 14px;
  line-height: 1;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
  color: var(--color-text);
}
.theme-toggle:hover {
  background: var(--color-hover);
}


.history-btn {
  font-size: 12px;
  padding: 2px 10px;
  border: none;
  border-radius: var(--radius-sm, 4px);
  background: transparent;
  color: var(--color-text-secondary, #666);
  cursor: pointer;
  transition: background 0.15s;
  font-family: var(--font-sans);
}

.history-btn:hover {
  background: var(--color-hover);
  color: var(--color-text);
}
</style>
