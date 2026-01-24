/**
 * DataSource Store - 数据源管理
 * 
 * 职责：
 * - 管理数据源的增删改查
 * - 调用 Tauri Commands 与后端通信
 * - 测试数据源连接
 * 
 * 验证需求: 1.1, 1.4, 1.5, 1.6, 9.2
 */

import { defineStore } from 'pinia';
import { ref } from 'vue';
import { invoke } from '@tauri-apps/api/core';
import type { DataSource, ConnectionResult } from '../types';

export const useDataSourceStore = defineStore('dataSource', () => {
  // ==================== 状态 ====================
  
  const dataSources = ref<DataSource[]>([]);
  const loading = ref(false);
  const error = ref<string | null>(null);

  // ==================== 方法 ====================

  /**
   * 获取所有数据源
   * 需求 1.1: 查看数据源配置
   */
  async function fetchDataSources(): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      const result = await invoke<DataSource[]>('list_data_sources');
      dataSources.value = result;
    } catch (e) {
      error.value = `获取数据源列表失败: ${e}`;
      console.error('fetchDataSources error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 获取单个数据源
   */
  async function getDataSource(id: string): Promise<DataSource | null> {
    try {
      const result = await invoke<DataSource>('get_data_source', { id });
      return result;
    } catch (e) {
      error.value = `获取数据源失败: ${e}`;
      console.error('getDataSource error:', e);
      return null;
    }
  }

  /**
   * 添加数据源
   * 需求 1.1: 添加新数据源
   */
  async function addDataSource(ds: Omit<DataSource, 'id' | 'createdAt' | 'updatedAt'>): Promise<string> {
    loading.value = true;
    error.value = null;
    
    try {
      const id = await invoke<string>('create_data_source', { dataSource: ds });
      await fetchDataSources(); // 刷新列表
      return id;
    } catch (e) {
      error.value = `添加数据源失败: ${e}`;
      console.error('addDataSource error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 更新数据源
   * 需求 1.4: 编辑数据源
   */
  async function updateDataSource(id: string, ds: Partial<DataSource>): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      await invoke('update_data_source', { id, dataSource: ds });
      await fetchDataSources(); // 刷新列表
    } catch (e) {
      error.value = `更新数据源失败: ${e}`;
      console.error('updateDataSource error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 删除数据源
   * 需求 1.5: 删除数据源
   */
  async function deleteDataSource(id: string): Promise<void> {
    loading.value = true;
    error.value = null;
    
    try {
      await invoke('delete_data_source', { id });
      await fetchDataSources(); // 刷新列表
    } catch (e) {
      error.value = `删除数据源失败: ${e}`;
      console.error('deleteDataSource error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
  }

  /**
   * 测试连接
   * 需求 1.6: 测试数据源连接
   */
  async function testConnection(id: string): Promise<ConnectionResult> {
    loading.value = true;
    error.value = null;
    
    try {
      const result = await invoke<ConnectionResult>('test_connection', { id });
      return result;
    } catch (e) {
      error.value = `测试连接失败: ${e}`;
      console.error('testConnection error:', e);
      throw e;
    } finally {
      loading.value = false;
    }
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
    dataSources,
    loading,
    error,
    
    // 方法
    fetchDataSources,
    getDataSource,
    addDataSource,
    updateDataSource,
    deleteDataSource,
    testConnection,
    clearError,
  };
});
