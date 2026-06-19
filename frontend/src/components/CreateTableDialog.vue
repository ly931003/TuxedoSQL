<script setup lang="ts">
import { ref, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { ConnectionService } from '../../bindings/tuxedosql/internal/service'
import * as models from '../../bindings/tuxedosql/internal/model/models'

const props = defineProps<{
  visible: boolean
  connectionId: string
  databaseName: string
}>()

const emit = defineEmits<{
  close: []
  saved: []
}>()

const tableName = ref('')
const comment = ref('')
const charset = ref('utf8mb4')
const collation = ref('utf8mb4_unicode_ci')
const saving = ref(false)

const charsetList = ref<models.CharsetInfo[]>([])
const collationList = ref<string[]>([])
const loadingMeta = ref(false)

async function loadCharsets() {
  if (!props.connectionId) return
  loadingMeta.value = true
  try {
    charsetList.value = (await ConnectionService.GetCharsets(props.connectionId)) || []
    if (charsetList.value.length > 0) {
      charset.value = charsetList.value[0].charset
      await loadCollations(charset.value)
    }
  } catch {
    /* silently fallback to defaults */
  } finally {
    loadingMeta.value = false
  }
}

async function loadCollations(cs: string) {
  if (!props.connectionId || !cs) return
  try {
    collationList.value = (await ConnectionService.GetCollations(props.connectionId, cs)) || []
    if (!collationList.value.includes(collation.value)) {
      collation.value = collationList.value[0] || ''
    }
  } catch {
    /* silently fallback */
  }
}

watch(
  () => props.visible,
  (v) => {
    if (v) loadCharsets()
  },
)
watch(charset, (cs) => {
  if (cs) loadCollations(cs)
})

interface ColumnRow {
  name: string
  dataType: string
  nullable: boolean
  defaultValue: string
  autoIncrement: boolean
  unsigned: boolean
  comment: string
  isPrimaryKey: boolean
}

const columns = ref<ColumnRow[]>([
  {
    name: 'id',
    dataType: 'INT',
    nullable: false,
    defaultValue: '',
    autoIncrement: true,
    unsigned: true,
    comment: '主键ID',
    isPrimaryKey: true,
  },
  {
    name: '',
    dataType: 'VARCHAR(255)',
    nullable: true,
    defaultValue: '',
    autoIncrement: false,
    unsigned: false,
    comment: '',
    isPrimaryKey: false,
  },
])

function addColumn() {
  columns.value = [
    ...columns.value,
    {
      name: '',
      dataType: 'VARCHAR(255)',
      nullable: true,
      defaultValue: '',
      autoIncrement: false,
      unsigned: false,
      comment: '',
      isPrimaryKey: false,
    },
  ]
}

function removeColumn(index: number) {
  if (columns.value.length <= 1) return
  columns.value = columns.value.filter((_, i) => i !== index)
}

async function handleCreate() {
  if (!tableName.value.trim()) {
    ElMessage.warning('请输入表名')
    return
  }
  const validColumns = columns.value.filter((c) => c.name.trim())
  if (validColumns.length === 0) {
    ElMessage.warning('至少定义一个列')
    return
  }

  saving.value = true
  try {
    const params = new models.CreateTableParams({
      connectionId: props.connectionId,
      databaseName: props.databaseName,
      tableName: tableName.value.trim(),
      charset: charset.value,
      collation: collation.value,
      comment: comment.value,
      columns: validColumns.map(
        (c) =>
          new models.ColumnDef({
            name: c.name.trim(),
            dataType: c.dataType,
            nullable: c.nullable,
            defaultValue: c.defaultValue,
            autoIncrement: c.autoIncrement,
            unsigned: c.unsigned,
            comment: c.comment,
            isPrimaryKey: c.isPrimaryKey,
          }),
      ),
    })
    const result = await ConnectionService.CreateTable(params)
    if (result) {
      ElMessage({ message: result.sql, type: 'success', duration: 4000 })
    }
    emit('saved')
  } catch (err: unknown) {
    const msg = err instanceof Error ? err.message : String(err)
    ElMessage.error(msg)
  } finally {
    saving.value = false
  }
}
</script>

<template>
  <el-dialog
    :model-value="visible"
    title="新建表"
    width="720px"
    :close-on-click-modal="false"
    @close="emit('close')"
  >
    <div class="ct-body">
      <!-- Basic info -->
      <div class="ct-row-basic">
        <label>表名</label>
        <input v-model="tableName" class="ct-input" placeholder="表名" />
        <label>注释</label>
        <input v-model="comment" class="ct-input" placeholder="表注释（可选）" />
        <label>字符集</label>
        <select v-model="charset" class="ct-input ct-input--sm" :disabled="loadingMeta">
          <option v-for="c in charsetList" :key="c.charset" :value="c.charset">
            {{ c.charset }}
          </option>
        </select>
        <label>排序规则</label>
        <select
          v-model="collation"
          class="ct-input ct-input--sm"
          :disabled="loadingMeta || collationList.length === 0"
        >
          <option v-for="c in collationList" :key="c" :value="c">{{ c }}</option>
        </select>
      </div>

      <!-- Column grid -->
      <div class="ct-col-header">
        <span class="col-h col-h--name">列名</span>
        <span class="col-h col-h--type">类型</span>
        <span class="col-h col-h--chk">非空</span>
        <span class="col-h col-h--chk">无符号</span>
        <span class="col-h col-h--chk">自增</span>
        <span class="col-h col-h--chk">主键</span>
        <span class="col-h col-h--def">默认值</span>
        <span class="col-h col-h--cmt">注释</span>
        <span class="col-h col-h--act"></span>
      </div>
      <div v-for="(col, i) in columns" :key="i" class="ct-col-row">
        <input v-model="col.name" class="ct-cell ct-cell--name" placeholder="列名" />
        <input v-model="col.dataType" class="ct-cell ct-cell--type" placeholder="INT" />
        <span class="ct-cell ct-cell--chk"><input type="checkbox" v-model="col.nullable" /></span>
        <span class="ct-cell ct-cell--chk"><input type="checkbox" v-model="col.unsigned" /></span>
        <span class="ct-cell ct-cell--chk"
          ><input type="checkbox" v-model="col.autoIncrement"
        /></span>
        <span class="ct-cell ct-cell--chk"
          ><input type="checkbox" v-model="col.isPrimaryKey"
        /></span>
        <input v-model="col.defaultValue" class="ct-cell ct-cell--def" placeholder="NULL" />
        <input v-model="col.comment" class="ct-cell ct-cell--cmt" placeholder="注释" />
        <button
          class="ct-cell ct-cell--act ct-btn-del"
          @click="removeColumn(i)"
          :disabled="columns.length <= 1"
        >
          ×
        </button>
      </div>
      <button class="ct-add-col" @click="addColumn">+ 添加列</button>
    </div>
    <template #footer>
      <button class="ct-btn ct-btn--cancel" @click="emit('close')">取消</button>
      <button class="ct-btn ct-btn--confirm" :disabled="saving" @click="handleCreate">
        {{ saving ? '创建中...' : '创建表' }}
      </button>
    </template>
  </el-dialog>
</template>

<style scoped>
.ct-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.ct-row-basic {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.ct-row-basic label {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text);
  white-space: nowrap;
}
.ct-input {
  font-size: 12px;
  padding: 3px 8px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-input-bg);
  color: var(--color-text);
  outline: none;
  flex: 1;
  min-width: 80px;
}
.ct-input--sm {
  max-width: 100px;
  flex: 0;
}
.ct-input:focus {
  border-color: var(--color-accent);
}

.ct-col-header {
  display: flex;
  gap: 4px;
  font-size: 11px;
  font-weight: 600;
  color: var(--color-text-secondary);
  padding: 4px 0;
  border-bottom: 1px solid var(--color-border);
}
.col-h--name {
  flex: 2;
  min-width: 80px;
}
.col-h--type {
  flex: 2;
  min-width: 80px;
}
.col-h--chk {
  width: 36px;
  text-align: center;
}
.col-h--def {
  flex: 1;
  min-width: 50px;
}
.col-h--cmt {
  flex: 1.5;
  min-width: 60px;
}
.col-h--act {
  width: 24px;
}

.ct-col-row {
  display: flex;
  gap: 4px;
  align-items: center;
}
.ct-cell {
  font-size: 12px;
  padding: 3px 4px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-input-bg);
  color: var(--color-text);
  outline: none;
}
.ct-cell:focus {
  border-color: var(--color-accent);
}
.ct-cell--name {
  flex: 2;
  min-width: 80px;
}
.ct-cell--type {
  flex: 2;
  min-width: 80px;
}
.ct-cell--chk {
  width: 36px;
  border: none;
  background: transparent;
  display: flex;
  justify-content: center;
}
.ct-cell--def {
  flex: 1;
  min-width: 50px;
}
.ct-cell--cmt {
  flex: 1.5;
  min-width: 60px;
}
.ct-cell--act {
  width: 24px;
}

.ct-btn-del {
  border: none;
  background: transparent;
  color: #e74c3c;
  font-size: 16px;
  cursor: pointer;
  padding: 0;
}
.ct-btn-del:disabled {
  opacity: 0.3;
  cursor: not-allowed;
}

.ct-add-col {
  font-size: 12px;
  border: 1px dashed var(--color-border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-accent);
  cursor: pointer;
  padding: 4px;
  text-align: left;
}
.ct-add-col:hover {
  background: var(--color-hover);
}

.ct-btn {
  font-size: 13px;
  padding: 6px 16px;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.ct-btn--cancel {
  background: var(--color-hover);
  color: var(--color-text);
  margin-right: 8px;
}
.ct-btn--confirm {
  background: var(--color-accent);
  color: #fff;
}
</style>
