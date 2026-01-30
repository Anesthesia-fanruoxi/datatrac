import { invoke } from '@tauri-apps/api/core';
import type { DataSource, CreateDataSourceRequest } from '../types';

export interface ConnectionResult {
  success: boolean;
  message: string;
  durationMs: number;
}

export const datasourceApi = {
  async list(): Promise<DataSource[]> {
    return await invoke('list_datasources');
  },

  async create(request: CreateDataSourceRequest): Promise<string> {
    return await invoke('create_datasource', { request });
  },

  async update(id: string, datasource: DataSource): Promise<void> {
    return await invoke('update_datasource', { id, datasource });
  },

  async delete(id: string): Promise<void> {
    return await invoke('delete_datasource', { id });
  },

  async testConnection(id: string): Promise<ConnectionResult> {
    return await invoke('test_connection', { id });
  }
};
