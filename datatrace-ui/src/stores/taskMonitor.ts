import { computed, reactive } from 'vue'
import { taskApi } from '../api/task'
import { createTaskMonitorSSE } from '../api/task-monitor-sse'
import type {
  IncrementalDatabaseStats,
  LogCategory,
  TargetProgress,
  TaskDetail,
  TaskItem,
  TaskLogEntry,
  TaskProgress
} from '../types/task-monitor'

type TaskTab = 'full' | 'incremental' | 'structure'

interface TaskMonitorState {
  loading: boolean
  tasks: TaskItem[]
  selectedTab: TaskTab
  currentTask: TaskDetail | null
  currentProgress: TaskProgress | null
  currentTargetId: string | null
  currentDatabase: string | null
  logs: TaskLogEntry[]
  currentLogCategory: LogCategory
  expandedDatabase: string | null
}

const state = reactive<TaskMonitorState>({
  loading: false,
  tasks: [],
  selectedTab: 'incremental',
  currentTask: null,
  currentProgress: null,
  currentTargetId: null,
  currentDatabase: null,
  logs: [],
  currentLogCategory: 'all',
  expandedDatabase: null
})

let detailSSE: EventSource | null = null
let progressSSE: EventSource | null = null
let logsSSE: EventSource | null = null

const incrementalTargets = computed(() => state.currentTask?.target_conns ?? [])
const selectedTargetStats = computed<TargetProgress | null>(() => {
  const targets = state.currentProgress?.target_stats ?? []
  if (targets.length === 0) return null
  return targets.find((item) => item.target_id === state.currentTargetId) ?? targets[0] ?? null
})

const selectedDatabaseStats = computed<IncrementalDatabaseStats[]>(() => selectedTargetStats.value?.database_stats ?? [])

function filterTasksByTab(tab: TaskTab) {
  return state.tasks.filter((task) => (task.status === 'configured' || task.is_running || task.status === 'completed') && task.sync_mode === tab)
}

function closeAllSSE() {
  detailSSE?.close()
  progressSSE?.close()
  logsSSE?.close()
  detailSSE = null
  progressSSE = null
  logsSSE = null
}

function buildProgressUrl(taskId: string, targetId?: string | null, database?: string | null) {
  const params = new URLSearchParams()
  if (targetId) params.set('target_id', targetId)
  if (database) params.set('database', database)
  const query = params.toString()
  return `/api/v1/tasks/${taskId}/stream/progress${query ? `?${query}` : ''}`
}

function connectProgressSSE() {
  if (!state.currentTask?.id) return
  progressSSE?.close()
  progressSSE = createTaskMonitorSSE<TaskProgress>(
    buildProgressUrl(state.currentTask.id, state.currentTargetId, state.currentDatabase),
    'progress',
    (payload) => {
      state.currentProgress = payload
      if (!state.currentTargetId && payload.target_stats?.length) {
        state.currentTargetId = payload.target_stats[0].target_id
      }
    }
  )
  progressSSE.onerror = () => {
    console.warn('progress SSE 连接异常，等待自动重连')
  }
}

function connectDetailSSE(taskId: string) {
  detailSSE?.close()
  detailSSE = createTaskMonitorSSE<TaskDetail>(`/api/v1/tasks/${taskId}/stream/detail`, 'task_detail', (payload) => {
    state.currentTask = {
      ...(state.currentTask ?? { id: taskId, name: '', source_type: '', target_type: '', status: 'configured', is_running: false, sync_mode: 'incremental' }),
      ...payload
    }
    if (payload.target_conns?.length) {
      const matched = payload.target_conns.find((item) => item.id === state.currentTargetId)
      if (!matched) {
        state.currentTargetId = payload.target_conns[0].id
        state.currentDatabase = null
        connectProgressSSE()
      }
    }
    patchTask(payload)
  })
  detailSSE.onerror = () => {
    console.warn('detail SSE 连接异常，等待自动重连')
  }
}

function connectLogsSSE() {
  if (!state.currentTask?.id) return
  logsSSE?.close()
  state.logs = []
  logsSSE = createTaskMonitorSSE<TaskLogEntry[] | TaskLogEntry>(
    `/api/v1/tasks/${state.currentTask.id}/stream/logs?category=${state.currentLogCategory}`,
    'log',
    (payload) => {
      const entries = Array.isArray(payload) ? payload : [payload]
      state.logs.push(...entries)
      if (state.logs.length > 2000) {
        state.logs = state.logs.slice(-1000)
      }
    }
  )
  logsSSE.onopen = () => {
    state.logs = []
  }
  logsSSE.onerror = () => {
    console.warn('logs SSE 连接异常，等待自动重连')
  }
}

function patchTask(detail: Partial<TaskDetail>) {
  const index = state.tasks.findIndex((item) => item.id === detail.id)
  if (index >= 0) {
    state.tasks[index] = { ...state.tasks[index], ...detail } as TaskItem
  }
}

async function loadTasks(preferredTaskId?: string) {
  state.loading = true
  try {
    state.tasks = await taskApi.list()
    const selectedId = preferredTaskId || state.currentTask?.id || filterTasksByTab(state.selectedTab)[0]?.id
    if (selectedId) {
      await selectTask(selectedId)
    } else {
      closeAllSSE()
      state.currentTask = null
      state.currentProgress = null
      state.logs = []
      state.currentTargetId = null
      state.currentDatabase = null
      state.expandedDatabase = null
    }
  } finally {
    state.loading = false
  }
}

async function selectTask(taskId: string) {
  const task = state.tasks.find((item) => item.id === taskId)
  if (!task) return
  state.selectedTab = task.sync_mode
  state.currentTask = { ...task }
  state.currentProgress = null
  state.logs = []
  state.currentDatabase = null
  state.expandedDatabase = null
  if (!state.currentTargetId || !state.currentTask.target_conns?.some((item) => item.id === state.currentTargetId)) {
    state.currentTargetId = task.target_id ?? null
  }
  connectDetailSSE(taskId)
  connectProgressSSE()
  connectLogsSSE()
}

async function startTask(taskId: string) {
  await taskApi.start(taskId)
  patchTask({ id: taskId, is_running: true, status: 'running' })
  if (state.currentTask?.id === taskId) {
    state.currentTask = { ...state.currentTask, is_running: true, status: 'running' }
  }
}

async function pauseTask(taskId: string) {
  await taskApi.pause(taskId)
  patchTask({ id: taskId, is_running: false, status: 'paused' })
  if (state.currentTask?.id === taskId) {
    state.currentTask = { ...state.currentTask, is_running: false, status: 'paused' }
  }
}

async function stopTask(taskId: string) {
  await taskApi.stop(taskId)
  patchTask({ id: taskId, is_running: false, status: 'configured' })
  if (state.currentTask?.id === taskId) {
    state.currentTask = { ...state.currentTask, is_running: false, status: 'configured' }
  }
}

function setTab(tab: TaskTab) {
  state.selectedTab = tab
  const candidates = filterTasksByTab(tab)
  if (candidates.length > 0) {
    void selectTask(candidates[0].id)
  } else {
    closeAllSSE()
    state.currentTask = null
    state.currentProgress = null
    state.logs = []
    state.currentTargetId = null
    state.currentDatabase = null
    state.expandedDatabase = null
  }
}

function switchTarget(targetId: string) {
  if (state.currentTargetId === targetId) return
  state.currentTargetId = targetId
  state.currentDatabase = null
  state.expandedDatabase = null
  connectProgressSSE()
}

function switchDatabase(database: string | null) {
  state.currentDatabase = state.expandedDatabase === database ? null : database
  state.expandedDatabase = state.currentDatabase
  connectProgressSSE()
}

function switchLogCategory(category: LogCategory) {
  state.currentLogCategory = category
  connectLogsSSE()
}

function dispose() {
  closeAllSSE()
}

export function useTaskMonitorStore() {
  return {
    state,
    incrementalTargets,
    selectedTargetStats,
    selectedDatabaseStats,
    filterTasksByTab,
    loadTasks,
    selectTask,
    startTask,
    pauseTask,
    stopTask,
    setTab,
    switchTarget,
    switchDatabase,
    switchLogCategory,
    dispose
  }
}
