import { mount } from '@vue/test-utils'
import { nextTick } from 'vue'

const codemirrorMocks = vi.hoisted(() => {
  const dispatchMock = vi.fn()
  const destroyMock = vi.fn()
  const focusMock = vi.fn()
  const themeMock = vi.fn(() => ({ type: 'theme' }))
  let updateListenerExtension: ((update: { docChanged: boolean; state: { doc: { toString(): string } } }) => void) | undefined

  class MockEditorView {
    static lineWrapping = { type: 'lineWrapping' }
    static theme = themeMock
    static updateListener = {
      of: vi.fn((listener) => {
        updateListenerExtension = listener
        return { type: 'updateListener', listener }
      }),
    }

    state = {
      doc: {
        length: 0,
        toString: () => this.docValue,
      },
    }

    docValue: string

    constructor(options: { doc?: string }) {
      this.docValue = options.doc ?? ''
      this.state.doc.length = this.docValue.length
    }

    dispatch = dispatchMock
    destroy = destroyMock
    focus = focusMock
  }

  return {
    MockEditorView,
    dispatchMock,
    destroyMock,
    focusMock,
    themeMock,
    getUpdateListenerExtension: () => updateListenerExtension,
    resetUpdateListenerExtension: () => {
      updateListenerExtension = undefined
    },
  }
})

import SqlEditor from '../SqlEditor.vue'

vi.mock('@wailsio/runtime', () => ({}))
vi.mock('../../../bindings/tuxedosql/internal/service', () => ({
  QueryService: {
    GetDBSchemaForCompletion: vi.fn(),
  },
}))
vi.mock('@codemirror/view', () => ({
  EditorView: codemirrorMocks.MockEditorView,
  keymap: { of: vi.fn((value) => ({ type: 'keymap', value })) },
  lineNumbers: vi.fn(() => ({ type: 'lineNumbers' })),
  highlightActiveLine: vi.fn(() => ({ type: 'highlightActiveLine' })),
}))
vi.mock('@codemirror/state', () => ({
  EditorState: { tabSize: { of: vi.fn((value) => ({ type: 'tabSize', value })) } },
  Compartment: class {
    of(value: unknown) {
      return { type: 'compartmentOf', value }
    }
    reconfigure(value: unknown) {
      return { type: 'reconfigure', value }
    }
  },
}))
vi.mock('@codemirror/commands', () => ({
  defaultKeymap: [],
  history: vi.fn(() => ({ type: 'history' })),
  historyKeymap: [],
}))
vi.mock('@codemirror/lang-sql', () => ({
  MySQL: { dialect: 'mysql' },
  sql: vi.fn(() => ({ type: 'sqlLanguage' })),
}))
vi.mock('@codemirror/language', () => ({
  indentOnInput: vi.fn(() => ({ type: 'indentOnInput' })),
  bracketMatching: vi.fn(() => ({ type: 'bracketMatching' })),
}))
vi.mock('@codemirror/autocomplete', () => ({
  autocompletion: vi.fn(() => ({ type: 'autocompletion' })),
}))

const mountEditor = (props?: Partial<InstanceType<typeof SqlEditor>['$props']>) =>
  mount(SqlEditor, {
    props: {
      modelValue: 'SELECT 1',
      isExecuting: false,
      database: 'demo_db',
      schema: null,
      ...props,
    },
  })

describe('SqlEditor', () => {
  beforeEach(() => {
    codemirrorMocks.dispatchMock.mockClear()
    codemirrorMocks.destroyMock.mockClear()
    codemirrorMocks.focusMock.mockClear()
    codemirrorMocks.themeMock.mockClear()
    codemirrorMocks.resetUpdateListenerExtension()
  })

  it('mounts without errors', () => {
    const wrapper = mountEditor()

    expect(wrapper.exists()).toBe(true)
  })

  it('renders the editor container div', () => {
    const wrapper = mountEditor()

    expect(wrapper.find('div.sql-editor').exists()).toBe(true)
  })

  it('accepts the modelValue prop', () => {
    const wrapper = mountEditor({ modelValue: 'SELECT * FROM users' })

    expect(wrapper.props('modelValue')).toBe('SELECT * FROM users')
  })

  it('emits update:modelValue when the editor document changes', () => {
    const wrapper = mountEditor()

    codemirrorMocks.getUpdateListenerExtension()?.({
      docChanged: true,
      state: { doc: { toString: () => 'SELECT 2' } },
    })

    expect(wrapper.emitted('update:modelValue')).toEqual([['SELECT 2']])
  })

  it('syncs external modelValue changes back into the editor', async () => {
    const wrapper = mountEditor({ modelValue: 'SELECT 1' })

    await wrapper.setProps({ modelValue: 'SELECT 3' })
    await nextTick()

    expect(codemirrorMocks.dispatchMock).toHaveBeenCalledWith({
      changes: { from: 0, to: 8, insert: 'SELECT 3' },
    })
  })
})
