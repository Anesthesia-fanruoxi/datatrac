import { http } from './http'
import type { DataSourcePayload, DataSourceRecord, TestConnectionResult } from '../types/datasource'
import type { DataSourceItem, DatabaseTablesNode, TableColumnInfo } from '../types/task-config'

export const datasourceApi = {
  list() {
    return http.get<DataSourceRecord[]>('/api/v1/datasources')
  },
  getById(id: string) {
    return http.get<DataSourceRecord>(`/api/v1/datasources/${id}`)
  },
  create(payload: DataSourcePayload) {
    return http.post<DataSourceRecord>('/api/v1/datasources', payload)
  },
  update(id: string, payload: DataSourcePayload) {
    return http.put<DataSourceRecord>(`/api/v1/datasources/${id}`, payload)
  },
  remove(id: string) {
    return http.delete<null>(`/api/v1/datasources/${id}`)
  },
  testConnection(payload: DataSourcePayload) {
    return http.post<TestConnectionResult>('/api/v1/datasources/test', payload)
  },
  testById(id: string) {
    return http.post<TestConnectionResult>(`/api/v1/datasources/${id}/test`, {})
  },
  getDatabaseTables(datasourceId: string) {
    return http.get<DatabaseTablesNode[]>(`/api/v1/datasources/${datasourceId}/database-tables`)
  },
  getTableColumns(datasourceId: string, database: string, table: string) {
    return http.get<TableColumnInfo[]>(`/api/v1/datasources/${datasourceId}/tables/${encodeURIComponent(database)}/${encodeURIComponent(table)}/columns`)
  }
}
