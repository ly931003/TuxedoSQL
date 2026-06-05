<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useConnectionStore } from '../stores/connection'
import { ConnectionService } from '../../bindings/tuxedosql/internal/service'
import type { TestResult } from '../types/connection'

const store = useConnectionStore()
const emit = defineEmits<{ saved: [] }>()

interface FormData {
  name: string
  groupId: string
  host: string
  port: number
  username: string
  password: string
  database: string
}

const form = reactive<FormData>({
  name: '', groupId: '', host: '127.0.0.1', port: 3306,
  username: 'root', password: '', database: '',
})

const testResult = reactive<TestResult>({ success: false, message: '' })
const isTesting = ref(false)
const isSaving = ref(false)

const visible = computed({
  get: () => store.dialogVisible,
  set: (v) => { if (!v) store.closeDialog() },
})

watch(() => store.editingConnection, (conn) => {
  if (conn) {
    (Object.keys(form) as (keyof FormData)[]).forEach(k => { (form as Record<string, unknown>)[k] = (conn as Record<string, unknown>)[k] })
  } else {
    form.name = ''; form.groupId = ''; form.host = '127.0.0.1'; form.port = 3306
    form.username = 'root'; form.password = ''; form.database = ''
  }
  testResult.success = false; testResult.message = ''
})

async function handleSave() {
  if (!form.name.trim()) return
  isSaving.value = true
  try {
    if (store.editingConnection) {
      const conn = await ConnectionService.Update({
        id: store.editingConnection.id,
        name: form.name, groupId: form.groupId, host: form.host,
        port: form.port, username: form.username,
        password: form.password, database: form.database,
      })
      if (conn) store.updateConnection(conn)
    } else {
      const conn = await ConnectionService.Create({
        name: form.name, groupId: form.groupId, host: form.host,
        port: form.port, username: form.username,
        password: form.password, database: form.database,
      })
      if (conn) store.addConnection(conn)
    }
    store.closeDialog()
    emit('saved')
  } catch (err) { console.error('保存连接失败:', err) }
  finally { isSaving.value = false }
}

async function handleTest() {
  isTesting.value = true; testResult.success = false; testResult.message = ''
  try {
    let connId = store.editingConnection?.id
    if (!connId) {
      const temp = await ConnectionService.Create({
        name: '__temp_test__', groupId: '', host: form.host,
        port: form.port, username: form.username,
        password: form.password, database: form.database,
      })
      if (!temp) { testResult.message = '创建临时连接失败'; return }
      connId = temp.id
      const result = await ConnectionService.TestConnection(connId)
      Object.assign(testResult, result)
      await ConnectionService.Delete(connId)
    } else {
      const result = await ConnectionService.TestConnection(connId)
      Object.assign(testResult, result)
    }
  } catch (err) { testResult.success = false; testResult.message = `测试失败: ${err}` }
  finally { isTesting.value = false }
}

function handleClose() { store.closeDialog() }
</script>

<template>
  <el-dialog
    v-model="visible"
    :title="store.editingConnection ? '编辑连接' : '新建连接'"
    :draggable="true"
    width="460px"
    :close-on-click-modal="false"
    @close="handleClose"
  >
    <el-form :model="form" label-position="top" size="small">
      <el-form-item label="连接名称">
        <el-input v-model="form.name" placeholder="例如：本地开发库" />
      </el-form-item>
      <el-form-item label="所属分组">
        <el-select v-model="form.groupId" placeholder="无(未分组)" clearable class="full-width">
          <el-option label="无(未分组)" value="" />
          <el-option v-for="g in store.groups" :key="g.id" :label="g.name" :value="g.id" />
        </el-select>
      </el-form-item>
      <el-form-item label="主机">
        <el-input v-model="form.host" placeholder="127.0.0.1" />
      </el-form-item>
      <el-row :gutter="12">
        <el-col :span="8">
          <el-form-item label="端口">
            <el-input-number v-model="form.port" :min="1" :max="65535" class="full-width" />
          </el-form-item>
        </el-col>
        <el-col :span="16">
          <el-form-item label="默认数据库">
            <el-input v-model="form.database" placeholder="(可选)" />
          </el-form-item>
        </el-col>
      </el-row>
      <el-form-item label="用户名">
        <el-input v-model="form.username" placeholder="root" />
      </el-form-item>
      <el-form-item label="密码">
        <el-input v-model="form.password" type="password" show-password placeholder="数据库密码" />
      </el-form-item>
      <div v-if="testResult.message" class="test-result" :class="{ 'test-result--ok': testResult.success }">
        {{ testResult.message }}
      </div>
    </el-form>

    <template #footer>
      <el-button @click="handleTest" :loading="isTesting" :disabled="isTesting">
        {{ isTesting ? '测试中...' : '测试连接' }}
      </el-button>
      <el-button type="primary" @click="handleSave" :loading="isSaving" :disabled="isSaving">
        {{ isSaving ? '保存中...' : '保存' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.full-width { width: 100%; }

.test-result {
  padding: 8px 12px; border-radius: 6px; font-size: 12px;
  background: rgba(231,76,60,0.10); color: #d63031;
}
.test-result--ok {
  background: rgba(46,204,113,0.12); color: #27ae60;
}
</style>
