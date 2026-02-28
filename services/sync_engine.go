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
	e.logService.Info(taskID, fmt.Sprintf("Worker %d 启动", workerID))

	for {
		select {
		case <-ctx.Done():
			e.logService.Info(taskID, fmt.Sprintf("Worker %d 收到停止信号", workerID))
			return

		case unit, ok := <-taskQueue:
			if !ok {
				e.logService.Info(taskID, fmt.Sprintf("Worker %d 任务队列已空，退出", workerID))
				return
			}

			e.logService.Info(taskID, fmt.Sprintf("Worker %d 开始处理表: %s", workerID, unit.UnitName))

			// 执行同步
			err := e.SyncTable(ctx, taskID, unit)
			if err != nil {
				e.logService.Error(taskID, fmt.Sprintf("Worker %d 处理失败: %v", workerID, err))
			} else {
				e.logService.Info(taskID, fmt.Sprintf("Worker %d 完成表: %s", workerID, unit.UnitName))
			}
		}
	}
}

// SyncTable 同步单个表
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

	e.logService.Info(taskID, fmt.Sprintf("开始同步: %s.%s -> %s.%s", sourceDB, sourceTable, targetDB, targetTable))

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

	// 7. 获取源数据库的字符集和排序规则
	sourceCharset, sourceCollation, err := reader.GetDatabaseCharset(sourceDB)
	if err != nil {
		e.logService.Warning(taskID, fmt.Sprintf("获取源数据库字符集失败，使用默认值: %v", err))
	}
	e.logService.Info(taskID, fmt.Sprintf("源数据库 %s 字符集: %s, 排序规则: %s", sourceDB, sourceCharset, sourceCollation))

	// 8. 创建目标数据库（如果不存在）
	created, err := CreateDatabaseIfNotExists(
		task.TargetConn.Host,
		task.TargetConn.Port,
		task.TargetConn.Username,
		targetPassword,
		targetDB,
		sourceCharset,
		sourceCollation,
	)
	if err != nil {
		return e.failUnit(unit, fmt.Sprintf("创建目标数据库失败: %v", err))
	}
	e.logService.Info(taskID, fmt.Sprintf("目标数据库 %s 已就绪", targetDB))

	// 只有在实际创建数据库时才记录日志
	if created {
		// 添加创建日志
		e.logService.AddLog(taskID, "info", fmt.Sprintf("创建数据库: %s", targetDB), "create")
		// 发送创建日志
		createLog := TaskLog{
			Time:     formatLogTime(time.Now()),
			Level:    "info",
			Message:  fmt.Sprintf("创建数据库: %s", targetDB),
			Category: "create",
		}
		e.sseService.BroadcastLogUpdate(taskID, []TaskLog{createLog})
	}

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
		return e.failUnit(unit, fmt.Sprintf("创建Writer失败: %v", err))
	}
	defer writer.Close()

	// 10. 处理表存在策略
	if err := e.handleTableExistsStrategy(writer, reader.GetDB(), sourceTable, config.SyncConfig.TableExistsStrategy, taskID); err != nil {
		return e.failUnit(unit, fmt.Sprintf("处理表存在策略失败: %v", err))
	}

	// 11. 更新总记录数
	unit.TotalRecords = reader.GetTotalCount()
	database.DB.Save(unit)

	e.logService.Info(taskID, fmt.Sprintf("表 %s 总记录数: %d", unit.UnitName, unit.TotalRecords))

	// 检查是否为空表，如果是则直接标记为完成
	if unit.TotalRecords == 0 {
		// 标记为完成
		unit.Status = "completed"
		unit.ProcessedRecords = 0
		database.DB.Save(unit)

		// 生成完成日志
		completeMessage := fmt.Sprintf("表 %s 同步完成，共 %d 条记录", unit.UnitName, unit.ProcessedRecords)
		// 添加到内存日志
		e.logService.AddLog(taskID, "success", completeMessage, "complete")
		// 发送完成日志
		completeLog := TaskLog{
			Time:     formatLogTime(time.Now()),
			Level:    "success",
			Message:  completeMessage,
			Category: "complete",
		}
		e.sseService.BroadcastLogUpdate(taskID, []TaskLog{completeLog})

		// 实时推送完成状态更新
		e.sseService.BroadcastProgressUpdate(taskID)

		e.logService.Info(taskID, fmt.Sprintf("表 %s 为空表，跳过同步", unit.UnitName))
		return nil
	}

	// 12. 批量读取和写入
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
		// 添加到内存日志
		e.logService.AddLog(taskID, "info", logMessage, "sync")
		// 发送同步批次日志
		syncLog := TaskLog{
			Time:     formatLogTime(time.Now()),
			Level:    "info",
			Message:  logMessage,
			Category: "sync",
		}
		e.sseService.BroadcastLogUpdate(taskID, []TaskLog{syncLog})

		// 实时推送进度更新
		e.sseService.BroadcastProgressUpdate(taskID)

		e.logService.Info(taskID, fmt.Sprintf("表 %s 批次 %d: %d/%d (%.1f%%)",
			unit.UnitName, batchNum, unit.ProcessedRecords, unit.TotalRecords,
			safePercent(unit.ProcessedRecords, unit.TotalRecords)))
	}

	// 13. 标记为完成
	unit.Status = "completed"
	unit.ProcessedRecords = unit.TotalRecords
	database.DB.Save(unit)

	// 生成完成日志
	completeMessage := fmt.Sprintf("表 %s 同步完成，共 %d 条记录", unit.UnitName, unit.ProcessedRecords)
	// 添加到内存日志
	e.logService.AddLog(taskID, "success", completeMessage, "complete")
	// 发送完成日志
	completeLog := TaskLog{
		Time:     formatLogTime(time.Now()),
		Level:    "success",
		Message:  completeMessage,
		Category: "complete",
	}
	e.sseService.BroadcastLogUpdate(taskID, []TaskLog{completeLog})

	// 实时推送完成状态更新
	e.sseService.BroadcastProgressUpdate(taskID)

	e.logService.Info(taskID, fmt.Sprintf("表 %s 同步完成，共 %d 条记录", unit.UnitName, unit.ProcessedRecords))

	return nil
}

// 辅助方法已拆分到 sync_helpers.go
