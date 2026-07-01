export interface ApiResponse<T> {
  code: number
  message: string
  data: T
}

export interface PageResponse<T> extends ApiResponse<T> {
  total: number
  page: number
  page_size: number
}
