<script setup lang="ts">
import { ref, onMounted, onUnmounted, watch, shallowRef } from 'vue'
import { EditorView, keymap, lineNumbers, highlightActiveLine } from '@codemirror/view'
import { EditorState, Compartment } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { sql, MySQL } from '@codemirror/lang-sql'
import { indentOnInput, bracketMatching, type LanguageSupport } from '@codemirror/language'
import { autocompletion, type Completion } from '@codemirror/autocomplete'
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

const editorRef = ref<HTMLDivElement | null>(null)
const editorView = shallowRef<EditorView | null>(null)
// Use a Compartment so we can reconfigure the SQL language extension
// without destroying the editor (preserving undo history, selection, scroll).
const languageCompartment = new Compartment()

function buildSQLLanguage(schema?: DBSchemaForCompletion | null): LanguageSupport {
  const schemaMap: Record<string, (string | Completion)[]> = {}

  if (schema?.tables) {
    for (const [tableName, columns] of Object.entries(schema.tables)) {
      if (columns && columns.length > 0) {
        schemaMap[tableName] = columns.map(
          (col): Completion => ({
            label: col,
            type: 'column',
          }),
        )
      } else {
        schemaMap[tableName] = []
      }
    }
  }

  if (schema?.views) {
    for (const viewName of schema.views) {
      if (!schemaMap[viewName]) {
        schemaMap[viewName] = []
      }
    }
  }

  return sql({
    dialect: MySQL,
    upperCaseKeywords: true,
    schema: Object.keys(schemaMap).length > 0 ? schemaMap : undefined,
  })
}

function createEditor(): EditorView {
  const executeKeymap = keymap.of([
    {
      key: 'Ctrl-Enter',
      run: () => {
        if (!props.isExecuting) {
          emit('execute')
        }
        return true
      },
    },
  ])

  const updateListener = EditorView.updateListener.of((update) => {
    if (update.docChanged) {
      const val = update.state.doc.toString()
      emit('update:modelValue', val)
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
      // Language extension in a Compartment so we can reconfigure on schema changes
      languageCompartment.of(buildSQLLanguage(props.schema)),
      autocompletion({
        activateOnTyping: true,
        closeOnBlur: false,
      }),
      EditorView.lineWrapping,
      EditorState.tabSize.of(2),
      EditorView.theme({
        '&': { height: '100%' },
        '.cm-scroller': { overflow: 'auto' },
        '.cm-content': {
          fontFamily: 'var(--font-mono, "JetBrains Mono", "Fira Code", "Cascadia Code", monospace)',
          fontSize: '13.5px',
          lineHeight: '1.65',
          padding: '12px 0',
        },
        '.cm-gutters': {
          backgroundColor: 'var(--color-editor-gutter-bg, #f8f8fa)',
          borderRight: '1px solid var(--color-border, #e0e0e3)',
          color: 'var(--color-text-muted, #999)',
          fontSize: '11px',
        },
        '.cm-activeLineGutter': {
          backgroundColor: 'var(--color-editor-gutter-active, #ebebf0)',
        },
        '&.cm-focused .cm-selectionBackground, ::selection': {
          backgroundColor: 'var(--color-selected, rgba(99, 102, 241, 0.12)) !important',
        },
        '.cm-activeLine': {
          backgroundColor: 'var(--color-editor-active-line, rgba(0,0,0,0.03))',
        },
        '.cm-cursor': {
          borderLeftColor: 'var(--color-editor-cursor, #6366f1)',
        },
        '.cm-tooltip': {
          backgroundColor: 'var(--color-surface, #fff)',
          border: '1px solid var(--color-border, #e0e0e3)',
          borderRadius: '6px',
          boxShadow: '0 4px 16px rgba(0,0,0,0.10)',
          fontSize: '13px',
        },
        '.cm-tooltip-autocomplete': {
          '& > ul > li': {
            padding: '4px 10px',
            lineHeight: '1.5',
          },
          '& > ul > li[aria-selected]': {
            backgroundColor: 'var(--color-accent, #6366f1)',
            color: '#fff',
          },
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
  editorView.value = null
})

// Sync external modelValue changes back into editor (e.g. tab switch)
watch(
  () => props.modelValue,
  (newVal) => {
    const view = editorView.value
    if (!view) return
    if (newVal !== view.state.doc.toString()) {
      view.dispatch({
        changes: { from: 0, to: view.state.doc.length, insert: newVal },
      })
    }
  },
)

// Reconfigure language when schema changes (preserves undo history, selection, scroll)
watch(
  () => props.schema,
  (newSchema) => {
    const view = editorView.value
    if (!view) return
    view.dispatch({
      effects: languageCompartment.reconfigure(buildSQLLanguage(newSchema)),
    })
  },
)

defineExpose({
  focus() {
    editorView.value?.focus()
  },
})
</script>

<template>
  <div ref="editorRef" class="sql-editor" :class="{ 'is-executing': props.isExecuting }" />
</template>

<style scoped>
.sql-editor {
  height: 100%;
  overflow: hidden;
}

.sql-editor.is-executing {
  opacity: 0.6;
  pointer-events: none;
}
</style>
