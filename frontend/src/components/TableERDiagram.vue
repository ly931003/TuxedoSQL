<script setup lang="ts">
import { ref, onMounted, watch, computed } from 'vue'
import { QueryService } from '../../bindings/tuxedosql/internal/service'
import type { ForeignKey } from '../../bindings/tuxedosql/internal/model/models'
import { parseError } from '../composables/parseError'

const props = defineProps<{
  connectionId: string
  database: string
  tableName: string
}>()

const foreignKeys = ref<ForeignKey[]>([])
const loading = ref(false)
const error = ref('')

// ── Layout constants ──
const TABLE_MIN_WIDTH = 160
const COL_HEIGHT = 18
const HEADER_HEIGHT = 28
const H_GAP = 60
const V_GAP = 50
const SVG_PAD = 20

// Column width estimation: ~9px per char
function colWidth(name: string): number {
  return Math.max(name.length * 9 + 16, 80)
}

// ── Fetch foreign keys ──
async function loadForeignKeys() {
  if (!props.connectionId || !props.database || !props.tableName) return
  loading.value = true
  error.value = ''
  try {
    foreignKeys.value = await QueryService.GetForeignKeys(
      props.connectionId,
      props.database,
      props.tableName,
    )
  } catch (err: unknown) {
    error.value = parseError(err)
  } finally {
    loading.value = false
  }
}

onMounted(loadForeignKeys)
watch(() => [props.connectionId, props.database, props.tableName], loadForeignKeys)

// ── Compute table info from FK data ──
interface TableNode {
  name: string
  columns: string[]       // columns involved in FK relationships
  isMain: boolean
  fkRefs: FkEdge[]        // FK edges where this table is source
}

interface FkEdge {
  sourceTable: string
  sourceColumn: string
  targetTable: string
  targetColumn: string
}

interface LayoutNode extends TableNode {
  x: number
  y: number
  width: number
  height: number
}

const layout = computed<{ nodes: LayoutNode[]; edges: FkEdge[]; svgWidth: number; svgHeight: number }>(() => {
  const fks = foreignKeys.value
  if (fks.length === 0) {
    return { nodes: [], edges: [], svgWidth: 200, svgHeight: 100 }
  }

  // Group by table: collect all columns involved in FKs
  const tableCols = new Map<string, Set<string>>()
  const edges: FkEdge[] = []

  for (const fk of fks) {
    // Source table
    if (!tableCols.has(fk.sourceTable)) tableCols.set(fk.sourceTable, new Set())
    tableCols.get(fk.sourceTable)!.add(fk.sourceColumn)
    // Target table
    if (!tableCols.has(fk.targetTable)) tableCols.set(fk.targetTable, new Set())
    tableCols.get(fk.targetTable)!.add(fk.targetColumn)

    edges.push({
      sourceTable: fk.sourceTable,
      sourceColumn: fk.sourceColumn,
      targetTable: fk.targetTable,
      targetColumn: fk.targetColumn,
    })
  }

  // Build table nodes
  const mainTable = props.tableName
  const outgoing: TableNode[] = []  // tables referenced BY main
  const incoming: TableNode[] = []  // tables that reference main
  const other: TableNode[] = []     // indirect relationships (not involving main)

  for (const [name, cols] of tableCols) {
    const colArr = [...cols].sort()
    const node: TableNode = {
      name,
      columns: colArr,
      isMain: name === mainTable,
      fkRefs: edges.filter(e => e.sourceTable === name),
    }

    if (name === mainTable) {
      // main table handled separately below
      continue
    }

    // Check if this table is purely outgoing from main, incoming to main, or indirect
    const hasFromMain = edges.some(e => e.sourceTable === mainTable && e.targetTable === name)
    const hasToMain = edges.some(e => e.sourceTable === name && e.targetTable === mainTable)

    if (hasFromMain && !hasToMain) outgoing.push(node)
    else if (hasToMain && !hasFromMain) incoming.push(node)
    else other.push(node)
  }

  // Width of each table node
  const getNodeWidth = (node: TableNode): number => {
    const nameW = node.name.length * 9 + 24
    const colW = Math.max(...node.columns.map(colWidth), 0)
    return Math.max(nameW, colW, TABLE_MIN_WIDTH)
  }

  const getNodeHeight = (node: TableNode): number => {
    return HEADER_HEIGHT + node.columns.length * COL_HEIGHT + 12
  }

  // Layout: main table center, outgoing above, incoming below, others to sides
  // Row layout helper
  const layoutRow = (nodes: TableNode[], startY: number): { nodes: LayoutNode[]; maxH: number } => {
    if (nodes.length === 0) return { nodes: [], maxH: 0 }
    const totalW = nodes.reduce((sum, n) => sum + getNodeWidth(n), 0) + (nodes.length - 1) * H_GAP
    let x = SVG_PAD + Math.max(0, (Math.max(totalW, 400) - totalW) / 2)
    const result: LayoutNode[] = []
    let maxH = 0
    for (const node of nodes) {
      const w = getNodeWidth(node)
      const h = getNodeHeight(node)
      result.push({ ...node, x, y: startY, width: w, height: h })
      x += w + H_GAP
      maxH = Math.max(maxH, h)
    }
    return { nodes: result, maxH }
  }

  const layoutNodes: LayoutNode[] = []

  // Position outgoing tables (above main)
  const outY = SVG_PAD
  const outRow = layoutRow(outgoing, outY)
  layoutNodes.push(...outRow.nodes)

  // Main table position
  const mainNode = tableCols.has(mainTable)
    ? {
        name: mainTable,
        columns: [...tableCols.get(mainTable)!].sort(),
        isMain: true,
        fkRefs: edges.filter(e => e.sourceTable === mainTable),
      } as TableNode
    : null

  const mainW = mainNode ? getNodeWidth(mainNode) : TABLE_MIN_WIDTH
  const mainH = mainNode ? getNodeHeight(mainNode) : HEADER_HEIGHT + 12
  const mainY = outRow.maxH > 0 ? outY + outRow.maxH + V_GAP : SVG_PAD

  if (mainNode) {
    // Center the main table horizontally
    const allNodes = [...outgoing, ...incoming, ...other]
    let totalAboveW = outgoing.reduce((s, n) => s + getNodeWidth(n), 0) + (outgoing.length - 1) * H_GAP
    const totalBelowW = [...incoming, ...other].reduce((s, n) => s + getNodeWidth(n), 0) + ([...incoming, ...other].length - 1) * H_GAP
    const maxRowW = Math.max(totalAboveW, totalBelowW, mainW)
    const mainX = SVG_PAD + Math.max(0, (maxRowW - mainW) / 2)
    layoutNodes.push({ ...mainNode, x: mainX, y: mainY, width: mainW, height: mainH })
  }

  // Position incoming + other tables (below main)
  const belowTables = [...incoming, ...other]
  const inY = mainY + mainH + V_GAP
  const inRow = layoutRow(belowTables, inY)
  layoutNodes.push(...inRow.nodes)

  // SVG dimensions
  const allXCoords = layoutNodes.map(n => n.x + n.width)
  const allYCoords = layoutNodes.map(n => n.y + n.height)
  const svgWidth = Math.max(...allXCoords, 400) + SVG_PAD
  const svgHeight = Math.max(...allYCoords, 100) + SVG_PAD

  return { nodes: layoutNodes, edges, svgWidth, svgHeight }
})

// ── Compute column Y positions within each table box ──
function columnY(colIndex: number): number {
  return HEADER_HEIGHT + 4 + colIndex * COL_HEIGHT + COL_HEIGHT / 2
}

// ── Compute edge path ──
function edgePath(
  srcNode: LayoutNode,
  srcColIndex: number,
  tgtNode: LayoutNode,
  tgtColIndex: number,
): string {
  const srcX = srcNode.x + 8
  const srcY = srcNode.y + columnY(srcColIndex)
  const tgtX = tgtNode.x + tgtNode.width - 8
  const tgtY = tgtNode.y + columnY(tgtColIndex)

  // Calculate control points for a nice Bezier curve
  const cpOffset = Math.abs(tgtY - srcY) * 0.4 + 30
  return `M ${srcX} ${srcY} C ${srcX - cpOffset} ${srcY}, ${tgtX + cpOffset} ${tgtY}, ${tgtX} ${tgtY}`
}

// ── Color helper ──
function edgeColor(index: number): string {
  const colors = [
    'var(--color-accent)',
    '#e74c3c',
    '#27ae60',
    '#f39c12',
    '#9b59b6',
    '#1abc9c',
    '#e67e22',
    '#3498db',
  ]
  return colors[index % colors.length]
}
</script>

<template>
  <div class="er-diagram-panel">
    <div v-if="loading" class="erd-loading">加载中...</div>
    <div v-else-if="error" class="erd-error">{{ error }}</div>
    <div v-else-if="foreignKeys.length === 0" class="erd-empty">该表无外键关系</div>
    <div v-else class="erd-svg-wrapper">
      <svg
        :width="layout.svgWidth"
        :height="layout.svgHeight"
        :viewBox="`0 0 ${layout.svgWidth} ${layout.svgHeight}`"
        class="erd-svg"
      >
        <defs>
          <marker
            v-for="(color, i) in ['var(--color-accent)', '#e74c3c', '#27ae60', '#f39c12', '#9b59b6', '#1abc9c', '#e67e22', '#3498db']"
            :key="i"
            :id="`arrow-${i}`"
            viewBox="0 0 10 10"
            refX="9"
            refY="5"
            markerWidth="6"
            markerHeight="6"
            orient="auto-start-reverse"
          >
            <path d="M 0 0 L 10 5 L 0 10 z" :fill="color" />
          </marker>
        </defs>

        <!-- FK edges -->
        <g v-for="(edge, ei) in layout.edges" :key="'e' + ei">
          <path
            v-for="(path, pi) in (() => {
              const srcNode = layout.nodes.find(n => n.name === edge.sourceTable)
              const tgtNode = layout.nodes.find(n => n.name === edge.targetTable)
              if (!srcNode || !tgtNode) return []
              const srcIdx = srcNode.columns.indexOf(edge.sourceColumn)
              const tgtIdx = tgtNode.columns.indexOf(edge.targetColumn)
              if (srcIdx === -1 || tgtIdx === -1) return []
              return [{ d: edgePath(srcNode, srcIdx, tgtNode, tgtIdx), color: edgeColor(ei) }]
            })()"
            :key="'p' + pi"
            :d="path.d"
            :stroke="path.color"
            stroke-width="1.5"
            fill="none"
            :marker-end="`url(#arrow-${ei % 8})`"
            opacity="0.6"
          />
        </g>

        <!-- Table boxes -->
        <g v-for="node in layout.nodes" :key="node.name">
          <!-- Box background -->
          <rect
            :x="node.x"
            :y="node.y"
            :width="node.width"
            :height="node.height"
            :rx="node.isMain ? 6 : 4"
            class="erd-box"
            :class="{ 'erd-box-main': node.isMain }"
          />
          <!-- Header -->
          <rect
            :x="node.x"
            :y="node.y"
            :width="node.width"
            :height="HEADER_HEIGHT"
            :rx="node.isMain ? 6 : 4"
            class="erd-header"
            :class="{ 'erd-header-main': node.isMain }"
          />
          <!-- Header bottom corners reset -->
          <rect
            v-if="node.isMain"
            :x="node.x"
            :y="node.y + HEADER_HEIGHT - 6"
            :width="node.width"
            :height="6"
            class="erd-header"
            :class="{ 'erd-header-main': node.isMain }"
          />
          <!-- Table name -->
          <text
            :x="node.x + node.width / 2"
            :y="node.y + HEADER_HEIGHT / 2 + 4"
            text-anchor="middle"
            class="erd-table-name"
            :class="{ 'erd-table-name-main': node.isMain }"
          >
            {{ node.name }}
          </text>
          <!-- Column entries -->
          <template v-for="(col, ci) in node.columns" :key="ci">
            <!-- Column name background (alternating stripe) -->
            <rect
              v-if="ci % 2 === 0"
              :x="node.x + 2"
              :y="node.y + HEADER_HEIGHT + ci * COL_HEIGHT"
              :width="node.width - 4"
              :height="COL_HEIGHT"
              class="erd-col-stripe"
            />
            <!-- Column name -->
            <text
              :x="node.x + 7"
              :y="node.y + columnY(ci) + 4"
              class="erd-col-text"
            >
              <tspan v-if="node.columns.filter(c => c === col).length > 1">
                {{ col }} ({{ ci + 1 }})
              </tspan>
              <tspan v-else>{{ col }}</tspan>
            </text>
          </template>
        </g>
      </svg>
    </div>
  </div>
</template>

<style scoped>
.er-diagram-panel {
  height: 100%;
  overflow: auto;
  padding: 8px;
}

.erd-loading,
.erd-empty {
  font-size: 12px;
  color: var(--color-text-secondary);
  text-align: center;
  padding: 24px 16px;
}

.erd-error {
  font-size: 12px;
  color: var(--color-danger);
  text-align: center;
  padding: 24px 16px;
}

.erd-svg-wrapper {
  overflow: auto;
  max-height: 100%;
  width: 100%;
}

.erd-svg {
  display: block;
  min-width: 100%;
}

/* Table box */
.erd-box {
  fill: var(--color-surface);
  stroke: var(--color-border);
  stroke-width: 1;
}

.erd-box-main {
  stroke: var(--color-accent);
  stroke-width: 2;
}

/* Header */
.erd-header {
  fill: var(--color-result-header-bg);
}

.erd-header-main {
  fill: var(--color-accent);
}

/* Table name */
.erd-table-name {
  font-size: 12px;
  font-weight: 600;
  fill: var(--color-text);
  font-family: var(--font-sans);
}

.erd-table-name-main {
  fill: var(--color-text-on-accent);
}

/* Column stripe */
.erd-col-stripe {
  fill: var(--color-result-stripe);
  opacity: 0.5;
}

/* Column text */
.erd-col-text {
  font-size: 10px;
  fill: var(--color-text-secondary);
  font-family: var(--font-mono);
}
</style>
