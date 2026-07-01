import type { ApiResponse } from '../types/common'

async function request<T>(url: string, options: RequestInit = {}): Promise<T> {
  const response = await fetch(url, {
    headers: {
      'Content-Type': 'application/json',
      ...(options.headers || {})
    },
    ...options
  })

  const result = (await response.json()) as ApiResponse<T>
  if (!response.ok || result.code !== 200) {
    throw new Error(result.message || '请求失败')
  }

  return result.data
}

export const http = {
  get<T>(url: string) {
    return request<T>(url)
  },
  post<T>(url: string, body?: unknown) {
    return request<T>(url, { method: 'POST', body: JSON.stringify(body ?? {}) })
  },
  put<T>(url: string, body?: unknown) {
    return request<T>(url, { method: 'PUT', body: JSON.stringify(body ?? {}) })
  },
  delete<T>(url: string) {
    return request<T>(url, { method: 'DELETE' })
  }
}
