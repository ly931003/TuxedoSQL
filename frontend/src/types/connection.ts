export interface Connection {
  id: string
  name: string
  groupId: string
  host: string
  port: number
  username: string
  password: string
  database: string
  createdAt: string
  updatedAt: string
}

export interface ConnectionGroup {
  id: string
  name: string
  parentId: string
}

export interface CreateConnectionParams {
  name: string
  groupId: string
  host: string
  port: number
  username: string
  password: string
  database: string
}

export interface UpdateConnectionParams {
  id: string
  name: string
  groupId: string
  host: string
  port: number
  username: string
  password: string
  database: string
}

export interface TestResult {
  success: boolean
  message: string
}

export interface TreeNode {
  key: string
  label: string
  type: 'group' | 'connection' | 'database' | 'table'
  children?: TreeNode[]
  leaf: boolean
}

export interface UpdateGroupParams {
  id: string
  name: string
  parentId: string
}
