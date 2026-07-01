import { http } from './http'
import type { TaskItem } from '../types/task-monitor'

export const taskApi = {
  list() {
    return http.get<TaskItem[]>('/api/v1/tasks')
  },
  start(taskId: string) {
    return http.post<null>(`/api/v1/tasks/${taskId}/start`)
  },
  pause(taskId: string) {
    return http.post<null>(`/api/v1/tasks/${taskId}/pause`)
  },
  stop(taskId: string) {
    return http.post<null>(`/api/v1/tasks/${taskId}/stop`)
  }
}
