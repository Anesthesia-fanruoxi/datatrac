import { defineStore } from 'pinia';
import { listen } from '@tauri-apps/api/event';
import type { TaskProgressEvent, LogEntry } from '../types';

export const useTaskMonitorStore = defineStore('taskMonitor', {
  state: () => ({
    activeTaskId: null as string | null,
    progress: null as TaskProgressEvent | null,
    logs: [] as LogEntry[],
    maxLogs: 1000,
  }),

  actions: {
    setActiveTask(taskId: string) {
      this.activeTaskId = taskId;
      this.logs = [];
      this.progress = null;
    },

    async initListeners() {
      // 监听任务进度更新
      await listen<TaskProgressEvent>('task-progress', (event) => {
        if (this.activeTaskId === event.payload.taskId) {
          this.progress = event.payload;
        }
      });

      // 监听日志更新
      await listen<LogEntry>('task-log', (event) => {
        // 这里的逻辑可以根据 activeTaskId 过滤，或者存储所有日志
        this.logs.unshift(event.payload);
        if (this.logs.length > this.maxLogs) {
          this.logs.pop();
        }
      });
    }
  }
});
