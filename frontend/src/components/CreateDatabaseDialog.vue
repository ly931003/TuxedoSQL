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

const dbName = ref(props.databaseName)
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

async function handleCreate() {
  if (!dbName.value.trim()) {
    ElMessage.warning('请输入数据库名')
    return
  }
  saving.value = true
  try {
    const params = new models.CreateDatabaseParams({
      connectionId: props.connectionId,
      databaseName: dbName.value.trim(),
      charset: charset.value,
      collation: collation.value,
    })
    const result = await ConnectionService.CreateDatabase(params)
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
    title="新建数据库"
    width="420px"
    :close-on-click-modal="false"
    @close="emit('close')"
  >
    <div class="db-body">
      <label class="db-label">数据库名</label>
      <input
        v-model="dbName"
        class="db-input"
        placeholder="请输入数据库名..."
        @keydown.enter="handleCreate"
      />

      <label class="db-label">字符集</label>
      <select v-model="charset" class="db-input" :disabled="loadingMeta">
        <option v-for="c in charsetList" :key="c.charset" :value="c.charset">
          {{ c.charset }} - {{ c.description }}
        </option>
      </select>

      <label class="db-label">排序规则</label>
      <select
        v-model="collation"
        class="db-input"
        :disabled="loadingMeta || collationList.length === 0"
      >
        <option v-for="c in collationList" :key="c" :value="c">{{ c }}</option>
      </select>
    </div>
    <template #footer>
      <button class="db-btn db-btn--cancel" @click="emit('close')">取消</button>
      <button class="db-btn db-btn--confirm" :disabled="saving" @click="handleCreate">
        {{ saving ? '创建中...' : '创建' }}
      </button>
    </template>
  </el-dialog>
</template>

<style scoped>
.db-body {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.db-label {
  font-size: 12px;
  font-weight: 500;
  color: var(--color-text);
}
.db-input {
  font-size: 13px;
  padding: 6px 10px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: var(--color-input-bg);
  color: var(--color-text);
  outline: none;
}
.db-input:focus {
  border-color: var(--color-accent);
}
.db-btn {
  font-size: 13px;
  padding: 6px 16px;
  border: none;
  border-radius: var(--radius-sm);
  cursor: pointer;
}
.db-btn--cancel {
  background: var(--color-hover);
  color: var(--color-text);
  margin-right: 8px;
}
.db-btn--confirm {
  background: var(--color-accent);
  color: #fff;
}
</style>
