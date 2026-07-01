<template>
  <div v-if="store.state.loading || store.state.tasks.length > 0" class="task-monitor-layout">
    <TaskListPanel
      :tasks="tabTasks"
      :loading="store.state.loading"
      :selected-tab="store.state.selectedTab"
      :current-task-id="store.state.currentTask?.id"
      @select="handleSelect"
      @start="runAction('start', $event)"
      @pause="runAction('pause', $event)"
      @stop="runAction('stop', $event)"
      @tab-change="handleTabChange"
    />
    <div class="task-monitor-right">
      <TaskDetailPanel
        :task="store.state.currentTask"
        :progress="store.state.currentProgress"
        :current-target-id="store.state.currentTargetId"
        :database-stats="selectedDatabaseStats"
        :table-stats="store.state.currentProgress?.table_stats || []"
        :expanded-database="store.state.expandedDatabase"
        @switch-target="store.switchTarget"
        @switch-database="store.switchDatabase"
      />
      <LogPanel :logs="store.state.logs" :model-value="store.state.currentLogCategory" @update:model-value="store.switchLogCategory" />
    </div>
  </div>
  <section v-else class="page-card">
    <div class="page-header">
      <h2>任务监控</h2>
      <el-tag type="info">暂无任务</el-tag>
    </div>
    <p>当前没有可监控的任务，请先到任务配置页面创建并配置任务。</p>
  </section>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ElMessage, ElMessageBox } from 'element-plus'
import TaskListPanel from '../../components/task-monitor/TaskListPanel.vue'
import TaskDetailPanel from '../../components/task-monitor/TaskDetailPanel.vue'
import LogPanel from '../../components/task-monitor/LogPanel.vue'
import { useTaskMonitorStore } from '../../stores/taskMonitor'

const route = useRoute()
const router = useRouter()
const store = useTaskMonitorStore()

const tabTasks = computed(() => store.filterTasksByTab(store.state.selectedTab))
const selectedDatabaseStats = computed(() => store.selectedDatabaseStats.value)

async function initialize() {
  const taskId = typeof route.params.id === 'string' ? route.params.id : undefined
  await store.loadTasks(taskId)
  syncRouteWithCurrentTask()
}

async function handleSelect(taskId: string) {
  await store.selectTask(taskId)
  syncRouteWithCurrentTask()
}

function handleTabChange(tab: 'full' | 'incremental' | 'structure') {
  store.setTab(tab)
  syncRouteWithCurrentTask()
}

function syncRouteWithCurrentTask() {
  const taskId = store.state.currentTask?.id
  if (taskId) {
    router.replace({ name: 'task-monitor', params: { id: taskId } })
  } else if (route.params.id) {
    router.replace({ name: 'task-monitor' })
  }
}

async function runAction(type: 'start' | 'pause' | 'stop', taskId: string) {
  const map = {
    start: store.startTask,
    pause: store.pauseTask,
    stop: store.stopTask
  }
  try {
    if (type === 'stop') {
      await ElMessageBox.confirm('确定停止该任务吗？', '提示', { type: 'warning' })
    }
    await map[type](taskId)
    ElMessage.success(`${type === 'start' ? '启动' : type === 'pause' ? '暂停' : '停止'}成功`)
  } catch (error) {
    if (error !== 'cancel') {
      ElMessage.error(error instanceof Error ? error.message : '操作失败')
    }
  }
}

watch(
  () => route.params.id,
  async (value) => {
    if (typeof value === 'string' && value !== store.state.currentTask?.id) {
      await store.selectTask(value)
    } else if (!value && store.state.currentTask?.id) {
      syncRouteWithCurrentTask()
    }
  }
)

onMounted(initialize)
onBeforeUnmount(store.dispose)
</script>
