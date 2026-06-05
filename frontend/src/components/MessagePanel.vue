<script setup lang="ts">
import { ref, computed } from 'vue'

const props = defineProps<{
  messages: string[]
  messageType?: string
}>()

const collapsed = ref(false)  // 默认展开，让用户看到 SQL

const hasMessages = computed(() => props.messages.length > 0)

function getMessageClass(msg: string): string {
  if (props.messageType === 'success') return 'msg-success'
  if (props.messageType === 'error') return 'msg-error'
  if (msg.startsWith('✅')) return 'msg-audit'
  return 'msg-info'
}
</script>

<template>
  <div class="message-panel" :class="{ collapsed, empty: !hasMessages }">
    <div class="msg-header" @click="collapsed = !collapsed">
      <span class="msg-toggle">{{ collapsed ? '▶' : '▼' }}</span>
      <span class="msg-title">消息</span>
      <span class="msg-count">{{ messages.length }}</span>
    </div>
    <div v-if="!collapsed" class="msg-list">
      <div v-if="!hasMessages" class="msg-empty">暂无消息。编辑操作产生的 SQL 将显示在此处。</div>
      <div
        v-for="(msg, idx) in messages"
        :key="idx"
        class="msg-item"
        :class="getMessageClass(msg)"
      >
        {{ msg }}
      </div>
    </div>
  </div>
</template>

<style scoped>
.message-panel {
  border-top: 1px solid var(--color-border);
  background: var(--color-sidebar);
  display: flex;
  flex-direction: column;
  height: 100%;
  overflow: hidden;
}

.message-panel .msg-list {
  flex: 1;
  overflow-y: auto;
  max-height: none;
}

.message-panel.collapsed {
  height: auto;
  flex: 0 0 auto;
}

.msg-empty {
  font-size: 12px;
  color: var(--color-text-secondary);
  padding: 12px;
  text-align: center;
  font-style: italic;
}
.message-panel.collapsed {
  border-top: 1px solid var(--color-border);
}

.msg-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 3px 12px;
  cursor: pointer;
  user-select: none;
  font-size: 12px;
}
.msg-header:hover {
  background: var(--color-hover);
}

.msg-toggle {
  font-size: 8px;
  color: var(--color-text-secondary);
}
.msg-title {
  font-weight: 500;
  color: var(--color-text);
}
.msg-count {
  background: var(--color-border);
  color: var(--color-text-secondary);
  font-size: 10px;
  padding: 0 5px;
  border-radius: 8px;
}

.msg-list {
  max-height: 120px;
  overflow-y: auto;
  padding: 0 12px 4px;
}
.msg-item {
  font-size: 12px;
  font-family: var(--font-mono, monospace);
  padding: 1px 0;
  line-height: 1.5;
}

.msg-error { color: #e74c3c; }
.msg-success { color: #27ae60; }
.msg-info { color: var(--color-text-secondary); }
.msg-audit { color: var(--color-accent); font-weight: 500; }

/* Force MessagePanel to show scrollbar for audit SQL */
.message-panel:not(.collapsed) {
  max-height: none;
  flex-shrink: 1;
  min-height: 60px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.message-panel:not(.collapsed) .msg-list {
  flex: 1;
  max-height: 200px;
  overflow-y: auto;
}
</style>
