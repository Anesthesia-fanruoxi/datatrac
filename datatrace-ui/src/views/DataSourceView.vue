<template>
  <section class="page-card">
    <div class="page-header">
      <div>
        <h2>数据源配置</h2>
        <p class="page-subtitle">第一版已支持列表、新建、编辑、删除和测试连接。</p>
      </div>
      <el-button type="primary" @click="openCreate">新建数据源</el-button>
    </div>

    <el-table v-loading="loading" :data="records" border style="width: 100%">
      <el-table-column label="名称" min-width="160">
        <template #default="scope">
          <i class="ds-icon" :class="scope.row.type === 'mysql' ? 'bi-database-fill' : 'bi-search'"></i>
          <strong>{{ scope.row.name }}</strong>
        </template>
      </el-table-column>
      <el-table-column label="类型" min-width="120">
        <template #default="scope">
          <span class="ds-badge">{{ scope.row.type === 'mysql' ? 'MySQL' : 'Elasticsearch' }}</span>
        </template>
      </el-table-column>
      <el-table-column label="地址" min-width="180">
        <template #default="scope"><code>{{ scope.row.host }}:{{ scope.row.port }}</code></template>
      </el-table-column>
      <el-table-column label="连接状态" min-width="180">
        <template #default="scope">
          <span v-if="statusMap[scope.row.id] === 'success'" class="ds-status ds-status-ok">
            <i class="bi bi-check-circle-fill"></i> 正常
          </span>
          <span v-else-if="statusMap[scope.row.id] === 'failed'" class="ds-status ds-status-error">
            <i class="bi bi-x-circle-fill"></i> 失败
          </span>
          <span v-else class="ds-status ds-status-testing">
            <i class="bi bi-hourglass-split"></i> 检测中...
          </span>
        </template>
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

    <DataSourceFormDialog
      v-model="dialogVisible"
      :form-data="formData"
      :credentials="credentials"
      :saving="saving"
      :testing="testing"
      @save="handleSave"
      @test="handleTest"
    />
  </section>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import DataSourceFormDialog from '../components/datasource/DataSourceFormDialog.vue'
import { datasourceApi } from '../api/datasource'
import { credentialApi } from '../api/credential'
import { createDataSourceStatusSSE } from '../api/datasource-sse'
import type { CredentialOption, DataSourceFormData, DataSourceHealthStatus, DataSourcePayload, DataSourceRecord } from '../types/datasource'

const loading = ref(false)
const saving = ref(false)
const testing = ref(false)
const dialogVisible = ref(false)
const records = ref<DataSourceRecord[]>([])
const credentials = ref<CredentialOption[]>([])
const statusMap = reactive<Record<string, string>>({})
let statusSSE: EventSource | null = null

const formData = reactive<DataSourceFormData>({
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

async function loadData() {
  loading.value = true
  try {
    const [list, creds] = await Promise.all([datasourceApi.list(), credentialApi.list()])
    records.value = list
    credentials.value = creds
    records.value.forEach((item) => {
      if (!statusMap[item.id]) {
        statusMap[item.id] = 'testing'
      }
    })
  } finally {
    loading.value = false
  }
}

function startStatusSSE() {
  stopStatusSSE()
  statusSSE = createDataSourceStatusSSE((results: DataSourceHealthStatus[]) => {
    results.forEach((item) => {
      if (item.status === 'success') {
        statusMap[item.id] = 'success'
      } else if (item.status === 'failed') {
        statusMap[item.id] = 'failed'
      } else {
        statusMap[item.id] = 'testing'
      }
    })
  })
  statusSSE.onerror = () => {
    console.warn('数据源状态 SSE 连接异常，等待自动重连')
  }
}

function stopStatusSSE() {
  if (statusSSE) {
    statusSSE.close()
    statusSSE = null
  }
}

function resetForm() {
  Object.assign(formData, {
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
}

function openCreate() {
  resetForm()
  dialogVisible.value = true
}

async function openEdit(id: string) {
  const detail = await datasourceApi.getById(id)
  Object.assign(formData, {
    id: detail.id,
    name: detail.name,
    type: detail.type === 'elasticsearch' ? 'elasticsearch' : 'mysql',
    host: detail.host,
    port: detail.port,
    auth_type: detail.credential_id ? 'credential' : 'manual',
    credential_id: detail.credential_id || '',
    username: detail.username || '',
    password: '',
    database_name: detail.database_name || ''
  })
  dialogVisible.value = true
}

async function handleSave(payload: DataSourcePayload) {
  saving.value = true
  try {
    if (formData.id) {
      await datasourceApi.update(formData.id, payload)
      ElMessage.success('更新成功')
    } else {
      await datasourceApi.create(payload)
      ElMessage.success('创建成功')
    }
    dialogVisible.value = false
    await loadData()
    startStatusSSE()
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '保存失败')
  } finally {
    saving.value = false
  }
}

async function handleTest(payload: DataSourcePayload) {
  testing.value = true
  try {
    const result = await datasourceApi.testConnection(payload)
    ElMessage.success(`连接成功，版本：${result.version || '未知'}`)
  } catch (error) {
    ElMessage.error(error instanceof Error ? error.message : '连接失败')
  } finally {
    testing.value = false
  }
}

async function removeItem(id: string, name: string) {
  try {
    await ElMessageBox.confirm(`确定要删除数据源“${name}”吗？此操作不可恢复。`, '删除确认', { type: 'warning' })
    await datasourceApi.remove(id)
    ElMessage.success('删除成功')
    await loadData()
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error instanceof Error ? error.message : '删除失败')
    }
  }
}

onMounted(loadData)
onMounted(startStatusSSE)
onBeforeUnmount(stopStatusSSE)
</script>
