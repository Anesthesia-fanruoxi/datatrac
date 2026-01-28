<template>
  <div class="task-monitor-page">
    <!-- 左侧：任务列表 -->
    <TaskList
      :tasks="syncTaskStore.tasks"
      :selected-task-id="selectedTaskId"
      :get-task-progress="getTaskProgress"
      :get-data-source-name="getDataSourceName"
      @select="handleTaskSelect"
      @refresh="loadTasks"
    />

    <!-- 右侧：任务详情 -->
    <div class="task-detail-area">
      <n-empty
        v-if="!selectedTaskId"
        description="请从左侧选择一个任务"
        style="margin-top: 100px"
      />
      
      <div v-else class="task-detail-content">
        <!-- 任务控制区 -->
        <TaskControl
          :source-data-source="getSelectedSourceDataSource()"
          :target-data-source="getSelectedTargetDataSource()"
          :is-running="isRunning"
          :is-paused="isPaused"
          :loading="taskMonitorStore.loading"
          :speed="progress?.speed.toFixed(2) || '0'"
          :estimated-time="formatTime(progress?.estimatedTime || 0)"
          @start="handleStart"
          @pause="handlePause"
          @resume="handleResume"
        />

        <!-- 下方分栏区 -->
        <div class="task-data-section">
          <!-- 左侧：同步进度 -->
          <ProgressPanel
            :task-units="sortedTaskUnits"
          />

          <!-- 右侧：日志输出 -->
          <LogPanel
            :all-logs="allLogs"
            :detail-logs="detailLogs"
            :verify-logs="verifyLogs"
            :errors="taskMonitorStore.errors"
            @update:all-log-ref="allLogContentRef = $event"
            @update:detail-log-ref="detailLogContentRef = $event"
            @update:verify-log-ref="verifyLogContentRef = $event"
            @update:error-log-ref="errorLogContentRef = $event"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, onUnmounted, watch } from 'vue'
import { useRoute } from 'vue-router'
import { NEmpty } from 'naive-ui'
import TaskList from './TaskMonitor/components/TaskList.vue'
import TaskControl from './TaskMonitor/components/TaskControl.vue'
import ProgressPanel from './TaskMonitor/components/ProgressPanel.vue'
import LogPanel from './TaskMonitor/components/LogPanel.vue'
import { useTaskMonitor } from './TaskMonitor/composables/useTaskMonitor'

const route = useRoute()

const {
  selectedTaskId,
  allLogs,
  allLogContentRef,
  detailLogContentRef,
  verifyLogContentRef,
  errorLogContentRef,
  progress,
  detailLogs,
  verifyLogs,
  sortedTaskUnits,
  isRunning,
  isPaused,
  syncTaskStore,
  taskMonitorStore,
  getSelectedSourceDataSource,
  getSelectedTargetDataSource,
  getTaskProgress,
  getDataSourceName,
  formatTime,
  handleTaskSelect,
  handleStart,
  handlePause,
  handleResume,
  loadTasks,
  initialize,
  cleanup
} = useTaskMonitor()

onMounted(async () => {
  await initialize()
})

watch(() => route.query.taskId, (newTaskId) => {
  if (newTaskId && typeof newTaskId === 'string') {
    handleTaskSelect(newTaskId)
  }
})

onUnmounted(() => {
  cleanup()
})
</script>

<style scoped>
.task-monitor-page {
  display: flex;
  height: 100%;
  background-color: #f5f5f5;
}

.task-detail-area {
  flex: 1;
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.task-detail-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  padding: 16px;
  overflow: hidden;
}

.task-data-section {
  flex: 1;
  display: flex;
  gap: 16px;
  overflow: hidden;
  min-height: 0;
}

.task-data-section > * {
  flex: 1;
  min-width: 0;
}
</style>
