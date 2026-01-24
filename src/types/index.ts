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
  createdAt: string;
  updatedAt: string;
}

/**
 * 连接测试结果
 */
export interface ConnectionResult {
  success: boolean;
  message: string;
  details?: string;
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
 * 同步配置
 */
export interface SyncConfig {
  threadCount: number; // 1-32
  batchSize: number;
  errorStrategy: ErrorStrategy;
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
  createdAt: string;
  updatedAt: string;
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
