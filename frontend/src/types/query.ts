import type {
  ColumnInfo,
  QueryResult,
  TabState,
  TableSchema,
  FilterCondition,
  TableDataParams,
  PageResult,
  UpdateRowParams,
  UpdateRowResult,
} from '../../bindings/tuxedosql/internal/model/models'
import { SortOrder, FilterOperator } from '../../bindings/tuxedosql/internal/model/models'

export type {
  ColumnInfo,
  QueryResult,
  TabState,
  TableSchema,
  FilterCondition,
  TableDataParams,
  PageResult,
  UpdateRowParams,
  UpdateRowResult,
}
export { SortOrder, FilterOperator }

export const FILTER_OPERATOR_LABELS: Record<string, string> = {
  eq: '等于',
  neq: '不等于',
  contains: '包含',
  gt: '大于',
  lt: '小于',
  isnull: '为空',
  notnull: '不为空',
}

export interface QueryTab {
  id: string
  title: string
  connectionId: string
  database: string
  sql: string
  viewType: 'query' | 'table'
  result: QueryResult | null
  messages: string[]
  isExecuting: boolean
  tableName?: string
  lastExecutedSQL?: string
  page?: number
  pageSize?: number
  totalRows?: number
  totalPages?: number
  sortColumn?: string
  sortOrder?: SortOrder
  filters?: FilterCondition[]
}

/** Tracks one pending cell edit before applying to the database. */
export interface DirtyChange {
  rowIndex: number
  columnName: string
  oldValue: unknown
  newValue: unknown
  pkValues: Record<string, unknown>
}

/** Identifies the cell currently being edited. null = no cell is in edit mode. */
export interface EditingCell {
  rowIndex: number
  columnName: string
}
