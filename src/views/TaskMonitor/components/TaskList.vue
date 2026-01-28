<template>
  <div class="task-list-panel">
    <div class="task-list-header">
      <h3>任务列表</h3>
      <n-button size="small" @click="onRefresh">
        <template #icon>
          <n-icon><RefreshIcon /></n-icon>
        </template>
      </n-button>
    </div>
    
    <div class="task-list-content">
      <div
        v-for="task in tasks"
        :key="task.id"
        :class="['task-item', { 'active': selectedTaskId === task.id }]"
        @click="onSelect(task.id)"
      >
        <div class="task-item-header">
          <span class="task-item-name">{{ task.name }}</span>
          <n-tag
            v-if="getTaskProgress(task.id)"
            :type="getStatusTagType(getTaskProgress(task.id)?.status)"
            size="small"
          >
            {{ getStatusText(getTaskProgress(task.id)?.status) }}
          </n-tag>
        </div>
        <div class="task-item-info">
          {{ getDataSourceName(task.sourceId) }} → {{ getDataSourceName(task.targetId) }}
        </div>
        <n-progress
          v-if="getTaskProgress(task.id)"
          type="line"
          :percentage="getTaskProgress(task.id)?.percentage || 0"
          :height="6"
          :show-indicator="false"
          :status="getProgressStatus(getTaskProgress(task.id)?.status)"
        />
      </div>
      
      <n-empty
        v-if="tasks.length === 0"
        description="暂无任务"
        style="margin-top: 40px"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { NButton, NIcon, NTag, NProgress, NEmpty } from 'naive-ui'
import { Refresh as RefreshIcon } from '@vicons/ionicons5'

interface SyncTask {
  id: string
  name: string
  sourceId: string
  targetId: string
  sourceType: string
  targetType: string
  status: string
}

interface Props {
  tasks: SyncTask[]
  selectedTaskId: string
  getTaskProgress: (taskId: string) => any
  getDataSourceName: (dataSourceId: string) => string
}

interface Emits {
  (e: 'select', taskId: string): void
  (e: 'refresh'): void
}

defineProps<Props>()
const emit = defineEmits<Emits>()

function onSelect(taskId: string) {
  emit('select', taskId)
}

function onRefresh() {
  emit('refresh')
}

function getStatusText(status?: string): string {
  const statusMap: Record<string, string> = {
    running: '运行中',
    paused: '已暂停',
    completed: '已完成',
    failed: '失败'
  }
  return statusMap[status || ''] || '空闲'
}

function getStatusTagType(status?: string): any {
  const typeMap: Record<string, any> = {
    running: 'info',
    paused: 'warning',
    completed: 'success',
    failed: 'error'
  }
  return typeMap[status || ''] || 'default'
}

function getProgressStatus(status?: string) {
  if (status === 'completed') return 'success'
  if (status === 'failed') return 'error'
  return 'info'
}
</script>

<style scoped>
.task-list-panel {
  width: 320px;
  height: 100%;
  display: flex;
  flex-direction: column;
  background-color: #fff;
  border-right: 1px solid #e0e0e6;
}

.task-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #e0e0e6;
  flex-shrink: 0;
}

.task-list-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.task-list-content {
  flex: 1;
  padding: 12px;
  overflow-y: auto;
}

.task-item {
  padding: 12px;
  margin-bottom: 8px;
  border: 1px solid #e0e0e6;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  background-color: #fff;
}

.task-item:hover {
  border-color: #2080f0;
  box-shadow: 0 2px 8px rgba(32, 128, 240, 0.1);
}

.task-item.active {
  border-color: #2080f0;
  background-color: #f0f7ff;
  box-shadow: 0 2px 8px rgba(32, 128, 240, 0.15);
}

.task-item-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.task-item-name {
  font-weight: 500;
  font-size: 14px;
  color: #333;
}

.task-item-info {
  font-size: 12px;
  color: #666;
  margin-bottom: 8px;
}
</style>
