import { createPinia, setActivePinia } from 'pinia'

vi.mock('../../types/query', () => ({
  SortOrder: {
    SortASC: 'asc',
  },
}))

import { useQueryStore } from '../query'

describe('useQueryStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  const createTabInput = (overrides: Record<string, unknown> = {}) => ({
    connectionId: 'conn-1',
    database: 'app',
    ...overrides,
  })

  it('initializes with no tabs and no active tab', () => {
    const store = useQueryStore()

    expect(store.tabs).toEqual([])
    expect(store.activeTabId).toBeNull()
    expect(store.activeTab).toBeNull()
  })

  it('addTab creates a query tab with generated id, title, and empty SQL', () => {
    const store = useQueryStore()

    const tab = store.addTab(createTabInput())

    expect(tab.id).toMatch(/^tab_/)
    expect(tab.title).toMatch(/^Query \d+$/)
    expect(tab.sql).toBe('')
    expect(tab.messages).toEqual([])
    expect(store.activeTabId).toBe(tab.id)
  })

  it('addTab allows overriding the title', () => {
    const store = useQueryStore()

    const tab = store.addTab(createTabInput({ title: 'Custom Query' }))

    expect(tab.title).toBe('Custom Query')
  })

  it('addTab increments tab count across multiple calls', () => {
    const store = useQueryStore()

    store.addTab(createTabInput())
    store.addTab(createTabInput({ database: 'analytics' }))
    store.addTab(createTabInput({ connectionId: 'conn-2' }))

    expect(store.tabs).toHaveLength(3)
    expect(new Set(store.tabs.map((tab) => tab.id)).size).toBe(3)
  })

  it('removeTab removes a tab and keeps the last remaining tab active when removing the active one', () => {
    const store = useQueryStore()

    const first = store.addTab(createTabInput({ title: 'First' }))
    const second = store.addTab(createTabInput({ title: 'Second' }))

    store.removeTab(second.id)

    expect(store.tabs.map((tab) => tab.id)).toEqual([first.id])
    expect(store.activeTabId).toBe(first.id)
  })

  it('removeTab clears activeTabId when all tabs are removed', () => {
    const store = useQueryStore()

    const tab = store.addTab(createTabInput())

    store.removeTab(tab.id)

    expect(store.tabs).toEqual([])
    expect(store.activeTabId).toBeNull()
  })

  it('setActiveTab updates activeTabId', () => {
    const store = useQueryStore()

    const first = store.addTab(createTabInput({ title: 'First' }))
    const second = store.addTab(createTabInput({ title: 'Second' }))

    store.setActiveTab(first.id)

    expect(store.activeTabId).toBe(first.id)
    expect(store.activeTab?.id).toBe(first.id)
    expect(second.id).not.toBe(store.activeTabId)
  })

  it('updateTabSQL updates sql on the matching tab only', () => {
    const store = useQueryStore()

    const first = store.addTab(createTabInput({ title: 'First' }))
    const second = store.addTab(createTabInput({ title: 'Second' }))

    store.updateTabSQL(first.id, 'SELECT 1')

    expect(store.tabs.find((tab) => tab.id === first.id)?.sql).toBe('SELECT 1')
    expect(store.tabs.find((tab) => tab.id === second.id)?.sql).toBe('')
  })

  it('addMessage appends messages to the target tab', () => {
    const store = useQueryStore()

    const first = store.addTab(createTabInput())
    const second = store.addTab(createTabInput({ title: 'Second' }))

    store.addMessage(first.id, 'Started')
    store.addMessage(first.id, 'Finished')

    expect(store.tabs.find((tab) => tab.id === first.id)?.messages).toEqual(['Started', 'Finished'])
    expect(store.tabs.find((tab) => tab.id === second.id)?.messages).toEqual([])
  })

  it('clearMessages removes all messages from the target tab', () => {
    const store = useQueryStore()

    const tab = store.addTab(createTabInput())
    store.addMessage(tab.id, 'Started')
    store.addMessage(tab.id, 'Finished')

    store.clearMessages(tab.id)

    expect(store.tabs.find((item) => item.id === tab.id)?.messages).toEqual([])
  })

  it('activeTab getter returns the tab matching activeTabId', () => {
    const store = useQueryStore()

    const first = store.addTab(createTabInput({ title: 'First' }))
    const second = store.addTab(createTabInput({ title: 'Second' }))

    store.setActiveTab(first.id)

    expect(store.activeTab).toMatchObject({ id: first.id, title: 'First' })
    expect(store.activeTab?.id).not.toBe(second.id)
  })
})
