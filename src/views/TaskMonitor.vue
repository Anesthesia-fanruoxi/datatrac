<template>
  <div class="task-monitor">
    <n-layout has-sider style="height: 100%">
      <!-- 左侧：任务列表 -->
      <n-layout-sider
        bordered
        :width="320"
        :native-scrollbar="false"
        style="background-color: #fff"
      >
        <div class="task-list-header">
          <h3>任务列表</h3>
          <n-button size="small" @click="loadTasks">
            <template #icon>
              <n-icon><RefreshIcon /></n-icon>
            </template>
          </n-button>
        </div>
        
        <div class="task-list">
          <div
            v-for="task in syncTaskStore.tasks"
            :key="task.id"
            :class="['task-card', { 'is-selected': selectedTaskId === task.id }]"
            @click="handleTaskSelect(task.id)"
          >
            <div class="task-card-header">
              <span class="task-name">{{ task.name }}</span>
              <n-tag
                v-if="getTaskProgress(task.id)"
                :type="getStatusTagType(getTaskProgress(task.id)?.status)"
                size="small"
              >
                {{ getStatusText(getTaskProgress(task.id)?.status) }}
              </n-tag>
            </div>
            <div class="task-card-info">
              <span class="task-source">{{ task.sourceName }} → {{ task.targetName }}</span>
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
            v-if="syncTaskStore.tasks.length === 0"
            description="暂无任务"
            style="margin-top: 40px"
          />
        </div>
      </n-layout-sider>

      <!-- 右侧：任务详情 -->
      <n-layout-content :native-scrollbar="false" style="padding: 16px">
        <n-empty
          v-if="!selectedTaskId"
          description="请从左侧选择一个任务"
          style="margin-top: 100px"
        />
        
        <n-space v-else vertical :size="16">
          <!-- 任务控制 -->
          <n-card size="small">
            <n-space align="center" justify="space-between">
              <!-- 左侧：数据源信息 -->
              <n-space :size="8" align="center">
                <n-icon 
                  :size="24" 
                  :color="getSelectedTask()?.sourceType === 'mysql' ? '#2080f0' : '#18a058'"
                >
                  <component :is="getSelectedTask()?.sourceType === 'mysql' ? DatabaseIcon : SearchIcon" />
                </n-icon>
                <span style="font-size: 14px; font-weight: 500;">{{ getSelectedSourceDataSource()?.name || '-' }}</span>
                <n-icon :size="20" color="#999">
                  <ArrowForwardIcon />
                </n-icon>
                <n-icon 
                  :size="24" 
                  :color="getSelectedTask()?.targetType === 'mysql' ? '#2080f0' : '#18a058'"
                >
                  <component :is="getSelectedTask()?.targetType === 'mysql' ? DatabaseIcon : SearchIcon" />
                </n-icon>
                <span style="font-size: 14px; font-weight: 500;">{{ getSelectedTargetDataSource()?.name || '-' }}</span>
              </n-space>
              
              <!-- 右侧：控制按钮 -->
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
              
              <!-- 右侧：统计信息 -->
              <n-space>
                <n-statistic label="同步速度" :value="progress?.speed.toFixed(2) || 0">
                  <template #suffix>记录/秒</template>
                </n-statistic>
                <n-statistic label="预计剩余" :value="formatTime(progress?.estimatedTime || 0)" />
              </n-space>
            </n-space>
          </n-card>

          <!-- 进度信息 -->
          <n-card size="small" title="同步进度">
            <n-space vertical :size="12">
              <div style="display: flex; justify-content: space-between">
                <span>{{ progress?.processedRecords.toLocaleString() || 0 }} / {{ progress?.totalRecords.toLocaleString() || 0 }}</span>
                <span style="font-weight: bold">{{ progress?.percentage.toFixed(2) || 0 }}%</span>
              </div>
              <n-progress
                type="line"
                :percentage="progress?.percentage || 0"
                :status="progressStatus"
                :height="24"
              />
              <div v-if="progress?.currentTable" style="color: #666; font-size: 13px">
                当前表：{{ progress.currentTable }}
              </div>
            </n-space>
          </n-card>

          <!-- 表进度 -->
          <n-card
            v-if="progress?.tableProgress && progress.tableProgress.length > 0"
            size="small"
            title="表同步进度"
          >
            <template #header-extra>
              <n-button size="small" @click="scrollToCurrentTable">
                定位当前表
              </n-button>
            </template>
            
            <div class="table-progress-container" ref="tableProgressRef">
              <div
                v-for="table in progress.tableProgress"
                :key="table.tableName"
                :class="['table-progress-item', { 'is-current': table.status === 'running' }]"
                :ref="table.status === 'running' ? 'currentTableRef' : undefined"
              >
                <div class="table-header">
                  <n-tag
                    :type="getTableStatusType(table.status)"
                    size="small"
                    style="margin-right: 8px"
                  >
                    {{ getTableStatusText(table.status) }}
                  </n-tag>
                  <span class="table-name">{{ table.tableName }}</span>
                  <span class="table-percentage">{{ table.percentage.toFixed(1) }}%</span>
                </div>
                <n-progress
                  type="line"
                  :percentage="table.percentage"
                  :status="getTableProgressStatus(table.status)"
                  :height="6"
                  :show-indicator="false"
                />
                <div class="table-info">
                  {{ table.processedRecords.toLocaleString() }} / {{ table.totalRecords.toLocaleString() }}
                </div>
              </div>
            </div>
          </n-card>

          <!-- 日志区域 -->
          <n-card size="small">
            <n-tabs type="line" animated>
              <!-- 实时日志 -->
              <n-tab-pane name="logs" tab="实时日志">
                <template #tab>
                  实时日志
                  <n-badge
                    v-if="allLogs.length > 0"
                    :value="allLogs.length"
                    :max="999"
                    style="margin-left: 8px"
                  />
                </template>
                <div class="log-container">
                  <div ref="allLogContentRef" class="log-content">
                    <div
                      v-for="(log, index) in allLogs"
                      :key="index"
                      :class="['log-entry', `log-${log.level}`]"
                    >
                      <span class="log-timestamp">{{ log.timestamp }}</span>
                      <span class="log-level">{{ getLevelText(log.level) }}</span>
                      <span class="log-message">{{ log.message }}</span>
                    </div>
                    <div v-if="allLogs.length === 0" class="empty-log">
                      暂无日志
                    </div>
                  </div>
                </div>
              </n-tab-pane>

              <!-- 明细日志 -->
              <n-tab-pane name="detail" tab="明细日志">
                <template #tab>
                  明细日志
                  <n-badge
                    v-if="detailLogs.length > 0"
                    :value="detailLogs.length"
                    :max="999"
                    style="margin-left: 8px"
                  />
                </template>
                <div class="log-container">
                  <div ref="detailLogContentRef" class="log-content">
                    <div
                      v-for="(log, index) in detailLogs"
                      :key="index"
                      :class="['log-entry', `log-${log.level}`]"
                    >
                      <span class="log-timestamp">{{ log.timestamp }}</span>
                      <span class="log-level">{{ getLevelText(log.level) }}</span>
                      <span class="log-message">{{ log.message }}</span>
                    </div>
                    <div v-if="detailLogs.length === 0" class="empty-log">
                      暂无明细日志
                    </div>
                  </div>
                </div>
              </n-tab-pane>

              <!-- 校验日志 -->
              <n-tab-pane name="verify" tab="校验日志">
                <template #tab>
                  校验日志
                  <n-badge
                    v-if="verifyLogs.length > 0"
                    :value="verifyLogs.length"
                    :max="999"
                    style="margin-left: 8px"
                  />
                </template>
                <div class="log-container">
                  <div ref="verifyLogContentRef" class="log-content">
                    <div
                      v-for="(log, index) in verifyLogs"
                      :key="index"
                      :class="['log-entry', `log-${log.level}`]"
                    >
                      <span class="log-timestamp">{{ log.timestamp }}</span>
                      <span class="log-level">{{ getLevelText(log.level) }}</span>
                      <span class="log-message">{{ log.message }}</span>
                    </div>
                    <div v-if="verifyLogs.length === 0" class="empty-log">
                      暂无校验日志
                    </div>
                  </div>
                </div>
              </n-tab-pane>

              <!-- 错误日志 -->
              <n-tab-pane name="errors" tab="错误日志">
                <template #tab>
                  错误日志
                  <n-badge
                    v-if="taskMonitorStore.errors.length > 0"
                    :value="taskMonitorStore.errors.length"
                    :max="99"
                    type="error"
                    style="margin-left: 8px"
                  />
                </template>
                
                <div class="log-container">
                  <div ref="errorLogContentRef" class="log-content">
                    <div
                      v-for="(error, index) in taskMonitorStore.errors"
                      :key="index"
                      class="error-entry"
                    >
                      <div class="error-header">
                        <span class="error-timestamp">{{ new Date(error.timestamp).toLocaleString() }}</span>
                        <n-tag type="error" size="small">{{ error.errorType }}</n-tag>
                      </div>
                      <div class="error-message">{{ error.message }}</div>
                      <div v-if="error.data" class="error-details">
                        <n-collapse>
                          <n-collapse-item title="详细信息">
                            <pre>{{ JSON.stringify(error.data, null, 2) }}</pre>
                          </n-collapse-item>
                        </n-collapse>
                      </div>
                    </div>
                    <div v-if="taskMonitorStore.errors.length === 0" class="empty-log">
                      暂无错误
                    </div>
                  </div>
                </div>
              </n-tab-pane>
            </n-tabs>
          </n-card>
        </n-space>
      </n-layout-content>
    </n-layout>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted, nextTick, watch } from 'vue'
import { useRoute } from 'vue-router'
import {
  NLayout,
  NLayoutSider,
  NLayoutContent,
  NSpace,
  NCard,
  NButton,
  NIcon,
  NTag,
  NProgress,
  NEmpty,
  NStatistic,
  NTabs,
  NTabPane,
  NBadge,
  NCollapse,
  NCollapseItem
} from 'naive-ui'
import {
  Play as PlayIcon,
  Pause as PauseIcon,
  Refresh as RefreshIcon,
  ArrowForward as ArrowForwardIcon,
  Search as SearchIcon,
  Server as DatabaseIcon
} from '@vicons/ionicons5'
import { useSyncTaskStore } from '../stores/syncTask'
import { useTaskMonitorStore } from '../stores/taskMonitor'
import { useDataSourceStore } from '../stores/dataSource'
import { showSuccess, handleApiError } from '../utils/message'
import { listen, type UnlistenFn } from '@tauri-apps/api/event'

const syncTaskStore = useSyncTaskStore()
const taskMonitorStore = useTaskMonitorStore()
const dataSourceStore = useDataSourceStore()
const route = useRoute()

const selectedTaskId = ref<string>('')
const allLogs = ref<any[]>([])
const detailLogs = computed(() => {
  // 明细日志：只显示批次进度（"批次 X/Y 完成，已同步: X 条，剩余: Y 批次"）
  return allLogs.value.filter(log => 
    log.message.includes('批次') && 
    log.message.includes('完成') && 
    log.message.includes('已同步') && 
    log.message.includes('剩余')
  )
})
const verifyLogs = computed(() => {
  // 校验日志：只显示数据校验结果（✓ 通过 或 ✗ 失败）
  return allLogs.value.filter(log => 
    log.message.includes('数据校验通过') ||
    log.message.includes('数据校验失败')
  )
})
const currentTableRef = ref<HTMLElement>()
const allLogContentRef = ref<HTMLElement>()
const detailLogContentRef = ref<HTMLElement>()
const verifyLogContentRef = ref<HTMLElement>()
const errorLogContentRef = ref<HTMLElement>()
const tableProgressRef = ref<HTMLElement>()

let unlistenLog: UnlistenFn | null = null
let unlistenError: UnlistenFn | null = null

// 当前进度
const progress = computed(() => {
  if (!selectedTaskId.value) return null
  // 如果是当前选中的任务，返回 store 中的进度
  return taskMonitorStore.progress
})

// 任务状态
const isRunning = computed(() => progress.value?.status === 'running')
const isPaused = computed(() => progress.value?.status === 'paused')

const progressStatus = computed(() => {
  if (progress.value?.status === 'completed') return 'success'
  if (progress.value?.status === 'failed') return 'error'
  return 'info'
})

// 获取选中的任务对象
function getSelectedTask() {
  return syncTaskStore.tasks.find(t => t.id === selectedTaskId.value)
}

// 获取选中任务的源数据源
function getSelectedSourceDataSource() {
  const task = getSelectedTask()
  if (!task) return null
  return dataSourceStore.dataSources.find(ds => ds.id === task.sourceId)
}

// 获取选中任务的目标数据源
function getSelectedTargetDataSource() {
  const task = getSelectedTask()
  if (!task) return null
  return dataSourceStore.dataSources.find(ds => ds.id === task.targetId)
}

// 获取任务进度
function getTaskProgress(taskId: string) {
  // 只有当前选中的任务才有进度信息
  if (taskId === selectedTaskId.value) {
    return taskMonitorStore.progress
  }
  return null
}

// 获取状态文本
function getStatusText(status?: string): string {
  const statusMap: Record<string, string> = {
    running: '运行中',
    paused: '已暂停',
    completed: '已完成',
    failed: '失败'
  }
  return statusMap[status || ''] || '空闲'
}

// 获取状态标签类型
function getStatusTagType(status?: string): any {
  const typeMap: Record<string, any> = {
    running: 'info',
    paused: 'warning',
    completed: 'success',
    failed: 'error'
  }
  return typeMap[status || ''] || 'default'
}

// 获取进度状态
function getProgressStatus(status?: string) {
  if (status === 'completed') return 'success'
  if (status === 'failed') return 'error'
  return 'info'
}

// 获取表状态文本
function getTableStatusText(status: string): string {
  const statusMap: Record<string, string> = {
    waiting: '等待',
    running: '同步中',
    completed: '完成',
    failed: '失败'
  }
  return statusMap[status] || status
}

// 获取表状态类型
function getTableStatusType(status: string): any {
  const typeMap: Record<string, any> = {
    waiting: 'default',
    running: 'info',
    completed: 'success',
    failed: 'error'
  }
  return typeMap[status] || 'default'
}

// 获取表进度状态
function getTableProgressStatus(status: string) {
  if (status === 'completed') return 'success'
  if (status === 'failed') return 'error'
  if (status === 'running') return 'info'
  return 'default'
}

// 获取日志级别文本
function getLevelText(level: string): string {
  const levelMap: Record<string, string> = {
    info: 'INFO',
    warn: 'WARN',
    error: 'ERROR',
  }
  return levelMap[level] || level.toUpperCase()
}

// 格式化时间
function formatTime(seconds: number): string {
  if (!seconds || seconds <= 0) return '-'
  
  const hours = Math.floor(seconds / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  const secs = Math.floor(seconds % 60)
  
  if (hours > 0) {
    return `${hours}h${minutes}m`
  } else if (minutes > 0) {
    return `${minutes}m${secs}s`
  } else {
    return `${secs}s`
  }
}

// 选择任务
function handleTaskSelect(taskId: string) {
  selectedTaskId.value = taskId
  allLogs.value = []
  loadProgress()
  loadErrors()
  loadLogs()
}

// 启动任务
async function handleStart() {
  if (!selectedTaskId.value) return
  
  try {
    await taskMonitorStore.startTask(selectedTaskId.value)
    showSuccess('任务已启动')
  } catch (error) {
    handleApiError(error, '启动任务失败')
  }
}

// 暂停任务
async function handlePause() {
  if (!selectedTaskId.value) return
  
  try {
    await taskMonitorStore.pauseTask(selectedTaskId.value)
    showSuccess('任务已暂停')
  } catch (error) {
    handleApiError(error, '暂停任务失败')
  }
}

// 恢复任务
async function handleResume() {
  if (!selectedTaskId.value) return
  
  try {
    await taskMonitorStore.resumeTask(selectedTaskId.value)
    showSuccess('任务已恢复')
  } catch (error) {
    handleApiError(error, '恢复任务失败')
  }
}

// 加载任务列表
async function loadTasks() {
  try {
    await syncTaskStore.fetchTasks()
  } catch (error) {
    handleApiError(error, '加载任务列表失败')
  }
}

// 加载进度
async function loadProgress() {
  if (!selectedTaskId.value) return
  
  try {
    await taskMonitorStore.getProgress(selectedTaskId.value)
  } catch (error) {
    // 静默失败
  }
}

// 加载错误日志
async function loadErrors() {
  if (!selectedTaskId.value) return
  
  try {
    await taskMonitorStore.getErrors(selectedTaskId.value)
  } catch (error) {
    // 静默失败
  }
}

// 加载日志
async function loadLogs() {
  if (!selectedTaskId.value) return
  
  try {
    allLogs.value = await taskMonitorStore.getTaskLogs(selectedTaskId.value)
    await nextTick()
    scrollToBottom(allLogContentRef.value)
  } catch (error) {
    // 静默失败
  }
}

// 滚动到当前表
async function scrollToCurrentTable() {
  await nextTick()
  if (currentTableRef.value) {
    currentTableRef.value.scrollIntoView({ behavior: 'smooth', block: 'center' })
  }
}

// 滚动到底部
function scrollToBottom(element?: HTMLElement) {
  if (element) {
    element.scrollTop = element.scrollHeight
  }
}

// 开始监听事件
async function startListening() {
  try {
    // 监听日志事件
    unlistenLog = await listen('task-log', (event: any) => {
      const logEvent = event.payload
      if (logEvent.taskId === selectedTaskId.value) {
        allLogs.value.push(logEvent.log)
        if (allLogs.value.length > 1000) {
          allLogs.value.shift()
        }
        nextTick(() => {
          scrollToBottom(allLogContentRef.value)
          scrollToBottom(detailLogContentRef.value)
          scrollToBottom(verifyLogContentRef.value)
        })
      }
    })
    
    // 监听错误事件
    unlistenError = await listen('task-error', (event: any) => {
      const errorEvent = event.payload
      if (errorEvent.taskId === selectedTaskId.value) {
        loadErrors()
        nextTick(() => scrollToBottom(errorLogContentRef.value))
      }
    })
  } catch (error) {
    console.error('监听事件失败:', error)
  }
}

// 停止监听事件
function stopListening() {
  if (unlistenLog) {
    unlistenLog()
    unlistenLog = null
  }
  if (unlistenError) {
    unlistenError()
    unlistenError = null
  }
}

onMounted(async () => {
  await dataSourceStore.fetchDataSources()
  await loadTasks()
  await taskMonitorStore.startEventListeners()
  startListening()
  
  // 检查 URL 参数，如果有 taskId 则自动选中该任务
  const taskId = route.query.taskId as string
  if (taskId) {
    selectedTaskId.value = taskId
    loadProgress()
    loadErrors()
    loadLogs()
  }
})

// 监听路由变化，支持动态切换任务
watch(() => route.query.taskId, (newTaskId) => {
  if (newTaskId && typeof newTaskId === 'string') {
    selectedTaskId.value = newTaskId
    allLogs.value = []
    loadProgress()
    loadErrors()
    loadLogs()
  }
})

onUnmounted(() => {
  taskMonitorStore.stopEventListeners()
  stopListening()
})
</script>

<style scoped>
.task-monitor {
  height: 100%;
}

/* 任务列表 */
.task-list-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px;
  border-bottom: 1px solid #e0e0e6;
}

.task-list-header h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.task-list {
  padding: 12px;
}

.task-card {
  padding: 12px;
  margin-bottom: 8px;
  border: 1px solid #e0e0e6;
  border-radius: 6px;
  cursor: pointer;
  transition: all 0.2s;
  background-color: #fff;
}

.task-card:hover {
  border-color: #2080f0;
  box-shadow: 0 2px 8px rgba(32, 128, 240, 0.1);
}

.task-card.is-selected {
  border-color: #2080f0;
  background-color: #f0f7ff;
  box-shadow: 0 2px 8px rgba(32, 128, 240, 0.15);
}

.task-card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.task-name {
  font-weight: 500;
  font-size: 14px;
  color: #333;
}

.task-card-info {
  font-size: 12px;
  color: #666;
  margin-bottom: 8px;
}

/* 表进度 */
.table-progress-container {
  max-height: 300px;
  overflow-y: auto;
}

.table-progress-item {
  padding: 12px;
  margin-bottom: 8px;
  border: 1px solid #e0e0e6;
  border-radius: 4px;
  background-color: #fafafa;
}

.table-progress-item.is-current {
  border-color: #2080f0;
  background-color: #f0f7ff;
}

.table-header {
  display: flex;
  align-items: center;
  margin-bottom: 8px;
}

.table-name {
  flex: 1;
  font-size: 13px;
  color: #333;
}

.table-percentage {
  font-weight: bold;
  font-size: 13px;
  color: #2080f0;
}

.table-info {
  margin-top: 4px;
  font-size: 12px;
  color: #666;
}

/* 日志区域 */
.log-container {
  height: 400px;
  border: 1px solid #e0e0e6;
  border-radius: 4px;
  overflow: hidden;
  margin-top: 12px;
}

.log-content {
  height: 100%;
  overflow-y: auto;
  padding: 12px;
  background-color: #1e1e1e;
  font-family: 'Consolas', 'Monaco', 'Courier New', monospace;
  font-size: 13px;
  line-height: 1.6;
}

.log-entry {
  margin-bottom: 4px;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-timestamp {
  color: #858585;
  margin-right: 8px;
}

.log-level {
  display: inline-block;
  width: 50px;
  margin-right: 8px;
  font-weight: bold;
}

.log-message {
  color: #d4d4d4;
}

.log-info .log-level {
  color: #4ec9b0;
}

.log-warn .log-level {
  color: #dcdcaa;
}

.log-error .log-level {
  color: #f48771;
}

.error-entry {
  margin-bottom: 12px;
  padding: 12px;
  background-color: #2d2d2d;
  border-radius: 4px;
  border-left: 3px solid #f48771;
}

.error-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}

.error-timestamp {
  color: #858585;
  font-size: 12px;
}

.error-message {
  color: #f48771;
  margin-bottom: 8px;
}

.error-details pre {
  color: #d4d4d4;
  font-size: 12px;
  margin: 0;
  white-space: pre-wrap;
  word-break: break-all;
}

.empty-log {
  color: #858585;
  text-align: center;
  padding: 20px;
}

/* 滚动条样式 */
.log-content::-webkit-scrollbar,
.table-progress-container::-webkit-scrollbar {
  width: 6px;
}

.log-content::-webkit-scrollbar-track {
  background: #2d2d2d;
}

.table-progress-container::-webkit-scrollbar-track {
  background: #f1f1f1;
}

.log-content::-webkit-scrollbar-thumb,
.table-progress-container::-webkit-scrollbar-thumb {
  background: #555;
  border-radius: 3px;
}

.log-content::-webkit-scrollbar-thumb:hover,
.table-progress-container::-webkit-scrollbar-thumb:hover {
  background: #666;
}
</style>
