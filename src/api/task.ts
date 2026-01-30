import { invoke } from '@tauri-apps/api/core';
import type { SyncTask } from '../types';

export const taskApi = {
  async list(): Promise<SyncTask[]> {
    return await invoke('list_tasks');
  },

  async create(task: Partial<SyncTask>): Promise<string> {
    return await invoke('create_task', { task });
  },

  async update(id: string, task: Partial<SyncTask>): Promise<void> {
    return await invoke('update_task', { id, task });
  },

  async delete(id: string): Promise<void> {
    return await invoke('delete_task', { id });
  },

  // 元数据查询
  async getDatabases(id: string): Promise<string[]> {
    return await invoke('get_databases', { id });
  },

  async getTables(id: string, database: string): Promise<string[]> {
    return await invoke('get_tables', { id, database });
  },

  async getIndices(id: string): Promise<string[]> {
    return await invoke('get_indices', { id });
  }
};
