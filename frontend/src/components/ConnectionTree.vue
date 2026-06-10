<script setup lang="ts">
import { ref } from 'vue'
import type { TreeNode } from '../types/connection'

const props = defineProps<{
  nodes: TreeNode[]
  loadFn: (node: TreeNode, resolve: (children: TreeNode[]) => void) => void
}>()

const emit = defineEmits<{
  'node-click': [node: TreeNode]
  'node-dblclick': [node: TreeNode]
  'node-drag-end': [draggingNode: TreeNode, dropNode: TreeNode, dropType: string]
  'edit-connection': [connectionId: string]
  'delete-connection': [connectionId: string]
  'edit-group': [groupId: string]
  'delete-group': [groupId: string]
  'query-table': [node: TreeNode]
  'create-database': [connectionId: string, databaseName: string]
  'create-table': [connectionId: string, databaseName: string]
  'drop-database': [connectionId: string, databaseName: string]
  'drop-table': [connectionId: string, databaseName: string, tableName: string]
}>()

function allowDrag(node: any): boolean {
  const data = node.data ?? node
  return data.type === 'connection' || data.type === 'group'
}

function allowDrop(draggingNode: any, dropNode: any, type: string): boolean {
  const dragging = draggingNode.data ?? draggingNode
  const target = dropNode.data ?? dropNode

  // Can't drop onto itself
  if (dragging.key === target.key) return false

  // database/table nodes are not valid drop targets — silently reject
  if (target.type === 'database' || target.type === 'table') return false

  if (dragging.type === 'connection') {
    if (target.type === 'group') return type === 'inner'
    if (target.type === 'connection') return type === 'before' || type === 'after'
    return false
  }

  if (dragging.type === 'group') {
    if (target.type === 'group') return type === 'inner' || type === 'before' || type === 'after'
    if (target.type === 'connection') return type === 'before' || type === 'after'
    return false
  }

  return false
}

function handleDragEnd(draggingNode: any, dropNode: any, dropType: string) {
  if (!dropNode) return
  const dragging = draggingNode.data ?? draggingNode
  const target = dropNode.data ?? dropNode
  emit('node-drag-end', dragging, target, dropType)
}

const treeRef = ref()

function getIcon(type: string): string {
  switch (type) {
    case 'group': return '\u{1F4C1}'
    case 'connection': return '\u{1F517}'
    case 'database': return '\u{1F5C4}'
    case 'table': return '\u{1F4CB}'
    default: return '\u{1F4C4}'
  }
}

const ctxVisible = ref(false)
const ctxNode = ref<TreeNode | null>(null)
const ctxPos = ref({ x: 0, y: 0 })

function getMenuLabel(): string {
  return ctxNode.value?.type === 'group' ? '编辑分组' : '编辑连接'
}

function getDeleteLabel(): string {
  return ctxNode.value?.type === 'group' ? '删除分组' : '删除连接'
}

function handleNodeClick(data: TreeNode) {
  emit('node-click', data)
}

function handleNodeDblClick(data: TreeNode) {
  if (data.type === 'database' || data.type === 'table') {
    emit('node-dblclick', data)
  }
}

function handleContextMenu(event: MouseEvent, data: TreeNode) {
  event.preventDefault()
  event.stopPropagation()
  if (data.type === 'connection' || data.type === 'group' || data.type === 'table' || data.type === 'database') {
    ctxNode.value = data
    ctxPos.value = { x: event.clientX, y: event.clientY }
    ctxVisible.value = true
  }
}

function closeContextMenu() {
  ctxVisible.value = false
  ctxNode.value = null
}

function handleEdit() {
  if (!ctxNode.value) return
  if (ctxNode.value.type === 'connection') emit('edit-connection', ctxNode.value.key)
  else if (ctxNode.value.type === 'group') emit('edit-group', ctxNode.value.key)
  closeContextMenu()
}

function handleDelete() {
  if (!ctxNode.value) return
  if (ctxNode.value.type === 'connection') emit('delete-connection', ctxNode.value.key)
  else if (ctxNode.value.type === 'group') emit('delete-group', ctxNode.value.key)
  closeContextMenu()
}

function handleQueryTable() {
  if (!ctxNode.value) return
  if (ctxNode.value.type === 'table') emit('query-table', ctxNode.value)
  closeContextMenu()
}

function parseNodeKey(key: string): { connId: string; db: string; table: string } {
  const parts = key.split('/')
  const connId = parts[0]
  const table = parts.length > 2 ? parts[parts.length - 1] : ''
  const db = parts.slice(1, table ? -1 : parts.length).join('/')
  return { connId, db, table }
}

function handleCreateDatabase() {
  if (!ctxNode.value) return
  const { connId, db } = parseNodeKey(ctxNode.value.key)
  emit('create-database', connId, db)
  closeContextMenu()
}

function handleCreateTable() {
  if (!ctxNode.value) return
  const { connId, db } = parseNodeKey(ctxNode.value.key)
  emit('create-table', connId, db)
  closeContextMenu()
}

function handleDropDatabase() {
  if (!ctxNode.value) return
  const { connId, db } = parseNodeKey(ctxNode.value.key)
  emit('drop-database', connId, db)
  closeContextMenu()
}

function handleDropTable() {
  if (!ctxNode.value) return
  const { connId, db, table } = parseNodeKey(ctxNode.value.key)
  emit('drop-table', connId, db, table)
  closeContextMenu()
}

defineExpose({ treeRef })
</script>

<template>
  <div class="tree-container" @click="closeContextMenu">
    <el-tree
      ref="treeRef"
      :data="props.nodes"
      node-key="key"
      :indent="16"
      :expand-on-click-node="true"
      highlight-current
      draggable
      :allow-drag="allowDrag"
      :allow-drop="allowDrop"
      lazy
      :load="(node: any, resolve: any) => {
        if (node.level === 0) {
          resolve(props.nodes)
        } else {
          props.loadFn(node.data ?? node, resolve)
        }
      }"
      @node-click="handleNodeClick"
      @node-drag-end="handleDragEnd"
      @node-contextmenu="handleContextMenu"
    >
      <template #default="{ data }">
        <span class="custom-tree-node" @dblclick.stop="handleNodeDblClick(data)">
          <span class="tree-icon">{{ getIcon(data.type) }}</span>
          <span class="tree-label">{{ data.label }}</span>
        </span>
      </template>
    </el-tree>

    <Teleport to="body">
      <div v-if="ctxVisible" class="ctx-fixed" @click="closeContextMenu" />
      <div v-if="ctxVisible" class="ctx-menu" :style="{ left: ctxPos.x + 'px', top: ctxPos.y + 'px' }">
        <template v-if="ctxNode?.type === 'table'">
          <div class="ctx-item ctx-item--query" @click="handleQueryTable">🔍 查询表</div>
          <div class="ctx-item ctx-item--danger" @click="handleDropTable">🗑 删除表</div>
        </template>
        <template v-else-if="ctxNode?.type === 'database'">
          <div class="ctx-item" @click="handleCreateTable">📄 新建表</div>
          <div class="ctx-item ctx-item--danger" @click="handleDropDatabase">🗑 删除数据库</div>
        </template>
        <template v-else>
          <div class="ctx-item" @click="handleEdit">{{ getMenuLabel() }}</div>
          <div class="ctx-item ctx-item--danger" @click="handleDelete">{{ getDeleteLabel() }}</div>
          <div v-if="ctxNode?.type === 'connection'" class="ctx-item" @click="handleCreateDatabase">🗄 新建数据库</div>
        </template>
      </div>
    </Teleport>
  </div>
</template>

<style scoped>
.tree-container {
  padding: 4px 0;
  overflow-y: auto;
  flex: 1;
}

.custom-tree-node {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.tree-icon {
  flex-shrink: 0;
  font-size: 14px;
}

.tree-label {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* Override Element Plus tree node selected style */
:deep(.el-tree-node.is-current > .el-tree-node__content) {
  background-color: var(--color-tree-selected) !important;
}
</style>

<style>
.ctx-fixed { position: fixed; inset: 0; z-index: 9998; }
.ctx-menu {
  position: fixed; z-index: 9999;
  background: var(--color-dropdown-bg); border: 1px solid var(--color-border);
  border-radius: 6px; padding: 4px 0; min-width: 140px;
  box-shadow: 0 4px 16px var(--color-dropdown-shadow);
}
.ctx-item {
  padding: 6px 16px; font-size: 13px; cursor: pointer;
  color: var(--color-text); transition: background 0.1s;
  white-space: nowrap;
}
.ctx-item:hover { background: var(--color-dropdown-hover); }
.ctx-item--danger { color: #e74c3c; }
.ctx-item--query { border-top: 1px solid var(--color-border); margin-top: 2px; padding-top: 8px; }

/* Suppress emoji drop-indicator drawn by Element Plus during tree drag */
body .el-tree__drop-indicator {
  display: none !important;
}
</style>
