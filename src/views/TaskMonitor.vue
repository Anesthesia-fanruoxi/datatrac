<template>
  <div class="task-monitor">
    <n-space vertical :size="16">
      <!-- 任务选择 -->
      <n-card title="选择任务">
        <n-space align="center">
          <n-select
            v-model:value="selectedTaskId"
            :options="taskOptions"
            placeholder="请选择要监控的任务"
            style="width: 300px"
            @update:value="handleTaskSelect"
          />
          <n-button @click="loadTasks">
            <template #icon>
              <n-icon><RefreshIcon /></n-icon>
            </template>
            刷新任务列表
          </n-button>
        </n-space>
      </n-card>

      <!-- 任务控制和状态 -->
      <n-card v-if="selectedTaskId" title="任务控制">
        <n-space vertical :size="16">
          <!-- 控制按钮 -->
          <n-space>
            <n-button
              type="primary"
              :disabled="isRunning || isPaused"
              :loading="taskMonitorStore.loading"
              @click="handleStart"
            >
              <template #icon>
                <n-icon><PlayIcon /></n-icon>
              </template>
              启动
            </n-button>
            <n-button
              type="warning"
              :disabled="!isRunning"
              :loading="taskMonitorStore.loading"
              @click="handlePause"
            >
              <template #icon>
                <n-icon><PauseIcon /></n-icon>
              </template>
              暂停
            </n-button>
            <n-button
              type="info"
              :disabled="!isPaused"
              :loading="taskMonitorStore.loading"
              @click="handleResume"
            >
              <template #icon>
                <n-icon><PlayIcon /></n-icon>
              </template>
              恢复
            </n-button>
          </n-space>

          <!-- 任务状态 -->
          <n-descriptions bordered :column="2">
            <n-descriptions-item label="任务状态">
              <n-tag :type="statusTagType">
                {{ statusText }}
              </n-tag>
            </n-descriptions-item>
            <n-descriptions-item label="当前表/索引">
              {{ progress?.currentTable || '-' }}
            </n-descriptions-item>
            <n-descriptions-item label="已同步记录">
              {{ progress?.processedRecords.toLocaleString() || 0 }}
            </n-descriptions-item>
            <n-descriptions-item label="总记录数">
              {{ progress?.totalRecords.toLocaleString() || 0 }}
            </n-descriptions-item>
            <n-descriptions-item label="同步速度">
              {{ progress?.speed.toFixed(2) || 0 }} 记录/秒
            </n-descriptions-item>
            <n-descriptions-item label="预计剩余时间">
              {{ formatTime(progress?.estimatedTime || 0) }}
            </n-descriptions-item>
            <n-descriptions-item label="开始时间">
              {{ progress?.startTime ? new Date(progress.startTime).toLocaleString() : '-' }}
            </n-descriptions-item>
          </n-descriptions>

          <!-- 进度条 -->
          <n-space vertical>
            <div style="display: flex; justify-content: space-between; align-items: center">
              <span>同步进度</span>
              <span style="font-weight: bold">{{ progress?.percentage.toFixed(2) || 0 }}%</span>
            </div>
            <n-progress
              type="line"
              :percentage="progress?.percentage || 0"
              :status="progressStatus"
              :height="24"
              :border-radius="4"
              :fill-border-radius="4"
            />
          </n-space>
        </n-space>
      </n-card>

      <!-- 错误日志 -->
      <n-card v-if="selectedTaskId" title="错误日志">
        <template #header-extra>
          <n-space>
            <n-tag type="error" v-if="taskMonitorStore.errors.length > 0">
              {{ taskMonitorStore.errors.length }} 个错误
            </n-tag>
            <n-button size="small" @click="loadErrors">
              <template #icon>
                <n-icon><RefreshIcon /></n-icon>
              </template>
              刷新
            </n-button>
          </n-space>
        </template>

        <n-empty
          v-if="taskMonitorStore.errors.length === 0"
          description="暂无错误日志"
        />

        <n-data-table
          v-else
          :columns="errorColumns"
          :data="taskMonitorStore.errors"
          :pagination="errorPagination"
          :max-height="400"
          :bordered="false"
        />
      </n-card>
    </n-space>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import {
  NSpace,
  NCard,
  NSelect,
  NButton,
  NIcon,
  NTag,
  NDescriptions,
  NDescriptionsItem,
  NProgress,
  NEmpty,
  NDataTable,
  type DataTableColumns
} from 'naive-ui'
import {
  Play as PlayIcon,
  Pause as PauseIcon,
  Refresh as RefreshIcon
} from '@vicons/ionicons5'
import { useSyncTaskStore } from '../stores/syncTask'
import { useTaskMonitorStore } from '../stores/taskMonitor'
import { showSuccess, handleApiError } from '../utils/message'
import type { ErrorLog } from '../types'

const syncTaskStore = useSyncTaskStore()
const taskMonitorStore = useTaskMonitorStore()

const selectedTaskId = ref<string>('')

// 任务选项
const taskOptions = computed(() => {
  return syncTaskStore.tasks.map(task => ({
    label: task.name,
    value: task.id
  }))
})

// 当前进度
const progress = computed(() => taskMonitorStore.progress)

// 任务状态
const isRunning = computed(() => progress.value?.status === 'running')
const isPaused = computed(() => progress.value?.status === 'paused')

const statusText = computed(() => {
  const statusMap: Record<string, string> = {
    running: '运行中',
    paused: '已暂停',
    completed: '已完成',
    failed: '失败'
  }
  return statusMap[progress.value?.status || ''] || '空闲'
})

const statusTagType = computed(() => {
  const typeMap: Record<string, any> = {
    running: 'info',
    paused: 'warning',
    completed: 'success',
    failed: 'error'
  }
  return typeMap[progress.value?.status || ''] || 'default'
})

const progressStatus = computed(() => {
  if (progress.value?.status === 'completed') return 'success'
  if (progress.value?.status === 'failed') return 'error'
  return 'info'
})

// 错误日志表格
const errorPagination = {
  pageSize: 10
}

const errorColumns: DataTableColumns<ErrorLog> = [
  {
    title: '时间',
    key: 'timestamp',
    width: 180,
    render: (row) => new Date(row.timestamp).toLocaleString()
  },
  {
    title: '错误类型',
    key: 'errorType',
    width: 150
  },
  {
    title: '错误消息',
    key: 'message',
    ellipsis: {
      tooltip: true
    }
  },
  {
    title: '相关数据',
    key: 'data',
    width: 200,
    ellipsis: {
      tooltip: true
    },
    render: (row) => row.data ? JSON.stringify(row.data) : '-'
  }
]

// 方法
function handleTaskSelect(taskId: string) {
  selectedTaskId.value = taskId
  loadProgress()
  loadErrors()
}

async function handleStart() {
  if (!selectedTaskId.value) return
  
  try {
    await taskMonitorStore.startTask(selectedTaskId.value)
    showSuccess('任务已启动')
  } catch (error) {
    handleApiError(error, '启动任务失败')
  }
}

async function handlePause() {
  if (!selectedTaskId.value) return
  
  try {
    await taskMonitorStore.pauseTask(selectedTaskId.value)
    showSuccess('任务已暂停')
  } catch (error) {
    handleApiError(error, '暂停任务失败')
  }
}

async function handleResume() {
  if (!selectedTaskId.value) return
  
  try {
    await taskMonitorStore.resumeTask(selectedTaskId.value)
    showSuccess('任务已恢复')
  } catch (error) {
    handleApiError(error, '恢复任务失败')
  }
}

async function loadTasks() {
  try {
    await syncTaskStore.fetchTasks()
  } catch (error) {
    handleApiError(error, '加载任务列表失败')
  }
}

async function loadProgress() {
  if (!selectedTaskId.value) return
  
  try {
    await taskMonitorStore.getProgress(selectedTaskId.value)
  } catch (error) {
    // 静默失败，因为可能任务还没开始
  }
}

async function loadErrors() {
  if (!selectedTaskId.value) return
  
  try {
    await taskMonitorStore.getErrors(selectedTaskId.value)
  } catch (error) {
    handleApiError(error, '加载错误日志失败')
  }
}

function formatTime(seconds: number): string {
  if (!seconds || seconds <= 0) return '-'
  
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)
  
  if (hours > 0) {
    return `${hours}小时${minutes}分钟${secs}秒`
  } else if (minutes > 0) {
    return `${minutes}分钟${secs}秒`
  } else {
    return `${secs}秒`
  }
}

// 定时刷新进度
let progressInterval: number | null = null

function startProgressPolling() {
  if (progressInterval) return
  
  progressInterval = window.setInterval(() => {
    if (selectedTaskId.value && isRunning.value) {
      loadProgress()
    }
  }, 1000) // 每秒刷新一次
}

function stopProgressPolling() {
  if (progressInterval) {
    clearInterval(progressInterval)
    progressInterval = null
  }
}

onMounted(async () => {
  await loadTasks()
  startProgressPolling()
  
  // 启动事件监听
  await taskMonitorStore.startEventListeners()
})

onUnmounted(() => {
  stopProgressPolling()
  taskMonitorStore.stopEventListeners()
})
</script>

<style scoped>
.task-monitor {
  height: 100%;
}
</style>
