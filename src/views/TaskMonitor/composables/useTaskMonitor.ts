import { ref, computed } from 'vue'
import { useRoute } from 'vue-router'
import { useSyncTaskStore } from '../../../stores/syncTask'
import { useTaskMonitorStore } from '../../../stores/taskMonitor'
import { useDataSourceStore } from '../../../stores/dataSource'
import { showSuccess, handleApiError } from '../../../utils/message'
import { listen, type UnlistenFn } from '@tauri-apps/api/event'

export function useTaskMonitor() {
  const syncTaskStore = useSyncTaskStore()
  const taskMonitorStore = useTaskMonitorStore()
  const dataSourceStore = useDataSourceStore()
  const route = useRoute()

  // ========== 状态 ==========
  const selectedTaskId = ref<string>('')
  const allLogs = ref<any[]>([])
  const allLogContentRef = ref<HTMLElement>()
  const detailLogContentRef = ref<HTMLElement>()
  const verifyLogContentRef = ref<HTMLElement>()
  const errorLogContentRef = ref<HTMLElement>()

  let unlistenLog: UnlistenFn | null = null
  let unlistenError: UnlistenFn | null = null

  // ========== 计算属性 ==========
  const progress = computed(() => {
    if (!selectedTaskId.value) return null
    return taskMonitorStore.progress
  })

  const detailLogs = computed(() => {
    return allLogs.value.filter(log => log.category === 'summary')
  })

  const verifyLogs = computed(() => {
    return allLogs.value.filter(log => log.category === 'verify')
  })

  const sortedTaskUnits = computed(() => {
    if (!progress.value?.taskUnits) return []
    
    const units = [...progress.value.taskUnits]
    const statusOrder: Record<string, number> = { running: 1, pending: 2, completed: 3, failed: 4 }
    
    return units.sort((a, b) => {
      const statusDiff = (statusOrder[a.status] || 99) - (statusOrder[b.status] || 99)
      if (statusDiff !== 0) return statusDiff
      return a.name.localeCompare(b.name)
    })
  })

  const isRunning = computed(() => progress.value?.status === 'running')
  const isPaused = computed(() => progress.value?.status === 'paused')

  // ========== 辅助函数 ==========
  function getSelectedTask() {
    return syncTaskStore.tasks.find(t => t.id === selectedTaskId.value)
  }

  function getSelectedSourceDataSource() {
    const task = getSelectedTask()
    if (!task) return null
    return dataSourceStore.dataSources.find(ds => ds.id === task.sourceId)
  }

  function getSelectedTargetDataSource() {
    const task = getSelectedTask()
    if (!task) return null
    return dataSourceStore.dataSources.find(ds => ds.id === task.targetId)
  }

  function getTaskProgress(taskId: string) {
    if (taskId === selectedTaskId.value) {
      return taskMonitorStore.progress
    }
    return null
  }

  function getDataSourceName(dataSourceId: string): string {
    const ds = dataSourceStore.dataSources.find(d => d.id === dataSourceId)
    return ds?.name || '未知数据源'
  }

  function getUnitCountByStatus(status: string): number {
    if (!progress.value?.taskUnits) return 0
    return progress.value.taskUnits.filter((unit: any) => unit.status === status).length
  }

  function formatTime(seconds: number): string {
    if (!seconds || seconds <= 0) return '-'
    
    const hours = Math.floor(seconds / 3600)
    const minutes = Math.floor((seconds % 3600) / 60)
    const secs = Math.floor(seconds % 60)
    
    if (hours > 0) return `${hours}h${minutes}m`
    else if (minutes > 0) return `${minutes}m${secs}s`
    else return `${secs}s`
  }

  function scrollToBottom(element?: HTMLElement) {
    if (element) {
      element.scrollTop = element.scrollHeight
    }
  }

  // ========== 任务操作 ==========
  function handleTaskSelect(taskId: string) {
    selectedTaskId.value = taskId
    allLogs.value = []
    loadProgress()
    loadErrors()
    loadLogs()
  }

  async function handleStart() {
    if (!selectedTaskId.value) return
    
    const task = getSelectedTask()
    if (!task) return
    
    if (!task.sourceId || !task.targetId) {
      handleApiError(new Error('任务未配置数据源，请先完成任务配置'), '启动任务失败')
      return
    }
    
    const hasMySQLConfig = task.mysqlConfig?.databases?.length && task.mysqlConfig.databases.length > 0
    const hasESIndices = task.esConfig?.indices?.length && task.esConfig.indices.length > 0
    const hasESSelectedIndices = task.esConfig?.selectedIndices?.length && task.esConfig.selectedIndices.length > 0
    
    if (!hasMySQLConfig && !hasESIndices && !hasESSelectedIndices) {
      handleApiError(new Error('任务未配置同步的表/索引，请先完成任务配置'), '启动任务失败')
      return
    }
    
    try {
      await taskMonitorStore.startTask(selectedTaskId.value)
      showSuccess('任务已启动')
    } catch (error) {
      handleApiError(error, '启动任务失败')
    }
  }

  async function handlePause() {
    if (!selectedTaskId.value) return
    try {
      await taskMonitorStore.pauseTask(selectedTaskId.value)
      showSuccess('任务已暂停')
    } catch (error) {
      handleApiError(error, '暂停任务失败')
    }
  }

  async function handleResume() {
    if (!selectedTaskId.value) return
    try {
      await taskMonitorStore.resumeTask(selectedTaskId.value)
      showSuccess('任务已恢复')
    } catch (error) {
      handleApiError(error, '恢复任务失败')
    }
  }

  // ========== 数据加载 ==========
  async function loadTasks() {
    try {
      await syncTaskStore.fetchTasks()
    } catch (error) {
      handleApiError(error, '加载任务列表失败')
    }
  }

  async function loadProgress() {
    if (!selectedTaskId.value) return
    try {
      await taskMonitorStore.getProgress(selectedTaskId.value)
    } catch (error) {
      // 静默失败
    }
  }

  async function loadErrors() {
    if (!selectedTaskId.value) return
    try {
      await taskMonitorStore.getErrors(selectedTaskId.value)
    } catch (error) {
      // 静默失败
    }
  }

  async function loadLogs() {
    if (!selectedTaskId.value) return
    try {
      allLogs.value = await taskMonitorStore.getTaskLogs(selectedTaskId.value)
      setTimeout(() => scrollToBottom(allLogContentRef.value), 100)
    } catch (error) {
      // 静默失败
    }
  }

  // ========== 事件监听 ==========
  async function startListening() {
    try {
      unlistenLog = await listen('task-log', (event: any) => {
        const logEvent = event.payload
        if (logEvent.taskId === selectedTaskId.value) {
          allLogs.value.push(logEvent.log)
          if (allLogs.value.length > 5000) {
            allLogs.value.shift()
          }
          setTimeout(() => {
            scrollToBottom(allLogContentRef.value)
            scrollToBottom(detailLogContentRef.value)
            scrollToBottom(verifyLogContentRef.value)
          }, 50)
        }
      })
      
      unlistenError = await listen('task-error', (event: any) => {
        const errorEvent = event.payload
        if (errorEvent.taskId === selectedTaskId.value) {
          loadErrors()
          setTimeout(() => scrollToBottom(errorLogContentRef.value), 50)
        }
      })
    } catch (error) {
      console.error('监听事件失败:', error)
    }
  }

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

  // ========== 初始化 ==========
  async function initialize() {
    await dataSourceStore.fetchDataSources()
    await loadTasks()
    await taskMonitorStore.startEventListeners()
    await startListening()
    
    const taskId = route.query.taskId as string
    if (taskId) {
      selectedTaskId.value = taskId
      loadProgress()
      loadErrors()
      loadLogs()
    }
  }

  function cleanup() {
    taskMonitorStore.stopEventListeners()
    stopListening()
  }

  return {
    // 状态
    selectedTaskId,
    allLogs,
    allLogContentRef,
    detailLogContentRef,
    verifyLogContentRef,
    errorLogContentRef,
    // 计算属性
    progress,
    detailLogs,
    verifyLogs,
    sortedTaskUnits,
    isRunning,
    isPaused,
    // Store
    syncTaskStore,
    taskMonitorStore,
    dataSourceStore,
    // 方法
    getSelectedTask,
    getSelectedSourceDataSource,
    getSelectedTargetDataSource,
    getTaskProgress,
    getDataSourceName,
    getUnitCountByStatus,
    formatTime,
    handleTaskSelect,
    handleStart,
    handlePause,
    handleResume,
    loadTasks,
    initialize,
    cleanup
  }
}
