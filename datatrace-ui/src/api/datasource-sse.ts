import type { DataSourceHealthStatus } from '../types/datasource'

export function createDataSourceStatusSSE(handler: (results: DataSourceHealthStatus[]) => void) {
  const source = new EventSource('/api/v1/datasources/test/stream')
  source.addEventListener('test', (event) => {
    const payload = JSON.parse((event as MessageEvent).data) as DataSourceHealthStatus[]
    if (Array.isArray(payload)) {
      handler(payload)
    }
  })
  return source
}
