package services

import (
	"context"
	"database/sql"
	"datatrace/database"
	"datatrace/models"
	"fmt"
	"sync"
	"time"
)

// initDatabaseConnections 初始化数据库连接
func (s *IncrementalSync) initDatabaseConnections() error {
	// 解密密码
	sourcePassword, err := s.dsService.crypto.Decrypt(s.task.SourceConn.Password)
	if err != nil {
		return fmt.Errorf("解密源数据库密码失败: %v", err)
	}

	targetPassword, err := s.dsService.crypto.Decrypt(s.task.TargetConn.Password)
	if err != nil {
		return fmt.Errorf("解密目标数据库密码失败: %v", err)
	}

	// 连接源库
	sourceDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
		s.task.SourceConn.Username,
		sourcePassword,
		s.task.SourceConn.Host,
		s.task.SourceConn.Port,
	)
	s.sourceDB, err = sql.Open("mysql", sourceDSN)
	if err != nil {
		return fmt.Errorf("连接源数据库失败: %v", err)
	}

	// 连接目标库
	targetDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/",
		s.task.TargetConn.Username,
		targetPassword,
		s.task.TargetConn.Host,
		s.task.TargetConn.Port,
	)
	s.targetDB, err = sql.Open("mysql", targetDSN)
	if err != nil {
		return fmt.Errorf("连接目标数据库失败: %v", err)
	}

	return nil
}

// closeDatabaseConnections 关闭数据库连接
func (s *IncrementalSync) closeDatabaseConnections() {
	if s.sourceDB != nil {
		s.sourceDB.Close()
	}
	if s.targetDB != nil {
		s.targetDB.Close()
	}
}

// checkBinlogConfig 检查 Binlog 配置
func (s *IncrementalSync) checkBinlogConfig() error {
	// 检查 Binlog 是否开启
	var logBin string
	err := s.sourceDB.QueryRow("SHOW VARIABLES LIKE 'log_bin'").Scan(new(string), &logBin)
	if err != nil {
		return fmt.Errorf("查询 log_bin 失败: %v", err)
	}

	if logBin != "ON" {
		return fmt.Errorf("源数据库未开启 Binlog，请设置 log_bin=ON")
	}

	// 检查 Binlog 格式
	var binlogFormat string
	err = s.sourceDB.QueryRow("SHOW VARIABLES LIKE 'binlog_format'").Scan(new(string), &binlogFormat)
	if err != nil {
		return fmt.Errorf("查询 binlog_format 失败: %v", err)
	}

	if binlogFormat != "ROW" {
		return fmt.Errorf("Binlog 格式必须为 ROW，当前为: %s", binlogFormat)
	}

	s.logService.Info(s.taskID, "Binlog 配置检查通过")
	return nil
}

// captureSnapshot 捕获快照点
func (s *IncrementalSync) captureSnapshot() (*BinlogSnapshot, error) {
	s.logService.Info(s.taskID, "正在捕获快照点...")

	// 开启一致性快照事务
	tx, err := s.sourceDB.Begin()
	if err != nil {
		return nil, fmt.Errorf("开启事务失败: %v", err)
	}
	defer tx.Rollback()

	// 设置隔离级别为可重复读
	_, err = tx.Exec("SET TRANSACTION ISOLATION LEVEL REPEATABLE READ")
	if err != nil {
		return nil, fmt.Errorf("设置隔离级别失败: %v", err)
	}

	// 获取 Binlog 位置
	var file string
	var pos uint32
	err = tx.QueryRow("SHOW MASTER STATUS").Scan(&file, &pos, new(string), new(string), new(string))
	if err != nil {
		return nil, fmt.Errorf("获取 Binlog 位置失败: %v", err)
	}

	// 提交事务
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("提交事务失败: %v", err)
	}

	snapshot := &BinlogSnapshot{
		File:      file,
		Position:  pos,
		Timestamp: time.Now(),
	}

	// 保存快照点到数据库
	database.DB.Model(&models.SyncTask{}).
		Where("id = ?", s.taskID).
		Updates(map[string]interface{}{
			"snapshot_binlog_file": file,
			"snapshot_binlog_pos":  pos,
			"updated_at":           time.Now(),
		})

	s.logService.Info(s.taskID, fmt.Sprintf("快照点捕获成功: %s:%d", file, pos))
	return snapshot, nil
}

// createQueue 创建队列
func (s *IncrementalSync) createQueue() error {
	queueType := s.task.QueueType
	if queueType == "" {
		queueType = "memory" // 默认使用内存队列
	}

	s.logService.Info(s.taskID, fmt.Sprintf("创建队列，类型: %s", queueType))

	switch queueType {
	case "memory":
		s.queue = NewMemoryQueue(s.config.SyncConfig.BatchSize * 10)

	case "redis":
		// 使用全局 Redis 客户端
		if database.RedisClient == nil {
			return fmt.Errorf("Redis 未启用，请在配置文件中启用 Redis")
		}
		queue, err := NewRedisQueue(database.RedisClient, s.taskID, "worker-1")
		if err != nil {
			return fmt.Errorf("创建 Redis 队列失败: %v", err)
		}
		s.queue = queue

	default:
		return fmt.Errorf("不支持的队列类型: %s", queueType)
	}

	return nil
}

// startBinlogListener 启动 Binlog 监听器
func (s *IncrementalSync) startBinlogListener(snapshot *BinlogSnapshot) error {
	s.logService.Info(s.taskID, "启动 Binlog 监听器...")

	// 解密密码
	sourcePassword, err := s.dsService.crypto.Decrypt(s.task.SourceConn.Password)
	if err != nil {
		return fmt.Errorf("解密源数据库密码失败: %v", err)
	}

	// 构建表过滤配置
	tables := make(map[string][]string)
	for _, dbSelection := range s.config.SelectedDatabases {
		sourceDB := dbSelection.SourceDatabase
		if sourceDB == "" {
			sourceDB = dbSelection.Database
		}

		if tables[sourceDB] == nil {
			tables[sourceDB] = []string{}
		}

		for _, tableConfig := range dbSelection.Tables {
			tables[sourceDB] = append(tables[sourceDB], tableConfig.SourceTable)
		}
	}

	// 创建监听器
	listenerConfig := &BinlogListenerConfig{
		Host:      s.task.SourceConn.Host,
		Port:      s.task.SourceConn.Port,
		Username:  s.task.SourceConn.Username,
		Password:  sourcePassword,
		ServerID:  100, // 可以配置化
		StartFile: snapshot.File,
		StartPos:  snapshot.Position,
		Tables:    tables,
		Queue:     s.queue,
		TaskID:    s.taskID,
	}

	listener, err := NewBinlogListener(listenerConfig)
	if err != nil {
		return fmt.Errorf("创建 Binlog 监听器失败: %v", err)
	}

	s.listener = listener

	// 启动监听
	if err := listener.Start(); err != nil {
		return fmt.Errorf("启动 Binlog 监听器失败: %v", err)
	}

	s.logService.Info(s.taskID, "Binlog 监听器已启动")
	return nil
}

// executeFullSync 执行全量同步
func (s *IncrementalSync) executeFullSync() error {
	s.logService.Info(s.taskID, "开始执行全量同步阶段...")

	// 1. 初始化任务单元
	if err := s.initTaskUnits(); err != nil {
		return fmt.Errorf("初始化任务单元失败: %v", err)
	}

	// 2. 执行初始化阶段（创建数据库和表结构）
	if err := s.executeInitialize(); err != nil {
		return fmt.Errorf("初始化阶段失败: %v", err)
	}

	// 3. 执行数据同步阶段
	if err := s.executeSyncData(); err != nil {
		return fmt.Errorf("数据同步阶段失败: %v", err)
	}

	s.logService.Info(s.taskID, "全量同步阶段完成")
	return nil
}

// executeInitialize 执行初始化阶段（创建数据库和表结构）
func (s *IncrementalSync) executeInitialize() error {
	s.logService.Info(s.taskID, "开始初始化阶段...")

	// 1. 更新任务步骤
	database.DB.Model(&models.SyncTask{}).
		Where("id = ?", s.taskID).
		Update("current_step", "initialize")

	// 2. 查询所有待初始化的单元
	var units []*models.TaskUnitRuntime
	database.DB.Where("task_id = ? AND status = ?", s.taskID, "pending").Find(&units)

	if len(units) == 0 {
		s.logService.Info(s.taskID, "没有需要初始化的表")
		return nil
	}

	// 3. 创建同步引擎
	engine := NewSyncEngine()

	// 4. 创建 context
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	// 5. 执行初始化
	if err := engine.InitializeWorker(ctx, s.taskID, units); err != nil {
		return fmt.Errorf("初始化失败: %v", err)
	}

	s.logService.Info(s.taskID, "初始化阶段完成")
	return nil
}

// initTaskUnits 初始化任务单元（从 TaskControlService 复制）
func (s *IncrementalSync) initTaskUnits() error {
	s.logService.Info(s.taskID, "初始化任务单元...")

	// 1. 删除旧的任务单元
	database.DB.Where("task_id = ?", s.taskID).Delete(&models.TaskUnitRuntime{})
	database.DB.Where("task_id = ?", s.taskID).Delete(&models.TaskUnitConfig{})

	// 2. 创建任务单元
	for _, dbSelection := range s.config.SelectedDatabases {
		sourceDB := dbSelection.SourceDatabase
		if sourceDB == "" {
			sourceDB = dbSelection.Database
		}

		for _, tableConfig := range dbSelection.Tables {
			unitName := fmt.Sprintf("%s.%s", sourceDB, tableConfig.SourceTable)

			// 创建配置单元
			configUnit := &models.TaskUnitConfig{
				TaskID:   s.taskID,
				UnitName: unitName,
				UnitType: "table",
			}
			database.DB.Create(configUnit)

			// 创建运行时单元
			runtimeUnit := &models.TaskUnitRuntime{
				TaskID:           s.taskID,
				UnitName:         unitName,
				Status:           "pending",
				TotalRecords:     0,
				ProcessedRecords: 0,
			}
			database.DB.Create(runtimeUnit)
		}
	}

	s.logService.Info(s.taskID, "任务单元初始化完成")
	return nil
}

// executeSyncData 执行数据同步（从 TaskControlService 复制并简化）
func (s *IncrementalSync) executeSyncData() error {
	s.logService.Info(s.taskID, "开始数据同步阶段...")

	// 1. 更新任务步骤
	database.DB.Model(&models.SyncTask{}).
		Where("id = ?", s.taskID).
		Update("current_step", "sync_data")

	// 2. 查询所有待同步的单元
	var units []models.TaskUnitRuntime
	database.DB.Where("task_id = ? AND status = ?", s.taskID, "initialized").Find(&units)

	if len(units) == 0 {
		s.logService.Info(s.taskID, "没有需要同步的表")
		return nil
	}

	// 3. 外键排序
	sortedUnits, hasForeignKeys, _, err := s.sortUnitsByForeignKeys(units)
	if err != nil {
		s.logService.Warning(s.taskID, fmt.Sprintf("外键排序失败，使用原顺序: %v", err))
		sortedUnits = make([]*models.TaskUnitRuntime, len(units))
		for i := range units {
			sortedUnits[i] = &units[i]
		}
		hasForeignKeys = false
	}

	// 4. 创建任务队列
	taskQueue := make(chan *models.TaskUnitRuntime, len(sortedUnits))

	// 5. 创建 context
	ctx, cancel := context.WithCancel(s.ctx)
	defer cancel()

	// 6. 启动 Worker 池
	var wg sync.WaitGroup
	threadCount := s.config.SyncConfig.ThreadCount
	if threadCount <= 0 {
		threadCount = 4
	}

	engine := NewSyncEngine()

	// 如果有外键，启动外键 Worker
	if hasForeignKeys {
		wg.Add(1)
		go func() {
			defer wg.Done()
			engine.Worker(ctx, s.taskID, taskQueue, -1)
		}()
	}

	// 启动普通 Worker
	for i := 0; i < threadCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			engine.Worker(ctx, s.taskID, taskQueue, workerID)
		}(i)
	}

	// 7. 分发任务
	for _, unit := range sortedUnits {
		taskQueue <- unit
	}
	close(taskQueue)

	// 8. 等待完成
	wg.Wait()

	s.logService.Info(s.taskID, "数据同步阶段完成")
	return nil
}

// sortUnitsByForeignKeys 外键排序（简化版）
func (s *IncrementalSync) sortUnitsByForeignKeys(units []models.TaskUnitRuntime) ([]*models.TaskUnitRuntime, bool, map[string]bool, error) {
	// 解析数据库和表名
	tablesByDB := make(map[string][]string)
	unitMap := make(map[string]*models.TaskUnitRuntime)

	for i := range units {
		unit := &units[i]
		parts := splitUnitName(unit.UnitName)
		if len(parts) != 2 {
			continue
		}
		db, table := parts[0], parts[1]
		tablesByDB[db] = append(tablesByDB[db], table)
		unitMap[unit.UnitName] = unit
	}

	// 对每个数据库进行外键排序
	sortedUnits := make([]*models.TaskUnitRuntime, 0, len(units))
	hasForeignKeys := false
	fkTables := make(map[string]bool)

	for db, tables := range tablesByDB {
		sorted, hasFk, fkMap, err := SortTablesForDropWithFKList(s.sourceDB, db, tables)
		if err != nil {
			return nil, false, nil, err
		}

		if hasFk {
			hasForeignKeys = true
			for table := range fkMap {
				fkTables[fmt.Sprintf("%s.%s", db, table)] = true
			}
		}

		for _, table := range sorted {
			unitName := fmt.Sprintf("%s.%s", db, table)
			if unit, ok := unitMap[unitName]; ok {
				sortedUnits = append(sortedUnits, unit)
			}
		}
	}

	return sortedUnits, hasForeignKeys, fkTables, nil
}

// markFullSyncCompleted 标记全量同步完成
func (s *IncrementalSync) markFullSyncCompleted() {
	database.DB.Model(&models.SyncTask{}).
		Where("id = ?", s.taskID).
		Updates(map[string]interface{}{
			"full_sync_completed": true,
			"updated_at":          time.Now(),
		})

	s.logService.Info(s.taskID, "全量同步已完成，标记已更新")
}

// startIncrementalConsumer 启动增量消费
func (s *IncrementalSync) startIncrementalConsumer() error {
	s.logService.Info(s.taskID, "启动增量消费引擎...")

	// 创建消费者
	consumerConfig := &IncrementalConsumerConfig{
		TaskID:        s.taskID,
		Queue:         s.queue,
		TargetDB:      s.targetDB,
		ErrorStrategy: s.config.SyncConfig.ErrorStrategy,
		SaveInterval:  5 * time.Second,
	}

	consumer := NewIncrementalConsumer(consumerConfig)
	s.consumer = consumer

	// 启动消费
	if err := consumer.Start(); err != nil {
		return fmt.Errorf("启动增量消费失败: %v", err)
	}

	s.logService.Info(s.taskID, "增量消费引擎已启动")
	return nil
}

// splitUnitName 分割单元名称（格式：database.table）
func splitUnitName(unitName string) []string {
	result := []string{}
	parts := []rune(unitName)
	start := 0
	for i, ch := range parts {
		if ch == '.' {
			result = append(result, string(parts[start:i]))
			start = i + 1
		}
	}
	if start < len(parts) {
		result = append(result, string(parts[start:]))
	}
	return result
}
