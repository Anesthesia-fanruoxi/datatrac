<template>
  <section class="page-card">
    <div class="page-header">
      <div>
        <h2>凭据管理</h2>
        <p class="page-subtitle">凭据用于数据源连接时的身份认证。</p>
      </div>
      <el-button type="primary" @click="openCreate">新建凭据</el-button>
    </div>

    <el-table v-loading="loading" :data="records" border style="width: 100%">
      <el-table-column prop="name" label="名称" min-width="160" />
      <el-table-column prop="username" label="用户名" min-width="140" />
      <el-table-column prop="description" label="描述" min-width="200">
        <template #default="scope">{{ scope.row.description || '-' }}</template>
      </el-table-column>
      <el-table-column label="创建时间" min-width="160">
        <template #default="scope">{{ formatDate(scope.row.created_at) }}</template>
      </el-table-column>
      <el-table-column label="操作" min-width="200" fixed="right">
        <template #default="scope">
          <div class="action-row">
            <el-button size="small" plain @click="openEdit(scope.row.id)">编辑</el-button>
            <el-button size="small" type="danger" plain @click="removeItem(scope.row.id, scope.row.name)">删除</el-button>
          </div>
        </template>
      </el-table-column>
    </el-table>

    <el-dialog v-model="dialogVisible" :title="editingId ? '编辑凭据' : '新建凭据'" width="500px" destroy-on-close>
      <el-form :model="form" label-position="top">
        <el-form-item label="名称">
          <el-input v-model="form.name" placeholder="例如：测试环境凭据" />
        </el-form-item>
        <el-form-item label="用户名">
          <el-input v-model="form.username" placeholder="数据库用户名" />
        </el-form-item>
        <el-form-item label="密码">
          <el-input
            v-model="form.password"
            type="password"
            show-password
            :placeholder="editingId ? '留空表示不修改' : '请输入密码'"
          />
        </el-form-item>
        <el-form-item label="描述">
          <el-input v-model="form.description" type="textarea" :rows="3" placeholder="可选，描述此凭据的用途" />
        </el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="dialogVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="handleSave">保存</el-button>
      </template>
    </el-dialog>
  </section>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { credentialApi } from '../api/credential'
import type { CredentialRecord, CreateCredentialRequest } from '../types/credential'

const loading = ref(false)
const saving = ref(false)
const dialogVisible = ref(false)
const editingId = ref('')
const records = ref<CredentialRecord[]>([])

const form = reactive<CreateCredentialRequest>({
  name: '',
  username: '',
  password: '',
  description: ''
})

async function loadData() {
  loading.value = true
  try {
    records.value = await credentialApi.list()
  } finally {
    loading.value = false
  }
}

function resetForm() {
  editingId.value = ''
  form.name = ''
  form.username = ''
  form.password = ''
  form.description = ''
}

function openCreate() {
  resetForm()
  dialogVisible.value = true
}

async function openEdit(id: string) {
  const detail = await credentialApi.getById(id)
  editingId.value = detail.id
  form.name = detail.name
  form.username = detail.username
  form.password = ''
  form.description = detail.description || ''
  dialogVisible.value = true
}

async function handleSave() {
  if (!form.name.trim()) return ElMessage.warning('请输入名称')
  if (!form.username.trim()) return ElMessage.warning('请输入用户名')
  if (!editingId.value && !form.password) return ElMessage.warning('请输入密码')

  saving.value = true
  try {
    const payload: CreateCredentialRequest = {
      name: form.name.trim(),
      username: form.username.trim(),
      description: (form.description || '').trim()
    }
    if (form.password) {
      payload.password = form.password
    }

    if (editingId.value) {
      await credentialApi.update(editingId.value, payload)
      ElMessage.success('更新成功')
    } else {
      await credentialApi.create(payload)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await loadData()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function removeItem(id: string, name: string) {
  try {
    await ElMessageBox.confirm(
      `确定要删除凭据"${name}"吗？\n\n注意：如果有数据源正在使用此凭据，将无法删除。`,
      '删除确认',
      { type: 'warning' }
    )
    await credentialApi.remove(id)
    ElMessage.success('删除成功')
    await loadData()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error instanceof Error ? error.message : '删除失败')
    }
  }
}

function formatDate(dateStr?: string) {
  if (!dateStr) return '-'
  const date = new Date(dateStr)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit'
  })
}

onMounted(loadData)
</script>
