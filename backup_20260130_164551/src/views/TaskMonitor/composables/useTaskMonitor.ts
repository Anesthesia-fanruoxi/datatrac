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
    console.log('[sortedTaskUnits] 计算中...')
    console.log('[sortedTaskUnits] progress.value:', progress.value)
    console.log('[sortedTaskUnits] progress.value?.taskUnits:', progress.value?.taskUnits)
    
    if (!progress.value?.taskUnits) {
      console.log('[sortedTaskUnits] 返回空数组')
      return []
    }
    
    const units = [...progress.value.taskUnits]
    console.log('[sortedTaskUnits] units 数量:', units.length)
    
    // 排序: 进行中 > 等待 > 失败 > 已完成
    const statusOrder: Record<string, number> = { 
      running: 1, 
      pending: 2, 
      failed: 3, 
      completed: 4 
    }
    
    const sorted = units.sort((a, b) => {
      const statusDiff = (statusOrder[a.status] || 99) - (statusOrder[b.status] || 99)
      if (statusDiff !== 0) return statusDiff
      return a.name.localeCompare(b.name)
    })
    
    console.log('[sortedTaskUnits] 排序后返回:', sorted.length, '个单元')
    return sorted
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
    if (taskId === selectedTaskId.value && taskMonitorStore.progress) {
      // 返回当前选中任务的运行时进度
      return taskMonitorStore.progress
    }
    
    // 返回任务的基本状态信息
    const task = syncTaskStore.tasks.find(t => t.id === taskId)
    if (task) {
      return {
        status: task.status || 'idle',
        percentage: 0
      }
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
  async function handleTaskSelect(taskId: string) {
    // 如果选择的是同一个任务，不需要重新加载
    if (selectedTaskId.value === taskId) {
      return
    }
    
    // 立即更新选中的任务 ID
    selectedTaskId.value = taskId
    
    // 清空日志和错误（这些是任务特定的）
    allLogs.value = []
    taskMonitorStore.errors = []
    
    // 同步加载新任务的数据（不清空 progress，让它自然更新）
    try {
      await loadProgress()
      await loadErrors()
      await loadLogs()
    } catch (error) {
      console.error('[useTaskMonitor] 加载任务数据失败:', error)
    }
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
      console.log('[handleStart] 开始启动任务:', selectedTaskId.value)
      await taskMonitorStore.startTask(selectedTaskId.value)
      console.log('[handleStart] 任务启动成功')
      
      showSuccess('任务已启动')
      
      // 重新加载任务列表，更新任务状态
      console.log('[handleStart] 重新加载任务列表')
      await loadTasks()
      console.log('[handleStart] 任务列表加载完成，tasks 数量:', syncTaskStore.tasks.length)
      
      // 重新加载任务单元（因为 startTask 会清空 progress）
      console.log('[handleStart] 重新加载任务单元')
      await loadTaskUnits()
      console.log('[handleStart] 任务单元加载完成')
    } catch (error) {
      console.error('[handleStart] 启动任务失败:', error)
      handleApiError(error, '启动任务失败')
    }
  }

  async function handlePause() {
    if (!selectedTaskId.value) return
    try {
      await taskMonitorStore.pauseTask(selectedTaskId.value)
      showSuccess('任务已暂停')
      // 重新加载任务列表，更新任务状态
      await loadTasks()
    } catch (error) {
      handleApiError(error, '暂停任务失败')
    }
  }

  async function handleResume() {
    if (!selectedTaskId.value) return
    try {
      await taskMonitorStore.resumeTask(selectedTaskId.value)
      showSuccess('任务已恢复')
      // 重新加载任务列表，更新任务状态
      await loadTasks()
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
      await loadTaskUnits()
    } catch (error) {
      console.warn('[useTaskMonitor] 加载进度失败:', error)
    }
  }

  async function loadTaskUnits() {
    if (!selectedTaskId.value) return
    
    try {
      console.log('[loadTaskUnits] 开始加载任务单元，taskId:', selectedTaskId.value)
      const response = await taskMonitorStore.getTaskUnits(selectedTaskId.value)
      console.log('[loadTaskUnits] 获取到响应:', response)
      
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
      
      console.log('[loadTaskUnits] newUnits 数量:', newUnits.length, 'completedUnits 数量:', completedUnits.length)
      
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
          isNew: true,
        }
      })
      
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
          isNew: false,
        }
      })
      
      const allTaskUnits = [...newTaskUnits, ...completedTaskUnits]
      console.log('[loadTaskUnits] 合并后的任务单元数量:', allTaskUnits.length)
      
      // 无论 progress 是否存在，都要确保创建或更新
      if (!taskMonitorStore.progress || taskMonitorStore.progress.taskId !== selectedTaskId.value) {
        console.log('[loadTaskUnits] progress 为空或任务ID不匹配，创建新的 progress 对象')
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
        console.log('[loadTaskUnits] 更新现有 progress 的 taskUnits')
        // 使用 Object.assign 强制触发响应式更新
        Object.assign(taskMonitorStore.progress, {
          taskUnits: allTaskUnits,
          statistics: statistics
        })
      }
      
      console.log('[loadTaskUnits] 最终 progress:', taskMonitorStore.progress)
      console.log('[loadTaskUnits] 最终 progress.taskUnits.length:', taskMonitorStore.progress?.taskUnits?.length)
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
      
      // 启动定期轮询
      startPolling()
    } catch (error) {
      console.error('监听事件失败:', error)
    }
  }
  
  // 定期轮询任务单元状态
  let pollingInterval: number | null = null
  
  function startPolling() {
    stopPolling()
    pollingInterval = window.setInterval(() => {
      if (selectedTaskId.value && isRunning.value) {
        console.log('[polling] 轮询更新任务单元')
        loadTaskUnits().catch(() => {})
      }
    }, 2000) // 每2秒轮询一次
  }
  
  function stopPolling() {
    if (pollingInterval) {
      window.clearInterval(pollingInterval)
      pollingInterval = null
    }
  }

  function stopListening() {
    if (unlistenLog) {
      unlistenLog()
      unlistenLog = null
    }
    stopPolling()
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
