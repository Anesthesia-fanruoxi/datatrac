/**
 * SyncTask Store - 同步任务管理
 * 
 * 职责：
 * - 管理同步任务的增删改查
 * - 查询数据库元数据（数据库、表、索引）
 * - 调用 Tauri Commands 与后端通信
 * 
 * 验证需求: 3.1, 3.2, 3.3, 3.4, 3.5, 3.6, 3.9, 9.3
 */

import { defineStore } from 'pinia';
import { ref } from 'vue';
import { invoke } from '@tauri-apps/api/core';
import type { SyncTask, IndexMatchResult } from '../types';

export const useSyncTaskStore = defineStore('syncTask', () => {
  // ==================== 状态 ====================
  
  const tasks = ref<SyncTask[]>([]);
  const currentTask = ref<SyncTask | null>(null);
  const loading = ref(false);
  const error = ref<string | null>(null);

  // ==================== 任务管理方法 ====================

  /**
   * 获取所有任务
   * 需求 3.9: 保存任务配置
   */
  async function fetchTasks(): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      const result = await invoke<any[]>('list_tasks');
      
      // 转换后端数据格式为前端格式
      tasks.value = result.map(task => {
        let config: any = { mysqlConfig: undefined, esConfig: undefined, syncConfig: undefined };
        
        try {
          if (task.config) {
            config = JSON.parse(task.config);
          }
        } catch (e) {
          console.error('解析任务配置失败:', e);
        }
        
        return {
          id: task.id,
          name: task.name,
          sourceId: task.sourceId,
          targetId: task.targetId,
          sourceType: task.sourceType,
          targetType: task.targetType,
          mysqlConfig: config.mysqlConfig,
          esConfig: config.esConfig,
          syncConfig: config.syncConfig && Object.keys(config.syncConfig).length > 0
            ? config.syncConfig
            : {
                threadCount: 4,
                batchSize: 2500,
                errorStrategy: 'skip' as const,
                tableExistsStrategy: 'drop' as const
              },
          status: task.status,
          createdAt: task.createdAt,
          updatedAt: task.updatedAt
        } as SyncTask;
      });
    } catch (e) {
      error.value = `获取任务列表失败: ${e}`;
      console.error('fetchTasks error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 获取单个任务
   */
  async function getTask(id: string): Promise<SyncTask | null> {
    try {
      const result = await invoke<any>('get_task', { id });
      
      if (!result) {
        return null;
      }
      
      let config: any = { mysqlConfig: undefined, esConfig: undefined, syncConfig: undefined };
      
      try {
        if (result.config) {
          config = JSON.parse(result.config);
        }
      } catch (e) {
        console.error('解析任务配置失败:', e);
      }
      
      const task: SyncTask = {
        id: result.id,
        name: result.name,
        sourceId: result.sourceId,
        targetId: result.targetId,
        sourceType: result.sourceType,
        targetType: result.targetType,
        mysqlConfig: config.mysqlConfig,
        esConfig: config.esConfig,
        syncConfig: config.syncConfig && Object.keys(config.syncConfig).length > 0 
          ? config.syncConfig 
          : {
              threadCount: 4,
              batchSize: 2500,
              errorStrategy: 'skip' as const,
              tableExistsStrategy: 'drop' as const
            },
        status: result.status,
        createdAt: result.createdAt,
        updatedAt: result.updatedAt
      };
      
      currentTask.value = task;
      return task;
    } catch (e) {
      error.value = `获取任务失败: ${e}`;
      console.error('getTask error:', e);
      return null;
    }
  }

  /**
   * 创建任务
   * 需求 3.1: 创建同步任务
   */
  async function createTask(task: Omit<SyncTask, 'id' | 'createdAt' | 'updatedAt'>): Promise<string> {
    loading.value = true;
    error.value = null;
    
    try {
      // 生成 UUID
      const id = crypto.randomUUID();
      const now = Date.now();
      
      // 构建配置对象
      const config = {
        mysqlConfig: task.mysqlConfig,
        esConfig: task.esConfig,
        syncConfig: task.syncConfig
      };
      
      // 构建后端期望的 SyncTask 格式
      const backendTask = {
        id,
        name: task.name,
        sourceId: task.sourceId,
        targetId: task.targetId,
        sourceType: task.sourceType,
        targetType: task.targetType,
        config: JSON.stringify(config),
        status: task.status,
        createdAt: now,
        updatedAt: now
      };
      
      await invoke('create_task', { task: backendTask });
      await fetchTasks(); // 刷新列表
      return id;
    } catch (e) {
      error.value = `创建任务失败: ${e}`;
      console.error('createTask error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 更新任务
   */
  async function updateTask(id: string, task: Partial<SyncTask>): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      const now = Date.now();
      
      // 构建配置对象
      const config = {
        mysqlConfig: task.mysqlConfig,
        esConfig: task.esConfig,
        syncConfig: task.syncConfig
      };
      
      console.log('updateTask - 准备更新任务:', {
        id,
        name: task.name,
        sourceId: task.sourceId,
        targetId: task.targetId,
        sourceType: task.sourceType,
        targetType: task.targetType,
        mysqlConfig: task.mysqlConfig,
        esConfig: task.esConfig,
        syncConfig: task.syncConfig
      });
      
      // 构建后端期望的 SyncTask 格式
      const backendTask = {
        id,
        name: task.name,
        sourceId: task.sourceId,
        targetId: task.targetId,
        sourceType: task.sourceType,
        targetType: task.targetType,
        config: JSON.stringify(config),
        status: task.status,
        createdAt: task.createdAt || now,
        updatedAt: now
      };
      
      console.log('updateTask - 发送到后端的数据:', backendTask);
      console.log('updateTask - config JSON:', backendTask.config);
      
      await invoke('update_task', { id, task: backendTask });
      
      console.log('updateTask - 更新成功');
      
      await fetchTasks(); // 刷新列表
    } catch (e) {
      error.value = `更新任务失败: ${e}`;
      console.error('updateTask error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 删除任务
   */
  async function deleteTask(id: string): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      await invoke('delete_task', { id });
      await fetchTasks(); // 刷新列表
    } catch (e) {
      error.value = `删除任务失败: ${e}`;
      console.error('deleteTask error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  // ==================== 元数据查询方法 ====================

  /**
   * 获取 MySQL 数据库列表
   * 需求 3.2: 选择要同步的数据库
   */
  async function fetchDatabases(sourceId: string): Promise<string[]> {
    loading.value = true;
    error.value = null;
    
    try {
      const result = await invoke<string[]>('get_databases', { sourceId });
      return result;
    } catch (e) {
      error.value = `获取数据库列表失败: ${e}`;
      console.error('fetchDatabases error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 获取 MySQL 表列表
   * 需求 3.3: 显示数据库下的所有表
   */
  async function fetchTables(sourceId: string, database: string): Promise<string[]> {
    loading.value = true;
    error.value = null;
    
    try {
      const result = await invoke<string[]>('get_tables', { sourceId, database });
      return result;
    } catch (e) {
      error.value = `获取表列表失败: ${e}`;
      console.error('fetchTables error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 获取 ES 索引列表
   * 需求 3.4: 选择要同步的索引
   */
  async function fetchIndices(sourceId: string): Promise<string[]> {
    loading.value = true;
    error.value = null;
    
    try {
      const result = await invoke<string[]>('get_indices', { sourceId });
      return result;
    } catch (e) {
      error.value = `获取索引列表失败: ${e}`;
      console.error('fetchIndices error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 通配符匹配 ES 索引
   * 需求 3.5, 3.6: 支持通配符匹配并显示匹配结果
   */
  async function matchIndices(sourceId: string, pattern: string): Promise<IndexMatchResult> {
    loading.value = true;
    error.value = null;
    
    try {
      const result = await invoke<IndexMatchResult>('match_indices', { sourceId, pattern });
      return result;
    } catch (e) {
      error.value = `匹配索引失败: ${e}`;
      console.error('matchIndices error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 设置当前任务
   */
  function setCurrentTask(task: SyncTask | null): void {
    currentTask.value = task;
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
    tasks,
    currentTask,
    loading,
    error,
    
    // 任务管理方法
    fetchTasks,
    getTask,
    createTask,
    updateTask,
    deleteTask,
    
    // 元数据查询方法
    fetchDatabases,
    fetchTables,
    fetchIndices,
    matchIndices,
    
    // 工具方法
    setCurrentTask,
    clearError,
  };
});
