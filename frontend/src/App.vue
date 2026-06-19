<script setup lang="ts">
import { ref } from 'vue'
import Sidebar from './components/Sidebar.vue'
import QueryTabs from './components/QueryTabs.vue'
import ConnectionDialog from './components/ConnectionDialog.vue'
import BottomBar from './components/BottomBar.vue'
import ResizableSplitter from './components/ResizableSplitter.vue'
import { useLayoutStore } from './stores/layout'

const layoutStore = useLayoutStore()
const sidebarRef = ref<InstanceType<typeof Sidebar> | null>(null)
const isDark = ref(document.documentElement.getAttribute('data-theme') === 'dark')

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
            class="theme-toggle"
            :title="isDark ? '切换到浅色主题' : '切换到暗色主题'"
            @click="toggleTheme"
          >
            {{ isDark ? '☀' : '☾' }}
          </button>
        </div>
        <QueryTabs />
      </div>
    </div>
    <BottomBar />
    <ConnectionDialog @saved="refreshTree" />
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
</style>
