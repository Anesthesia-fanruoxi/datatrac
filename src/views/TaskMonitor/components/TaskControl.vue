<template>
  <div class="task-control-panel">
    <n-card size="small">
      <div class="control-row">
        <!-- 数据源信息 -->
        <div class="datasource-info">
          <n-icon 
            :size="24" 
            :color="sourceDataSource?.type === 'mysql' ? '#2080f0' : '#18a058'"
          >
            <component :is="sourceDataSource?.type === 'mysql' ? DatabaseIcon : SearchIcon" />
          </n-icon>
          <span class="datasource-name">{{ sourceDataSource?.name || '-' }}</span>
          <n-icon :size="20" color="#999">
            <ArrowForwardIcon />
          </n-icon>
          <n-icon 
            :size="24" 
            :color="targetDataSource?.type === 'mysql' ? '#2080f0' : '#18a058'"
          >
            <component :is="targetDataSource?.type === 'mysql' ? DatabaseIcon : SearchIcon" />
          </n-icon>
          <span class="datasource-name">{{ targetDataSource?.name || '-' }}</span>
        </div>
        
        <!-- 控制按钮 -->
        <div class="control-buttons">
          <n-button
            type="primary"
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
            type="warning"
            :disabled="!isRunning"
            :loading="loading"
            @click="emit('pause')"
          >
            <template #icon>
              <n-icon><PauseIcon /></n-icon>
            </template>
            暂停1
          </n-button>
          <n-button
            type="info"
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
          <n-statistic label="同步速度" :value="speed">
            <template #suffix>记录/秒</template>
          </n-statistic>
          <n-statistic label="预计剩余" :value="estimatedTime" />
        </div>
      </div>
    </n-card>
  </div>
</template>

<script setup lang="ts">
import { NCard, NButton, NIcon, NStatistic } from 'naive-ui'
import {
  Play as PlayIcon,
  Pause as PauseIcon,
  ArrowForward as ArrowForwardIcon,
  Search as SearchIcon,
  Server as DatabaseIcon
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
}

interface Emits {
  (e: 'start'): void
  (e: 'pause'): void
  (e: 'resume'): void
}

defineProps<Props>()
const emit = defineEmits<Emits>()
</script>

<style scoped>
.task-control-panel {
  flex-shrink: 0;
  margin-bottom: 16px;
}

.control-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 24px;
}

.datasource-info {
  display: flex;
  align-items: center;
  gap: 8px;
}

.datasource-name {
  font-size: 14px;
  font-weight: 500;
}

.control-buttons {
  display: flex;
  gap: 8px;
}

.statistics-info {
  display: flex;
  gap: 24px;
}
</style>
