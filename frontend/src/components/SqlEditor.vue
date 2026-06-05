<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, shallowRef } from 'vue'
import { EditorView, keymap, lineNumbers, highlightActiveLine } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { sql, MySQL } from '@codemirror/lang-sql'
import { indentOnInput, bracketMatching } from '@codemirror/language'

const props = defineProps<{
  modelValue: string
  isExecuting: boolean
  database: string
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
  execute: []
  stop: []
}>()

const editorRef = ref<HTMLDivElement | null>(null)
const editorView = shallowRef<EditorView | null>(null)

function createEditor(): EditorView {
  const executeKeymap = keymap.of([{
    key: 'Ctrl-Enter',
    run: () => {
      if (!props.isExecuting) {
        emit('execute')
      }
      return true
    },
  }])

  const updateListener = EditorView.updateListener.of((update) => {
    if (update.docChanged) {
      const newValue = update.state.doc.toString()
      emit('update:modelValue', newValue)
    }
  })

  return new EditorView({
    doc: props.modelValue,
    extensions: [
      lineNumbers(),
      highlightActiveLine(),
      bracketMatching(),
      history(),
      indentOnInput(),
      keymap.of([...defaultKeymap, ...historyKeymap]),
      executeKeymap,
      updateListener,
      sql({ dialect: MySQL }),
      EditorView.lineWrapping,
      EditorState.tabSize.of(2),
      EditorView.theme({
        '&': { height: '100%' },
        '.cm-scroller': { overflow: 'auto' },
        '.cm-content': {
          fontFamily: 'var(--font-mono, monospace)',
          fontSize: '13px',
          lineHeight: '1.6',
          padding: '8px 0',
        },
        '.cm-gutters': {
          backgroundColor: 'var(--color-editor-gutter-bg, #f5f5f7)',
          borderRight: '1px solid var(--color-border, #d9d9dc)',
          color: 'var(--color-text-secondary, #6e6e80)',
        },
        '.cm-activeLineGutter': {
          backgroundColor: 'var(--color-editor-gutter-active, #e8e8ec)',
        },
        '&.cm-focused .cm-selectionBackground, ::selection': {
          backgroundColor: 'var(--color-selected, rgba(99, 102, 241, 0.10)) !important',
        },
        '.cm-activeLine': {
          backgroundColor: 'var(--color-editor-active-line, rgba(0,0,0,0.02))',
        },
        '.cm-cursor': {
          borderLeftColor: 'var(--color-editor-cursor, #6366f1)',
        },
      }),
    ],
    parent: editorRef.value!,
  })
}

onMounted(() => {
  editorView.value = createEditor()
})

onUnmounted(() => {
  editorView.value?.destroy()
})

watch(() => props.modelValue, (newVal) => {
  const view = editorView.value
  if (!view) return
  if (newVal !== view.state.doc.toString()) {
    view.dispatch({
      changes: { from: 0, to: view.state.doc.length, insert: newVal },
    })
  }
})

watch(() => props.isExecuting, () => {
  // 只读模式通过 CSS 类控制，无需在编辑器层面处理
})

defineExpose({
  focus() {
    editorView.value?.focus()
  },
})
</script>

<template>
  <div
    ref="editorRef"
    class="sql-editor"
    :class="{ 'is-executing': props.isExecuting }"
  />
</template>

<style scoped>
.sql-editor {
  height: 100%;
  overflow: hidden;
}

.sql-editor.is-executing {
  opacity: 0.7;
  pointer-events: none;
}
</style>
