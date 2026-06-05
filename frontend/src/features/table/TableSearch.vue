<script setup lang="ts">
import { ref } from 'vue'

const props = defineProps<{
  columns: string[]
  loading?: boolean
}>()

const emit = defineEmits<{
  search: [params: { column: string; keyword: string }]
  reset: []
}>()

const selectedColumn = ref('')
const keyword = ref('')

function handleSearch() {
  if (!selectedColumn.value.trim()) return
  emit('search', { column: selectedColumn.value.trim(), keyword: keyword.value })
}

function handleReset() {
  selectedColumn.value = ''
  keyword.value = ''
  emit('reset')
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter') handleSearch()
}
</script>

<template>
  <div class="table-search">
    <select
      v-model="selectedColumn"
      class="search-col-select"
      :disabled="loading"
    >
      <option value="" disabled>选择列...</option>
      <option v-for="col in columns" :key="col" :value="col">{{ col }}</option>
    </select>
    <input
      v-model="keyword"
      class="search-input"
      type="text"
      placeholder="搜索关键词..."
      :disabled="loading || !selectedColumn"
      @keydown.enter="handleSearch"
    />
    <button
      class="search-btn"
      :disabled="loading || !selectedColumn"
      @click="handleSearch"
    >
      🔍
    </button>
    <button
      class="search-btn"
      :disabled="loading"
      title="重置搜索"
      @click="handleReset"
    >
      ✕
    </button>
  </div>
</template>

<style scoped>
.table-search {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-sidebar);
}

.search-col-select {
  font-size: 12px;
  padding: 2px 6px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm, 4px);
  background: var(--color-input-bg);
  color: var(--color-text);
  outline: none;
  min-width: 120px;
}

.search-input {
  flex: 1;
  font-size: 12px;
  padding: 2px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm, 4px);
  background: var(--color-input-bg);
  color: var(--color-text);
  outline: none;
}

.search-input:focus {
  border-color: var(--color-accent);
}

.search-btn {
  width: 24px;
  height: 24px;
  border: none;
  border-radius: var(--radius-sm, 4px);
  background: transparent;
  cursor: pointer;
  font-size: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: background 0.15s;
  color: var(--color-text-secondary);
}

.search-btn:hover:not(:disabled) {
  background: var(--color-hover);
}

.search-btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}
</style>
