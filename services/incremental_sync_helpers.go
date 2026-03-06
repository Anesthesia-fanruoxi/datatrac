package services

import (
	"database/sql"
	"datatrace/database"
	"datatrace/models"
	"fmt"
	"sync"
	"time"

	_ "github.com/go-sql-driver/mysql"
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

	// 连接源数据库（不指定具体数据库）
	sourceDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		s.task.SourceConn.Username, sourcePassword,
		s.task.SourceConn.Host, s.task.SourceConn.Port)

	s.sourceDB, err = sql.Open("mysql", sourceDSN)
	if err != nil {
		return fmt.Errorf("连接源数据库失败: %v", err)
	}

	if err := s.sourceDB.Ping(); err != nil {
		return fmt.Errorf("源数据库连接测试失败: %v", err)
	}

	// 连接目标数据库（不指定具体数据库）
	targetDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		s.task.TargetConn.Username, targetPassword,
		s.task.TargetConn.Host, s.task.TargetConn.Port)

	s.targetDB, err = sql.Open("mysql", targetDSN)
	if err != nil {
		return fmt.Errorf("连接目标数据库失败: %v", err)
	}

	if err := s.targetDB.Ping(); err != nil {
		return fmt.Errorf("目标数据库连接测试失败: %v", err)
	}

	s.logService.Info(s.taskID, "数据库连接初始化成功")
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

// checkBinlogConfig 检查源库 Binlog 配置
func (s *IncrementalSync) checkBinlogConfig() error {
	s.logService.Info(s.taskID, "检查源库 Binlog 配置...")

	// 检查 binlog_format
	var binlogFormat string
	err := s.sourceDB.QueryRow("SELECT @@binlog_format").Scan(&binlogFormat)
	if err != nil {
		return fmt.Errorf("查询 binlog_format 失败: %v", err)
	}

	if binlogFormat != "ROW" {
		return fmt.Errorf("binlog_format 必须为 ROW，当前为: %s", binlogFormat)
	}

	// 检查 binlog_row_image
	var binlogRowImage string
	err = s.sourceDB.QueryRow("SELECT @@binlog_row_image").Scan(&binlogRowImage)
	if err != nil {
		return fmt.Errorf("查询 binlog_row_image 失败: %v", err)
	}

	if binlogRowImage != "FULL" {
		s.logService.Warning(s.taskID, fmt.Sprintf("建议 binlog_row_image 设置为 FULL，当前为: %s", binlogRowImage))
	}

	s.logService.Info(s.taskID, fmt.Sprintf("Binlog 配置检查通过 (format=%s, row_image=%s)", binlogFormat, binlogRowImage))
	return nil
}

// captureSnapshot 捕获 Binlog 快照点
func (s *IncrementalSync) captureSnapshot() (*BinlogSnapshot, error) {
	s.logService.Info(s.taskID, "捕获 Binlog 快照点...")

	var file string
	var position uint32
	var binlogDoDB, binlogIgnoreDB, executedGtidSet string

	err := s.sourceDB.QueryRow("SHOW MASTER STATUS").Scan(
		&file, &position, &binlogDoDB, &binlogIgnoreDB, &executedGtidSet,
	)
	if err != nil {
		return nil, fmt.Errorf("获取 MASTER STATUS 失败: %v", err)
	}

	snapshot := &BinlogSnapshot{
		File:      file,
		Position:  position,
		Timestamp: time.Now(),
	}

	s.logService.Info(s.taskID, fmt.Sprintf("快照点: %s:%d", snapshot.File, snapshot.Position))
	return snapshot, nil
}

// createQueue 创建 Binlog 队列
func (s *IncrementalSync) createQueue() error {
	s.logService.Info(s.taskID, "创建 Binlog 队列...")

	queueType := s.task.QueueType
	if queueType == "" {
		queueType = "memory" // 默认使用内存队列
	}

	var err error
	if queueType == "redis" {
		s.queue, err = NewRedisQueue(database.RedisClient, s.taskID, "worker-1")
	} else {
		s.queue = NewMemoryQueue(10000) // 缓冲区大小10000
	}

	if err != nil {
		return fmt.Errorf("创建队列失败: %w", err)
	}

	s.logService.Info(s.taskID, fmt.Sprintf("队列创建成功 (类型: %s)", queueType))
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

	// 提取需要监听的数据库和表
	tableFilters := make(map[string][]string)

	for _, dbSel := range s.config.SelectedDatabases {
		sourceDB := dbSel.SourceDatabase
		var tables []string
		for _, tbl := range dbSel.Tables {
			tables = append(tables, tbl.SourceTable)
		}
		tableFilters[sourceDB] = tables
	}

	// 创建监听器配置
	config := &BinlogListenerConfig{
		Host:      s.task.SourceConn.Host,
		Port:      s.task.SourceConn.Port,
		Username:  s.task.SourceConn.Username,
		Password:  sourcePassword,
		ServerID:  100,
		StartFile: snapshot.File,
		StartPos:  snapshot.Position,
		Tables:    tableFilters,
		Queue:     s.queue,
		TaskID:    s.taskID,
		SourceDB:  s.sourceDB,
	}

	// 创建监听器
	s.listener, err = NewBinlogListener(config)
	if err != nil {
		return fmt.Errorf("创建监听器失败: %v", err)
	}

	// 启动监听
	if err := s.listener.Start(); err != nil {
		return fmt.Errorf("启动监听器失败: %v", err)
	}

	s.logService.Info(s.taskID, "Binlog 监听器启动成功")
	return nil
}

// executeFullSync 执行全量同步（复用全量同步逻辑）
func (s *IncrementalSync) executeFullSync() error {
	s.logService.Info(s.taskID, "========== 开始全量同步阶段 ==========")

	// 0. 清理旧的Redis统计数据（避免显示旧的源表统计）
	statsService := NewIncrementalStatsService()
	statsService.ClearTaskStats(s.taskID)
	s.logService.Info(s.taskID, "已清理旧的Redis统计数据")

	// 1. 从Redis获取配置
	configCache := NewConfigCacheService()
	config, err := configCache.GetTaskConfigWithFallback(s.taskID)
	if err != nil {
		return fmt.Errorf("获取任务配置失败: %v", err)
	}

	threadCount := config.SyncConfig.ThreadCount
	if threadCount <= 0 {
		threadCount = 1
	}

	// 2. 生成任务单元列表
	var unitNames []string
	for _, db := range config.SelectedDatabases {
		for _, tableConfig := range db.Tables {
			targetDatabase := db.Database
			targetTable := tableConfig.TargetTable
			if targetTable == "" {
				targetTable = tableConfig.SourceTable
			}
			unitName := fmt.Sprintf("%s.%s", targetDatabase, targetTable)
			unitNames = append(unitNames, unitName)
		}
	}

	if len(unitNames) == 0 {
		return fmt.Errorf("没有待处理的任务单元")
	}

	// 3. 初始化内存进度
	progressManager := GetProgressManager()
	progressManager.InitTask(s.taskID, unitNames)

	// 4. 初始化Redis表统计结构（用于SSE推送，使用目标表名）
	for _, db := range config.SelectedDatabases {
		targetDB := db.Database
		for _, tableConfig := range db.Tables {
			targetTable := tableConfig.TargetTable
			if targetTable == "" {
				targetTable = tableConfig.SourceTable
			}
			statsService.InitTableStats(s.taskID, targetDB, targetTable)
		}
	}

	// 5. 外键分析和排序
	s.logService.Info(s.taskID, "开始分析表的外键依赖关系...")

	fkSorter := NewTaskForeignKeySorter()
	sortedUnitNames, hasForeignKeys, fkTableSet, err := fkSorter.SortUnitsByForeignKeys(s.task, unitNames, config)
	if err != nil {
		s.logService.Error(s.taskID, fmt.Sprintf("外键分析失败: %v", err))
		return fmt.Errorf("外键分析失败: %w", err)
	}

	if hasForeignKeys {
		s.logService.Info(s.taskID, "检测到外键依赖,将按依赖顺序初始化")
	} else {
		s.logService.Info(s.taskID, "未检测到外键依赖")
	}

	// 6. 更新任务步骤
	database.DB.Model(&models.SyncTask{}).
		Where("id = ?", s.taskID).
		Update("current_step", "initialize")
	progressManager.UpdateTaskStep(s.taskID, "initialize")

	// 广播任务详情更新
	s.sseService.BroadcastTaskDetailUpdate(s.taskID)

	// 7. 创建同步引擎
	engine := NewSyncEngine()

	// 8. 初始化阶段
	s.logService.Info(s.taskID, "步骤 1/2: 初始化阶段")
	if err := engine.InitializeWorker(s.ctx, s.taskID, sortedUnitNames); err != nil {
		s.logService.Error(s.taskID, fmt.Sprintf("初始化阶段失败: %v", err))
		return fmt.Errorf("初始化阶段失败: %v", err)
	}

	// 检查是否被取消
	select {
	case <-s.ctx.Done():
		s.logService.Info(s.taskID, "任务被取消")
		return fmt.Errorf("任务被取消")
	default:
	}

	// 9. 数据同步阶段
	s.logService.Info(s.taskID, "步骤 2/2: 数据同步阶段")

	database.DB.Model(&models.SyncTask{}).
		Where("id = ?", s.taskID).
		Update("current_step", "sync_data")
	progressManager.UpdateTaskStep(s.taskID, "sync_data")

	s.sseService.BroadcastTaskDetailUpdate(s.taskID)

	// 分离有外键和无外键的表
	var fkUnitNames []string
	var normalUnitNames []string

	if hasForeignKeys {
		for _, unitName := range sortedUnitNames {
			if fkTableSet[unitName] {
				fkUnitNames = append(fkUnitNames, unitName)
			} else {
				normalUnitNames = append(normalUnitNames, unitName)
			}
		}

		// 反转外键表顺序
		for i, j := 0, len(fkUnitNames)-1; i < j; i, j = i+1, j-1 {
			fkUnitNames[i], fkUnitNames[j] = fkUnitNames[j], fkUnitNames[i]
		}

		s.logService.Info(s.taskID, fmt.Sprintf("外键表: %d 个（串行）, 普通表: %d 个（并发）", len(fkUnitNames), len(normalUnitNames)))
	} else {
		normalUnitNames = sortedUnitNames
	}

	// 创建队列
	fkQueue := make(chan string, len(fkUnitNames))
	normalQueue := make(chan string, len(normalUnitNames))

	for _, unitName := range fkUnitNames {
		fkQueue <- unitName
	}
	close(fkQueue)

	for _, unitName := range normalUnitNames {
		normalQueue <- unitName
	}
	close(normalQueue)

	var syncWg sync.WaitGroup

	// 外键Worker
	if len(fkUnitNames) > 0 {
		syncWg.Add(1)
		go func() {
			defer syncWg.Done()
			s.logService.Info(s.taskID, "外键Worker 启动")
			engine.Worker(s.ctx, s.taskID, fkQueue, -1)
			s.logService.Info(s.taskID, "外键Worker 完成")
		}()
	}

	// 普通Worker池
	for i := 0; i < threadCount; i++ {
		syncWg.Add(1)
		go func(workerID int) {
			defer syncWg.Done()
			s.logService.Info(s.taskID, fmt.Sprintf("Worker %d 启动", workerID))
			engine.Worker(s.ctx, s.taskID, normalQueue, workerID)
			s.logService.Info(s.taskID, fmt.Sprintf("Worker %d 完成", workerID))
		}(i)
	}

	syncWg.Wait()

	s.logService.Info(s.taskID, "========== 全量同步阶段完成 ==========")
	return nil
}

// startIncrementalConsumer 启动增量消费
func (s *IncrementalSync) startIncrementalConsumer() error {
	s.logService.Info(s.taskID, "启动增量消费...")

	// 解密密码
	targetPassword, err := s.dsService.crypto.Decrypt(s.task.TargetConn.Password)
	if err != nil {
		return fmt.Errorf("解密目标数据库密码失败: %v", err)
	}

	// 连接目标数据库（不指定具体数据库）
	targetDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		s.task.TargetConn.Username, targetPassword,
		s.task.TargetConn.Host, s.task.TargetConn.Port)

	targetDB, err := sql.Open("mysql", targetDSN)
	if err != nil {
		return fmt.Errorf("连接目标数据库失败: %v", err)
	}

	if err := targetDB.Ping(); err != nil {
		return fmt.Errorf("目标数据库连接测试失败: %v", err)
	}

	// 创建消费者配置
	consumerConfig := &IncrementalConsumerConfig{
		TaskID:        s.taskID,
		Queue:         s.queue,
		TargetDB:      targetDB,
		ErrorStrategy: s.config.SyncConfig.ErrorStrategy,
		SaveInterval:  5 * time.Second,
		TaskConfig:    s.config,
	}

	// 创建消费者
	s.consumer = NewIncrementalConsumer(consumerConfig)

	// 启动消费
	if err := s.consumer.Start(); err != nil {
		return fmt.Errorf("启动消费者失败: %v", err)
	}

	s.logService.Info(s.taskID, "增量消费启动成功")
	return nil
}

// markFullSyncCompleted 标记全量同步完成
func (s *IncrementalSync) markFullSyncCompleted() {
	s.logService.Info(s.taskID, "标记全量同步完成")
	// 可以在这里记录一些元数据到Redis，比如全量完成时间等
}
