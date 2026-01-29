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

  let unlistenLog: UnlistenFn | null = null

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

  const sortedTaskUnits = computed<import('../../../types').TaskUnit[]>(() => {
    if (!progress.value?.taskUnits) return []
    
    const units = [...progress.value.taskUnits]
    // 排序: 进行中 > 等待 > 失败 > 已完成
    const statusOrder: Record<string, number> = { 
      running: 1, 
      pending: 2, 
      failed: 3, 
      completed: 4 
    }
    
    return units.sort((a, b) => {
      const statusDiff = (statusOrder[a.status] || 99) - (statusOrder[b.status] || 99)
      if (statusDiff !== 0) return statusDiff
      return a.name.localeCompare(b.name)
    })
  })

  const isRunning = computed(() => progress.value?.status === 'running')
  const isPaused = computed(() => progress.value?.status === 'paused')

  // 已处理记录数（格式化显示）
  const processedRecordsFormatted = computed(() => {
    const count = progress.value?.processedRecords || 0
    if (count >= 1000000) {
      return (count / 1000000).toFixed(2) + 'M'
    } else if (count >= 1000) {
      return (count / 1000).toFixed(1) + 'K'
    }
    return count.toLocaleString()
  })

  // 总进度百分比
  const totalProgressFormatted = computed(() => {
    const percentage = progress.value?.percentage || 0
    return percentage.toFixed(1)
  })

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

  // ========== 任务操作 ==========
  function handleTaskSelect(taskId: string) {
    selectedTaskId.value = taskId
    allLogs.value = []
    // 使用 Promise.all 确保所有异步操作都正确处理
    Promise.all([
      loadProgress().catch(() => {}),
      loadErrors().catch(() => {}),
      loadLogs().catch(() => {})
    ]).catch(() => {})
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
      // 同时加载任务单元
      await loadTaskUnits()
    } catch (error) {
      // 静默失败
      console.warn('[useTaskMonitor] 加载进度失败:', error)
    }
  }

  async function loadTaskUnits() {
    if (!selectedTaskId.value) return
    try {
      const response = await taskMonitorStore.getTaskUnits(selectedTaskId.value)
      
      // 响应包含 newUnits, completedUnits 和 statistics
      const newUnits = response.newUnits || []
      const completedUnits = response.completedUnits || []
      const statistics = response.statistics || {
        total: 0,
        pending: 0,
        running: 0,
        completed: 0,
        failed: 0,
        paused: 0
      }
      
      // 转换新增单元数据
      const newTaskUnits = newUnits.map((unit: any) => {
        const totalRecords = unit.totalRecords || 0
        const processedRecords = unit.processedRecords || 0
        const percentage = totalRecords > 0 
          ? (processedRecords / totalRecords) * 100 
          : 0
        
        return {
          id: unit.id,
          name: unit.unitName,
          status: unit.status,
          totalRecords: totalRecords,
          processedRecords: processedRecords,
          percentage: percentage,
          errorMessage: unit.errorMessage,
          searchPattern: unit.searchPattern,
          isNew: true, // 标记为新增
        }
      })
      
      // 转换已完成单元数据
      const completedTaskUnits = completedUnits.map((unit: any) => {
        return {
          id: unit.id,
          name: unit.unitName,
          status: 'completed',
          totalRecords: unit.totalRecords || 0,
          processedRecords: unit.totalRecords || 0,
          percentage: 100,
          errorMessage: null,
          searchPattern: unit.searchPattern,
          completedAt: unit.completedAt,
          duration: unit.duration,
          isNew: false, // 标记为已完成
        }
      })
      
      // 合并所有单元
      const allTaskUnits = [...newTaskUnits, ...completedTaskUnits]
      
      // 将任务单元数据设置到 progress
      if (!progress.value) {
        // 如果 progress 为空,创建一个初始对象
        taskMonitorStore.progress = {
          taskId: selectedTaskId.value,
          status: 'idle',
          totalRecords: 0,
          processedRecords: 0,
          percentage: 0,
          speed: 0,
          estimatedTime: 0,
          startTime: '',
          taskUnits: allTaskUnits,
          statistics: statistics
        }
      } else {
        progress.value.taskUnits = allTaskUnits
        progress.value.statistics = statistics
      }
    } catch (error) {
      console.error('[useTaskMonitor] 加载任务单元失败:', error)
    }
  }


  async function loadErrors() {
    if (!selectedTaskId.value) return
    try {
      await taskMonitorStore.getErrors(selectedTaskId.value)
    } catch (error) {
      // 静默失败
      console.warn('[useTaskMonitor] 加载错误日志失败:', error)
    }
  }

  async function loadLogs() {
    if (!selectedTaskId.value) return
    try {
      allLogs.value = await taskMonitorStore.getTaskLogs(selectedTaskId.value)
    } catch (error) {
      // 静默失败
      console.warn('[useTaskMonitor] 加载日志失败:', error)
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
        }
      })
      
      // 监听任务单元更新事件
      await listen('task-units-updated', (event: any) => {
        const { taskId } = event.payload
        if (taskId === selectedTaskId.value) {
          loadTaskUnits().catch(() => {})
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
  }

  // ========== 初始化 ==========
  async function initialize() {
    try {
      await dataSourceStore.fetchDataSources()
      await loadTasks()
      await taskMonitorStore.startEventListeners()
      await startListening()
      
      const taskId = route.query.taskId as string
      if (taskId) {
        selectedTaskId.value = taskId
        // 使用 Promise.all 确保所有异步操作都正确处理
        await Promise.all([
          loadProgress().catch(() => {}),
          loadErrors().catch(() => {}),
          loadLogs().catch(() => {})
        ])
      }
    } catch (error) {
      console.error('[useTaskMonitor] 初始化失败:', error)
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
    // 计算属性
    progress,
    detailLogs,
    verifyLogs,
    sortedTaskUnits,
    isRunning,
    isPaused,
    processedRecordsFormatted,
    totalProgressFormatted,
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
