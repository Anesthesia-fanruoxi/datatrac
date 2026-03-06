package services

import (
	"context"
	"datatrace/database"
	"datatrace/models"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// startFullSyncTask 启动全量同步任务
func (s *TaskControlService) startFullSyncTask(taskID string) error {
	// 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在")
	}

	// 清理之前的日志文件
	logDir := filepath.Join("logs", taskID)
	if _, err := os.Stat(logDir); err == nil {
		os.RemoveAll(logDir)
		fmt.Printf("已清除任务 %s 的旧日志文件\n", taskID)
	}

	// 创建日志服务
	logService := NewTaskLogService()

	// 从Redis获取配置（带降级）
	configCache := NewConfigCacheService()
	config, err := configCache.GetTaskConfigWithFallback(taskID)
	if err != nil {
		return fmt.Errorf("获取任务配置失败: %w", err)
	}

	threadCount := config.SyncConfig.ThreadCount
	if threadCount <= 0 {
		threadCount = 1 // 默认1个线程
	}

	// 生成任务单元列表（从配置中）
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

	// 初始化内存进度
	progressManager := GetProgressManager()
	progressManager.InitTask(taskID, unitNames)

	// 外键分析和排序
	logService.Info(taskID, "开始分析表的外键依赖关系...")

	fkSorter := NewTaskForeignKeySorter()
	sortedUnitNames, hasForeignKeys, fkTableSet, err := fkSorter.SortUnitsByForeignKeys(&task, unitNames, config)
	if err != nil {
		logService.Error(taskID, fmt.Sprintf("外键分析失败: %v", err))
		return fmt.Errorf("外键分析失败: %w", err)
	}

	if hasForeignKeys {
		logService.Info(taskID, "检测到外键依赖,将按依赖顺序初始化")
	} else {
		logService.Info(taskID, "未检测到外键依赖")
	}

	// 更新任务为运行状态
	task.IsRunning = true
	task.CurrentStep = "initialize"
	if err := database.DB.Save(&task).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 广播任务详情更新
	sseService := NewTaskSSEService()
	sseService.BroadcastTaskDetailUpdate(taskID)

	// 创建context和execution
	ctx, cancel := context.WithCancel(context.Background())
	wg := &sync.WaitGroup{}

	execManager := GetExecutionManager()
	execManager.StoreExecution(taskID, cancel, wg)

	// 启动异步执行流程
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			execManager.DeleteExecution(taskID)

			// 获取当前任务状态，判断是否正常完成
			var currentTask models.SyncTask
			database.DB.First(&currentTask, "id = ?", taskID)

			// 如果任务正常完成（不是被取消或失败），设置为completed步骤
			updateData := map[string]interface{}{
				"is_running": false,
			}

			// 如果是正常完成（没有错误），设置completed步骤
			if currentTask.CurrentStep == "sync_data" {
				updateData["current_step"] = "completed"
			}

			// 更新任务状态
			database.DB.Model(&models.SyncTask{}).
				Where("id = ?", taskID).
				Updates(updateData)

			// 广播任务详情更新
			sseService := NewTaskSSEService()
			sseService.BroadcastTaskDetailUpdate(taskID)
		}()

		engine := NewSyncEngine()

		// ========== 阶段1: 初始化（创建数据库和表结构） ==========
		logService.Info(taskID, "========== 开始全量同步 ==========")
		logService.Info(taskID, "步骤 1/2: 初始化阶段")

		if err := engine.InitializeWorker(ctx, taskID, sortedUnitNames); err != nil {
			logService.Error(taskID, fmt.Sprintf("初始化阶段失败: %v", err))
			return
		}

		// 检查是否被取消
		select {
		case <-ctx.Done():
			logService.Info(taskID, "任务被取消")
			return
		default:
		}

		// ========== 阶段2: 数据同步 ==========
		logService.Info(taskID, "步骤 2/2: 数据同步阶段")

		// 更新任务步骤
		database.DB.Model(&models.SyncTask{}).
			Where("id = ?", taskID).
			Update("current_step", "sync_data")
		progressManager.UpdateTaskStep(taskID, "sync_data")

		// 广播任务详情更新
		sseService.BroadcastTaskDetailUpdate(taskID)

		// 分离有外键和无外键的表
		var fkUnitNames []string     // 有外键依赖的表
		var normalUnitNames []string // 无外键依赖的表

		if hasForeignKeys {
			for _, unitName := range sortedUnitNames {
				if fkTableSet[unitName] {
					fkUnitNames = append(fkUnitNames, unitName)
				} else {
					normalUnitNames = append(normalUnitNames, unitName)
				}
			}

			// 反转外键表顺序，使其变成数据同步顺序（父表在前，子表在后）
			for i, j := 0, len(fkUnitNames)-1; i < j; i, j = i+1, j-1 {
				fkUnitNames[i], fkUnitNames[j] = fkUnitNames[j], fkUnitNames[i]
			}

			logService.Info(taskID, fmt.Sprintf("外键表: %d 个（外键Worker串行）, 普通表: %d 个（Worker池并发）", len(fkUnitNames), len(normalUnitNames)))
		} else {
			normalUnitNames = sortedUnitNames
		}

		// 创建队列
		fkQueue := make(chan string, len(fkUnitNames))
		normalQueue := make(chan string, len(normalUnitNames))

		// 填充外键队列
		for _, unitName := range fkUnitNames {
			fkQueue <- unitName
		}
		close(fkQueue)

		// 填充普通队列
		for _, unitName := range normalUnitNames {
			normalQueue <- unitName
		}
		close(normalQueue)

		var syncWg sync.WaitGroup

		// 如果有外键表，启动外键专用Worker
		if len(fkUnitNames) > 0 {
			syncWg.Add(1)
			go func() {
				defer syncWg.Done()
				logService.Info(taskID, "外键Worker 启动（串行处理外键表）")
				engine.Worker(ctx, taskID, fkQueue, -1) // workerID=-1 表示外键Worker
				logService.Info(taskID, "外键Worker 完成")
			}()
		}

		// 启动普通Worker池并发同步普通表
		for i := 0; i < threadCount; i++ {
			syncWg.Add(1)
			go func(workerID int) {
				defer syncWg.Done()
				logService.Info(taskID, fmt.Sprintf("Worker %d 启动", workerID))
				engine.Worker(ctx, taskID, normalQueue, workerID)
				logService.Info(taskID, fmt.Sprintf("Worker %d 完成", workerID))
			}(i)
		}

		// 等待所有Worker完成
		syncWg.Wait()

		logService.Info(taskID, "========== 全量同步完成 ==========")
	}()

	return nil
}

// startIncrementalTask 启动增量同步任务
func (s *TaskControlService) startIncrementalTask(taskID string) error {
	// 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在")
	}

	// 清理之前的日志文件
	logDir := filepath.Join("logs", taskID)
	if _, err := os.Stat(logDir); err == nil {
		os.RemoveAll(logDir)
		fmt.Printf("已清除任务 %s 的旧日志文件\n", taskID)
	}

	// 创建日志服务
	logService := NewTaskLogService()

	// 创建增量同步引擎
	incrementalSync, err := NewIncrementalSync(taskID)
	if err != nil {
		return fmt.Errorf("创建增量同步引擎失败: %v", err)
	}

	// 更新任务状态
	task.IsRunning = true
	if err := database.DB.Save(&task).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 广播任务详情更新
	sseService := NewTaskSSEService()
	sseService.BroadcastTaskDetailUpdate(taskID)

	// 存储增量同步实例
	execManager := GetExecutionManager()
	execManager.StoreIncrementalSync(taskID, incrementalSync)

	// 启动增量同步（异步）
	go func() {
		defer func() {
			// 更新任务状态
			database.DB.Model(&models.SyncTask{}).
				Where("id = ?", taskID).
				Updates(map[string]interface{}{
					"is_running": false,
				})
			// 广播任务详情更新
			sseService := NewTaskSSEService()
			sseService.BroadcastTaskDetailUpdate(taskID)
			// 清理增量同步实例
			execManager.DeleteIncrementalSync(taskID)
		}()

		if err := incrementalSync.Start(); err != nil {
			logService.Error(taskID, fmt.Sprintf("增量同步失败: %v", err))
		}
	}()

	return nil
}
