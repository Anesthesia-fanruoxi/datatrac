import { invoke } from '@tauri-apps/api/core';
import type { TaskUnit, TaskProgressEvent, LogEntry } from '../types';

export interface TaskUnitsResponse {
  units: TaskUnit[];
  statistics: {
    total: number;
    pending: number;
    running: number;
    completed: number;
    failed: number;
  };
}

export const monitorApi = {
  async startTask(taskId: string): Promise<void> {
    return await invoke('start_sync', { taskId });
  },

  async pauseTask(taskId: string): Promise<void> {
    return await invoke('pause_sync', { taskId });
  },

  async resumeTask(taskId: string): Promise<void> {
    return await invoke('resume_sync', { taskId });
  },

  async getProgress(taskId: string): Promise<TaskProgressEvent | null> {
    return await invoke('get_progress', { taskId });
  },

  async getLogs(taskId: string): Promise<LogEntry[]> {
    return await invoke('get_logs', { taskId });
  },

  async getTaskUnits(taskId: string): Promise<TaskUnitsResponse> {
    return await invoke('get_task_units', { taskId });
  },

  async resetFailedUnits(taskId: string): Promise<number> {
    return await invoke('reset_failed_units', { taskId });
  }
};
