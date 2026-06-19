import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useLayoutStore } from '../../stores/layout'
import { useQueryStore } from '../../stores/query'

const queryServiceMocks = vi.hoisted(() => ({
  loadTabsMock: vi.fn(),
  saveTabsMock: vi.fn(),
  executeMock: vi.fn(),
  getSchemaMock: vi.fn(),
}))

vi.mock('../../../bindings/tuxedosql/internal/service', () => ({
  QueryService: {
    LoadTabs: queryServiceMocks.loadTabsMock,
    SaveTabs: queryServiceMocks.saveTabsMock,
    Execute: queryServiceMocks.executeMock,
    GetDBSchemaForCompletion: queryServiceMocks.getSchemaMock,
  },
  ConnectionService: {},
}))

import QueryTabs from '../QueryTabs.vue'

const mountTabs = () => {
  const pinia = createPinia()
  setActivePinia(pinia)

  const queryStore = useQueryStore()
  const layoutStore = useLayoutStore()

  queryStore.tabs = [
    {
      id: 'tab-1',
      title: 'Query 1',
      connectionId: 'conn-1',
      database: 'demo_db',
      sql: 'SELECT 1',
      result: null,
      messages: [],
      isExecuting: false,
      viewType: 'query',
    },
  ]
  queryStore.activeTabId = 'tab-1'
  layoutStore.rightSidebarVisible = true

  return mount(QueryTabs, {
    global: {
      plugins: [pinia],
      stubs: {
        QueryEditor: { template: '<div class="query-editor-stub" />' },
        QueryResult: { template: '<div class="query-result-stub" />' },
        MessagePanel: { template: '<div class="message-panel-stub" />' },
        ResizableSplitter: { template: '<div class="resizable-splitter-stub" />' },
        TableView: { template: '<div class="table-view-stub" />' },
        TableInfoPanel: { template: '<div class="table-info-panel-stub" />' },
        TableDDLPanel: { template: '<div class="table-ddl-panel-stub" />' },
      },
    },
  })
}

describe('QueryTabs', () => {
  beforeEach(() => {
    queryServiceMocks.loadTabsMock.mockResolvedValue([])
    queryServiceMocks.saveTabsMock.mockResolvedValue(undefined)
    queryServiceMocks.executeMock.mockResolvedValue({ cancel: vi.fn() })
    queryServiceMocks.getSchemaMock.mockResolvedValue(null)
  })

  it('mounts without errors', () => {
    const wrapper = mountTabs()

    expect(wrapper.exists()).toBe(true)
  })

  it('renders the tab bar structure for an active tab', () => {
    const wrapper = mountTabs()

    expect(wrapper.find('.tab-bar').exists()).toBe(true)
    expect(wrapper.find('.tab-list').exists()).toBe(true)
    expect(wrapper.findAll('.tab-item')).toHaveLength(1)
  })

  it('shows the active tab title and database subtitle', () => {
    const wrapper = mountTabs()

    expect(wrapper.find('.tab-title').text()).toBe('Query 1')
    expect(wrapper.find('.tab-subtitle').text()).toBe('demo_db')
  })

  it('renders the query mode child stubs', () => {
    const wrapper = mountTabs()

    expect(wrapper.find('.query-editor-stub').exists()).toBe(true)
    expect(wrapper.find('.query-result-stub').exists()).toBe(true)
    expect(wrapper.find('.message-panel-stub').exists()).toBe(true)
  })
})
