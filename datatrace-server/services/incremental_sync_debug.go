package services

import (
	"context"
	"database/sql"
	"datatrace/database"
	"datatrace/models"
	"encoding/json"
	"fmt"
	"time"
)

// IncrementalSyncDebugger 增量同步调试器
type IncrementalSyncDebugger struct {
	taskID     string
	logService *TaskLogService
}

// NewIncrementalSyncDebugger 创建增量同步调试器
func NewIncrementalSyncDebugger(taskID string) *IncrementalSyncDebugger {
	return &IncrementalSyncDebugger{
		taskID:     taskID,
		logService: NewTaskLogService(),
	}
}

// DiagnoseIncrementalSync 诊断增量同步问题
func (d *IncrementalSyncDebugger) DiagnoseIncrementalSync() error {
	d.logService.Info(d.taskID, "========== 开始增量同步诊断 ==========")

	// 1. 检查任务配置
	if err := d.checkTaskConfig(); err != nil {
		return fmt.Errorf("任务配置检查失败: %v", err)
	}

	// 2. 检查源库binlog配置
	if err := d.checkSourceBinlogConfig(); err != nil {
		return fmt.Errorf("源库binlog配置检查失败: %v", err)
	}

	// 3. 检查队列状态
	if err := d.checkQueueStatus(); err != nil {
		return fmt.Errorf("队列状态检查失败: %v", err)
	}

	// 4. 检查增量同步实例状态
	if err := d.checkIncrementalSyncStatus(); err != nil {
		return fmt.Errorf("增量同步实例状态检查失败: %v", err)
	}

	d.logService.Info(d.taskID, "========== 增量同步诊断完成 ==========")
	return nil
}

// checkTaskConfig 检查任务配置
func (d *IncrementalSyncDebugger) checkTaskConfig() error {
	d.logService.Info(d.taskID, "检查任务配置...")

	// 查询任务
	var task models.SyncTask
	if err := database.DB.Preload("SourceConn").Preload("TargetConn").
		First(&task, "id = ?", d.taskID).Error; err != nil {
		return fmt.Errorf("查询任务失败: %v", err)
	}

	d.logService.Info(d.taskID, fmt.Sprintf("任务名称: %s", task.Name))
	d.logService.Info(d.taskID, fmt.Sprintf("同步模式: %s", task.SyncMode))
	d.logService.Info(d.taskID, fmt.Sprintf("队列类型: %s", task.QueueType))
	d.logService.Info(d.taskID, fmt.Sprintf("运行状态: %t", task.IsRunning))
	d.logService.Info(d.taskID, fmt.Sprintf("当前步骤: %s", task.CurrentStep))
	d.logService.Info(d.taskID, fmt.Sprintf("全量同步完成: %t", task.FullSyncCompleted))

	if task.SyncMode != "incremental" {
		return fmt.Errorf("任务不是增量同步模式，当前模式: %s", task.SyncMode)
	}

	// 解析配置
	var config TaskConfig
	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		return fmt.Errorf("解析任务配置失败: %v", err)
	}

	d.logService.Info(d.taskID, fmt.Sprintf("源数据源ID: %s", config.SourceID))
	d.logService.Info(d.taskID, fmt.Sprintf("目标数据源ID: %s", config.TargetID))
	d.logService.Info(d.taskID, fmt.Sprintf("选中数据库数量: %d", len(config.SelectedDatabases)))

	for _, db := range config.SelectedDatabases {
		d.logService.Info(d.taskID, fmt.Sprintf("  数据库: %s -> %s, 表数量: %d",
			db.SourceDatabase, db.Database, len(db.Tables)))
	}

	return nil
}

// checkSourceBinlogConfig 检查源库binlog配置
func (d *IncrementalSyncDebugger) checkSourceBinlogConfig() error {
	d.logService.Info(d.taskID, "检查源库binlog配置...")

	// 查询任务和数据源
	var task models.SyncTask
	if err := database.DB.Preload("SourceConn").First(&task, "id = ?", d.taskID).Error; err != nil {
		return fmt.Errorf("查询任务失败: %v", err)
	}

	if task.SourceConn == nil {
		return fmt.Errorf("源数据源不存在")
	}

	// 获取数据源服务
	dsService := NewDataSourceService()

	// 解密密码
	sourcePassword, err := dsService.crypto.Decrypt(task.SourceConn.Password)
	if err != nil {
		return fmt.Errorf("解密源数据库密码失败: %v", err)
	}

	// 连接源数据库
	sourceDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		task.SourceConn.Username, sourcePassword,
		task.SourceConn.Host, task.SourceConn.Port)

	sourceDB, err := sql.Open("mysql", sourceDSN)
	if err != nil {
		return fmt.Errorf("连接源数据库失败: %v", err)
	}
	defer sourceDB.Close()

	if err := sourceDB.Ping(); err != nil {
		return fmt.Errorf("源数据库连接测试失败: %v", err)
	}

	d.logService.Info(d.taskID, "源数据库连接成功")

	// 检查binlog配置
	var logBin string
	err = sourceDB.QueryRow("SELECT @@log_bin").Scan(&logBin)
	if err != nil {
		return fmt.Errorf("查询log_bin失败: %v", err)
	}
	d.logService.Info(d.taskID, fmt.Sprintf("log_bin: %s", logBin))

	if logBin != "1" && logBin != "ON" {
		return fmt.Errorf("binlog未开启，log_bin=%s", logBin)
	}

	var binlogFormat string
	err = sourceDB.QueryRow("SELECT @@binlog_format").Scan(&binlogFormat)
	if err != nil {
		return fmt.Errorf("查询binlog_format失败: %v", err)
	}
	d.logService.Info(d.taskID, fmt.Sprintf("binlog_format: %s", binlogFormat))

	if binlogFormat != "ROW" {
		return fmt.Errorf("binlog_format必须为ROW，当前为: %s", binlogFormat)
	}

	var binlogRowImage string
	err = sourceDB.QueryRow("SELECT @@binlog_row_image").Scan(&binlogRowImage)
	if err != nil {
		d.logService.Warning(d.taskID, fmt.Sprintf("查询binlog_row_image失败: %v", err))
	} else {
		d.logService.Info(d.taskID, fmt.Sprintf("binlog_row_image: %s", binlogRowImage))
	}

	// 检查当前binlog位置
	var file string
	var position uint32
	var binlogDoDB, binlogIgnoreDB, executedGtidSet string

	err = sourceDB.QueryRow("SHOW MASTER STATUS").Scan(
		&file, &position, &binlogDoDB, &binlogIgnoreDB, &executedGtidSet,
	)
	if err != nil {
		return fmt.Errorf("获取MASTER STATUS失败: %v", err)
	}

	d.logService.Info(d.taskID, fmt.Sprintf("当前binlog位置: %s:%d", file, position))

	return nil
}

// checkQueueStatus 检查队列状态
func (d *IncrementalSyncDebugger) checkQueueStatus() error {
	d.logService.Info(d.taskID, "检查队列状态...")

	// 获取任务执行管理器
	execManager := GetExecutionManager()

	// 检查是否有增量同步实例
	if incrementalSync, ok := execManager.GetIncrementalSync(d.taskID); ok {
		d.logService.Info(d.taskID, "找到增量同步实例")

		// 获取状态
		status := incrementalSync.GetStatus()
		statusJSON, _ := json.MarshalIndent(status, "", "  ")
		d.logService.Info(d.taskID, fmt.Sprintf("增量同步状态: %s", string(statusJSON)))

		return nil
	} else {
		d.logService.Warning(d.taskID, "未找到增量同步实例")
		return fmt.Errorf("增量同步实例不存在")
	}
}

// checkIncrementalSyncStatus 检查增量同步实例状态
func (d *IncrementalSyncDebugger) checkIncrementalSyncStatus() error {
	d.logService.Info(d.taskID, "检查增量同步实例状态...")

	// 检查Redis中的统计数据
	statsService := NewIncrementalStatsService()

	// 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", d.taskID).Error; err != nil {
		return fmt.Errorf("查询任务失败: %v", err)
	}

	// 获取任务统计
	taskStats, err := statsService.GetTaskStats(d.taskID, task.TargetID)
	if err != nil {
		d.logService.Warning(d.taskID, fmt.Sprintf("获取Redis任务统计失败: %v", err))
	} else if taskStats != nil {
		d.logService.Info(d.taskID, fmt.Sprintf("Redis任务统计 - 总事件: %d, INSERT: %d, UPDATE: %d, DELETE: %d, 失败: %d",
			taskStats.TotalEvents, taskStats.InsertCount, taskStats.UpdateCount, taskStats.DeleteCount, taskStats.FailedCount))
		d.logService.Info(d.taskID, fmt.Sprintf("当前binlog位置: %s:%d",
			taskStats.CurrentBinlogFile, taskStats.CurrentBinlogPos))
	} else {
		d.logService.Warning(d.taskID, "Redis中未找到任务统计数据")
	}

	// 获取表统计
	tableStats, err := statsService.GetTableStatsList(d.taskID, task.TargetID)
	if err != nil {
		d.logService.Warning(d.taskID, fmt.Sprintf("获取Redis表统计失败: %v", err))
	} else {
		d.logService.Info(d.taskID, fmt.Sprintf("表统计数量: %d", len(tableStats)))

		for _, stat := range tableStats {
			d.logService.Info(d.taskID, fmt.Sprintf("表 %s.%s - INSERT: %d, UPDATE: %d, DELETE: %d",
				stat.Database, stat.Table,
				stat.InsertCount, stat.UpdateCount, stat.DeleteCount))
		}
	}

	return nil
}

// TestBinlogListener 测试binlog监听器
func (d *IncrementalSyncDebugger) TestBinlogListener(duration time.Duration) error {
	d.logService.Info(d.taskID, fmt.Sprintf("开始测试binlog监听器，持续时间: %v", duration))

	// 查询任务
	var task models.SyncTask
	if err := database.DB.Preload("SourceConn").First(&task, "id = ?", d.taskID).Error; err != nil {
		return fmt.Errorf("查询任务失败: %v", err)
	}

	// 解析配置
	var config TaskConfig
	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		return fmt.Errorf("解析任务配置失败: %v", err)
	}

	// 获取数据源服务
	dsService := NewDataSourceService()

	// 解密密码
	sourcePassword, err := dsService.crypto.Decrypt(task.SourceConn.Password)
	if err != nil {
		return fmt.Errorf("解密源数据库密码失败: %v", err)
	}

	// 连接源数据库
	sourceDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		task.SourceConn.Username, sourcePassword,
		task.SourceConn.Host, task.SourceConn.Port)

	sourceDB, err := sql.Open("mysql", sourceDSN)
	if err != nil {
		return fmt.Errorf("连接源数据库失败: %v", err)
	}
	defer sourceDB.Close()

	// 获取当前binlog位置
	var file string
	var position uint32
	var binlogDoDB, binlogIgnoreDB, executedGtidSet string

	err = sourceDB.QueryRow("SHOW MASTER STATUS").Scan(
		&file, &position, &binlogDoDB, &binlogIgnoreDB, &executedGtidSet,
	)
	if err != nil {
		return fmt.Errorf("获取MASTER STATUS失败: %v", err)
	}

	d.logService.Info(d.taskID, fmt.Sprintf("测试起始位置: %s:%d", file, position))

	// 创建内存队列用于测试
	testQueue := NewMemoryQueue(1000)
	defer testQueue.Close()

	// 提取需要监听的数据库和表
	tableFilters := make(map[string][]string)
	for _, dbSel := range config.SelectedDatabases {
		sourceDB := dbSel.SourceDatabase
		var tables []string
		for _, tbl := range dbSel.Tables {
			tables = append(tables, tbl.SourceTable)
		}
		tableFilters[sourceDB] = tables
	}

	// 创建测试监听器配置
	listenerConfig := &BinlogListenerConfig{
		Host:      task.SourceConn.Host,
		Port:      task.SourceConn.Port,
		Username:  task.SourceConn.Username,
		Password:  sourcePassword,
		ServerID:  200, // 使用不同的ServerID避免冲突
		StartFile: file,
		StartPos:  position,
		Tables:    tableFilters,
		Queue:     testQueue,
		TaskID:    d.taskID + "-test",
		SourceDB:  sourceDB,
	}

	// 创建测试监听器
	listener, err := NewBinlogListener(listenerConfig)
	if err != nil {
		return fmt.Errorf("创建测试监听器失败: %v", err)
	}

	// 启动监听器
	if err := listener.Start(); err != nil {
		return fmt.Errorf("启动测试监听器失败: %v", err)
	}
	defer listener.Stop()

	d.logService.Info(d.taskID, "测试监听器启动成功，开始监听binlog事件...")

	// 监听指定时间
	ctx, cancel := context.WithTimeout(context.Background(), duration)
	defer cancel()

	eventCount := 0
	for {
		select {
		case <-ctx.Done():
			d.logService.Info(d.taskID, fmt.Sprintf("测试完成，共捕获 %d 个事件", eventCount))
			return nil
		default:
			// 尝试从队列获取事件
			event, err := testQueue.Pop(context.WithValue(context.Background(), "timeout", 1*time.Second))
			if err != nil {
				continue
			}
			if event != nil {
				eventCount++
				d.logService.Info(d.taskID, fmt.Sprintf("捕获事件 #%d: %s %s.%s",
					eventCount, event.Type, event.Database, event.Table))
			}
		}
	}
}
