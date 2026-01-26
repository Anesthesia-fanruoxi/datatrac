/**
 * 数据同步工具 - TypeScript 类型定义
 */

// ==================== 数据源相关类型 ====================

/**
 * 数据源类型
 */
export type DataSourceType = 'mysql' | 'elasticsearch';

/**
 * 数据源接口
 */
export interface DataSource {
  id: string;
  name: string;
  type: DataSourceType;
  host: string;
  port: number;
  username: string;
  password: string; // 加密存储
  database?: string; // MySQL 专用
  createdAt: number; // 毫秒时间戳
  updatedAt: number; // 毫秒时间戳
}

/**
 * 连接测试结果
 */
export interface ConnectionResult {
  success: boolean;
  message: string;
  details?: string;
  steps?: ConnectionTestStep[]; // 测试步骤详情
}

/**
 * 连接测试步骤
 */
export interface ConnectionTestStep {
  step: number;
  name: string;
  success: boolean;
  message: string;
  duration?: number; // 耗时（毫秒）
}

/**
 * 批量测试单个数据源的结果
 */
export interface BatchTestDataSourceResult {
  dataSourceId: string;
  dataSourceName: string;
  steps: ConnectionTestStep[];
  success: boolean;
}

/**
 * 批量测试结果
 */
export interface BatchTestResult {
  total: number;
  success: number;
  failed: number;
  results: BatchTestDataSourceResult[];
}

// ==================== 同步任务相关类型 ====================

/**
 * 任务状态
 */
export type TaskStatus = 'idle' | 'running' | 'paused' | 'completed' | 'failed';

/**
 * 错误处理策略
 */
export type ErrorStrategy = 'skip' | 'pause';

/**
 * 数据库选择配置
 */
export interface DatabaseSelection {
  database: string;
  tables: string[]; // 选中的表
}

/**
 * 索引选择配置
 */
export interface IndexSelection {
  pattern: string; // 索引名或通配符 (如 logs-*)
  matchedIndices?: string[]; // 匹配到的索引
}

/**
 * 数据库名称转换配置
 */
export interface DbNameTransform {
  enabled: boolean;
  mode: 'prefix' | 'suffix';
  sourcePattern: string;
  targetPattern: string;
}

/**
 * 目标表存在时的处理策略（与后端保持一致）
 */
export type TableExistsStrategy = 'drop' | 'truncate' | 'backup';

/**
 * 同步配置
 */
export interface SyncConfig {
  threadCount: number; // 1-32
  batchSize: number;
  errorStrategy: ErrorStrategy;
  tableExistsStrategy: TableExistsStrategy;
  dbNameTransform?: DbNameTransform;
}

/**
 * 同步任务接口
 */
export interface SyncTask {
  id: string;
  name: string;
  sourceId: string;
  targetId: string;
  sourceType: DataSourceType;
  targetType: DataSourceType;
  
  // MySQL 配置
  mysqlConfig?: {
    databases: DatabaseSelection[];
  };
  
  // ES 配置
  esConfig?: {
    indices: IndexSelection[];
  };
  
  // 同步配置
  syncConfig: SyncConfig;
  
  status: TaskStatus;
  createdAt: number; // 毫秒时间戳
  updatedAt: number; // 毫秒时间戳
}

/**
 * 索引匹配结果
 */
export interface IndexMatchResult {
  pattern: string;
  count: number;
  preview: string[]; // 前 10 个
}

// ==================== 任务监控相关类型 ====================

/**
 * 表同步状态
 */
export type TableStatus = 'waiting' | 'running' | 'completed' | 'failed';

/**
 * 表进度
 */
export interface TableProgress {
  tableName: string;
  status: TableStatus;
  totalRecords: number;
  processedRecords: number;
  percentage: number;
}

/**
 * 日志级别
 */
export type LogLevel = 'info' | 'warn' | 'error';

/**
 * 日志条目
 */
export interface LogEntry {
  timestamp: string;
  level: LogLevel;
  message: string;
}

/**
 * 任务进度
 */
export interface TaskProgress {
  taskId: string;
  status: TaskStatus;
  totalRecords: number;
  processedRecords: number;
  percentage: number;
  speed: number; // 记录/秒
  estimatedTime: number; // 秒
  startTime: string;
  currentTable?: string; // 当前处理的表/索引
  tableProgress?: TableProgress[]; // 表进度列表
}

/**
 * 错误日志
 */
export interface ErrorLog {
  timestamp: string;
  errorType: string;
  message: string;
  data?: any;
}
