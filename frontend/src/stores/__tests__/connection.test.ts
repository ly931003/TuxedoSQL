import { createPinia, setActivePinia } from 'pinia'

import { useConnectionStore } from '../connection'

describe('useConnectionStore', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
  })

  const createConnection = (overrides: Record<string, unknown> = {}) => ({
    id: 'conn-1',
    name: 'Primary',
    groupId: 'group-a',
    host: 'localhost',
    port: 3306,
    username: 'root',
    password: 'secret',
    database: 'app',
    driver: 'mysql',
    timezone: 'UTC',
    ssh: { enabled: false, host: '', port: 22, user: '', password: '', privateKeyPath: '', privateKeyPass: '', hostKeyAlgo: '' },
    createdAt: '2026-01-01T00:00:00Z',
    updatedAt: '2026-01-01T00:00:00Z',
    ...overrides,
  })

  it('initializes with empty lists and closed dialog state', () => {
    const store = useConnectionStore()

    expect(store.connections).toEqual([])
    expect(store.groups).toEqual([])
    expect(store.selectedKey).toBeNull()
    expect(store.dialogVisible).toBe(false)
    expect(store.editingConnection).toBeNull()
  })

  it('setConnections replaces the connections list', () => {
    const store = useConnectionStore()

    const list = [createConnection(), createConnection({ id: 'conn-2', name: 'Replica' })]
    store.setConnections(list)

    expect(store.connections).toEqual(list)
  })

  it('connectionsByGroup getter filters by groupId', () => {
    const store = useConnectionStore()

    store.setConnections([
      createConnection(),
      createConnection({ id: 'conn-2', groupId: 'group-a' }),
      createConnection({ id: 'conn-3', groupId: 'group-b' }),
    ])

    expect(store.connectionsByGroup('group-a').map((conn) => conn.id)).toEqual(['conn-1', 'conn-2'])
    expect(store.connectionsByGroup('group-b').map((conn) => conn.id)).toEqual(['conn-3'])
  })

  it('openCreateDialog clears editingConnection and shows dialog', () => {
    const store = useConnectionStore()

    store.openEditDialog(createConnection())
    store.openCreateDialog()

    expect(store.editingConnection).toBeNull()
    expect(store.dialogVisible).toBe(true)
  })

  it('openEditDialog stores the connection being edited and shows dialog', () => {
    const store = useConnectionStore()

    const connection = createConnection()
    store.openEditDialog(connection)

    expect(store.editingConnection).toEqual(connection)
    expect(store.dialogVisible).toBe(true)
  })

  it('closeDialog hides dialog and clears editingConnection', () => {
    const store = useConnectionStore()

    store.openEditDialog(createConnection())
    store.closeDialog()

    expect(store.dialogVisible).toBe(false)
    expect(store.editingConnection).toBeNull()
  })

  it('addConnection pushes a connection into the array', () => {
    const store = useConnectionStore()

    const first = createConnection()
    const second = createConnection({ id: 'conn-2', name: 'Replica' })
    store.addConnection(first)
    store.addConnection(second)

    expect(store.connections).toEqual([first, second])
  })

  it('removeConnection filters out the matching id', () => {
    const store = useConnectionStore()

    store.setConnections([
      createConnection(),
      createConnection({ id: 'conn-2', name: 'Replica' }),
    ])

    store.removeConnection('conn-1')

    expect(store.connections.map((conn) => conn.id)).toEqual(['conn-2'])
  })
})
