export interface DataSourceItem {
  id: string
  name: string
  type: string
  host?: string
  port?: number
}

export interface CreateTaskRequest {
  name: string
  source_type: string
  target_type: string
  remark?: string
}

export interface ConfigViewTable {
  source_table: string
  target_table: string
  is_modified?: boolean
  selected_fields?: string[]
}

export interface ConfigViewDatabase {
  database: string
  source_database: string
  is_database_modified?: boolean
  tables: ConfigViewTable[]
}

export interface TaskConfigViewResult {
  task: {
    id: string
    name: string
    source_type: string
    target_type: string
    source_conn?: { name?: string }
    target_conn?: { name?: string }
  }
  config?: {
    source_id?: string
    target_id?: string
    target_ids?: string[]
    selected_databases?: ConfigViewDatabase[]
    sync_config?: {
      sync_mode?: string
      error_strategy?: string
      table_exists_strategy?: string
      sync_structure_only?: boolean
    }
  }
  real_time_info?: Record<string, string[]>
}

export interface DatabaseTablesNode {
  database: string
  tables: string[]
}

export interface TableColumnInfo {
  name: string
  is_primary?: boolean
}

export interface TableSelectionItem {
  source_table: string
  target_table: string
  is_modified?: boolean
  selected_fields?: string[]
}

export interface DatabaseSelectionItem {
  database: string
  source_database: string
  is_database_modified?: boolean
  tables: TableSelectionItem[]
}

export interface SyncConfigPayload {
  sync_mode: string
  error_strategy: string
  table_exists_strategy: string
  sync_structure_only: boolean
}

export interface UpdateTaskConfigPayload {
  source_id: string
  target_id?: string
  target_ids: string[]
  selected_databases: DatabaseSelectionItem[]
  sync_config: SyncConfigPayload
}
