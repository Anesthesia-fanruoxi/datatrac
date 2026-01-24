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
      
      // 调用后端启动任务
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
        progress.value = event.payload;
        console.log('Progress update:', event.payload);
      });
      
      // 监听错误事件
      errorUnlisten = await listen<ErrorLog>('task-error', (event) => {
        errors.value.push(event.payload);
        console.log('Error logged:', event.payload);
      });
      
      console.log('Event listeners started');
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
    
    console.log('Event listeners stopped');
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
    
    // 事件监听方法
    startEventListeners,
    stopEventListeners,
    
    // 工具方法
    clearMonitorData,
    clearError,
  };
});
