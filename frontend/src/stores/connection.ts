import { defineStore } from 'pinia'
import type { Connection, ConnectionGroup } from '../types/connection'

interface ConnectionState {
  connections: Connection[]
  groups: ConnectionGroup[]
  selectedKey: string | null
  dialogVisible: boolean
  editingConnection: Connection | null
}

export const useConnectionStore = defineStore('connection', {
  state: (): ConnectionState => ({
    connections: [],
    groups: [],
    selectedKey: null,
    dialogVisible: false,
    editingConnection: null,
  }),

  getters: {
    connectionsByGroup:
      (state) =>
      (groupId: string): Connection[] => {
        return state.connections.filter((c) => c.groupId === groupId)
      },
    ungroupedConnections: (state): Connection[] => {
      return state.connections.filter((c) => !c.groupId)
    },
  },

  actions: {
    setConnections(list: Connection[]) {
      this.connections = list
    },
    setGroups(list: ConnectionGroup[]) {
      this.groups = list
    },
    selectNode(key: string | null) {
      this.selectedKey = key
    },
    openCreateDialog() {
      this.editingConnection = null
      this.dialogVisible = true
    },
    openEditDialog(conn: Connection) {
      this.editingConnection = conn
      this.dialogVisible = true
    },
    closeDialog() {
      this.dialogVisible = false
      this.editingConnection = null
    },
    addConnection(conn: Connection) {
      this.connections.push(conn)
    },
    updateConnection(conn: Connection) {
      const idx = this.connections.findIndex((c) => c.id === conn.id)
      if (idx !== -1) {
        this.connections[idx] = conn
      }
    },
    removeConnection(id: string) {
      this.connections = this.connections.filter((c) => c.id !== id)
    },
    addGroup(group: ConnectionGroup) {
      this.groups.push(group)
    },
    removeGroup(id: string) {
      this.groups = this.groups.filter((g) => g.id !== id)
      this.connections = this.connections.map((c) => (c.groupId === id ? { ...c, groupId: '' } : c))
    },
  },
})
