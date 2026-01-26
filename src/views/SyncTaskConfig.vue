<template>
  <div class="sync-task-config">
    <n-space vertical :size="16">
      <!-- 操作栏 -->
      <n-space justify="space-between">
        <n-button type="primary" @click="handleAdd">
          <template #icon>
            <n-icon><AddIcon /></n-icon>
          </template>
          创建任务
        </n-button>
        <n-button @click="loadTasks">
          <template #icon>
            <n-icon><RefreshIcon /></n-icon>
          </template>
          刷新
        </n-button>
      </n-space>

      <!-- 任务列表 -->
      <n-data-table
        :columns="columns"
        :data="syncTaskStore.tasks"
        :loading="syncTaskStore.loading"
        :pagination="pagination"
        :bordered="false"
      />
    </n-space>

    <!-- 任务创建/编辑向导 -->
    <SyncTaskWizard
      v-model="showModal"
      :is-edit="isEdit"
      :form-data="formData"
      @submit="handleSubmit"
    />

    <!-- 删除确认对话框 -->
    <n-modal
      v-model:show="showDeleteModal"
      preset="dialog"
      title="确认删除"
      content="确定要删除这个任务吗？此操作不可恢复。"
      positive-text="删除"
      negative-text="取消"
      @positive-click="confirmDelete"
    />
  </div>
</template>

<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import {
  NSpace,
  NButton,
  NDataTable,
  NModal,
  NIcon,
  NTag,
  type DataTableColumns
} from 'naive-ui'
import {
  Add as AddIcon,
  Refresh as RefreshIcon,
  Create as EditIcon,
  Trash as DeleteIcon
} from '@vicons/ionicons5'
import { useDataSourceStore } from '../stores/dataSource'
import { useSyncTaskStore } from '../stores/syncTask'
import { showSuccess, handleApiError } from '../utils/message'
import SyncTaskWizard from '../components/SyncTaskWizard/index.vue'
import type { SyncTask } from '../types'

const dataSourceStore = useDataSourceStore()
const syncTaskStore = useSyncTaskStore()
const router = useRouter()

// 表格配置
const pagination = { pageSize: 10 }

// 表格列定义
const columns: DataTableColumns<SyncTask> = [
  { title: '任务名称', key: 'name', width: 150 },
  {
    title: '源类型',
    key: 'sourceType',
    width: 120,
    render: (row) => h(
      NTag,
      { type: row.sourceType === 'mysql' ? 'info' : 'success' },
      { default: () => row.sourceType === 'mysql' ? 'MySQL' : 'Elasticsearch' }
    )
  },
  {
    title: '目标类型',
    key: 'targetType',
    width: 120,
    render: (row) => h(
      NTag,
      { type: row.targetType === 'mysql' ? 'info' : 'success' },
      { default: () => row.targetType === 'mysql' ? 'MySQL' : 'Elasticsearch' }
    )
  },
  {
    title: '状态',
    key: 'status',
    width: 100,
    render: (row) => {
      const statusMap: Record<string, { type: any; label: string }> = {
        idle: { type: 'default', label: '空闲' },
        running: { type: 'info', label: '运行中' },
        paused: { type: 'warning', label: '已暂停' },
        completed: { type: 'success', label: '已完成' },
        failed: { type: 'error', label: '失败' }
      }
      const status = statusMap[row.status] || statusMap.idle
      return h(NTag, { type: status.type }, { default: () => status.label })
    }
  },
  {
    title: '线程数',
    key: 'syncConfig.threadCount',
    width: 80,
    render: (row) => row.syncConfig.threadCount
  },
  {
    title: '批量大小',
    key: 'syncConfig.batchSize',
    width: 100,
    render: (row) => row.syncConfig.batchSize
  },
  {
    title: '创建时间',
    key: 'createdAt',
    width: 180,
    render: (row) => new Date(row.createdAt).toLocaleString()
  },
  {
    title: '操作',
    key: 'actions',
    width: 150,
    render: (row) => h(
      NSpace,
      {},
      {
        default: () => [
          h(
            NButton,
            { size: 'small', onClick: () => handleEdit(row) },
            { default: () => '编辑', icon: () => h(NIcon, null, { default: () => h(EditIcon) }) }
          ),
          h(
            NButton,
            { size: 'small', type: 'error', onClick: () => handleDelete(row) },
            { default: () => '删除', icon: () => h(NIcon, null, { default: () => h(DeleteIcon) }) }
          )
        ]
      }
    )
  }
]

// 表单相关
const showModal = ref(false)
const isEdit = ref(false)
const formData = ref<Partial<SyncTask> & { targetDatabase?: string }>({
  name: '',
  sourceId: '',
  targetId: '',
  sourceType: 'mysql',
  targetType: 'mysql',
  targetDatabase: '',
  mysqlConfig: { databases: [] },
  esConfig: { indices: [] },
  syncConfig: {
    threadCount: 4,
    batchSize: 1000,
    errorStrategy: 'skip',
    tableExistsStrategy: 'drop',
    dbNameTransform: {
      enabled: false,
      mode: 'prefix',
      sourcePattern: '',
      targetPattern: ''
    }
  },
  status: 'idle'
})

// 删除相关
const showDeleteModal = ref(false)
const deleteTarget = ref<SyncTask | null>(null)

// 方法
function handleAdd() {
  isEdit.value = false
  formData.value = {
    name: '',
    sourceId: '',
    targetId: '',
    sourceType: 'mysql',
    targetType: 'mysql',
    targetDatabase: '',
    mysqlConfig: { databases: [] },
    esConfig: { indices: [] },
    syncConfig: {
      threadCount: 4,
      batchSize: 1000,
      errorStrategy: 'skip',
      tableExistsStrategy: 'drop',
      dbNameTransform: {
        enabled: false,
        mode: 'prefix',
        sourcePattern: '',
        targetPattern: ''
      }
    },
    status: 'idle'
  }
  showModal.value = true
}

function handleEdit(row: SyncTask) {
  isEdit.value = true
  formData.value = { ...row }
  showModal.value = true
}

async function handleSubmit(data: Partial<SyncTask>) {
  try {
    let taskId: string
    
    if (isEdit.value && data.id) {
      await syncTaskStore.updateTask(data.id, data as SyncTask)
      taskId = data.id
      showSuccess('任务更新成功')
    } else {
      // createTask 返回新创建的任务 ID
      taskId = await syncTaskStore.createTask(data as SyncTask)
      showSuccess('任务创建成功，正在跳转到任务监控...')
    }
    
    showModal.value = false
    await loadTasks()
    
    // 如果是新建任务，跳转到任务监控页面
    if (!isEdit.value) {
      // 使用 setTimeout 确保提示消息显示后再跳转
      setTimeout(() => {
        router.push({
          path: '/task-monitor',
          query: { taskId }
        })
      }, 500)
    }
  } catch (error) {
    handleApiError(error, isEdit.value ? '更新任务失败' : '创建任务失败')
  }
}

function handleDelete(row: SyncTask) {
  deleteTarget.value = row
  showDeleteModal.value = true
}

async function confirmDelete() {
  if (!deleteTarget.value) return
  
  try {
    await syncTaskStore.deleteTask(deleteTarget.value.id)
    showSuccess('任务删除成功')
    await loadTasks()
  } catch (error) {
    handleApiError(error, '删除任务失败')
  } finally {
    deleteTarget.value = null
  }
}

async function loadTasks() {
  try {
    await syncTaskStore.fetchTasks()
    console.log('任务列表加载完成:', syncTaskStore.tasks)
  } catch (error) {
    handleApiError(error, '加载任务列表失败')
  }
}

onMounted(async () => {
  await dataSourceStore.fetchDataSources()
  await loadTasks()
})
</script>

<style scoped>
.sync-task-config {
  height: 100%;
}
</style>
