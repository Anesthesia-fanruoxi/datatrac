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

    <!-- 统计信息 -->
    <div class="task-stats">
      <div class="stat-item">
        <span class="stat-label">总计:</span>
        <span class="stat-value">{{ statistics.total }}</span>
      </div>
      <div class="stat-item stat-waiting">
        <span class="stat-label">等待:</span>
        <span class="stat-value">{{ statistics.waiting }}</span>
      </div>
      <div class="stat-item stat-running">
        <span class="stat-label">进行中:</span>
        <span class="stat-value">{{ statistics.running }}</span>
      </div>
      <div class="stat-item stat-completed">
        <span class="stat-label">完成:</span>
        <span class="stat-value">{{ statistics.completed }}</span>
      </div>
      <div class="stat-item stat-failed">
        <span class="stat-label">失败:</span>
        <span class="stat-value">{{ statistics.failed }}</span>
      </div>
    </div>
    
    <div class="task-list-content">
      <!-- 进行中 -->
      <n-collapse :default-expanded-names="['running']" arrow-placement="right">
        <n-collapse-item name="running">
          <template #header>
            <div class="collapse-header">
              <n-icon :size="18" color="#2080f0"><PlayCircleIcon /></n-icon>
              <span class="collapse-title">进行中</span>
              <n-tag size="small" type="info" round>{{ runningTasks.length }}</n-tag>
            </div>
          </template>
          
          <div class="task-group">
            <div
              v-for="task in runningTasks"
              :key="task.id"
              :class="['task-item', { 'active': selectedTaskId === task.id }]"
              @click="onSelect(task.id)"
            >
              <div class="task-item-header">
                <span class="task-item-name">{{ task.name }}</span>
                <n-tag type="info" size="small">
                  {{ getStatusText(getTaskProgress(task.id)?.status) }}
                </n-tag>
              </div>
              <div class="task-item-info">
                {{ getDataSourceName(task.sourceId) }} → {{ getDataSourceName(task.targetId) }}
              </div>
              <n-progress
                type="line"
                :percentage="getTaskProgress(task.id)?.percentage || 0"
                :height="6"
                :show-indicator="false"
                status="info"
              />
            </div>
            
            <n-empty
              v-if="runningTasks.length === 0"
              description="暂无进行中的任务"
              size="small"
              style="margin: 20px 0"
            />
          </div>
        </n-collapse-item>

        <!-- 未启动 -->
        <n-collapse-item name="idle">
          <template #header>
            <div class="collapse-header">
              <n-icon :size="18" color="#909399"><PauseIcon /></n-icon>
              <span class="collapse-title">未启动</span>
              <n-tag size="small" type="default" round>{{ idleTasks.length }}</n-tag>
            </div>
          </template>
          
          <div class="task-group">
            <div
              v-for="task in idleTasks"
              :key="task.id"
              :class="['task-item', { 'active': selectedTaskId === task.id }]"
              @click="onSelect(task.id)"
            >
              <div class="task-item-header">
                <span class="task-item-name">{{ task.name }}</span>
                <n-tag
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
                type="line"
                :percentage="getTaskProgress(task.id)?.percentage || 0"
                :height="6"
                :show-indicator="false"
                :status="getProgressStatus(getTaskProgress(task.id)?.status)"
              />
            </div>
            
            <n-empty
              v-if="idleTasks.length === 0"
              description="暂无未启动任务"
              size="small"
              style="margin: 20px 0"
            />
          </div>
        </n-collapse-item>
      </n-collapse>
      
      <n-empty
        v-if="tasks.length === 0"
        description="暂无任务"
        style="margin-top: 40px"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NButton, NIcon, NTag, NProgress, NEmpty, NCollapse, NCollapseItem } from 'naive-ui'
import { 
  Refresh as RefreshIcon,
  PlayCircle as PlayCircleIcon,
  Pause as PauseIcon
} from '@vicons/ionicons5'

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

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 进行中：running 状态
const runningTasks = computed(() => {
  return props.tasks.filter(task => {
    const progress = props.getTaskProgress(task.id)
    return progress?.status === 'running'
  })
})

// 未启动：idle、pending、paused、failed、completed 等其他状态
const idleTasks = computed(() => {
  return props.tasks.filter(task => {
    const progress = props.getTaskProgress(task.id)
    const status = progress?.status || 'idle'
    return status !== 'running'
  })
})

// 统计信息
const statistics = computed(() => {
  const stats = {
    total: props.tasks.length,
    waiting: 0,
    running: 0,
    completed: 0,
    failed: 0
  }
  
  props.tasks.forEach(task => {
    const progress = props.getTaskProgress(task.id)
    const status = progress?.status || 'idle'
    
    if (status === 'running') {
      stats.running++
    } else if (status === 'completed') {
      stats.completed++
    } else if (status === 'failed') {
      stats.failed++
    } else {
      stats.waiting++
    }
  })
  
  return stats
})

function onSelect(taskId: string) {
  emit('select', taskId)
}

function onRefresh() {
  emit('refresh')
}

function getStatusText(status?: string): string {
  const statusMap: Record<string, string> = {
    running: '进行中',
    paused: '已暂停',
    completed: '完成',
    failed: '失败',
    pending: '待同步',
    idle: '未启动'
  }
  return statusMap[status || ''] || '未启动'
}

function getStatusTagType(status?: string): any {
  const typeMap: Record<string, any> = {
    running: 'info',
    paused: 'warning',
    completed: 'success',
    failed: 'error',
    pending: 'default',
    idle: 'default'
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

.task-stats {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 12px;
  background-color: #f8f9fa;
  border-bottom: 1px solid #e0e0e6;
  flex-shrink: 0;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 4px;
  padding: 4px 8px;
  background-color: #fff;
  border-radius: 4px;
  font-size: 12px;
}

.stat-label {
  color: #666;
}

.stat-value {
  font-weight: 600;
  color: #333;
}

.stat-waiting .stat-value {
  color: #909399;
}

.stat-running .stat-value {
  color: #2080f0;
}

.stat-completed .stat-value {
  color: #18a058;
}

.stat-failed .stat-value {
  color: #d03050;
}

.task-list-content {
  flex: 1;
  padding: 12px;
  overflow-y: auto;
}

.task-list-content :deep(.n-collapse) {
  background-color: transparent;
}

.task-list-content :deep(.n-collapse-item) {
  margin-bottom: 12px;
}

.task-list-content :deep(.n-collapse-item__header) {
  padding: 8px 12px;
  background-color: #f8f9fa;
  border-radius: 6px;
  font-weight: 500;
}

.collapse-header {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.collapse-title {
  flex: 1;
  font-size: 14px;
  font-weight: 500;
}

.task-group {
  padding-top: 8px;
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
  flex: 1;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  margin-right: 8px;
}

.task-item-info {
  font-size: 12px;
  color: #666;
  margin-bottom: 8px;
}
</style>
