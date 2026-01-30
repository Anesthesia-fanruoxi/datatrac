<template>
  <div class="page-container">
    <!-- 任务概览 -->
    <n-card v-if="task" :title="task.name" class="monitor-header">
      <template #header-extra>
        <n-space>
          <n-button v-if="task.status === 'idle' || task.status === 'failed' || task.status === 'completed'" type="primary" @click="handleStart">
            <template #icon><n-icon><PlayIcon /></n-icon></template>
            开始同步
          </n-button>
          <n-button v-if="task.status === 'running'" type="warning" @click="handlePause">
            <template #icon><n-icon><PauseIcon /></n-icon></template>
            暂停
          </n-button>
          <n-button v-if="task.status === 'paused'" type="primary" @click="handleResume">
            <template #icon><n-icon><PlayIcon /></n-icon></template>
            继续
          </n-button>
          <n-button v-if="task.status !== 'idle'" type="error" @click="handleStop">
            <template #icon><n-icon><StopIcon /></n-icon></template>
            终止
          </n-button>
        </n-space>
      </template>

      <n-grid :cols="4">
        <n-gi>
          <n-statistic label="总体进度">
            <n-progress
              type="circle"
              :percentage="overallPercentage"
              :status="progressStatus"
            />
          </n-statistic>
        </n-gi>
        <n-gi>
          <n-statistic label="已处理 / 总记录">
            <div class="stat-value">
              {{ formatNumber(processedRecords) }} / {{ formatNumber(totalRecords) }}
            </div>
          </n-statistic>
        </n-gi>
        <n-gi>
          <n-statistic label="实时速率">
            <div class="stat-value">{{ speed }} 条/秒</div>
          </n-statistic>
        </n-gi>
        <n-gi>
          <n-statistic label="预计剩余时间">
            <div class="stat-value">{{ remainingTime }}</div>
          </n-statistic>
        </n-gi>
      </n-grid>
    </n-card>

    <n-tabs type="line" animated>
      <n-tab-pane name="units" tab="同步单元">
        <n-data-table
          :columns="unitColumns"
          :data="taskUnits"
          :max-height="400"
        />
        <div v-if="hasFailedUnits" class="failed-actions">
          <n-button type="warning" @click="handleResetFailed">重置失败单元并继续</n-button>
        </div>
      </n-tab-pane>
      <n-tab-pane name="logs" tab="运行日志">
        <div class="log-container" ref="logContainer">
          <div v-for="(log, index) in monitorStore.logs" :key="index" :class="['log-item', log.level]">
            <span class="log-time">[{{ log.timestamp }}]</span>
            <span class="log-category">[{{ log.category.toUpperCase() }}]</span>
            <span class="log-message">{{ log.message }}</span>
          </div>
        </div>
      </n-tab-pane>
    </n-tabs>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted, computed, watch, onUnmounted } from 'vue';
import { useRoute } from 'vue-router';
import { 
  NCard, NGrid, NGi, NStatistic, NProgress, NTabs, NTabPane, 
  NDataTable, NButton, NIcon, NSpace, NTag 
} from 'naive-ui';
import { 
  Play as PlayIcon, 
  Pause as PauseIcon, 
  Stop as StopIcon,
  Refresh as RefreshIcon
} from '@vicons/ionicons5';
import { useTaskMonitorStore } from '../stores/taskMonitor';
import { taskApi } from '../api/task';
import { monitorApi } from '../api/monitor';
import type { SyncTask, TaskUnit } from '../types';
import { formatNumber, formatPercentage } from '../utils/format';
import { showSuccess, showError } from '../utils/message';

const route = useRoute();
const monitorStore = useTaskMonitorStore();
const task = ref<SyncTask | null>(null);
const taskUnits = ref<TaskUnit[]>([]);
const logContainer = ref<HTMLElement | null>(null);

const taskId = computed(() => route.query.taskId as string);

// 统计数据
const overallPercentage = computed(() => monitorStore.progress?.percentage || 0);
const totalRecords = computed(() => monitorStore.progress?.totalRecords || 0);
const processedRecords = computed(() => monitorStore.progress?.processedRecords || 0);
const speed = computed(() => monitorStore.progress?.speed || 0);
const remainingTime = computed(() => {
  const seconds = monitorStore.progress?.estimatedTime || 0;
  if (seconds < 60) return `${seconds}秒`;
  const minutes = Math.floor(seconds / 60);
  return `${minutes}分${seconds % 60}秒`;
});

const progressStatus = computed(() => {
  if (task.value?.status === 'failed') return 'error';
  if (task.value?.status === 'completed') return 'success';
  return 'info';
});

const hasFailedUnits = computed(() => taskUnits.value.some(u => u.status === 'failed'));

const unitColumns = [
  { title: '单元名称', key: 'unitName' },
  { 
    title: '状态', 
    key: 'status',
    render(row: TaskUnit) {
      const typeMap: any = { pending: 'default', running: 'primary', completed: 'success', failed: 'error' };
      const textMap: any = { pending: '等待', running: '进行中', completed: '已完成', failed: '失败' };
      return h(NTag, { type: typeMap[row.status], size: 'small' }, { default: () => textMap[row.status] });
    }
  },
  { 
    title: '进度', 
    key: 'progress',
    render(row: TaskUnit) {
      const percentage = row.totalRecords > 0 
        ? Math.round((row.processedRecords / row.totalRecords) * 100) 
        : (row.status === 'completed' ? 100 : 0);
      return h(NProgress, {
        type: 'line',
        percentage: percentage,
        indicatorPlacement: 'inside',
        processing: row.status === 'running'
      });
    }
  },
  { title: '记录数', key: 'totalRecords', render: (row: TaskUnit) => formatNumber(row.totalRecords) },
  { title: '错误信息', key: 'errorMessage', ellipsis: true }
];

async function loadTask() {
  if (!taskId.value) return;
  try {
    task.value = (await taskApi.list()).find(t => t.id === taskId.value) || null;
    const response = await monitorApi.getTaskUnits(taskId.value);
    taskUnits.value = response.units;
    
    // 如果没有实时进度，则从 API 获取一次
    if (!monitorStore.progress) {
      const progress = await monitorApi.getProgress(taskId.value);
      if (progress) {
        monitorStore.progress = progress;
      }
    }
    
    // 如果没有日志，获取一次
    if (monitorStore.logs.length === 0) {
      monitorStore.logs = await monitorApi.getLogs(taskId.value);
    }
  } catch (e) {
    showError('加载任务详情失败');
  }
}

async function handleStart() {
  try {
    await monitorApi.startTask(taskId.value);
    showSuccess('任务已启动');
    loadTask();
  } catch (e) {
    showError('启动失败: ' + e);
  }
}

async function handlePause() {
  try {
    await monitorApi.pauseTask(taskId.value);
    showSuccess('任务已暂停');
    loadTask();
  } catch (e) {
    showError('暂停失败: ' + e);
  }
}

async function handleResume() {
  try {
    await monitorApi.resumeTask(taskId.value);
    showSuccess('任务已继续');
    loadTask();
  } catch (e) {
    showError('继续失败: ' + e);
  }
}

async function handleStop() {
  try {
    await monitorApi.stopTask(taskId.value);
    showSuccess('任务已停止');
    loadTask();
  } catch (e) {
    showError('停止失败: ' + e);
  }
}

async function handleResetFailed() {
  try {
    const count = await monitorApi.resetFailedUnits(taskId.value);
    showSuccess(`已重置 ${count} 个失败单元`);
    handleResume();
  } catch (e) {
    showError('重置失败: ' + e);
  }
}

// 自动滚动日志
watch(() => monitorStore.logs.length, () => {
  if (logContainer.value) {
    setTimeout(() => {
      logContainer.value!.scrollTop = 0; // 最新日志在顶部，所以滚回顶
    }, 0);
  }
});

onMounted(() => {
  loadTask();
  monitorStore.setActiveTask(taskId.value);
  monitorStore.initListeners();
});

onUnmounted(() => {
  monitorStore.setActiveTask('');
});
</script>

<style scoped>
.page-container {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.monitor-header {
  margin-bottom: 20px;
}

.stat-value {
  font-size: 18px;
  font-weight: bold;
}

.failed-actions {
  margin-top: 16px;
  display: flex;
  justify-content: center;
}

.log-container {
  height: 500px;
  background-color: #1e1e1e;
  color: #d4d4d4;
  padding: 12px;
  font-family: 'Fira Code', monospace;
  font-size: 13px;
  overflow-y: auto;
  border-radius: 4px;
  display: flex;
  flex-direction: column;
}

.log-item {
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
}

.log-item.info { color: #9cdcfe; }
.log-item.warn { color: #ce9178; }
.log-item.error { color: #f44747; }

.log-time { color: #808080; margin-right: 8px; }
.log-category { color: #569cd6; margin-right: 8px; }
</style>
