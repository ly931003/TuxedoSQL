<script setup lang="ts">
import { ref } from 'vue'

defineProps<{
  direction?: 'horizontal' | 'vertical'
}>()

const emit = defineEmits<{
  resize: [percent: number]
}>()

const splitterRef = ref<HTMLElement | null>(null)
let dragging = false

function onMouseDown(e: MouseEvent) {
  e.preventDefault()
  dragging = true
  document.addEventListener('mousemove', onMouseMove)
  document.addEventListener('mouseup', onMouseUp)
  document.body.style.cursor = 'row-resize'
  document.body.style.userSelect = 'none'
}

function onMouseMove(e: MouseEvent) {
  if (!dragging || !splitterRef.value) return
  const container = splitterRef.value.parentElement
  if (!container) return
  const rect = container.getBoundingClientRect()
  const percent = ((e.clientY - rect.top) / rect.height) * 100
  emit('resize', Math.max(15, Math.min(85, percent)))
}

function onMouseUp() {
  dragging = false
  document.removeEventListener('mousemove', onMouseMove)
  document.removeEventListener('mouseup', onMouseUp)
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
}
</script>

<template>
  <div
    ref="splitterRef"
    class="resizable-splitter"
    :class="{ 'splitter-horizontal': direction === 'horizontal', 'splitter-vertical': direction !== 'horizontal' }"
    @mousedown="onMouseDown"
  >
    <div class="splitter-handle" />
  </div>
</template>

<style scoped>
.resizable-splitter {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--color-splitter, #e5e7eb);
  transition: background 0.15s;
  z-index: 10;
}
.resizable-splitter:hover {
  background: var(--color-splitter-hover, #6366f1);
}
.splitter-vertical {
  height: 4px;
  cursor: row-resize;
}
.splitter-horizontal {
  width: 4px;
  cursor: col-resize;
}
.splitter-handle {
  width: 32px;
  height: 3px;
  border-radius: 2px;
  background: var(--color-border, #d9d9dc);
  transition: background 0.15s;
}
.resizable-splitter:hover .splitter-handle {
  background: #fff;
}
</style>
