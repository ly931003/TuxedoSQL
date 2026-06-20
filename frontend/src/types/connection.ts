export interface SSHConfig {
  enabled: boolean
  host: string
  port: number
  user: string
  password: string
  privateKeyPath: string
  privateKeyPass: string
}

export interface Connection {
  id: string
  name: string
  groupId: string
  host: string
  port: number
  username: string
  password: string
  database: string
  timezone: string
  ssh: SSHConfig
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
  timezone: string
  ssh: SSHConfig
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
  timezone: string
  ssh: SSHConfig
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
