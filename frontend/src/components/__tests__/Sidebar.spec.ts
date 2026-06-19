import { mount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { useConnectionStore } from '../../stores/connection'
import { useLayoutStore } from '../../stores/layout'
import { useQueryStore } from '../../stores/query'

const sidebarMocks = vi.hoisted(() => ({
  listMock: vi.fn(),
  listGroupsMock: vi.fn(),
  getDatabasesMock: vi.fn(),
  getTablesMock: vi.fn(),
  successMock: vi.fn(),
  errorMock: vi.fn(),
}))

vi.mock('element-plus', () => ({
  ElMessage: Object.assign(vi.fn(), {
    success: sidebarMocks.successMock,
    error: sidebarMocks.errorMock,
  }),
}))

vi.mock('../../../bindings/tuxedosql/internal/service', () => ({
  QueryService: {},
  ConnectionService: {
    List: sidebarMocks.listMock,
    ListGroups: sidebarMocks.listGroupsMock,
    GetDatabases: sidebarMocks.getDatabasesMock,
    GetTables: sidebarMocks.getTablesMock,
    Delete: vi.fn(),
    DeleteGroup: vi.fn(),
    Update: vi.fn(),
    UpdateGroup: vi.fn(),
    DropDatabase: vi.fn(),
    DropTable: vi.fn(),
  },
}))

import Sidebar from '../Sidebar.vue'

const mountSidebar = async () => {
  const pinia = createPinia()
  setActivePinia(pinia)

  const connectionStore = useConnectionStore()
  const queryStore = useQueryStore()
  const layoutStore = useLayoutStore()

  connectionStore.groups = []
  connectionStore.connections = []
  queryStore.tabs = []
  layoutStore.leftSidebarWidth = 320

  const wrapper = mount(Sidebar, {
    global: {
      plugins: [pinia],
      stubs: {
        ConnectionTree: { template: '<div class="connection-tree-stub" />' },
        GroupDialog: { template: '<div class="group-dialog-stub" />' },
        CreateDatabaseDialog: { template: '<div class="create-database-dialog-stub" />' },
        CreateTableDialog: { template: '<div class="create-table-dialog-stub" />' },
      },
    },
  })

  await Promise.resolve()
  return wrapper
}

describe('Sidebar', () => {
  beforeEach(() => {
    sidebarMocks.listMock.mockResolvedValue([])
    sidebarMocks.listGroupsMock.mockResolvedValue([])
    sidebarMocks.getDatabasesMock.mockResolvedValue([])
    sidebarMocks.getTablesMock.mockResolvedValue([])
    sidebarMocks.successMock.mockClear()
    sidebarMocks.errorMock.mockClear()
  })

  it('mounts without errors', async () => {
    const wrapper = await mountSidebar()

    expect(wrapper.exists()).toBe(true)
  })

  it('renders the sidebar shell structure', async () => {
    const wrapper = await mountSidebar()

    expect(wrapper.find('.sidebar').exists()).toBe(true)
    expect(wrapper.find('.sidebar-header').exists()).toBe(true)
    expect(wrapper.find('.header-btns').exists()).toBe(true)
  })

  it('renders stubbed child components', async () => {
    const wrapper = await mountSidebar()

    expect(wrapper.find('.connection-tree-stub').exists()).toBe(true)
    expect(wrapper.find('.group-dialog-stub').exists()).toBe(true)
    expect(wrapper.find('.create-database-dialog-stub').exists()).toBe(true)
    expect(wrapper.find('.create-table-dialog-stub').exists()).toBe(true)
  })

  it('applies width from the layout store', async () => {
    const wrapper = await mountSidebar()

    expect(wrapper.attributes('style')).toContain('width: 320px;')
  })
})
