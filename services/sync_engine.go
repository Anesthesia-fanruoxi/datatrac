package services

import (
	"context"
	"datatrace/database"
	"datatrace/models"
	"fmt"
)

// SyncEngine 同步引擎
type SyncEngine struct {
	logService *TaskLogService
	sseService *TaskSSEService
	dsService  *DataSourceService
}

// NewSyncEngine 创建同步引擎
func NewSyncEngine() *SyncEngine {
	return &SyncEngine{
		logService: NewTaskLogService(),
		sseService: NewTaskSSEService(),
		dsService:  NewDataSourceService(),
	}
}

// Worker Worker执行逻辑
func (e *SyncEngine) Worker(ctx context.Context, taskID string, taskQueue <-chan string, workerID int) {
	isFK := workerID == -1
	workerName := fmt.Sprintf("Worker %d", workerID)
	if isFK {
		workerName = "外键Worker"
	}

	e.logService.Info(taskID, fmt.Sprintf("%s 启动", workerName))

	for {
		select {
		case <-ctx.Done():
			e.logService.Info(taskID, fmt.Sprintf("%s 收到停止信号", workerName))
			return

		case unitName, ok := <-taskQueue:
			if !ok {
				e.logService.Info(taskID, fmt.Sprintf("%s 任务队列已空，退出", workerName))
				return
			}

			if isFK {
				e.logService.Info(taskID, fmt.Sprintf(">>> %s 开始处理外键表: %s", workerName, unitName))
			}

			// 执行同步
			err := e.SyncTable(ctx, taskID, unitName)
			if err != nil {
				e.logService.Error(taskID, fmt.Sprintf("%s 处理失败: %v", workerName, err))
			} else {
				if isFK {
					e.logService.Info(taskID, fmt.Sprintf("<<< %s 完成外键表: %s", workerName, unitName))
				}
			}
		}
	}
}

// SyncTable 同步单个表的数据（表结构已在初始化阶段处理）
func (e *SyncEngine) SyncTable(ctx context.Context, taskID string, unitName string) error {
	progressManager := GetProgressManager()

	// 1. 更新状态为running
	progressManager.UpdateUnitStatus(taskID, unitName, "running")

	// 2. 查询任务配置
	var task models.SyncTask
	if err := database.DB.Preload("SourceConn").Preload("TargetConn").First(&task, "id = ?", taskID).Error; err != nil {
		return e.failUnit(taskID, unitName, fmt.Sprintf("查询任务失败: %v", err))
	}

	// 3. 从Redis获取配置
	configCache := NewConfigCacheService()
	config, err := configCache.GetTaskConfigWithFallback(taskID)
	if err != nil {
		return e.failUnit(taskID, unitName, fmt.Sprintf("获取配置失败: %v", err))
	}

	// 4. 解析表名（格式：database.table）
	sourceDB, sourceTable, targetDB, targetTable, err := e.parseUnitName(unitName, config)
	if err != nil {
		return e.failUnit(taskID, unitName, fmt.Sprintf("解析表名失败: %v", err))
	}

	// 5. 解密密码
	sourcePassword, err := e.dsService.crypto.Decrypt(task.SourceConn.Password)
	if err != nil {
		return e.failUnit(taskID, unitName, fmt.Sprintf("解密源数据库密码失败: %v", err))
	}

	targetPassword, err := e.dsService.crypto.Decrypt(task.TargetConn.Password)
	if err != nil {
		return e.failUnit(taskID, unitName, fmt.Sprintf("解密目标数据库密码失败: %v", err))
	}

	// 6. 获取字段配置
	selectedFields := e.getSelectedFields(config, sourceDB, sourceTable)

	// 7. 计算自适应批次大小
	batchSize := e.calculateAdaptiveBatchSize(task.SourceConn, sourceDB, sourceTable, sourcePassword)

	// 8. 创建Reader（支持字段选择和自适应批次）
	reader, err := NewMySQLReaderWithFields(
		task.SourceConn.Host,
		task.SourceConn.Port,
		task.SourceConn.Username,
		sourcePassword,
		sourceDB,
		sourceTable,
		batchSize,
		selectedFields,
	)
	if err != nil {
		return e.failUnit(taskID, unitName, fmt.Sprintf("创建Reader失败: %v", err))
	}
	defer reader.Close()

	// 9. 创建Writer
	writer, err := NewMySQLWriter(
		task.TargetConn.Host,
		task.TargetConn.Port,
		task.TargetConn.Username,
		targetPassword,
		targetDB,
		targetTable,
	)
	if err != nil {
		return e.failUnit(taskID, unitName, fmt.Sprintf("创建Writer失败: %v", err))
	}
	defer writer.Close()

	// 8. 获取总记录数
	unit := progressManager.GetUnit(taskID, unitName)
	if unit == nil {
		return e.failUnit(taskID, unitName, "获取单元信息失败")
	}

	// 检查是否为空表
	if unit.TotalRecords == 0 {
		progressManager.UpdateUnitStatus(taskID, unitName, "completed")
		completeMessage := fmt.Sprintf("表 %s 同步完成，共 0 条记录", unitName)
		e.logService.AddLog(taskID, "success", completeMessage, "complete")
		e.sseService.BroadcastProgressUpdate(taskID)
		return nil
	}

	// 9. 批量读取和写入数据
	batchNum := 0
	for reader.HasMore() {
		// 检查context是否被取消
		select {
		case <-ctx.Done():
			return e.pauseUnit(taskID, unitName, "任务被暂停")
		default:
		}

		batchNum++

		// 读取批次
		records, err := reader.ReadBatch()
		if err != nil {
			return e.failUnit(taskID, unitName, fmt.Sprintf("读取数据失败: %v", err))
		}

		if len(records) == 0 {
			break
		}

		// 写入批次
		if err := writer.WriteBatch(records); err != nil {
			// 根据错误策略处理
			if config.SyncConfig.ErrorStrategy == "pause" {
				return e.failUnit(taskID, unitName, fmt.Sprintf("写入数据失败: %v", err))
			} else {
				e.logService.Error(taskID, fmt.Sprintf("批次 %d 写入失败(跳过): %v", batchNum, err))
				continue
			}
		}

		// 更新进度（内存）
		unit.ProcessedRecords += int64(len(records))
		progressManager.UpdateUnitProgress(taskID, unitName, unit.TotalRecords, unit.ProcessedRecords)

		// 生成同步批次日志
		logMessage := fmt.Sprintf("表 %s 批次 %d: %d/%d (%.1f%%)",
			unitName, batchNum, unit.ProcessedRecords, unit.TotalRecords,
			safePercent(unit.ProcessedRecords, unit.TotalRecords))
		e.logService.AddLog(taskID, "info", logMessage, "sync")
		e.sseService.BroadcastProgressUpdate(taskID)
	}

	// 10. 标记为完成
	progressManager.UpdateUnitStatus(taskID, unitName, "completed")
	progressManager.UpdateUnitProgress(taskID, unitName, unit.TotalRecords, unit.TotalRecords)

	completeMessage := fmt.Sprintf("表 %s 同步完成，共 %d 条记录", unitName, unit.TotalRecords)
	e.logService.AddLog(taskID, "success", completeMessage, "complete")
	e.sseService.BroadcastProgressUpdate(taskID)

	return nil
}
