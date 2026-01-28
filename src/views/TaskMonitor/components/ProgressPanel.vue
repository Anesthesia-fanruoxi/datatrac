<template>
  <div class="progress-panel">
    <n-card
      v-if="taskUnits && taskUnits.length > 0"
      size="small"
      title="表/索引同步进度"
      :content-style="{ padding: 0, height: '100%', overflow: 'hidden' }"
    >
      <template #header-extra>
        <div class="progress-stats">
          <n-tag :bordered="false" size="small">总计: {{ taskUnits.length }}</n-tag>
          <n-tag :bordered="false" type="default" size="small">等待: {{ getCountByStatus('pending') }}</n-tag>
          <n-tag :bordered="false" type="info" size="small">进行中: {{ getCountByStatus('running') }}</n-tag>
          <n-tag :bordered="false" type="success" size="small">完成: {{ getCountByStatus('completed') }}</n-tag>
          <n-tag :bordered="false" type="error" size="small">失败: {{ getCountByStatus('failed') }}</n-tag>
        </div>
      </template>
      
      <div class="progress-list-wrapper">
        <div
          v-for="unit in taskUnits"
          :key="unit.id"
          :class="['progress-item', `status-${unit.status}`]"
        >
          <div class="progress-item-header">
            <n-tag :type="getStatusType(unit.status)" size="small">
              {{ getStatusText(unit.status) }}
            </n-tag>
            <span class="progress-item-name">{{ unit.name }}</span>
            <span v-if="unit.status === 'running' || unit.status === 'completed'" class="progress-item-percentage">
              {{ unit.percentage.toFixed(1) }}%
            </span>
          </div>
          
          <n-progress
            v-if="unit.status === 'running' || unit.status === 'completed'"
            type="line"
            :percentage="unit.percentage"
            :status="getProgressStatus(unit.status)"
            :height="6"
            :show-indicator="false"
          />
          
          <div v-if="unit.totalRecords > 0" class="progress-item-info">
            {{ unit.processedRecords.toLocaleString() }} / {{ unit.totalRecords.toLocaleString() }} 条记录
          </div>
          
          <div v-if="unit.status === 'failed' && unit.errorMessage" class="progress-item-error">
            <n-icon :size="14" color="#d03050">
              <AlertCircleIcon />
            </n-icon>
            {{ unit.errorMessage }}
          </div>
        </div>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { NCard, NTag, NProgress, NIcon } from 'naive-ui'
import { AlertCircle as AlertCircleIcon } from '@vicons/ionicons5'

interface TaskUnit {
  id: string
  name: string
  status: 'pending' | 'running' | 'completed' | 'failed'
  percentage: number
  processedRecords: number
  totalRecords: number
  errorMessage?: string
}

interface Props {
  taskUnits: TaskUnit[]
}

const props = defineProps<Props>()

function getCountByStatus(status: string): number {
  return props.taskUnits.filter(unit => unit.status === status).length
}

function getStatusText(status: string): string {
  const statusMap: Record<string, string> = {
    pending: '等待',
    running: '进行中',
    completed: '完成',
    failed: '失败'
  }
  return statusMap[status] || status
}

function getStatusType(status: string): any {
  const typeMap: Record<string, any> = {
    pending: 'default',
    running: 'info',
    completed: 'success',
    failed: 'error'
  }
  return typeMap[status] || 'default'
}

function getProgressStatus(status: string) {
  if (status === 'completed') return 'success'
  if (status === 'failed') return 'error'
  if (status === 'running') return 'info'
  return 'default'
}
</script>

<style scoped>
.progress-panel {
  flex: 1;
  height: 100%;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.progress-panel > .n-card {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.progress-stats {
  display: flex;
  gap: 8px;
}

.progress-list-wrapper {
  height: 100%;
  overflow-y: auto;
  padding: 16px;
}

.progress-item {
  padding: 12px;
  margin-bottom: 8px;
  border: 1px solid #e0e0e6;
  border-radius: 4px;
  background-color: #fafafa;
  transition: all 0.2s;
}

.progress-item.status-pending {
  border-color: #d0d0d0;
  background-color: #f5f5f5;
}

.progress-item.status-running {
  border-color: #2080f0;
  background-color: #f0f7ff;
  box-shadow: 0 2px 8px rgba(32, 128, 240, 0.1);
}

.progress-item.status-completed {
  border-color: #18a058;
  background-color: #f0fdf4;
}

.progress-item.status-failed {
  border-color: #d03050;
  background-color: #fef0f0;
}

.progress-item-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 8px;
}

.progress-item-name {
  flex: 1;
  font-size: 13px;
  color: #333;
  font-weight: 500;
}

.progress-item-percentage {
  font-weight: bold;
  font-size: 13px;
  color: #2080f0;
}

.progress-item-info {
  margin-top: 4px;
  font-size: 12px;
  color: #666;
}

.progress-item-error {
  margin-top: 8px;
  padding: 8px;
  background-color: #fff;
  border-radius: 4px;
  font-size: 12px;
  color: #d03050;
  display: flex;
  align-items: flex-start;
  gap: 4px;
}
</style>
