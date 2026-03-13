package services

import (
	"context"
	"database/sql"
	"datatrace/database"
	"datatrace/models"
	"fmt"
	"strings"
	"time"
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

// TargetConnWithPassword 目标连接（包含解密后的密码）
type TargetConnWithPassword struct {
	Conn     *models.DataSource
	Password string
}

// loadTargetConns 加载多个目标源连接
func (e *SyncEngine) loadTargetConns(targetIDs []string) ([]TargetConnWithPassword, error) {
	if len(targetIDs) == 0 {
		return nil, fmt.Errorf("没有目标源")
	}

	var conns []TargetConnWithPassword
	for _, targetID := range targetIDs {
		var ds models.DataSource
		if err := database.DB.First(&ds, "id = ?", targetID).Error; err != nil {
			return nil, fmt.Errorf("查询目标源 %s 失败: %w", targetID, err)
		}

		password, err := e.dsService.crypto.Decrypt(ds.Password)
		if err != nil {
			return nil, fmt.Errorf("解密目标源 %s 密码失败: %w", targetID, err)
		}

		conns = append(conns, TargetConnWithPassword{
			Conn:     &ds,
			Password: password,
		})
	}

	return conns, nil
}

// ensureTargetTableExists 确保目标表存在，不存在则创建
func (e *SyncEngine) ensureTargetTableExists(targetDB, sourceDB *sql.DB, sourceDatabase, sourceTable, targetTable string, selectedFields []string, targetDatabase string) error {
	// 检查目标表是否存在
	var count int
	query := "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = ? AND table_name = ?"
	err := targetDB.QueryRow(query, targetDatabase, targetTable).Scan(&count)
	if err != nil {
		// 查询失败，尝试创建表
		goto createTable
	}
	if count > 0 {
		// 表已存在
		return nil
	}

createTable:
	// 表不存在，创建表结构
	if err := e.createTableLike(targetDB, sourceDB, sourceDatabase, sourceTable, targetTable, selectedFields); err != nil {
		return fmt.Errorf("创建表结构失败: %w", err)
	}
	return nil
}

// createTableLike 根据源表结构创建目标表
func (e *SyncEngine) createTableLike(targetDB, sourceDB *sql.DB, sourceDatabase, sourceTable, targetTable string, selectedFields []string) error {
	// 获取源表的CREATE TABLE语句
	var tableName, createSQL string
	query := fmt.Sprintf("SHOW CREATE TABLE `%s`", sourceTable)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := sourceDB.QueryRowContext(ctx, query).Scan(&tableName, &createSQL)
	if err != nil {
		return fmt.Errorf("获取源表结构失败: %w", err)
	}

	// 如果指定了字段列表，过滤字段并重建CREATE TABLE语句
	if len(selectedFields) > 0 {
		modifier := NewTableStructureModifier()
		createSQL, err = modifier.FilterFieldsAndRebuild(createSQL, selectedFields, targetTable)
		if err != nil {
			return fmt.Errorf("过滤字段失败: %w", err)
		}
	} else {
		// 没有指定字段，只替换表名
		oldTableDef := fmt.Sprintf("CREATE TABLE `%s`", sourceTable)
		newTableDef := fmt.Sprintf("CREATE TABLE `%s`", targetTable)
		createSQL = strings.Replace(createSQL, oldTableDef, newTableDef, 1)
	}

	// 执行创建表
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	_, err = targetDB.ExecContext(ctx2, createSQL)
	if err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}

	return nil
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
// 支持多目标源同步
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

	// 获取多目标源ID列表
	targetIDs := config.TargetIDs
	if len(targetIDs) == 0 {
		// 兼容旧配置
		targetIDs = []string{config.TargetID}
	}

	// 4. 解析表名（格式：database.table）
	sourceDB, sourceTable, targetDB, targetTable, err := e.parseUnitName(unitName, config)
	if err != nil {
		return e.failUnit(taskID, unitName, fmt.Sprintf("解析表名失败: %v", err))
	}

	// 5. 解密源数据库密码
	sourcePassword, err := e.dsService.crypto.Decrypt(task.SourceConn.Password)
	if err != nil {
		return e.failUnit(taskID, unitName, fmt.Sprintf("解密源数据库密码失败: %v", err))
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

	// 9. 加载多个目标源连接
	targetConns, err := e.loadTargetConns(targetIDs)
	if err != nil {
		return e.failUnit(taskID, unitName, fmt.Sprintf("加载目标源失败: %v", err))
	}

	// 10. 获取总记录数
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

	// 11. 对每个目标源进行同步
	for targetIdx, targetConn := range targetConns {
		e.logService.Info(taskID, fmt.Sprintf("同步到目标 %d/%d: %s", targetIdx+1, len(targetConns), targetConn.Conn.Name))

		// 初始化目标源单元进度
		progressManager.InitTargetUnit(taskID, targetConn.Conn.ID, targetConn.Conn.Name, unitName, unit.TotalRecords)
		progressManager.UpdateTargetUnitStatus(taskID, targetConn.Conn.ID, unitName, "running")

		// 确保目标数据库存在
		created, err := CreateDatabaseIfNotExists(
			targetConn.Conn.Host,
			targetConn.Conn.Port,
			targetConn.Conn.Username,
			targetConn.Password,
			targetDB,
			"utf8mb4",
			"utf8mb4_general_ci",
		)
		if err != nil {
			return e.failUnit(taskID, unitName, fmt.Sprintf("目标 %s 创建数据库失败: %v", targetConn.Conn.Name, err))
		}
		if created {
			e.logService.Info(taskID, fmt.Sprintf("目标 %s 创建数据库: %s", targetConn.Conn.Name, targetDB))
		}

		// 创建Writer
		writer, err := NewMySQLWriter(
			targetConn.Conn.Host,
			targetConn.Conn.Port,
			targetConn.Conn.Username,
			targetConn.Password,
			targetDB,
			targetTable,
		)
		if err != nil {
			return e.failUnit(taskID, unitName, fmt.Sprintf("目标 %s 创建Writer失败: %v", targetConn.Conn.Name, err))
		}

		// 检查目标表是否存在，不存在则创建
		if err := e.ensureTargetTableExists(writer.GetDB(), reader.GetDB(), sourceDB, sourceTable, targetTable, selectedFields, targetDB); err != nil {
			writer.Close()
			return e.failUnit(taskID, unitName, fmt.Sprintf("目标 %s 创建表结构失败: %v", targetConn.Conn.Name, err))
		}

		// 重新从头读取数据（每个目标源都需要从头读取）
		reader.Reset()

		// 批量读取和写入数据
		batchNum := 0
		// 当前目标源已处理的记录数（不累加到整体进度）
		targetProcessed := int64(0)
		for reader.HasMore() {
			// 检查context是否被取消
			select {
			case <-ctx.Done():
				writer.Close()
				return e.pauseUnit(taskID, unitName, "任务被暂停")
			default:
			}

			batchNum++

			// 读取批次
			records, err := reader.ReadBatch()
			if err != nil {
				writer.Close()
				return e.failUnit(taskID, unitName, fmt.Sprintf("目标 %s 读取数据失败: %v", targetConn.Conn.Name, err))
			}

			if len(records) == 0 {
				break
			}

			// 写入批次
			if err := writer.WriteBatch(records); err != nil {
				writer.Close()
				// 根据错误策略处理
				if config.SyncConfig.ErrorStrategy == "pause" {
					return e.failUnit(taskID, unitName, fmt.Sprintf("目标 %s 写入数据失败: %v", targetConn.Conn.Name, err))
				} else {
					e.logService.Error(taskID, fmt.Sprintf("目标 %s 批次 %d 写入失败(跳过): %v", targetConn.Conn.Name, batchNum, err))
					continue
				}
			}

			// 更新当前目标源的进度（临时变量，不累加到整体）
			targetProcessed += int64(len(records))

			// 更新目标源级别进度
			progressManager.UpdateTargetUnitProgress(taskID, targetConn.Conn.ID, unitName, targetProcessed, batchNum)

			// 生成同步批次日志
			logMessage := fmt.Sprintf("目标 %s 表 %s 批次 %d: %d/%d (%.1f%%)",
				targetConn.Conn.Name, unitName, batchNum, targetProcessed, unit.TotalRecords,
				safePercent(targetProcessed, unit.TotalRecords))
			e.logService.AddLog(taskID, "info", logMessage, "sync")
		}

		writer.Close()

		// 标记目标源单元完成
		progressManager.UpdateTargetUnitStatus(taskID, targetConn.Conn.ID, unitName, "completed")
		progressManager.UpdateTargetUnitProgress(taskID, targetConn.Conn.ID, unitName, unit.TotalRecords, batchNum)

		e.logService.Info(taskID, fmt.Sprintf("目标 %s 同步完成: %s", targetConn.Conn.Name, unitName))
	}

	// 12. 标记为完成
	progressManager.UpdateUnitStatus(taskID, unitName, "completed")
	progressManager.UpdateUnitProgress(taskID, unitName, unit.TotalRecords, unit.TotalRecords)

	completeMessage := fmt.Sprintf("表 %s 同步完成，共 %d 条记录（%d个目标）", unitName, unit.TotalRecords, len(targetConns))
	e.logService.AddLog(taskID, "success", completeMessage, "complete")
	e.sseService.BroadcastProgressUpdate(taskID)

	return nil
}
