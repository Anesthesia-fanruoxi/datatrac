/**
 * TaskMonitor Store - 任务执行监控
 * 
 * 职责：
 * - 管理任务执行状态
 * - 监听后端进度和错误事件
 * - 提供任务控制方法（启动、暂停、恢复）
 * 
 * 验证需求: 5.1, 5.2, 5.3, 6.1, 6.2, 6.3, 6.4, 6.5, 7.3, 9.4
 */

import { defineStore } from 'pinia';
import { ref, computed } from 'vue';
import { invoke } from '@tauri-apps/api/core';
import { listen, type UnlistenFn } from '@tauri-apps/api/event';
import type { TaskProgress, ErrorLog } from '../types';

export const useTaskMonitorStore = defineStore('taskMonitor', () => {
  // ==================== 状态 ====================
  
  const progress = ref<TaskProgress | null>(null);
  const errors = ref<ErrorLog[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);
  
  // 事件监听器引用
  let progressUnlisten: UnlistenFn | null = null;
  let errorUnlisten: UnlistenFn | null = null;

  // ==================== 计算属性 ====================

  /**
   * 是否有任务正在运行
   */
  const isRunning = computed(() => {
    return progress.value?.status === 'running';
  });

  /**
   * 是否任务已暂停
   */
  const isPaused = computed(() => {
    return progress.value?.status === 'paused';
  });

  /**
   * 错误总数
   */
  const errorCount = computed(() => {
    return errors.value.length;
  });

  // ==================== 任务控制方法 ====================

  /**
   * 启动同步任务
   * 需求 5.1: 启动任务
   */
  async function startTask(taskId: string): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      // 清空之前的进度和错误
      progress.value = null;
      errors.value = [];
      
      // 启动事件监听
      await startEventListeners();
      
      // 传递任务 ID
      await invoke('start_sync', { taskId });
    } catch (e) {
      error.value = `启动任务失败: ${e}`;
      console.error('startTask error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 暂停同步任务
   * 需求 5.2: 暂停任务
   */
  async function pauseTask(taskId: string): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      await invoke('pause_sync', { taskId });
    } catch (e) {
      error.value = `暂停任务失败: ${e}`;
      console.error('pauseTask error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 恢复同步任务
   * 需求 5.3: 恢复任务
   */
  async function resumeTask(taskId: string): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      await invoke('resume_sync', { taskId });
    } catch (e) {
      error.value = `恢复任务失败: ${e}`;
      console.error('resumeTask error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 获取任务进度
   * 需求 6.1, 6.2, 6.3, 6.4, 6.5: 显示进度信息
   */
  async function getProgress(taskId: string): Promise<TaskProgress | null> {
    try {
      const result = await invoke<TaskProgress>('get_progress', { taskId });
      progress.value = result;
      return result;
    } catch (e) {
      error.value = `获取进度失败: ${e}`;
      console.error('getProgress error:', e);
      return null;
    }
  }

  /**
   * 获取错误日志
   * 需求 7.3: 查看错误日志
   */
  async function getErrors(taskId: string): Promise<ErrorLog[]> {
    try {
      const result = await invoke<ErrorLog[]>('get_errors', { taskId });
      errors.value = result;
      return result;
    } catch (e) {
      error.value = `获取错误日志失败: ${e}`;
      console.error('getErrors error:', e);
      return [];
    }
  }

  /**
   * 获取任务日志
   */
  async function getTaskLogs(taskId: string): Promise<any[]> {
    try {
      const result = await invoke<any[]>('get_task_logs', { taskId });
      return result;
    } catch (e) {
      error.value = `获取任务日志失败: ${e}`;
      console.error('getTaskLogs error:', e);
      return [];
    }
  }

  /**
   * 获取任务单元列表（包含状态统计）
   * 用于断点续传和进度展示
   * 返回: { newUnits: 新增同步, completedUnits: 已完成同步, statistics: 统计 }
   */
  async function getTaskUnits(taskId: string): Promise<{ newUnits: any[], completedUnits: any[], statistics: any }> {
    try {
      const result = await invoke<{ newUnits: any[], completedUnits: any[], statistics: any }>('get_task_units', { taskId });
      return result;
    } catch (e) {
      error.value = `获取任务单元失败: ${e}`;
      console.error('getTaskUnits error:', e);
      return { newUnits: [], completedUnits: [], statistics: { total: 0, pending: 0, running: 0, completed: 0, failed: 0, paused: 0 } };
    }
  }

  // ==================== 事件监听方法 ====================

  /**
   * 启动事件监听器
   * 监听后端发送的进度和错误事件
   */
  async function startEventListeners(): Promise<void> {
    // 如果已有监听器，先清理
    await stopEventListeners();
    
    try {
      // 监听进度更新事件
      progressUnlisten = await listen<TaskProgress>('task-progress', (event) => {
        console.log('[task-progress] 收到进度更新事件:', event.payload)
        
        // 合并数据而不是直接覆盖，保留 taskUnits
        if (progress.value) {
          // 保存当前的 taskUnits
          const currentTaskUnits = progress.value.taskUnits
          
          // 更新其他字段
          Object.assign(progress.value, event.payload)
          
          // 恢复 taskUnits（因为后端推送的 taskUnits 是空的）
          progress.value.taskUnits = currentTaskUnits
          
          console.log('[task-progress] 更新后 taskUnits 数量:', progress.value.taskUnits?.length)
        } else {
          progress.value = event.payload
        }
      });
      
      // 监听错误事件
      errorUnlisten = await listen<ErrorLog>('task-error', (event) => {
        errors.value.push(event.payload);
      });
    } catch (e) {
      console.error('Failed to start event listeners:', e);
      throw e;
    }
  }

  /**
   * 停止事件监听器
   */
  async function stopEventListeners(): Promise<void> {
    if (progressUnlisten) {
      progressUnlisten();
      progressUnlisten = null;
    }
    
    if (errorUnlisten) {
      errorUnlisten();
      errorUnlisten = null;
    }
  }

  /**
   * 清空进度和错误
   */
  function clearMonitorData(): void {
    progress.value = null;
    errors.value = [];
  }

  /**
   * 清除错误信息
   */
  function clearError(): void {
    error.value = null;
  }

  /**
   * 重置失败的任务单元
   */
  async function resetFailedUnits(taskId: string): Promise<number> {
    try {
      const count = await invoke<number>('reset_failed_units', { taskId });
      return count;
    } catch (e) {
      error.value = `重置失败单元失败: ${e}`;
      console.error('resetFailedUnits error:', e);
      throw e;
    }
  }

  /**
   * 重置指定的任务单元
   */
  async function resetUnit(unitId: string): Promise<void> {
    try {
      await invoke('reset_unit', { unitId });
    } catch (e) {
      error.value = `重置单元失败: ${e}`;
      console.error('resetUnit error:', e);
      throw e;
    }
  }
  
  /**
   * 按搜索关键字清除任务单元记录
   */
  async function clearTaskUnitsByPattern(taskId: string, pattern: string): Promise<number> {
    try {
      const count = await invoke<number>('clear_task_units_by_pattern', { taskId, pattern });
      return count;
    } catch (e) {
      error.value = `清除记录失败: ${e}`;
      console.error('clearTaskUnitsByPattern error:', e);
      throw e;
    }
  }

  // ==================== 返回 ====================

  return {
    // 状态
    progress,
    errors,
    loading,
    error,
    
    // 计算属性
    isRunning,
    isPaused,
    errorCount,
    
    // 任务控制方法
    startTask,
    pauseTask,
    resumeTask,
    getProgress,
    getErrors,
    getTaskLogs,
    getTaskUnits,
    resetFailedUnits,
    resetUnit,
    clearTaskUnitsByPattern,
    
    // 事件监听方法
    startEventListeners,
    stopEventListeners,
    
    // 工具方法
    clearMonitorData,
    clearError,
  };
});
