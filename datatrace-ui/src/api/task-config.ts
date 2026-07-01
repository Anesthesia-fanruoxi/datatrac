import { http } from './http'
import type { CreateTaskRequest, TaskConfigViewResult, UpdateTaskConfigPayload } from '../types/task-config'
import type { TaskItem } from '../types/task-monitor'

export const taskConfigApi = {
  list() {
    return http.get<TaskItem[]>('/api/v1/tasks')
  },
  create(payload: CreateTaskRequest) {
    return http.post<TaskItem>('/api/v1/tasks', payload)
  },
  remove(taskId: string) {
    return http.delete<null>(`/api/v1/tasks/${taskId}`)
  },
  getConfigView(taskId: string) {
    return http.get<TaskConfigViewResult>(`/api/v1/tasks/${taskId}/config-view`)
  },
  updateConfig(taskId: string, payload: UpdateTaskConfigPayload) {
    return http.put<TaskItem>(`/api/v1/tasks/${taskId}/config`, payload)
  }
}
