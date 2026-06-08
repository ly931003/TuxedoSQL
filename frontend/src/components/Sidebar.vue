<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useConnectionStore } from '../stores/connection'
import { useQueryStore } from '../stores/query'
import { useLayoutStore } from '../stores/layout'
import ConnectionTree from './ConnectionTree.vue'
import GroupDialog from './GroupDialog.vue'
import { ConnectionService } from '../../bindings/tuxedosql/internal/service'
import type { ConnectionGroup, Connection, TreeNode } from '../types/connection'

const store = useConnectionStore()
const queryStore = useQueryStore()
const layoutStore = useLayoutStore()
const treeCompRef = ref<InstanceType<typeof ConnectionTree> | null>(null)
const treeData = ref<TreeNode[]>([])
const groupDialogVisible = ref(false)
const editingGroup = ref<{ id: string; name: string; parentId: string } | null>(null)

function showToast(message: string, type: 'success' | 'error' = 'error') {
  if (type === 'success') ElMessage.success(message)
  else ElMessage.error(message)
}

function parseError(err: unknown): string {
  if (err instanceof Error) {
    try { const p = JSON.parse(err.message); if (p?.message) return String(p.message) } catch {}
    return err.message
  }
  if (err && typeof err === 'object') {
    const msg = (err as Record<string, unknown>).message
    if (typeof msg === 'string') return msg
  }
  const raw = String(err)
  try { const p = JSON.parse(raw); if (p?.message) return String(p.message) } catch {}
  return raw
}

function buildGroupSubtree(
  parentId: string,
  groups: ConnectionGroup[],
  connections: Connection[],
): TreeNode[] {
  const children: TreeNode[] = []

  for (const g of groups.filter(g => g.parentId === parentId)) {
    children.push({
      key: g.id, label: g.name, type: 'group',
      leaf: false,
      children: buildGroupSubtree(g.id, groups, connections),
    })
  }

  for (const c of connections.filter(c => c.groupId === parentId)) {
    children.push({
      key: c.id, label: c.name, type: 'connection', leaf: false,
    })
  }

  return children
}

function buildTreeNodes(groups: ConnectionGroup[], connections: Connection[]): TreeNode[] {
  return buildGroupSubtree('', groups, connections)
}

async function loadData() {
  try {
    const [conns, grps] = await Promise.all([
      ConnectionService.List(), ConnectionService.ListGroups(),
    ])
    store.setConnections(conns)
    store.setGroups(grps)
    treeData.value = buildTreeNodes(grps, conns)
  } catch (err) { showToast(parseError(err)) }
}

async function handleLoadNode(node: TreeNode, resolve: (children: TreeNode[]) => void) {
  if (node.type === 'group') {
    resolve(node.children ?? [])
    return
  }

  if (node.type === 'connection') {
    try {
      const databases = await ConnectionService.GetDatabases(node.key)
      const children: TreeNode[] = databases.map((db: string) => ({
        key: `${node.key}/${db}`, label: db, type: 'database', leaf: false,
      }))
      resolve(children)
    } catch (err) {
      showToast(parseError(err))
      resolve([])
    }
    return
  }

  if (node.type === 'database') {
    const parts = node.key.split('/')
    try {
      const tables = await ConnectionService.GetTables(parts[0], parts.slice(1).join('/'))
      const children: TreeNode[] = tables.map((t: string) => ({
        key: `${node.key}/${t}`, label: t, type: 'table', leaf: true,
      }))
      resolve(children)
    } catch (err) {
      showToast(parseError(err))
      resolve([])
    }
    return
  }

  resolve([])
}

function handleCreateConnection() { store.openCreateDialog() }
function handleEditConnection(connId: string) {
  const conn = store.connections.find(c => c.id === connId)
  if (conn) store.openEditDialog(conn)
}
function handleDeleteConnection(connId: string) {
  if (confirm('确定要删除此连接吗？')) {
    ConnectionService.Delete(connId).then(() => {
      store.removeConnection(connId)
      treeData.value = buildTreeNodes(store.groups, store.connections)
    }).catch((err: unknown) => showToast(parseError(err)))
  }
}

function handleCreateGroup(parentId: string = '') {
  editingGroup.value = null
  groupDialogVisible.value = true
}
function handleEditGroup(groupId: string) {
  const g = store.groups.find(x => x.id === groupId)
  if (g) { editingGroup.value = { id: g.id, name: g.name, parentId: g.parentId }; groupDialogVisible.value = true }
}
function handleDeleteGroup(groupId: string) {
  const g = store.groups.find(x => x.id === groupId)
  if (g && confirm(`确定要删除分组 "${g.name}" 吗？其中的连接将移至未分组。`)) {
    ConnectionService.DeleteGroup(groupId).then(() => {
      store.removeGroup(groupId)
      treeData.value = buildTreeNodes(store.groups, store.connections)
    }).catch((err: unknown) => showToast(parseError(err)))
  }
}

async function handleGroupSaved() {
  groupDialogVisible.value = false
  await loadData()
}

function handleNodeClick(node: TreeNode) {
  store.selectNode(node.key)
}

function handleNodeDblClick(node: TreeNode) {
  // Parse connectionID and database from node key (format: connID/dbName/...)
  const parts = node.key.split('/')
  const connectionId = parts[0]

  if (node.type === 'database') {
    const database = parts.slice(1).join('/')
    queryStore.openTab({ connectionId, database, sql: '' })
  } else if (node.type === 'table') {
    // parts: [connID, dbName, tableName]
    const tableName = parts[parts.length - 1]
    const dbName = parts.slice(1, -1).join('/')
    // 双击表名 → 打开表格视图
    queryStore.openTableView({
      connectionId: parts[0],
      database: dbName,
      tableName,
    })
  }
}

function handleQueryTable(node: TreeNode) {
  // Parse: connID/dbName/tableName
  const parts = node.key.split('/')
  const connectionId = parts[0]
  const tableName = parts[parts.length - 1]
  const dbName = parts.slice(1, -1).join('/')

  // 打开查询标签，预填 SELECT * FROM table
  queryStore.openTab({
    connectionId,
    database: dbName,
    sql: `SELECT * FROM \`${tableName}\` LIMIT 100`,
  })
}

async function handleNodeDragEnd(dragging: TreeNode, target: TreeNode, dropType: string) {
  if (dragging.type === 'connection') {
    const conn = store.connections.find(c => c.id === dragging.key)
    if (!conn) return
    let newGroupId = ''
    if (dropType === 'inner') {
      newGroupId = target.key
    } else {
      const targetConn = store.connections.find(c => c.id === target.key)
      newGroupId = targetConn?.groupId ?? ''
    }

    try {
      await ConnectionService.Update({
        id: conn.id, name: conn.name, groupId: newGroupId,
        host: conn.host, port: conn.port, username: conn.username,
        password: conn.password, database: conn.database,
      })
      await loadData()
      ElMessage.success(`已将 "${conn.name}" 移至${newGroupId ? '分组' : '未分组'}`)
    } catch (err) {
      showToast(parseError(err))
      await loadData()
    }
    return
  }

  // Dragging a group
  if (dragging.type === 'group') {
    const group = store.groups.find(g => g.id === dragging.key)
    if (!group) return
    let newParentId = ''
    if (dropType === 'inner') {
      newParentId = target.key
    } else {
      const targetConn = store.connections.find(c => c.id === target.key)
      if (targetConn) {
        newParentId = targetConn.groupId ?? ''
      } else {
        const targetGroup = store.groups.find(g => g.id === target.key)
        newParentId = targetGroup?.parentId ?? ''
      }
    }

    try {
      await ConnectionService.UpdateGroup({ id: group.id, name: group.name, parentId: newParentId })
      await loadData()
      ElMessage.success(`已将分组 "${group.name}" 移动`)
    } catch (err) {
      showToast(parseError(err))
      await loadData()
    }
  }
}

defineExpose({ loadData })
onMounted(() => loadData())
</script>

<template>
  <div class="sidebar" :style="{ width: layoutStore.leftSidebarWidth + 'px' }">
    <div class="sidebar-header">
      <h2>连接</h2>
      <div class="header-btns">
        <button class="btn-add" @click="handleCreateGroup()" title="新建分组">📁+</button>
        <button class="btn-add" @click="handleCreateConnection" title="新建连接">+</button>
      </div>
    </div>
    <ConnectionTree
      ref="treeCompRef"
      :nodes="treeData"
      :load-fn="handleLoadNode"
      @node-click="handleNodeClick"
      @node-dblclick="handleNodeDblClick"
      @node-drag-end="handleNodeDragEnd"
      @edit-connection="handleEditConnection"
      @delete-connection="handleDeleteConnection"
      @edit-group="handleEditGroup"
      @delete-group="handleDeleteGroup"
      @query-table="handleQueryTable"
    />
    <GroupDialog
      :visible="groupDialogVisible"
      :editing="editingGroup"
      :groups="store.groups"
      @saved="handleGroupSaved"
      @close="groupDialogVisible = false"
    />
  </div>
</template>

<style scoped>
.sidebar {
  height: 100%;
  background: var(--color-sidebar);
  border-right: 1px solid var(--color-border);
  display: flex; flex-direction: column;
  user-select: none;
}
.sidebar-header {
  display: flex; align-items: center; justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid var(--color-border);
}
.sidebar-header h2 {
  font-size: 13px; font-weight: 600;
  color: var(--color-text-secondary);
  text-transform: uppercase; letter-spacing: 0.5px;
  margin: 0;
}
.header-btns { display: flex; gap: 4px; }
.btn-add {
  width: 24px; height: 24px;
  border: none; border-radius: 4px;
  background: var(--color-hover);
  color: var(--color-text);
  font-size: 12px; line-height: 1;
  cursor: pointer;
  display: flex; align-items: center; justify-content: center;
  transition: background 0.15s;
}
.btn-add:hover { background: var(--color-accent); color: #fff; }
</style>
