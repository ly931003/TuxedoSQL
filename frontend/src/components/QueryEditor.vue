<script setup lang="ts">
import { ref } from 'vue'
import SqlEditor from './SqlEditor.vue'
import type { DBSchemaForCompletion } from '../types/query'

const props = defineProps<{
  modelValue: string
  isExecuting: boolean
  database: string
  schema?: DBSchemaForCompletion | null
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  execute: []
  stop: []
}>()

const sqlEditorRef = ref<InstanceType<typeof SqlEditor> | null>(null)

function focus() {
  sqlEditorRef.value?.focus()
}

defineExpose({ focus })
</script>

<template>
  <div class="query-editor">
    <div class="editor-toolbar">
      <span class="toolbar-db">{{ database }}</span>
      <div class="toolbar-actions">
        <button
          class="btn-execute"
          :disabled="isExecuting"
          title="执行 (Ctrl+Enter)"
          @click="emit('execute')"
        >
          ▶ 执行
        </button>
        <button
          v-if="isExecuting"
          class="btn-stop"
          title="停止"
          @click="emit('stop')"
        >
          ■ 停止
        </button>
      </div>
    </div>
    <div class="editor-body">
      <SqlEditor
        ref="sqlEditorRef"
        :model-value="modelValue"
        :is-executing="isExecuting"
        :database="database"
        :schema="schema"
        @update:model-value="emit('update:modelValue', $event)"
        @execute="emit('execute')"
        @stop="emit('stop')"
      />
    </div>
  </div>
</template>

<style scoped>
.query-editor {
  display: flex;
  flex-direction: column;
  flex: 1;
  min-height: 0;
  background: var(--color-editor-bg, #fafbfc);
}

.editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 6px 12px;
  border-bottom: 1px solid var(--color-border, #d9d9dc);
  background: var(--color-surface, #fff);
  flex-shrink: 0;
}

.toolbar-db {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-accent, #6366f1);
  background: var(--color-selected, rgba(99, 102, 241, 0.10));
  padding: 2px 8px;
  border-radius: var(--radius-sm, 4px);
}

.toolbar-actions {
  display: flex;
  gap: 6px;
}

.btn-execute,
.btn-stop {
  font-size: 12px;
  padding: 4px 12px;
  border: none;
  border-radius: var(--radius-sm, 4px);
  cursor: pointer;
  transition: background 0.15s;
  font-family: var(--font-sans);
}

.btn-execute {
  background: var(--color-accent, #6366f1);
  color: #fff;
}
.btn-execute:hover:not(:disabled) {
  background: #4f46e5;
}
.btn-execute:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.btn-stop {
  background: #e74c3c;
  color: #fff;
}
.btn-stop:hover {
  background: #c0392b;
}

.editor-body {
  flex: 1;
  min-height: 0;
  overflow: hidden;
}
</style>
