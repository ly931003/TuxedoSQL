<script setup lang="ts">
import { ref } from 'vue'
import { FilterOperator, LogicOp } from '../../types/query'
import { FILTER_OPERATOR_LABELS } from '../../types/query'
import type { FilterGroup } from '../../types/query'

const props = defineProps<{
  columns: string[]
  loading?: boolean
}>()

const emit = defineEmits<{
  search: [group: FilterGroup | null]
}>()

// ── Local tree model ──

/** A tree node for the query builder UI. */
interface TreeNode {
  id: number
  type: 'group' | 'leaf'
  logic: LogicOp // only used when type='group'
  children: number[] // only used when type='group' (child node ids)
  column: string
  operator: FilterOperator
  value: string
  depth: number
}

let nextId = 1
function makeLeaf(): TreeNode {
  return {
    id: nextId++,
    type: 'leaf',
    logic: LogicOp.LogicAND,
    children: [],
    column: '',
    operator: FilterOperator.OpContains,
    value: '',
    depth: 0,
  }
}
function makeGroup(logic: LogicOp): TreeNode {
  return {
    id: nextId++,
    type: 'group',
    logic,
    children: [nextId++, nextId++],
    column: '',
    operator: FilterOperator.OpContains,
    value: '',
    depth: 0,
  }
}

const root = ref<TreeNode>({
  id: nextId++,
  type: 'group',
  logic: LogicOp.LogicAND,
  children: [nextId++, nextId++],
  column: '',
  operator: FilterOperator.OpContains,
  value: '',
  depth: 0,
})
const nodes = ref<Record<number, TreeNode>>({})
const flatOrder = ref<number[]>([]) // pre-order traversal order for rendering

function addLeaf(id: number) {
  const n = nextId++
  const leaf: TreeNode = {
    id: n,
    type: 'leaf',
    logic: LogicOp.LogicAND,
    children: [],
    column: '',
    operator: FilterOperator.OpContains,
    value: '',
    depth: 0,
  }
  nodes.value[n] = leaf
  const parent = nodes.value[id] ?? root.value
  parent.children = [...parent.children, n]
  rebuildFlat()
}

function addGroup(id: number, logic: LogicOp) {
  const ng = nextId++
  const nl1 = nextId++
  const nl2 = nextId++
  const group: TreeNode = {
    id: ng,
    type: 'group',
    logic,
    children: [nl1, nl2],
    column: '',
    operator: FilterOperator.OpContains,
    value: '',
    depth: 0,
  }
  nodes.value[ng] = group
  nodes.value[nl1] = {
    id: nl1,
    type: 'leaf',
    logic: LogicOp.LogicAND,
    children: [],
    column: '',
    operator: FilterOperator.OpContains,
    value: '',
    depth: 0,
  }
  nodes.value[nl2] = {
    id: nl2,
    type: 'leaf',
    logic: LogicOp.LogicAND,
    children: [],
    column: '',
    operator: FilterOperator.OpContains,
    value: '',
    depth: 0,
  }
  const parent = nodes.value[id] ?? root.value
  parent.children = [...parent.children, ng]
  rebuildFlat()
}

function removeNode(id: number) {
  function findParent(childId: number): { node: TreeNode; idx: number } | null {
    const ri = root.value.children.indexOf(childId)
    if (ri !== -1) return { node: root.value, idx: ri }
    for (const n of Object.values(nodes.value)) {
      const i = n.children.indexOf(childId)
      if (i !== -1) return { node: n, idx: i }
    }
    return null
  }
  const parent = findParent(id)
  if (!parent) return

  // If removing a leaf from a group with >2 children, just remove it
  if (parent.node.children.length > 2) {
    parent.node.children = parent.node.children.filter((c) => c !== id)
    delete nodes.value[id]
  } else {
    // Collapse: this is the last leaf in a group with exactly 2 children.
    // Find the sibling and promote it to replace this group.
    const siblingId = parent.node.children.find((c) => c !== id)
    if (parent.node === root.value) {
      // Root group: replace root children with [sibling]
      root.value.children = siblingId ? [siblingId] : []
      delete nodes.value[id]
    } else {
      // Find grandparent and replace this parent node with the sibling
      const gp = findParent(parent.node.id)
      if (gp) {
        gp.node.children = gp.node.children.map((c) => (c === parent.node.id ? siblingId! : c))
      }
      // Clean up parent and the removed child
      delete nodes.value[id]
      delete nodes.value[parent.node.id]
    }
  }
  rebuildFlat()
}

function rewriteNode(id: number, patch: Partial<TreeNode>) {
  const existing = nodes.value[id]
  if (existing) {
    nodes.value[id] = { ...existing, ...patch }
  }
}

function toggleLogic(id: number) {
  const existing = nodes.value[id]
  if (existing) {
    const nextLogic = existing.logic === LogicOp.LogicAND ? LogicOp.LogicOR : LogicOp.LogicAND
    nodes.value[id] = { ...existing, logic: nextLogic }
  } else if (root.value.id === id) {
    root.value = {
      ...root.value,
      logic: root.value.logic === LogicOp.LogicAND ? LogicOp.LogicOR : LogicOp.LogicAND,
    }
  }
}

function handleToggleLogic(id: number) {
  toggleLogic(id)
}

// ── Rebuild flat render order ──
function rebuildFlat() {
  const order: number[] = []
  function walk(id: number, depth: number) {
    ;(nodes.value[id] || root.value).depth = depth
    order.push(id)
    const n = nodes.value[id] ?? root.value
    if (n.type === 'group') {
      for (const c of n.children) walk(c, depth + 1)
    }
  }
  walk(root.value.id, 0)
  flatOrder.value = order
}

// Initialize
{
  const l1 = nextId++
  const l2 = nextId++
  root.value.children = [l1, l2]
  nodes.value[l1] = {
    id: l1,
    type: 'leaf',
    logic: LogicOp.LogicAND,
    children: [],
    column: '',
    operator: FilterOperator.OpContains,
    value: '',
    depth: 1,
  }
  nodes.value[l2] = {
    id: l2,
    type: 'leaf',
    logic: LogicOp.LogicAND,
    children: [],
    column: '',
    operator: FilterOperator.OpContains,
    value: '',
    depth: 1,
  }
  rebuildFlat()
}

// ── Convert tree to FilterGroup ──
function leafToFilterGroup(leaf: TreeNode): FilterGroup {
  return {
    logic: LogicOp.LogicAND,
    conditions: [],
    column: leaf.column,
    operator: leaf.operator,
    value: leaf.value,
  }
}

function treeToFilterGroup(node: TreeNode): FilterGroup | null {
  if (node.type === 'leaf') {
    if (!node.column.trim()) return null
    return leafToFilterGroup(node)
  }
  const children: FilterGroup[] = []
  for (const cid of node.children) {
    const child = nodes.value[cid]
    if (!child) continue
    const fg = treeToFilterGroup(child)
    if (fg) children.push(fg)
  }
  if (children.length === 0) return null
  if (children.length === 1) return children[0]
  return {
    logic: node.logic,
    conditions: children,
    column: '',
    operator: FilterOperator.OpEQ,
    value: '',
  }
}

function handleSearch() {
  const fg = treeToFilterGroup(root.value)
  emit('search', fg)
}

function handleReset() {
  flatOrder.value = []
  nodes.value = {}
  nextId = 1
  root.value = {
    id: nextId++,
    type: 'group',
    logic: LogicOp.LogicAND,
    children: [nextId++, nextId++],
    column: '',
    operator: FilterOperator.OpContains,
    value: '',
    depth: 0,
  }
  const l1 = nextId++
  const l2 = nextId++
  root.value.children = [l1, l2]
  nodes.value[l1] = {
    id: l1,
    type: 'leaf',
    logic: LogicOp.LogicAND,
    children: [],
    column: '',
    operator: FilterOperator.OpContains,
    value: '',
    depth: 1,
  }
  nodes.value[l2] = {
    id: l2,
    type: 'leaf',
    logic: LogicOp.LogicAND,
    children: [],
    column: '',
    operator: FilterOperator.OpContains,
    value: '',
    depth: 1,
  }
  rebuildFlat()
  emit('search', null)
}
</script>

<template>
  <div class="table-search">
    <div class="search-body">
      <template v-for="id in flatOrder" :key="id">
        <!-- Logic divider between siblings -->
        <div
          v-if="nodes[id]?.type === 'group' && nodes[id]!.children.length >= 2"
          class="group-wrapper"
          :style="{ paddingLeft: (nodes[id]?.depth ?? 0) * 16 + 'px' }"
        >
          <div class="group-header">
            <button class="logic-btn" :disabled="loading" @click="handleToggleLogic(id)">
              {{ nodes[id]?.logic === 'AND' ? '且' : '或' }}
            </button>
            <div class="group-actions">
              <button class="mini-btn" :disabled="loading" title="添加条件" @click="addLeaf(id)">
                +
              </button>
              <button
                class="mini-btn"
                :disabled="loading"
                title="添加括号组"
                @click="addGroup(id, LogicOp.LogicAND)"
              >
                ( )
              </button>
            </div>
          </div>
          <!-- spacer after group header; children render at their own depth -->
        </div>

        <div
          v-if="nodes[id]?.type === 'leaf'"
          class="leaf-row"
          :style="{ paddingLeft: (nodes[id]?.depth ?? 1) * 16 + 'px' }"
        >
          <span
            class="connector-text"
            v-if="
              flatOrder.indexOf(id) > 0 &&
              nodes[flatOrder[flatOrder.indexOf(id) - 1]]?.type !== 'group'
            "
          >
            {{
              /* find parent's logic */ (() => {
                const pid = Object.values(nodes).find((n) => n.children.includes(id))
                return pid ? pid.logic : root.logic
              })() === 'AND'
                ? '且'
                : '或'
            }}
          </span>
          <select
            class="col-select"
            :value="nodes[id]?.column ?? ''"
            :disabled="loading"
            @change="rewriteNode(id, { column: ($event.target as HTMLSelectElement).value })"
          >
            <option value="" disabled>选择列...</option>
            <option v-for="col in columns" :key="col" :value="col">{{ col }}</option>
          </select>
          <select
            class="op-select"
            :value="nodes[id]?.operator ?? ''"
            :disabled="loading"
            @change="
              rewriteNode(id, {
                operator: ($event.target as HTMLSelectElement).value as FilterOperator,
              })
            "
          >
            <option v-for="(label, op) in FILTER_OPERATOR_LABELS" :key="op" :value="op">
              {{ label }}
            </option>
          </select>
          <input
            v-if="nodes[id]?.operator !== 'isnull' && nodes[id]?.operator !== 'notnull'"
            class="val-input"
            :value="nodes[id]?.value ?? ''"
            type="text"
            placeholder="值..."
            :disabled="loading || !nodes[id]?.column"
            @input="rewriteNode(id, { value: ($event.target as HTMLInputElement).value })"
            @keydown.enter="handleSearch"
          />
          <button
            class="mini-btn mini-del"
            :disabled="loading"
            title="移除此条件"
            @click="removeNode(id)"
          >
            ×
          </button>
        </div>
      </template>
    </div>

    <div class="search-actions">
      <button class="mini-btn" :disabled="loading" @click="handleSearch" title="搜索">🔍</button>
      <button class="mini-btn" :disabled="loading" @click="handleReset" title="重置">✕</button>
    </div>
  </div>
</template>

<style scoped>
.table-search {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: 4px 8px;
  border-bottom: 1px solid var(--color-border);
  background: var(--color-sidebar);
  font-size: 12px;
}

.search-body {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.group-wrapper {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 4px;
}

.leaf-row {
  display: flex;
  align-items: center;
  gap: 4px;
  flex-wrap: wrap;
}

.connector-text {
  font-size: 10px;
  color: var(--color-accent);
  font-weight: 600;
  min-width: 18px;
  text-align: center;
}

.logic-btn {
  font-size: 10px;
  font-weight: 700;
  padding: 1px 6px;
  border: 1px solid var(--color-accent);
  border-radius: 3px;
  background: var(--color-accent);
  color: #fff;
  cursor: pointer;
  text-transform: uppercase;
}

.logic-btn:hover:not(:disabled) {
  opacity: 0.85;
}

.col-select,
.op-select {
  font-size: 11px;
  padding: 1px 4px;
  border: 1px solid var(--color-border);
  border-radius: 3px;
  background: var(--color-input-bg);
  color: var(--color-text);
  outline: none;
}

.col-select {
  min-width: 80px;
  max-width: 120px;
}

.val-input {
  font-size: 11px;
  padding: 1px 6px;
  border: 1px solid var(--color-border);
  border-radius: 3px;
  background: var(--color-input-bg);
  color: var(--color-text);
  outline: none;
  width: 100px;
}

.val-input:focus {
  border-color: var(--color-accent);
}

.mini-btn {
  width: 18px;
  height: 18px;
  border: 1px solid var(--color-border);
  border-radius: 3px;
  background: var(--color-input-bg);
  cursor: pointer;
  font-size: 11px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--color-text-secondary);
  padding: 0;
}

.mini-btn:hover:not(:disabled) {
  background: var(--color-hover);
}
.mini-btn:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}
.mini-del {
  color: #e74c3c;
}

.search-actions {
  display: flex;
  align-items: center;
  gap: 4px;
  padding-top: 2px;
  border-top: 1px solid var(--color-border);
}

.group-actions {
  display: flex;
  gap: 2px;
}
</style>
