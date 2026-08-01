<script setup lang="ts">
import { computed } from 'vue'
import { useLayoutStore } from '../stores/layout'
import { useQueryStore } from '../stores/query'

const layoutStore = useLayoutStore()
const queryStore = useQueryStore()

const activeTab = computed(() => queryStore.activeTab)

const displaySQL = computed(() => {
  const tab = activeTab.value
  if (!tab) return ''
  return tab.lastExecutedSQL ?? ''
})

const connectionInfo = computed(() => {
  const tab = activeTab.value
  if (!tab) return ''
  return `${tab.database}`
})

const zoomLabel = computed(() => `${Math.round(layoutStore.fontScale * 100)}%`)

function copySQL() {
  if (!displaySQL.value) return
  navigator.clipboard.writeText(displaySQL.value)
}
</script>

<template>
  <div class="bottom-bar">
    <div class="bar-left">
      <span v-if="connectionInfo" class="bar-connection-info">{{ connectionInfo }}</span>
    </div>
    <div class="bar-center">
      <span v-if="displaySQL" class="bar-sql" :title="displaySQL">{{ displaySQL }}</span>
      <button v-if="displaySQL" class="bar-copy-btn" title="复制 SQL" @click="copySQL">📋</button>
    </div>
    <div class="bar-right">
      <!-- Zoom controls -->
      <button
        class="bar-zoom-btn"
        title="缩小"
        :disabled="layoutStore.isMinZoom"
        @click="layoutStore.zoomOut()"
      >
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
          <rect x="2" y="5" width="8" height="2" rx="1" fill="currentColor" />
        </svg>
      </button>
      <button
        class="bar-zoom-label"
        :title="`重置为 100%（当前 ${zoomLabel}）`"
        @click="layoutStore.resetZoom()"
      >
        {{ zoomLabel }}
      </button>
      <button
        class="bar-zoom-btn"
        title="放大"
        :disabled="layoutStore.isMaxZoom"
        @click="layoutStore.zoomIn()"
      >
        <svg width="12" height="12" viewBox="0 0 12 12" fill="none">
          <rect x="2" y="5" width="8" height="2" rx="1" fill="currentColor" />
          <rect x="5" y="2" width="2" height="8" rx="1" fill="currentColor" />
        </svg>
      </button>
      <span class="bar-separator"></span>
      <!-- Sidebar toggles -->
      <button
        class="bar-toggle-btn"
        :class="{ active: !layoutStore.leftSidebarVisible }"
        title="切换左侧边栏"
        @click="layoutStore.toggleLeftSidebar()"
      >
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <rect
            x="1"
            y="1"
            width="4"
            height="12"
            rx="1"
            :fill="
              layoutStore.leftSidebarVisible ? 'var(--color-accent)' : 'var(--color-text-secondary)'
            "
          />
          <rect
            x="7"
            y="1"
            width="6"
            height="12"
            rx="1"
            fill="var(--color-text-secondary)"
            opacity="0.4"
          />
        </svg>
      </button>
      <button
        class="bar-toggle-btn"
        :class="{ active: !layoutStore.rightSidebarVisible }"
        title="切换右侧边栏"
        @click="layoutStore.toggleRightSidebar()"
      >
        <svg width="14" height="14" viewBox="0 0 14 14" fill="none">
          <rect
            x="7"
            y="1"
            width="6"
            height="12"
            rx="1"
            :fill="
              layoutStore.rightSidebarVisible
                ? 'var(--color-accent)'
                : 'var(--color-text-secondary)'
            "
          />
          <rect
            x="1"
            y="1"
            width="4"
            height="12"
            rx="1"
            fill="var(--color-text-secondary)"
            opacity="0.4"
          />
        </svg>
      </button>
    </div>
  </div>
</template>

<style scoped>
.bottom-bar {
  display: flex;
  align-items: center;
  height: 28px;
  background: var(--color-sidebar);
  border-top: 1px solid var(--color-border);
  padding: 0 8px;
  flex-shrink: 0;
  gap: 8px;
  user-select: none;
}

.bar-left {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 4px;
}

.bar-connection-info {
  font-size: 11px;
  color: var(--color-text-secondary);
  font-family: var(--font-sans);
}

.bar-center {
  flex: 1;
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  overflow: hidden;
}

.bar-sql {
  font-size: 11px;
  font-family: var(--font-mono);
  color: var(--color-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 100%;
}

.bar-copy-btn {
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 3px;
  background: transparent;
  cursor: pointer;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
  flex-shrink: 0;
}
.bar-copy-btn:hover {
  background: var(--color-hover);
}

.bar-right {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 2px;
}

/* ── Zoom controls ── */

.bar-zoom-btn {
  width: 20px;
  height: 20px;
  border: none;
  border-radius: 3px;
  background: transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
  color: var(--color-text-secondary);
  padding: 0;
}
.bar-zoom-btn:hover {
  background: var(--color-hover);
  color: var(--color-text);
}
.bar-zoom-btn:disabled {
  opacity: 0.35;
  cursor: not-allowed;
}
.bar-zoom-btn:disabled:hover {
  background: transparent;
  color: var(--color-text-secondary);
}

.bar-zoom-label {
  min-width: 36px;
  height: 20px;
  border: none;
  border-radius: 3px;
  background: transparent;
  cursor: pointer;
  font-size: 10px;
  font-family: var(--font-sans);
  color: var(--color-text-secondary);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 0 2px;
  transition: background 0.15s;
}
.bar-zoom-label:hover {
  background: var(--color-hover);
  color: var(--color-text);
}

.bar-separator {
  width: 1px;
  height: 16px;
  background: var(--color-border);
  margin: 0 4px;
}

/* ── Sidebar toggles ── */

.bar-toggle-btn {
  width: 24px;
  height: 24px;
  border: none;
  border-radius: 4px;
  background: transparent;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
  padding: 0;
}
.bar-toggle-btn:hover {
  background: var(--color-hover);
}
</style>
