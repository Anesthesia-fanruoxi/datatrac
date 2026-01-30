// 通用响应
export interface ApiResponse<T> {
  success: boolean;
  data: T;
  error?: {
    code: string;
    message: string;
    details?: any;
  };
}

// 数据源类型
export type DataSourceType = 'mysql' | 'elasticsearch';

// 数据源
export interface DataSource {
  id: string;
  name: string;
  type: DataSourceType; // 对应后端 #[serde(rename = "type")]
  host: string;
  port: number;
  username: string;
  password?: string;
  database?: string;
  createdAt: string;
  updatedAt: string;
}

export interface CreateDataSourceRequest {
  name: string;
  type: DataSourceType;
  host: string;
  port: number;
  username: string;
  password?: string;
  database?: string;
}

// 任务状态
export type TaskStatus = 'idle' | 'running' | 'paused' | 'completed' | 'failed';

// 任务单元状态
export type TaskUnitStatus = 'pending' | 'running' | 'completed' | 'failed';

// 同步任务
export interface SyncTask {
  id: string;
  name: string;
  sourceId: string;
  targetId: string;
  sourceType: DataSourceType;
  targetType: DataSourceType;
  config: string; // JSON 字符串
  status: TaskStatus;
  createdAt: string;
  updatedAt: string;
}

// 任务单元
export interface TaskUnit {
  id: string;
  taskId: string;
  unitName: string;
  unitType: 'table' | 'index';
  status: string;
  totalRecords: number;
  processedRecords: number;
  errorMessage?: string;
  createdAt: string;
  updatedAt: string;
  // 前端计算字段
  percentage?: number;
}

// 任务进度事件
export interface TaskProgressEvent {
  taskId: string;
  status: TaskStatus;
  totalRecords: number;
  processedRecords: number;
  percentage: number;
  speed: number;
  estimatedTime: number;
  taskUnits: TaskUnit[];
}

// 日志类别
export type LogCategory = 'realtime' | 'summary' | 'verify' | 'error';
export type LogLevel = 'info' | 'warn' | 'error';

// 日志分条
export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  category: LogCategory;
  message: string;
}

// 任务配置 (解析后的 config 字段)
export interface SyncConfig {
  batchSize: number;
  threadCount: number;
  errorStrategy: 'skip' | 'pause';
  tableExistsStrategy: 'drop' | 'truncate' | 'backup';
  selection?: string[]; // 选择的表或索引
  dbNameTransform?: {
    enabled: boolean;
    mode: 'prefix' | 'suffix';
    sourcePattern: string;
    targetPattern: string;
  };
}
