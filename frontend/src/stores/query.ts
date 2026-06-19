import { defineStore } from 'pinia'
import { SortOrder } from '../types/query'
import type { QueryTab, QueryResult, TabState, FilterGroup } from '../types/query'

let nextTabNum = 1

function genTitle(): string {
  return `Query ${nextTabNum++}`
}

function genID(): string {
  return `tab_${Date.now()}_${Math.random().toString(36).slice(2, 8)}`
}

// ── Tab view interface (subset needed for openTableView) ──

interface OpenTableViewParams {
  connectionId: string
  database: string
  tableName: string
  title?: string
}

interface OpenTabParams extends Omit<
  QueryTab,
  'id' | 'title' | 'result' | 'messages' | 'isExecuting' | 'viewType'
> {
  title?: string
}

interface AddTabParams extends Omit<OpenTabParams, 'sql'> {
  sql?: string
}

interface QueryState {
  tabs: QueryTab[]
  activeTabId: string | null
}

export const useQueryStore = defineStore('query', {
  state: (): QueryState => ({
    tabs: [],
    activeTabId: null,
  }),

  getters: {
    activeTab(state): QueryTab | null {
      return state.tabs.find((t) => t.id === state.activeTabId) ?? null
    },
    tabStates(state): TabState[] {
      return state.tabs.map((t) => ({
        id: t.id,
        title: t.title,
        connectionId: t.connectionId,
        database: t.database,
        sql: t.sql,
        viewType: t.viewType,
      }))
    },
  },

  actions: {
    addTab(tab: AddTabParams): QueryTab {
      return this.openTab({ ...tab, sql: tab.sql ?? '' })
    },

    openTab(
      tab: Omit<QueryTab, 'id' | 'title' | 'result' | 'messages' | 'isExecuting' | 'viewType'> & {
        title?: string
      },
    ): QueryTab {
      const id = genID()
      const title = tab.title ?? genTitle()
      const newTab: QueryTab = {
        id,
        title,
        connectionId: tab.connectionId,
        database: tab.database,
        sql: tab.sql,
        result: null,
        messages: [],
        isExecuting: false,
        viewType: 'query',
      }
      this.tabs = [...this.tabs, newTab]
      this.activeTabId = id
      return newTab
    },

    openTableView(tab: {
      connectionId: string
      database: string
      tableName: string
      title?: string
    }): QueryTab {
      const existingTab = this.tabs.find(
        (item) =>
          item.viewType === 'table' &&
          item.connectionId === tab.connectionId &&
          item.database === tab.database &&
          item.tableName === tab.tableName,
      )

      if (existingTab) {
        this.activeTabId = existingTab.id
        return existingTab
      }

      const id = genID()
      const title = tab.title ?? tab.tableName
      const newTab: QueryTab = {
        id,
        title,
        connectionId: tab.connectionId,
        database: tab.database,
        sql: `SELECT * FROM \`${tab.tableName}\` LIMIT 100`,
        result: null,
        messages: [],
        isExecuting: false,
        viewType: 'table',
        tableName: tab.tableName,
        page: 1,
        pageSize: 100,
        totalRows: 0,
        totalPages: 0,
        sortColumn: '',
        sortOrder: SortOrder.SortASC,
        filters: undefined,
      }
      this.tabs = [...this.tabs, newTab]
      this.activeTabId = id
      return newTab
    },

    closeTab(id: string): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return

      this.tabs = [...this.tabs.slice(0, idx), ...this.tabs.slice(idx + 1)]

      if (this.activeTabId === id) {
        this.activeTabId =
          this.tabs.length > 0 ? this.tabs[Math.min(idx, this.tabs.length - 1)].id : null
      }
    },

    removeTab(id: string): void {
      this.closeTab(id)
    },

    setActiveTab(id: string): void {
      this.activeTabId = id
    },

    updateSQL(id: string, sql: string): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      this.tabs = [
        ...this.tabs.slice(0, idx),
        { ...this.tabs[idx], sql },
        ...this.tabs.slice(idx + 1),
      ]
    },

    updateTabSQL(id: string, sql: string): void {
      this.updateSQL(id, sql)
    },

    setResult(id: string, result: QueryResult): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      const tab = this.tabs[idx]
      this.tabs = [...this.tabs.slice(0, idx), { ...tab, result }, ...this.tabs.slice(idx + 1)]
    },

    setExecuting(id: string, isExecuting: boolean): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      this.tabs = [
        ...this.tabs.slice(0, idx),
        { ...this.tabs[idx], isExecuting },
        ...this.tabs.slice(idx + 1),
      ]
    },

    addMessage(id: string, message: string): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      const tab = this.tabs[idx]
      this.tabs = [
        ...this.tabs.slice(0, idx),
        { ...tab, messages: [...tab.messages, message] },
        ...this.tabs.slice(idx + 1),
      ]
    },

    clearMessages(id: string): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      this.tabs = [
        ...this.tabs.slice(0, idx),
        { ...this.tabs[idx], messages: [] },
        ...this.tabs.slice(idx + 1),
      ]
    },

    renameTab(id: string, title: string): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      this.tabs = [
        ...this.tabs.slice(0, idx),
        { ...this.tabs[idx], title },
        ...this.tabs.slice(idx + 1),
      ]
    },

    // ── Table view actions ──

    setTablePage(id: string, page: number): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      this.tabs = [
        ...this.tabs.slice(0, idx),
        { ...this.tabs[idx], page },
        ...this.tabs.slice(idx + 1),
      ]
    },

    setTablePageSize(id: string, pageSize: number): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      this.tabs = [
        ...this.tabs.slice(0, idx),
        { ...this.tabs[idx], pageSize, page: 1 },
        ...this.tabs.slice(idx + 1),
      ]
    },

    setTableSorting(id: string, sortColumn: string, sortOrder: SortOrder): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      this.tabs = [
        ...this.tabs.slice(0, idx),
        { ...this.tabs[idx], sortColumn, sortOrder, page: 1 },
        ...this.tabs.slice(idx + 1),
      ]
    },

    setTableFilters(id: string, filters: FilterGroup | null | undefined): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      this.tabs = [
        ...this.tabs.slice(0, idx),
        { ...this.tabs[idx], filters: filters ?? undefined, page: 1 },
        ...this.tabs.slice(idx + 1),
      ]
    },

    setTablePageResult(id: string, total: number, totalPages: number): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      this.tabs = [
        ...this.tabs.slice(0, idx),
        { ...this.tabs[idx], totalRows: total, totalPages },
        ...this.tabs.slice(idx + 1),
      ]
    },

    updateLastExecutedSQL(id: string, sql: string): void {
      const idx = this.tabs.findIndex((t) => t.id === id)
      if (idx === -1) return
      this.tabs = [
        ...this.tabs.slice(0, idx),
        { ...this.tabs[idx], lastExecutedSQL: sql },
        ...this.tabs.slice(idx + 1),
      ]
    },
  },
})
