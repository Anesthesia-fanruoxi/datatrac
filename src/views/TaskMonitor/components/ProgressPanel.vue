<template>
  <div class="progress-panel">
    <n-card
      v-if="taskUnits && taskUnits.length > 0"
      size="small"
      title="表/索引同步进度"
      :content-style="{ padding: 0, height: '100%', display: 'flex', flexDirection: 'column', overflow: 'hidden' }"
    >
      <template #header-extra>
        <div class="progress-stats">
          <n-tag :bordered="false" size="small" class="stat-tag stat-tag-total">总计: {{ taskUnits.length }}</n-tag>
          <n-tag :bordered="false" size="small" class="stat-tag stat-tag-pending">等待: {{ getCountByStatus('pending') }}</n-tag>
          <n-tag :bordered="false" size="small" class="stat-tag stat-tag-running">进行中: {{ getCountByStatus('running') }}</n-tag>
          <n-tag :bordered="false" size="small" class="stat-tag stat-tag-completed">完成: {{ getCountByStatus('completed') }}</n-tag>
          <n-tag :bordered="false" size="small" class="stat-tag stat-tag-failed">失败: {{ getCountByStatus('failed') }}</n-tag>
        </div>
      </template>
      
      <div class="progress-list-wrapper">
        <!-- 按搜索关键字分组显示 - 手风琴模式 -->
        <n-collapse 
          v-if="groupedUnits.length > 0" 
          :default-expanded-names="['new-group']"
          accordion
          class="full-height-collapse"
        >
          <n-collapse-item
            v-for="group in groupedUnits"
            :key="group.isNewGroup ? 'new-group' : (group.pattern || 'unknown')"
            :name="group.isNewGroup ? 'new-group' : (group.pattern || 'unknown')"
            :title="getGroupTitle(group)"
          >
            <template #header-extra>
              <n-space :size="8">
                <n-tag size="small" :bordered="false" class="count-tag">
                  {{ group.units.length }} 个
                </n-tag>
                <!-- 历史记录组显示清除按钮 -->
                <n-button
                  v-if="group.pattern && group.isHistorical"
                  size="tiny"
                  type="error"
                  @click.stop="handleClearPattern(group.pattern)"
                  title="清除此关键字的所有记录"
                >
                  <template #icon>
                    <n-icon><TrashIcon /></n-icon>
                  </template>
                  清除
                </n-button>
              </n-space>
            </template>
            
            <div
              v-for="unit in group.units"
              :key="unit.id"
              :class="['progress-item', `status-${unit.status}`]"
            >
              <div class="progress-item-header">
                <n-tag :type="getStatusType(unit.status)" size="small">
                  {{ getStatusText(unit.status) }}
                </n-tag>
                <!-- 显示搜索关键字标签（所有单元都显示） -->
                <n-tag 
                  v-if="unit.searchPattern" 
                  size="small" 
                  :bordered="false"
                  :style="{ 
                    backgroundColor: getPatternColor(unit.searchPattern), 
                    color: '#fff',
                    borderRadius: '12px',
                    padding: '2px 10px',
                    fontWeight: '500'
                  }"
                >
                  {{ unit.searchPattern }}
                </n-tag>
                <span class="progress-item-name">{{ unit.name }}</span>
                <!-- 失败状态显示重试按钮 -->
                <n-button
                  v-if="unit.status === 'failed'"
                  size="tiny"
                  type="warning"
                  @click="handleRetryUnit(unit.id)"
                  title="重试此单元"
                >
                  <template #icon>
                    <n-icon><RefreshIcon /></n-icon>
                  </template>
                  重试
                </n-button>
              </div>
              
              <!-- 进度条和百分比 -->
              <div v-if="unit.status === 'running' || unit.status === 'completed'" class="progress-bar-wrapper">
                <n-progress
                  type="line"
                  :percentage="unit.percentage"
                  :status="getProgressStatus(unit.status)"
                  :height="6"
                  :show-indicator="false"
                  style="flex: 1;"
                />
                <span class="progress-percentage">{{ unit.percentage.toFixed(1) }}%</span>
              </div>
              
              <!-- 记录数信息 -->
              <div v-if="unit.totalRecords > 0" class="progress-item-info">
                {{ unit.processedRecords.toLocaleString() }} / {{ unit.totalRecords.toLocaleString() }} 条记录
              </div>
              
              <!-- 已完成单元显示完成时间和耗时 -->
              <div v-if="unit.status === 'completed' && unit.completedAt" class="progress-item-info">
                完成时间: {{ formatTimestamp(unit.completedAt) }} | 耗时: {{ formatDuration(unit.duration) }}
              </div>
              
              <!-- 错误信息 -->
              <div v-if="unit.status === 'failed' && unit.errorMessage" class="progress-item-error">
                <n-icon :size="14" color="#d03050">
                  <AlertCircleIcon />
                </n-icon>
                {{ unit.errorMessage }}
              </div>
            </div>
          </n-collapse-item>
        </n-collapse>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { NCard, NTag, NProgress, NIcon, NButton, NCollapse, NCollapseItem, NSpace } from 'naive-ui'
import { AlertCircle as AlertCircleIcon, Refresh as RefreshIcon, Trash as TrashIcon } from '@vicons/ionicons5'
import type { TaskUnit } from '../../../types'
import { useTaskMonitorStore } from '../../../stores/taskMonitor'
import { useRoute } from 'vue-router'
import { showSuccess, handleApiError } from '../../../utils/message'

interface Props {
  taskUnits: TaskUnit[]
}

const props = defineProps<Props>()
const taskMonitorStore = useTaskMonitorStore()
const route = useRoute()

// 按搜索关键字分组
interface UnitGroup {
  pattern: string | null
  units: TaskUnit[]
  isHistorical: boolean // 是否是历史记录
  isNewGroup: boolean // 是否是"新增同步"组
}

// 为不同的搜索关键字分配颜色
const patternColors = new Map<string, string>()
const colorPalette = [
  '#2080f0', // 蓝色
  '#18a058', // 绿色
  '#f0a020', // 橙色
  '#d03050', // 红色
  '#7c3aed', // 紫色
  '#0891b2', // 青色
  '#ea580c', // 深橙
  '#be185d', // 粉红
  '#4338ca', // 靛蓝
  '#059669', // 翠绿
]

function getPatternColor(pattern: string): string {
  if (!patternColors.has(pattern)) {
    const colorIndex = patternColors.size % colorPalette.length
    patternColors.set(pattern, colorPalette[colorIndex])
  }
  return patternColors.get(pattern)!
}

const groupedUnits = computed<UnitGroup[]>(() => {
  const newUnits: TaskUnit[] = []
  const completedUnits: TaskUnit[] = []
  
  // 分类：新增同步 vs 已完成同步
  for (const unit of props.taskUnits) {
    if (unit.status === 'completed') {
      // 已完成同步
      completedUnits.push(unit)
    } else {
      // 新增同步（待同步、进行中、失败、暂停）
      newUnits.push(unit)
    }
  }
  
  // 新增同步排序：进行中 → 待同步/暂停 → 失败
  newUnits.sort((a, b) => {
    const priority: Record<string, number> = {
      running: 1,
      pending: 2,
      paused: 2,
      failed: 3
    }
    return (priority[a.status] || 99) - (priority[b.status] || 99)
  })
  
  const result: UnitGroup[] = []
  
  // 1. 新增同步组（如果有）- 放在最前面，默认展开
  if (newUnits.length > 0) {
    result.push({
      pattern: null,
      units: newUnits,
      isHistorical: false,
      isNewGroup: true
    })
  }
  
  // 2. 已完成同步组（如果有）
  if (completedUnits.length > 0) {
    result.push({
      pattern: null,
      units: completedUnits,
      isHistorical: true,
      isNewGroup: false
    })
  }
  
  return result
})

function getGroupTitle(group: UnitGroup): string {
  if (group.isNewGroup) {
    return '🆕 新增同步'
  }
  if (group.isHistorical) {
    return '✅ 已完成同步'
  }
  return '📊 未分类'
}

// 清除指定关键字的记录
async function handleClearPattern(pattern: string) {
  try {
    const taskId = route.query.taskId as string
    if (!taskId) {
      handleApiError(new Error('未选择任务'), '清除失败')
      return
    }
    
    const count = await taskMonitorStore.clearTaskUnitsByPattern(taskId, pattern)
    showSuccess(`已清除 ${count} 条记录`)
  } catch (error) {
    handleApiError(error, '清除失败')
  }
}

// 重试单个失败的单元
async function handleRetryUnit(unitId: string) {
  try {
    await taskMonitorStore.resetUnit(unitId)
    showSuccess('单元已重置为等待状态')
  } catch (error) {
    handleApiError(error, '重试失败')
  }
}

function getCountByStatus(status: string): number {
  return props.taskUnits.filter(unit => unit.status === status).length
}

function getStatusText(status: string): string {
  const statusMap: Record<string, string> = {
    pending: '等待',
    running: '进行中',
    completed: '完成',
    failed: '失败',
    paused: '暂停'
  }
  return statusMap[status] || status
}

function getStatusType(status: string): any {
  const typeMap: Record<string, any> = {
    pending: 'default',
    running: 'info',
    completed: 'success',
    failed: 'error',
    paused: 'warning'
  }
  return typeMap[status] || 'default'
}

function getProgressStatus(status: string) {
  if (status === 'completed') return 'success'
  if (status === 'failed') return 'error'
  if (status === 'running') return 'info'
  if (status === 'paused') return 'warning'
  return 'default'
}

// 格式化时间戳
function formatTimestamp(timestamp: number): string {
  const date = new Date(timestamp)
  return date.toLocaleString('zh-CN', {
    year: 'numeric',
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit'
  })
}

// 格式化时长（毫秒）
function formatDuration(ms: number): string {
  const seconds = Math.floor(ms / 1000)
  const minutes = Math.floor(seconds / 60)
  const hours = Math.floor(minutes / 60)
  
  if (hours > 0) {
    return `${hours}小时${minutes % 60}分钟`
  } else if (minutes > 0) {
    return `${minutes}分钟${seconds % 60}秒`
  } else {
    return `${seconds}秒`
  }
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

.progress-panel :deep(.n-card) {
  height: 100%;
  display: flex;
  flex-direction: column;
}

.progress-panel :deep(.n-card__content) {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.progress-stats {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.progress-list-wrapper {
  flex: 1;
  overflow-y: auto;
  overflow-x: hidden;
  padding: 0;
}

.full-height-collapse {
  height: 100%;
}

.full-height-collapse :deep(.n-collapse-item__content-wrapper) {
  overflow-y: auto;
}

.full-height-collapse :deep(.n-collapse-item__content-inner) {
  padding: 12px;
}

.progress-item {
  padding: 12px;
  margin-bottom: 8px;
  border: 1px solid #e0e0e6;
  border-radius: 8px;
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

.progress-bar-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-bottom: 4px;
}

.progress-percentage {
  font-weight: bold;
  font-size: 13px;
  color: #2080f0;
  min-width: 45px;
  text-align: right;
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

/* 顶部统计标签样式 */
.stat-tag {
  border-radius: 12px !important;
  padding: 4px 12px !important;
  font-weight: 500 !important;
  font-size: 12px !important;
}

.stat-tag-total {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%) !important;
  color: #fff !important;
}

.stat-tag-pending {
  background: linear-gradient(135deg, #e8e8e8 0%, #d0d0d0 100%) !important;
  color: #666 !important;
}

.stat-tag-running {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%) !important;
  color: #fff !important;
}

.stat-tag-completed {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%) !important;
  color: #fff !important;
}

.stat-tag-failed {
  background: linear-gradient(135deg, #fa709a 0%, #fee140 100%) !important;
  color: #fff !important;
}

/* 数量标签 */
.count-tag {
  background: linear-gradient(135deg, #a8edea 0%, #fed6e3 100%) !important;
  color: #666 !important;
  border-radius: 12px !important;
  padding: 2px 10px !important;
  font-weight: 500 !important;
}

/* 状态标签样式优化 */
.progress-item-header :deep(.n-tag) {
  border-radius: 12px;
  padding: 2px 10px;
  font-weight: 500;
}

/* 等待状态 - 灰色 */
.progress-item-header :deep(.n-tag.n-tag--default-type) {
  background: linear-gradient(135deg, #e8e8e8 0%, #d0d0d0 100%);
  color: #666;
}

/* 进行中状态 - 蓝色 */
.progress-item-header :deep(.n-tag.n-tag--info-type) {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  color: #fff;
}

/* 完成状态 - 绿色 */
.progress-item-header :deep(.n-tag.n-tag--success-type) {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
  color: #fff;
}

/* 失败状态 - 红粉渐变 */
.progress-item-header :deep(.n-tag.n-tag--error-type) {
  background: linear-gradient(135deg, #fa709a 0%, #fee140 100%);
  color: #fff;
}
</style>


