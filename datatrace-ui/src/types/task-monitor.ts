export type SyncMode = 'full' | 'incremental' | 'structure'
export type TaskStatus = 'idle' | 'configured' | 'running' | 'completed' | 'paused' | string

export interface TargetConn {
  id: string
  name: string
}

export interface TaskItem {
  id: string
  name: string
  source_type: string
  target_type: string
  status: TaskStatus
  is_running: boolean
  sync_mode: SyncMode
  current_step?: string
  target_id?: string
  created_at?: string
}

export interface TaskDetail extends TaskItem {
  target_conns?: TargetConn[]
}

export interface IncrementalDatabaseStats {
  database: string
  total_insert_count: number
  total_update_count: number
  total_delete_count: number
  total_count: number
  today_count: number
  table_count: number
  full_sync_total_records: number
  full_sync_processed_records: number
  full_sync_progress: number
}

export interface IncrementalTableStats {
  database: string
  table: string
  total_count: number
  today_count: number
  insert_count: number
  update_count: number
  delete_count: number
  full_sync_total_records: number
  full_sync_progress: number
  replication_lag_seconds: number
  last_event_time?: string
}

export interface TargetProgress {
  target_id: string
  target_name: string
  status: string
  progress: number
  total_tables: number
  completed_tables: number
  total_records: number
  processed_records: number
  database_stats?: IncrementalDatabaseStats[]
}

export interface TaskProgress {
  task_id: string
  sync_mode: SyncMode
  current_step: string
  overall_progress?: number
  sync_speed?: number
  elapsed_time?: string
  estimated_time?: string
  total_tables?: number
  init_tables?: number
  completed_tables?: number
  total_records?: number
  processed_records?: number
  target_stats?: TargetProgress[]
  table_stats?: IncrementalTableStats[]
}

export interface TaskLogEntry {
  Time?: string
  Message?: string
  Level?: string
  time?: string
  message?: string
  level?: string
}

export type LogCategory = 'all' | 'initialize' | 'complete'
