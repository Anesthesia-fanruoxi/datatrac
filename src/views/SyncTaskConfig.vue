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
      @create="handleCreate"
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
    width: 180,
    render: (row) => h(
      NSpace,
      {},
      {
        default: () => [
          h(
            NButton,
            { size: 'small', onClick: () => handleEditName(row) },
            { default: () => '修改名称', icon: () => h(NIcon, null, { default: () => h(EditIcon) }) }
          ),
          h(
            NButton,
            { size: 'small', type: 'primary', onClick: () => handleConfigure(row) },
            { default: () => '配置任务', icon: () => h(NIcon, null, { default: () => h(EditIcon) }) }
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
    batchSize: 2500,
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
      batchSize: 2500,
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

// 修改任务名称（只修改名称，不进入配置步骤）
async function handleEditName(row: SyncTask) {
  const newName = prompt('请输入新的任务名称：', row.name)
  if (newName && newName.trim() && newName !== row.name) {
    try {
      await syncTaskStore.updateTask(row.id, { ...row, name: newName.trim() })
      showSuccess('任务名称修改成功')
      await loadTasks()
    } catch (error) {
      handleApiError(error, '修改任务名称失败')
    }
  }
}

// 配置任务（进入完整配置步骤）
function handleConfigure(row: SyncTask) {
  isEdit.value = true
  formData.value = { ...row }
  showModal.value = true
}

async function handleCreate(data: { name: string; sourceType: any; targetType: any }) {
  try {
    // 前置配置完成，创建任务记录（只保存名称和类型）
    await syncTaskStore.createTask({
      name: data.name,
      sourceId: '', // 暂时为空，后续步骤中填写
      targetId: '', // 暂时为空，后续步骤中填写
      sourceType: data.sourceType,
      targetType: data.targetType,
      mysqlConfig: { databases: [] },
      esConfig: { indices: [] },
      syncConfig: {
        threadCount: 4,
        batchSize: 2500,
        errorStrategy: 'skip',
        tableExistsStrategy: 'drop'
      },
      status: 'idle'
    })
    
    showSuccess('任务创建成功，请在列表中点击"编辑"按钮进行配置')
    
    // 关闭对话框
    showModal.value = false
    
    // 刷新任务列表
    await loadTasks()
  } catch (error) {
    handleApiError(error, '创建任务失败')
  }
}

async function handleSubmit(data: Partial<SyncTask>) {
  try {
    console.log('SyncTaskConfig.handleSubmit - 收到提交数据:', data)
    
    let taskId: string
    
    if (data.id) {
      // 更新现有任务配置
      console.log('SyncTaskConfig.handleSubmit - 更新任务:', data.id)
      await syncTaskStore.updateTask(data.id, data as SyncTask)
      taskId = data.id
      showSuccess('任务配置保存成功，正在跳转到任务监控...')
    } else {
      // 创建新任务（完整配置）
      console.log('SyncTaskConfig.handleSubmit - 创建新任务')
      const newTask = {
        ...data,
        status: 'idle' as const
      } as Omit<SyncTask, 'id' | 'createdAt' | 'updatedAt'>
      
      taskId = await syncTaskStore.createTask(newTask)
      showSuccess('任务创建成功，正在跳转到任务监控...')
    }
    
    showModal.value = false
    await loadTasks()
    
    // 跳转到任务监控页面
    setTimeout(() => {
      router.push({
        path: '/task-monitor',
        query: { taskId }
      })
    }, 500)
  } catch (error) {
    console.error('SyncTaskConfig.handleSubmit - 错误:', error)
    handleApiError(error, '保存任务失败')
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
  padding: 24px;
  box-sizing: border-box;
}
</style>
