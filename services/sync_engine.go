package services

import (
	"context"
	"datatrace/database"
	"datatrace/models"
	"encoding/json"
	"fmt"
	"time"
)

// SyncEngine 同步引擎
type SyncEngine struct {
	progressService *TaskProgressService
	logService      *TaskLogService
	sseService      *TaskSSEService
	dsService       *DataSourceService
}

// NewSyncEngine 创建同步引擎
func NewSyncEngine() *SyncEngine {
	return &SyncEngine{
		progressService: NewTaskProgressService(),
		logService:      NewTaskLogService(),
		sseService:      NewTaskSSEService(),
		dsService:       NewDataSourceService(),
	}
}

// Worker Worker执行逻辑
func (e *SyncEngine) Worker(ctx context.Context, taskID string, taskQueue <-chan *models.TaskUnitRuntime, workerID int) {
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

		case unit, ok := <-taskQueue:
			if !ok {
				e.logService.Info(taskID, fmt.Sprintf("%s 任务队列已空，退出", workerName))
				return
			}

			if isFK {
				e.logService.Info(taskID, fmt.Sprintf(">>> %s 开始处理外键表: %s", workerName, unit.UnitName))
			}

			// 执行同步
			err := e.SyncTable(ctx, taskID, unit)
			if err != nil {
				e.logService.Error(taskID, fmt.Sprintf("%s 处理失败: %v", workerName, err))
			} else {
				if isFK {
					e.logService.Info(taskID, fmt.Sprintf("<<< %s 完成外键表: %s", workerName, unit.UnitName))
				}
			}
		}
	}
}

// SyncTable 同步单个表的数据（表结构已在初始化阶段处理）
func (e *SyncEngine) SyncTable(ctx context.Context, taskID string, unit *models.TaskUnitRuntime) error {
	// 1. 更新状态为running
	now := time.Now()
	unit.Status = "running"
	unit.StartedAt = &now
	database.DB.Save(unit)

	// 2. 查询任务配置
	var task models.SyncTask
	if err := database.DB.Preload("SourceConn").Preload("TargetConn").First(&task, "id = ?", taskID).Error; err != nil {
		return e.failUnit(unit, fmt.Sprintf("查询任务失败: %v", err))
	}

	// 3. 解析配置
	var config TaskConfig
	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		return e.failUnit(unit, fmt.Sprintf("解析配置失败: %v", err))
	}

	// 4. 解析表名（格式：database.table）
	sourceDB, sourceTable, targetDB, targetTable, err := e.parseUnitName(unit.UnitName, &config)
	if err != nil {
		return e.failUnit(unit, fmt.Sprintf("解析表名失败: %v", err))
	}

	// 5. 解密密码
	sourcePassword, err := e.dsService.crypto.Decrypt(task.SourceConn.Password)
	if err != nil {
		return e.failUnit(unit, fmt.Sprintf("解密源数据库密码失败: %v", err))
	}

	targetPassword, err := e.dsService.crypto.Decrypt(task.TargetConn.Password)
	if err != nil {
		return e.failUnit(unit, fmt.Sprintf("解密目标数据库密码失败: %v", err))
	}

	// 6. 创建Reader
	reader, err := NewMySQLReader(
		task.SourceConn.Host,
		task.SourceConn.Port,
		task.SourceConn.Username,
		sourcePassword,
		sourceDB,
		sourceTable,
		config.SyncConfig.BatchSize,
	)
	if err != nil {
		return e.failUnit(unit, fmt.Sprintf("创建Reader失败: %v", err))
	}
	defer reader.Close()

	// 7. 创建Writer
	writer, err := NewMySQLWriter(
		task.TargetConn.Host,
		task.TargetConn.Port,
		task.TargetConn.Username,
		targetPassword,
		targetDB,
		targetTable,
	)
	if err != nil {
		return e.failUnit(unit, fmt.Sprintf("创建Writer失败: %v", err))
	}
	defer writer.Close()

	// 8. 检查是否为空表，如果是则直接标记为完成
	if unit.TotalRecords == 0 {
		unit.Status = "completed"
		unit.ProcessedRecords = 0
		database.DB.Save(unit)

		completeMessage := fmt.Sprintf("表 %s 同步完成，共 %d 条记录", unit.UnitName, unit.ProcessedRecords)
		e.logService.AddLog(taskID, "success", completeMessage, "complete")
		completeLog := TaskLog{
			Time:     formatLogTime(time.Now()),
			Level:    "success",
			Message:  completeMessage,
			Category: "complete",
		}
		e.sseService.BroadcastLogUpdate(taskID, completeLog)
		e.sseService.BroadcastProgressUpdate(taskID)

		return nil
	}

	// 9. 批量读取和写入数据
	batchNum := 0
	for reader.HasMore() {
		// 检查context是否被取消
		select {
		case <-ctx.Done():
			return e.pauseUnit(unit, "任务被暂停")
		default:
		}

		batchNum++

		// 读取批次
		records, err := reader.ReadBatch()
		if err != nil {
			return e.failUnit(unit, fmt.Sprintf("读取数据失败: %v", err))
		}

		if len(records) == 0 {
			break
		}

		// 写入批次
		if err := writer.WriteBatch(records); err != nil {
			// 根据错误策略处理
			if config.SyncConfig.ErrorStrategy == "pause" {
				return e.failUnit(unit, fmt.Sprintf("写入数据失败: %v", err))
			} else {
				e.logService.Error(taskID, fmt.Sprintf("批次 %d 写入失败(跳过): %v", batchNum, err))
				continue
			}
		}

		// 更新进度
		unit.ProcessedRecords += int64(len(records))
		database.DB.Model(unit).Updates(map[string]interface{}{
			"processed_records": unit.ProcessedRecords,
			"updated_at":        time.Now(),
		})

		// 生成同步批次日志
		logMessage := fmt.Sprintf("表 %s 批次 %d: %d/%d (%.1f%%)",
			unit.UnitName, batchNum, unit.ProcessedRecords, unit.TotalRecords,
			safePercent(unit.ProcessedRecords, unit.TotalRecords))
		e.logService.AddLog(taskID, "info", logMessage, "sync")
		syncLog := TaskLog{
			Time:     formatLogTime(time.Now()),
			Level:    "info",
			Message:  logMessage,
			Category: "sync",
		}
		e.sseService.BroadcastLogUpdate(taskID, syncLog)
		e.sseService.BroadcastProgressUpdate(taskID)
	}

	// 10. 标记为完成
	unit.Status = "completed"
	unit.ProcessedRecords = unit.TotalRecords
	database.DB.Save(unit)

	completeMessage := fmt.Sprintf("表 %s 同步完成，共 %d 条记录", unit.UnitName, unit.ProcessedRecords)
	e.logService.AddLog(taskID, "success", completeMessage, "complete")
	completeLog := TaskLog{
		Time:     formatLogTime(time.Now()),
		Level:    "success",
		Message:  completeMessage,
		Category: "complete",
	}
	e.sseService.BroadcastLogUpdate(taskID, completeLog)
	e.sseService.BroadcastProgressUpdate(taskID)

	return nil
}

// 辅助方法已拆分到 sync_helpers.go
