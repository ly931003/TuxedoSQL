<script setup lang="ts">
import { computed, reactive, ref, watch } from 'vue'
import { ConnectionService } from '../../bindings/tuxedosql/internal/service'
import type { ConnectionGroup } from '../types/connection'

const props = defineProps<{
  visible: boolean
  editing: { id: string; name: string; parentId: string } | null
  groups: ConnectionGroup[]
}>()
const emit = defineEmits<{ saved: []; close: [] }>()

const form = reactive({ name: '', parentId: '' })
const saving = ref(false)

const dialogVisible = computed({
  get: () => props.visible,
  set: (v) => { if (!v) emit('close') },
})

// Groups available as parent (exclude self when editing)
const parentOptions = computed(() => {
  if (props.editing) {
    return props.groups.filter(g => g.id !== props.editing!.id)
  }
  return props.groups
})

watch(() => props.visible, (v) => {
  if (v) {
    form.name = props.editing?.name || ''
    form.parentId = props.editing?.parentId || ''
  }
})

async function handleSave() {
  if (!form.name.trim()) return
  saving.value = true
  try {
    if (props.editing) {
      await ConnectionService.UpdateGroup({
        id: props.editing.id,
        name: form.name.trim(),
        parentId: form.parentId,
      })
    } else {
      await ConnectionService.CreateGroup({
        name: form.name.trim(),
        parentId: form.parentId,
      })
    }
    emit('saved')
  } catch (err) {
    console.error('保存分组失败:', err)
  } finally { saving.value = false }
}
</script>

<template>
  <el-dialog
    v-model="dialogVisible"
    :title="props.editing ? '编辑分组' : '新建分组'"
    :draggable="true"
    width="380px"
    :close-on-click-modal="false"
    @close="emit('close')"
  >
    <el-form :model="form" label-position="top" size="small" @submit.prevent="handleSave">
      <el-form-item label="分组名称">
        <el-input v-model="form.name" placeholder="例如：生产环境" @keyup.enter="handleSave" />
      </el-form-item>
      <el-form-item label="父分组">
        <el-select v-model="form.parentId" placeholder="无（顶级分组）" clearable class="full-width">
          <el-option label="无（顶级分组）" value="" />
          <el-option
            v-for="g in parentOptions"
            :key="g.id"
            :label="g.name"
            :value="g.id"
          />
        </el-select>
      </el-form-item>
    </el-form>

    <template #footer>
      <el-button @click="emit('close')">取消</el-button>
      <el-button type="primary" @click="handleSave" :loading="saving" :disabled="saving">
        {{ saving ? '保存中...' : '保存' }}
      </el-button>
    </template>
  </el-dialog>
</template>

<style scoped>
.full-width { width: 100%; }
</style>
