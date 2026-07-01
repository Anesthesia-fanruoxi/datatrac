<template>
  <el-dialog v-model="visible" :title="form.id ? '编辑数据源' : '新建数据源'" width="560px" destroy-on-close>
    <el-form :model="form" label-position="top">
      <el-form-item label="名称">
        <el-input v-model="form.name" />
      </el-form-item>
      <el-form-item label="类型">
        <el-select v-model="form.type" style="width: 100%">
          <el-option label="MySQL" value="mysql" />
          <el-option label="Elasticsearch" value="elasticsearch" />
        </el-select>
      </el-form-item>
      <div class="wizard-grid two-cols">
        <el-form-item label="主机">
          <el-input v-model="form.host" placeholder="localhost" />
        </el-form-item>
        <el-form-item label="端口">
          <el-input-number v-model="form.port" :min="1" :max="65535" style="width: 100%" />
        </el-form-item>
      </div>
      <el-form-item label="认证方式">
        <el-radio-group v-model="form.auth_type">
          <el-radio value="manual">手动输入</el-radio>
          <el-radio value="credential">使用凭据</el-radio>
        </el-radio-group>
      </el-form-item>
      <template v-if="form.auth_type === 'credential'">
        <el-form-item label="凭据">
          <el-select v-model="form.credential_id" filterable style="width: 100%">
            <el-option v-for="item in credentials" :key="item.id" :label="`${item.name} (${item.username})`" :value="item.id" />
          </el-select>
        </el-form-item>
      </template>
      <template v-else>
        <el-form-item label="用户名">
          <el-input v-model="form.username" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input v-model="form.password" type="password" :placeholder="form.id ? '留空表示不修改' : '请输入密码'" show-password />
        </el-form-item>
      </template>
      <el-form-item v-if="form.type === 'mysql'" label="数据库名称">
        <el-input v-model="form.database_name" placeholder="可选" />
      </el-form-item>
    </el-form>
    <template #footer>
      <div class="wizard-footer">
        <el-button @click="visible = false">取消</el-button>
        <div>
          <el-button :loading="testing" @click="handleTest">测试连接</el-button>
          <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
        </div>
      </div>
    </template>
  </el-dialog>
</template>

<script setup lang="ts">
import { computed, reactive, watch } from 'vue'
import { ElMessage } from 'element-plus'
import type { CredentialOption, DataSourceFormData, DataSourcePayload } from '../../types/datasource'

const props = defineProps<{
  modelValue: boolean
  formData: DataSourceFormData
  credentials: CredentialOption[]
  saving: boolean
  testing: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: boolean]
  save: [payload: DataSourcePayload]
  test: [payload: DataSourcePayload]
}>()

const visible = computed({
  get: () => props.modelValue,
  set: (value: boolean) => emit('update:modelValue', value)
})

const form = reactive<DataSourceFormData>({
  id: '',
  name: '',
  type: 'mysql',
  host: '',
  port: 3306,
  auth_type: 'manual',
  credential_id: '',
  username: '',
  password: '',
  database_name: ''
})

watch(
  () => props.formData,
  (value) => {
    Object.assign(form, value)
  },
  { immediate: true, deep: true }
)

function buildPayload(): DataSourcePayload | null {
  if (!form.name.trim()) return ElMessage.warning('请输入名称'), null
  if (!form.host.trim()) return ElMessage.warning('请输入主机'), null
  if (!form.port) return ElMessage.warning('请输入端口'), null
  const payload: DataSourcePayload = {
    name: form.name.trim(),
    type: form.type,
    host: form.host.trim(),
    port: Number(form.port),
    database_name: form.database_name.trim()
  }
  if (form.auth_type === 'credential') {
    if (!form.credential_id) return ElMessage.warning('请选择凭据'), null
    payload.credential_id = form.credential_id
  } else {
    if (!form.username.trim()) return ElMessage.warning('请输入用户名'), null
    if (!form.id && !form.password) return ElMessage.warning('请输入密码'), null
    payload.username = form.username.trim()
    if (form.password) payload.password = form.password
  }
  return payload
}

function handleSave() {
  const payload = buildPayload()
  if (!payload) return
  emit('save', payload)
}

function handleTest() {
  const payload = buildPayload()
  if (!payload) return
  emit('test', payload)
}
</script>
