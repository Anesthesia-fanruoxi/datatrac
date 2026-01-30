<template>
  <div class="task-control-panel">
    <n-card size="small" :bordered="false" class="control-card">
      <!-- 单行布局 -->
      <div class="control-row">
        <!-- 数据源信息 -->
        <div class="datasource-info">
          <div class="datasource-item">
            <span class="status-dot" :class="{ 'online': sourceDataSource }"></span>
            <n-icon 
              :size="20" 
              :color="sourceDataSource?.type === 'mysql' ? '#2080f0' : '#18a058'"
            >
              <component :is="sourceDataSource?.type === 'mysql' ? DatabaseIcon : SearchIcon" />
            </n-icon>
            <span class="datasource-name">{{ sourceDataSource?.name || '-' }}</span>
          </div>
          <n-icon :size="18" color="#999" class="arrow-icon">
            <ArrowForwardIcon />
          </n-icon>
          <div class="datasource-item">
            <span class="status-dot" :class="{ 'online': targetDataSource }"></span>
            <n-icon 
              :size="20" 
              :color="targetDataSource?.type === 'mysql' ? '#2080f0' : '#18a058'"
            >
              <component :is="targetDataSource?.type === 'mysql' ? DatabaseIcon : SearchIcon" />
            </n-icon>
            <span class="datasource-name">{{ targetDataSource?.name || '-' }}</span>
          </div>
        </div>

        <n-divider vertical />
        
        <!-- 控制按钮 -->
        <div class="control-buttons">
          <n-button
            :type="isRunning || isPaused ? 'default' : 'primary'"
            :disabled="isRunning || isPaused"
            :loading="loading"
            @click="emit('start')"
          >
            <template #icon>
              <n-icon><PlayIcon /></n-icon>
            </template>
            启动
          </n-button>
          <n-button
            :type="isRunning ? 'warning' : 'default'"
            :class="{ 'btn-pulse': isRunning }"
            :disabled="!isRunning"
            :loading="loading"
            @click="emit('pause')"
          >
            <template #icon>
              <n-icon><PauseIcon /></n-icon>
            </template>
            暂停
          </n-button>
          <n-button
            :type="isPaused ? 'success' : 'default'"
            :class="{ 'btn-pulse': isPaused }"
            :disabled="!isPaused"
            :loading="loading"
            @click="emit('resume')"
          >
            <template #icon>
              <n-icon><PlayIcon /></n-icon>
            </template>
            恢复
          </n-button>
        </div>

        <!-- 统计信息 -->
        <div class="statistics-info">
          <div class="stat-item">
            <n-icon :size="18" class="stat-icon speed"><FlashIcon /></n-icon>
            <span class="stat-label">同步速度</span>
            <span class="stat-value">{{ speed }}<span class="stat-unit">记录/秒</span></span>
          </div>
          <div class="stat-item">
            <n-icon :size="18" class="stat-icon time"><TimeIcon /></n-icon>
            <span class="stat-label">预计剩余</span>
            <span class="stat-value">{{ estimatedTime }}</span>
          </div>
          <div class="stat-item">
            <n-icon :size="18" class="stat-icon records"><DocumentIcon /></n-icon>
            <span class="stat-label">已处理</span>
            <span class="stat-value">{{ processedRecords }}<span class="stat-unit">条</span></span>
          </div>
          <div class="stat-item">
            <n-icon :size="18" class="stat-icon progress"><StatsChartIcon /></n-icon>
            <span class="stat-label">总进度</span>
            <span class="stat-value">{{ totalProgress }}<span class="stat-unit">%</span></span>
          </div>
        </div>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { NCard, NButton, NIcon, NDivider } from 'naive-ui'
import {
  Play as PlayIcon,
  Pause as PauseIcon,
  ArrowForward as ArrowForwardIcon,
  Search as SearchIcon,
  Server as DatabaseIcon,
  Flash as FlashIcon,
  Time as TimeIcon,
  Document as DocumentIcon,
  StatsChart as StatsChartIcon
} from '@vicons/ionicons5'

interface DataSource {
  id: string
  name: string
  type: string
}

interface Props {
  sourceDataSource: DataSource | null
  targetDataSource: DataSource | null
  isRunning: boolean
  isPaused: boolean
  loading: boolean
  speed: string
  estimatedTime: string
  processedRecords?: string
  totalProgress?: string
}

interface Emits {
  (e: 'start'): void
  (e: 'pause'): void
  (e: 'resume'): void
}

withDefaults(defineProps<Props>(), {
  processedRecords: '0',
  totalProgress: '0'
})
const emit = defineEmits<Emits>()
</script>

<style scoped>
.task-control-panel {
  flex-shrink: 0;
  margin-bottom: 16px;
}

.control-card {
  background: linear-gradient(135deg, #667eea08 0%, #764ba208 100%);
  border: 1px solid #e8e8e8;
  border-radius: 12px;
}

/* 单行布局 */
.control-row {
  display: flex;
  align-items: center;
  gap: 16px;
}

/* 数据源信息 */
.datasource-info {
  display: flex;
  align-items: center;
  gap: 12px;
  flex-shrink: 0;
}

.datasource-item {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 12px;
  background: #fff;
  border-radius: 8px;
  border: 1px solid #e8e8e8;
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: #d0d0d0;
}

.status-dot.online {
  background: #52c41a;
  box-shadow: 0 0 8px rgba(82, 196, 26, 0.6);
  animation: pulse-green 2s infinite;
}

@keyframes pulse-green {
  0%, 100% { box-shadow: 0 0 8px rgba(82, 196, 26, 0.6); }
  50% { box-shadow: 0 0 12px rgba(82, 196, 26, 0.9); }
}

.datasource-name {
  font-size: 14px;
  font-weight: 500;
  color: #333;
}

.arrow-icon {
  animation: arrow-move 1.5s ease-in-out infinite;
}

@keyframes arrow-move {
  0%, 100% { transform: translateX(0); }
  50% { transform: translateX(4px); }
}

/* 控制按钮 */
.control-buttons {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.btn-pulse {
  animation: btn-pulse-animation 1.5s ease-in-out infinite;
}

@keyframes btn-pulse-animation {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.02); box-shadow: 0 0 12px rgba(0, 0, 0, 0.15); }
}

/* 统计信息 */
.statistics-info {
  display: flex;
  gap: 24px;
  margin-left: auto;
  flex-shrink: 0;
}

.stat-item {
  display: flex;
  align-items: center;
  gap: 8px;
}

.stat-icon {
  border-radius: 6px;
  padding: 4px;
}

.stat-icon.speed {
  background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
  color: #fff;
}

.stat-icon.time {
  background: linear-gradient(135deg, #f093fb 0%, #f5576c 100%);
  color: #fff;
}

.stat-icon.records {
  background: linear-gradient(135deg, #4facfe 0%, #00f2fe 100%);
  color: #fff;
}

.stat-icon.progress {
  background: linear-gradient(135deg, #43e97b 0%, #38f9d7 100%);
  color: #fff;
}

.stat-label {
  font-size: 12px;
  color: #999;
  margin-right: 4px;
}

.stat-value {
  font-size: 18px;
  font-weight: 600;
  color: #333;
}

.stat-unit {
  font-size: 12px;
  font-weight: 400;
  color: #999;
  margin-left: 2px;
}
</style>
