<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { useConnectionStore } from '../stores/connection'
import { useQueryStore } from '../stores/query'
import { ConnectionService } from '../../bindings/tuxedosql/internal/service'
import { ElMessageBox } from 'element-plus'
import type { TestResult } from '../types/connection'

const store = useConnectionStore()
const queryStore = useQueryStore()
const emit = defineEmits<{ saved: [] }>()

interface FormData {
  name: string
  groupId: string
  host: string
  port: number
  username: string
  password: string
  database: string
  timezone: string
  sshEnabled: boolean
  sshHost: string
  sshPort: number
  sshUser: string
  sshPassword: string
  sshPrivateKeyPath: string
  sshPrivateKeyPass: string
}

const form = reactive<FormData>({
  name: '',
  groupId: '',
  host: '127.0.0.1',
  port: 3306,
  username: 'root',
  password: '',
  database: '',
  timezone: 'Local',
  sshEnabled: false,
  sshHost: '',
  sshPort: 22,
  sshUser: '',
  sshPassword: '',
  sshPrivateKeyPath: '',
  sshPrivateKeyPass: '',
})

const testResult = reactive<TestResult>({ success: false, message: '' })
const isTesting = ref(false)
const isSaving = ref(false)
const showSSH = ref(false)
const visible = computed({
  get: () => store.dialogVisible,
  set: (v) => {
    if (!v) store.closeDialog()
  },
})

watch(
  () => store.editingConnection,
  (conn) => {
    if (conn) {
      const connAny = conn as Record<string, unknown>
      const formAny = form as Record<string, unknown>
      ;(Object.keys(form) as (keyof FormData)[]).forEach((k) => {
        if (k.startsWith('ssh')) {
          const sshKey = k.slice(3).replace(/^[A-Z]/, (c) => c.toLowerCase())
          formAny[k] = (conn.ssh as Record<string, unknown>)?.[sshKey] ?? formAny[k]
        } else {
          formAny[k] = connAny[k]
        }
      })
    } else {
      form.name = ''
      form.groupId = ''
      form.host = '127.0.0.1'
      form.port = 3306
      form.username = 'root'
      form.password = ''
      form.database = ''
      form.timezone = 'Local'
      form.sshEnabled = false
      form.sshHost = ''
      form.sshPort = 22
      form.sshUser = ''
      form.sshPassword = ''
      form.sshPrivateKeyPath = ''
      form.sshPrivateKeyPass = ''
    }
    testResult.success = false
    testResult.message = ''
  },
)

async function handleSave() {
  if (!form.name.trim()) return

  // 编辑模式下：检查是否有活跃的查询标签页使用该连接
  if (store.editingConnection) {
    const editingId = store.editingConnection.id
    const activeTabs = queryStore.tabs.filter((t) => t.connectionId === editingId)
    if (activeTabs.length > 0) {
      const tabNames = activeTabs.map((t) => t.title).join('、')
      const confirmed = await ElMessageBox.confirm(
        `该连接有 ${activeTabs.length} 个活跃的查询标签页（${tabNames}），保存后所有连接将被重新建立，这些标签页的查询状态可能受影响。是否继续？`,
        '连接配置变更提醒',
        { confirmButtonText: '继续保存', cancelButtonText: '取消', type: 'warning' },
      ).catch(() => false)
      if (!confirmed) return
    }
  }

    isSaving.value = true
    const ssh = {
      enabled: form.sshEnabled,
      host: form.sshHost,
      port: form.sshPort,
      user: form.sshUser,
      password: form.sshPassword,
      privateKeyPath: form.sshPrivateKeyPath,
      privateKeyPass: form.sshPrivateKeyPass,
    }
    try {
    if (store.editingConnection) {
      const conn = await ConnectionService.Update({
        id: store.editingConnection.id,
        name: form.name,
        groupId: form.groupId,
        host: form.host,
        port: form.port,
        username: form.username,
        password: form.password,
        database: form.database,
        timezone: form.timezone,
        ssh,
      })
      if (conn) store.updateConnection(conn)
    } else {
      const conn = await ConnectionService.Create({
        name: form.name,
        groupId: form.groupId,
        host: form.host,
        port: form.port,
        username: form.username,
        password: form.password,
        database: form.database,
        timezone: form.timezone,
        ssh,
      })
      if (conn) store.addConnection(conn)
    }
    store.closeDialog()
    emit('saved')
  } catch (err) {
    console.error('保存连接失败:', err)
  } finally {
    isSaving.value = false
  }
}

async function handleTest() {
  isTesting.value = true
  testResult.success = false
  testResult.message = ''
  try {
    const ssh = {
      enabled: form.sshEnabled,
      host: form.sshHost,
      port: form.sshPort,
      user: form.sshUser,
      password: form.sshPassword,
      privateKeyPath: form.sshPrivateKeyPath,
      privateKeyPass: form.sshPrivateKeyPass,
    }
    let connId = store.editingConnection?.id
    if (!connId) {
      const temp = await ConnectionService.Create({
        name: '__temp_test__',
        groupId: '',
        host: form.host,
        port: form.port,
        username: form.username,
        password: form.password,
        database: form.database,
        timezone: form.timezone,
        ssh,
      })
      if (!temp) {
        testResult.message = '创建临时连接失败'
        return
      }
      connId = temp.id
      const result = await ConnectionService.TestConnection(connId)
      Object.assign(testResult, result)
      await ConnectionService.Delete(connId)
    } else {
      const result = await ConnectionService.TestConnection(connId)
      Object.assign(testResult, result)
    }
  } catch (err) {
    testResult.success = false
    testResult.message = `测试失败: ${err}`
  } finally {
    isTesting.value = false
  }
}

function handleClose() {
  store.closeDialog()
}
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
      <el-form-item label="时区">
        <el-select
          v-model="form.timezone"
          filterable
          allow-create
          placeholder="选择或输入时区"
          class="full-width"
        >
          <el-option-group label="常用">
            <el-option label="本机时区" value="Local" />
            <el-option label="UTC" value="UTC" />
          </el-option-group>
          <el-option-group label="亚洲">
            <el-option label="Asia/Shanghai" value="Asia/Shanghai" />
            <el-option label="Asia/Tokyo" value="Asia/Tokyo" />
            <el-option label="Asia/Hong_Kong" value="Asia/Hong_Kong" />
            <el-option label="Asia/Singapore" value="Asia/Singapore" />
            <el-option label="Asia/Kolkata" value="Asia/Kolkata" />
          </el-option-group>
          <el-option-group label="欧洲">
            <el-option label="Europe/London" value="Europe/London" />
            <el-option label="Europe/Berlin" value="Europe/Berlin" />
            <el-option label="Europe/Paris" value="Europe/Paris" />
            <el-option label="Europe/Moscow" value="Europe/Moscow" />
          </el-option-group>
          <el-option-group label="美洲">
            <el-option label="America/New_York" value="America/New_York" />
            <el-option label="America/Los_Angeles" value="America/Los_Angeles" />
            <el-option label="America/Chicago" value="America/Chicago" />
            <el-option label="America/Sao_Paulo" value="America/Sao_Paulo" />
          </el-option-group>
          <el-option-group label="固定偏移">
            <el-option label="+08:00" value="+08:00" />
            <el-option label="+00:00" value="+00:00" />
            <el-option label="-05:00" value="-05:00" />
          </el-option-group>
        </el-select>
      </el-form-item>

      <!-- SSH 隧道配置 -->
      <el-divider />
      <el-form-item>
        <el-checkbox v-model="form.sshEnabled" @change="showSSH = form.sshEnabled">
          通过 SSH 隧道连接
        </el-checkbox>
      </el-form-item>
      <template v-if="form.sshEnabled || showSSH">
        <el-form-item label="SSH 主机">
          <el-input v-model="form.sshHost" placeholder="例如：192.168.1.1" />
        </el-form-item>
        <el-row :gutter="12">
          <el-col :span="8">
            <el-form-item label="SSH 端口">
              <el-input-number v-model="form.sshPort" :min="1" :max="65535" class="full-width" />
            </el-form-item>
          </el-col>
          <el-col :span="16">
            <el-form-item label="SSH 用户">
              <el-input v-model="form.sshUser" placeholder="root" />
            </el-form-item>
          </el-col>
        </el-row>
        <el-form-item label="SSH 密码">
          <el-input v-model="form.sshPassword" type="password" show-password placeholder="SSH 登录密码（可选）" />
        </el-form-item>
        <el-form-item label="私钥路径">
          <el-input v-model="form.sshPrivateKeyPath" placeholder="~/.ssh/id_rsa（可选）" />
        </el-form-item>
        <el-form-item label="私钥口令">
          <el-input v-model="form.sshPrivateKeyPass" type="password" show-password placeholder="加密私钥的口令（可选）" />
        </el-form-item>
      </template>
      <div
        v-if="testResult.message"
        class="test-result"
        :class="{ 'test-result--ok': testResult.success }"
      >
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
.full-width {
  width: 100%;
}

.test-result {
  padding: 8px 12px;
  border-radius: 6px;
  font-size: 12px;
  background: rgba(231, 76, 60, 0.1);
  color: #d63031;
}
.test-result--ok {
  background: rgba(46, 204, 113, 0.12);
  color: #27ae60;
}
</style>
