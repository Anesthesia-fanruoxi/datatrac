package services

import (
	"context"
	"database/sql"
	"datatrace/database"
	"datatrace/models"
	"datatrace/utils"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
)

// TaskExecution 任务执行信息
type TaskExecution struct {
	TaskID    string
	Cancel    context.CancelFunc
	TaskQueue chan *models.TaskUnitRuntime
	WaitGroup *sync.WaitGroup
}

// TaskControlService 任务控制服务
type TaskControlService struct {
	executions       sync.Map // map[taskID]*TaskExecution
	incrementalSyncs sync.Map // map[taskID]*IncrementalSync
}

var (
	taskControlInstance *TaskControlService
	taskControlOnce     sync.Once
)

// NewTaskControlService 获取任务控制服务单例
func NewTaskControlService() *TaskControlService {
	taskControlOnce.Do(func() {
		taskControlInstance = &TaskControlService{}
	})
	return taskControlInstance
}

// StartTask 启动任务
func (s *TaskControlService) StartTask(taskID string) error {
	// 1. 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在")
	}

	// 2. 检查任务状态
	if task.IsRunning {
		return fmt.Errorf("任务已在运行中")
	}

	if task.Status != "configured" {
		return fmt.Errorf("任务未配置，无法启动")
	}

	// 3. 根据同步模式选择执行路径
	if task.SyncMode == "incremental" {
		return s.startIncrementalTask(taskID)
	}

	// 默认执行全量同步
	return s.startFullSyncTask(taskID)
}

// startFullSyncTask 启动全量同步任务
func (s *TaskControlService) startFullSyncTask(taskID string) error {
	// 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在")
	}

	// 3. 检查是否所有任务单元都已完成（允许重新启动）
	var completedCount int64
	var totalCount int64
	database.DB.Model(&models.TaskUnitRuntime{}).Where("task_id = ?", taskID).Count(&totalCount)
	database.DB.Model(&models.TaskUnitRuntime{}).Where("task_id = ? AND status = ?", taskID, "completed").Count(&completedCount)

	// 如果所有任务单元都已完成，重置它们的状态为pending
	if totalCount > 0 && completedCount == totalCount {
		database.DB.Model(&models.TaskUnitRuntime{}).
			Where("task_id = ?", taskID).
			Updates(map[string]interface{}{
				"status":            "pending",
				"processed_records": 0,
				"total_records":     0,
			})
	}

	// 4. 清理之前的日志文件
	logDir := filepath.Join("logs", taskID)
	if _, err := os.Stat(logDir); err == nil {
		// 目录存在，删除所有日志文件
		os.RemoveAll(logDir)
		fmt.Printf("已清除任务 %s 的旧日志文件\n", taskID)
	}

	// 5. 创建日志服务
	logService := NewTaskLogService()

	// 6. 解析配置获取线程数
	var config TaskConfig
	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		return fmt.Errorf("解析任务配置失败: %w", err)
	}
	threadCount := config.SyncConfig.ThreadCount
	if threadCount <= 0 {
		threadCount = 1 // 默认1个线程
	}

	// 6. 初始化任务单元运行记录（如果是首次启动）
	if err := s.initTaskUnits(taskID); err != nil {
		return fmt.Errorf("初始化任务单元失败: %w", err)
	}

	// 7. 查询所有待处理的任务单元
	var units []models.TaskUnitRuntime
	database.DB.Where("task_id = ? AND status IN ?", taskID, []string{"pending", "failed"}).Find(&units)

	if len(units) == 0 {
		return fmt.Errorf("没有待处理的任务单元")
	}

	// 8. 外键分析和排序
	logService.Info(taskID, "开始分析表的外键依赖关系...")

	sortedUnits, hasForeignKeys, _, err := s.sortUnitsByForeignKeys(&task, units, &config)
	if err != nil {
		logService.Error(taskID, fmt.Sprintf("外键分析失败: %v", err))
		return fmt.Errorf("外键分析失败: %w", err)
	}

	if hasForeignKeys {
		logService.Info(taskID, "检测到外键依赖,将按依赖顺序初始化")
	} else {
		logService.Info(taskID, "未检测到外键依赖")
	}

	// 9. 更新任务为运行状态
	task.IsRunning = true
	task.CurrentStep = "initialize"
	if err := database.DB.Save(&task).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 广播任务详情更新
	sseService := NewTaskSSEService()
	sseService.BroadcastTaskDetailUpdate(taskID)

	// 10. 创建context和execution
	ctx, cancel := context.WithCancel(context.Background())

	execution := &TaskExecution{
		TaskID:    taskID,
		Cancel:    cancel,
		TaskQueue: nil, // 新流程不使用队列
		WaitGroup: &sync.WaitGroup{},
	}
	s.executions.Store(taskID, execution)

	// 11. 启动异步执行流程
	execution.WaitGroup.Add(1)
	go func() {
		defer execution.WaitGroup.Done()
		defer func() {
			s.executions.Delete(taskID)

			// 获取当前任务状态，判断是否正常完成
			var currentTask models.SyncTask
			database.DB.First(&currentTask, "id = ?", taskID)

			// 如果任务正常完成（不是被取消或失败），设置为completed步骤
			// 否则保持当前步骤不变
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

		if err := engine.InitializeWorker(ctx, taskID, sortedUnits); err != nil {
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

		// 广播任务详情更新
		sseService := NewTaskSSEService()
		sseService.BroadcastTaskDetailUpdate(taskID)

		// 分离有外键和无外键的表
		var fkUnits []*models.TaskUnitRuntime     // 有外键依赖的表
		var normalUnits []*models.TaskUnitRuntime // 无外键依赖的表

		if hasForeignKeys {
			// 从 sortedUnits 中分离
			// 注意：sortedUnits 是删除顺序（子表在前），数据同步需要反转（父表在前）
			fkTableSet := make(map[string]bool)

			// 重新获取外键表集合
			_, _, fkSet, _ := s.sortUnitsByForeignKeys(&task, units, &config)
			fkTableSet = fkSet

			for _, unit := range sortedUnits {
				if fkTableSet[unit.UnitName] {
					fkUnits = append(fkUnits, unit)
				} else {
					normalUnits = append(normalUnits, unit)
				}
			}

			// 反转外键表顺序，使其变成数据同步顺序（父表在前，子表在后）
			for i, j := 0, len(fkUnits)-1; i < j; i, j = i+1, j-1 {
				fkUnits[i], fkUnits[j] = fkUnits[j], fkUnits[i]
			}

			logService.Info(taskID, fmt.Sprintf("外键表: %d 个（外键Worker串行）, 普通表: %d 个（Worker池并发）", len(fkUnits), len(normalUnits)))
		} else {
			normalUnits = sortedUnits
		}

		// 创建队列
		fkQueue := make(chan *models.TaskUnitRuntime, len(fkUnits))
		normalQueue := make(chan *models.TaskUnitRuntime, len(normalUnits))

		// 填充外键队列
		for _, unit := range fkUnits {
			fkQueue <- unit
		}
		close(fkQueue)

		// 填充普通队列
		for _, unit := range normalUnits {
			normalQueue <- unit
		}
		close(normalQueue)

		var syncWg sync.WaitGroup

		// 如果有外键表，启动外键专用Worker
		if len(fkUnits) > 0 {
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

// PauseTask 暂停任务
func (s *TaskControlService) PauseTask(taskID string) error {
	// 1. 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在")
	}

	// 2. 检查任务状态
	if !task.IsRunning {
		return fmt.Errorf("任务未在运行中")
	}

	// 3. 根据同步模式处理
	if task.SyncMode == "incremental" {
		// 处理增量同步暂停
		if _, ok := s.incrementalSyncs.Load(taskID); ok {
			// 增量同步不支持暂停，只能停止
			return fmt.Errorf("增量同步不支持暂停，请使用停止功能")
		}
	} else {
		// 处理全量同步暂停
		if exec, ok := s.executions.Load(taskID); ok {
			execution := exec.(*TaskExecution)
			execution.Cancel()         // 发送取消信号
			execution.WaitGroup.Wait() // 等待所有Worker退出
		}

		// 更新运行中的任务单元状态为paused
		database.DB.Model(&models.TaskUnitRuntime{}).
			Where("task_id = ? AND status = ?", taskID, "running").
			Update("status", "paused")
	}

	// 4. 更新任务状态
	task.IsRunning = false
	if err := database.DB.Save(&task).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 广播任务详情更新
	sseService := NewTaskSSEService()
	sseService.BroadcastTaskDetailUpdate(taskID)

	return nil
}

// StopTask 停止任务
func (s *TaskControlService) StopTask(taskID string) error {
	// 1. 查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在")
	}

	// 2. 根据同步模式处理
	if task.SyncMode == "incremental" {
		// 处理增量同步停止
		if incrementalSync, ok := s.incrementalSyncs.Load(taskID); ok {
			// 停止增量同步
			if err := incrementalSync.(*IncrementalSync).Stop(); err != nil {
				return fmt.Errorf("停止增量同步失败: %v", err)
			}
			// 等待增量同步停止
			time.Sleep(1 * time.Second)
			// 清理增量同步实例
			s.incrementalSyncs.Delete(taskID)
		}
	} else {
		// 处理全量同步停止
		if task.IsRunning {
			if exec, ok := s.executions.Load(taskID); ok {
				execution := exec.(*TaskExecution)
				execution.Cancel()         // 发送取消信号
				execution.WaitGroup.Wait() // 等待所有Worker退出
			}
		}

		// 清除任务单元的状态，而不是删除它们
		database.DB.Model(&models.TaskUnitRuntime{}).
			Where("task_id = ?", taskID).
			Updates(map[string]interface{}{
				"status":            "pending",
				"processed_records": 0,
				"total_records":     0,
				"started_at":        nil,
				"updated_at":        time.Now(),
			})
	}

	// 3. 更新任务状态
	task.IsRunning = false
	// 注意：不清空current_step，保留步骤信息用于判断任务状态
	// 只有在任务被完全重置时才清空步骤
	if err := database.DB.Save(&task).Error; err != nil {
		return fmt.Errorf("更新任务状态失败: %w", err)
	}

	// 广播任务详情更新
	sseService := NewTaskSSEService()
	sseService.BroadcastTaskDetailUpdate(taskID)

	return nil
}

// initTaskUnits 初始化任务单元运行记录
func (s *TaskControlService) initTaskUnits(taskID string) error {
	// 检查是否已有运行记录
	var count int64
	database.DB.Model(&models.TaskUnitRuntime{}).Where("task_id = ?", taskID).Count(&count)
	if count > 0 {
		// 已有记录，更新状态为pending
		database.DB.Model(&models.TaskUnitRuntime{}).
			Where("task_id = ? AND status IN ?", taskID, []string{"paused", "failed"}).
			Update("status", "pending")
		return nil
	}

	// 查询任务单元配置
	var configs []models.TaskUnitConfig
	if err := database.DB.Where("task_id = ?", taskID).Find(&configs).Error; err != nil {
		return err
	}

	if len(configs) == 0 {
		return fmt.Errorf("任务没有配置单元")
	}

	// 创建运行记录（pending状态不设置StartedAt，等实际开始运行时再设置）
	now := time.Now()
	var runtimes []models.TaskUnitRuntime
	for _, config := range configs {
		runtime := models.TaskUnitRuntime{
			ID:               uuid.New().String(),
			TaskID:           taskID,
			UnitName:         config.UnitName,
			Status:           "pending",
			TotalRecords:     0,
			ProcessedRecords: 0,
			StartedAt:        nil,
			UpdatedAt:        now,
		}
		runtimes = append(runtimes, runtime)
	}

	// 批量插入
	if err := database.DB.Create(&runtimes).Error; err != nil {
		return err
	}

	return nil
}

// sortUnitsByForeignKeys 按照外键依赖关系对任务单元进行排序
// 返回: 排序后的单元列表, 是否有外键依赖, 外键表集合(unitName -> bool), 错误
func (s *TaskControlService) sortUnitsByForeignKeys(task *models.SyncTask, units []models.TaskUnitRuntime, config *TaskConfig) ([]*models.TaskUnitRuntime, bool, map[string]bool, error) {
	logService := NewTaskLogService()
	fkTableSet := make(map[string]bool) // 记录哪些表有外键依赖

	// 如果只有一个表,直接返回
	if len(units) <= 1 {
		result := make([]*models.TaskUnitRuntime, len(units))
		for i := range units {
			result[i] = &units[i]
		}
		return result, false, fkTableSet, nil
	}

	// 需要先 Preload 关联的数据源
	if err := database.DB.Preload("SourceConn").First(task, "id = ?", task.ID).Error; err != nil {
		return nil, false, fkTableSet, fmt.Errorf("加载数据源信息失败: %w", err)
	}

	// 解密源数据库密码
	crypto := utils.NewCryptoService()
	sourcePassword, err := crypto.Decrypt(task.SourceConn.Password)
	if err != nil {
		return nil, false, fkTableSet, fmt.Errorf("解密源数据库密码失败: %w", err)
	}

	// 按数据库分组
	dbGroups := make(map[string][]models.TaskUnitRuntime)
	for _, unit := range units {
		parts := splitTableName(unit.UnitName)
		if len(parts) >= 1 {
			targetDBName := parts[0]
			dbGroups[targetDBName] = append(dbGroups[targetDBName], unit)
		}
	}

	logService.Info(task.ID, fmt.Sprintf("共 %d 个数据库需要分析外键关系", len(dbGroups)))

	// 对每个数据库的表进行排序
	var sortedUnits []*models.TaskUnitRuntime
	hasForeignKeys := false

	for targetDBName, dbUnits := range dbGroups {
		logService.Info(task.ID, fmt.Sprintf("正在分析数据库 %s 的 %d 个表...", targetDBName, len(dbUnits)))

		// 从配置中找到源数据库名
		var sourceDBName string
		for _, dbSel := range config.SelectedDatabases {
			if dbSel.Database == targetDBName {
				sourceDBName = dbSel.SourceDatabase
				break
			}
		}

		if sourceDBName == "" {
			logService.Warning(task.ID, fmt.Sprintf("未找到目标数据库 %s 对应的源数据库, 使用原顺序", targetDBName))
			for i := range dbUnits {
				sortedUnits = append(sortedUnits, &dbUnits[i])
			}
			continue
		}

		// 连接到源数据库
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
			task.SourceConn.Username, sourcePassword, task.SourceConn.Host, task.SourceConn.Port, sourceDBName)

		db, err := sql.Open("mysql", dsn)
		if err != nil {
			logService.Warning(task.ID, fmt.Sprintf("连接源数据库 %s 失败: %v, 使用原顺序", sourceDBName, err))
			for i := range dbUnits {
				sortedUnits = append(sortedUnits, &dbUnits[i])
			}
			continue
		}

		// 测试连接
		if err := db.Ping(); err != nil {
			logService.Warning(task.ID, fmt.Sprintf("源数据库 %s Ping 失败: %v, 使用原顺序", sourceDBName, err))
			db.Close()
			for i := range dbUnits {
				sortedUnits = append(sortedUnits, &dbUnits[i])
			}
			continue
		}

		// 提取源表名列表（需要从配置中查找映射）
		sourceTableNames := make([]string, 0, len(dbUnits))
		unitMap := make(map[string]*models.TaskUnitRuntime)

		for i, unit := range dbUnits {
			parts := splitTableName(unit.UnitName)
			if len(parts) >= 2 {
				targetTableName := parts[1]
				// 从配置中找到源表名
				var sourceTableName string
				for _, dbSel := range config.SelectedDatabases {
					if dbSel.Database == targetDBName {
						for _, tbl := range dbSel.Tables {
							if tbl.TargetTable == targetTableName {
								sourceTableName = tbl.SourceTable
								break
							}
						}
						break
					}
				}

				if sourceTableName != "" {
					sourceTableNames = append(sourceTableNames, sourceTableName)
					unitMap[sourceTableName] = &dbUnits[i]
				}
			}
		}

		// 对源表进行排序,并获取外键表列表
		sortedSourceTableNames, dbHasForeignKeys, fkTables, err := SortTablesForDropWithFKList(db, sourceDBName, sourceTableNames)
		db.Close()

		if err != nil {
			logService.Warning(task.ID, fmt.Sprintf("源数据库 %s 排序失败: %v, 使用原顺序", sourceDBName, err))
			for i := range dbUnits {
				sortedUnits = append(sortedUnits, &dbUnits[i])
			}
			continue
		}

		// 如果这个数据库有外键,标记整个任务有外键
		if dbHasForeignKeys {
			hasForeignKeys = true
			logService.Info(task.ID, fmt.Sprintf("源数据库 %s 检测到 %d 个外键表,已按依赖顺序排序", sourceDBName, len(fkTables)))

			// 记录哪些表有外键(使用完整的 unitName: 目标库.目标表)
			for sourceTableName := range fkTables {
				// 找到对应的 unit
				if unit, ok := unitMap[sourceTableName]; ok {
					fkTableSet[unit.UnitName] = true
				}
			}
		} else {
			logService.Info(task.ID, fmt.Sprintf("源数据库 %s 未检测到外键依赖", sourceDBName))
		}

		// 按照排序后的顺序添加到结果中
		for _, sourceTableName := range sortedSourceTableNames {
			if unit, ok := unitMap[sourceTableName]; ok {
				sortedUnits = append(sortedUnits, unit)
			}
		}
	}

	logService.Info(task.ID, "外键分析完成")
	return sortedUnits, hasForeignKeys, fkTableSet, nil
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
		// 目录存在，删除所有日志文件
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
	s.incrementalSyncs.Store(taskID, incrementalSync)

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
			s.incrementalSyncs.Delete(taskID)
		}()

		if err := incrementalSync.Start(); err != nil {
			logService.Error(taskID, fmt.Sprintf("增量同步失败: %v", err))
		}
	}()

	return nil
}

// GetIncrementalSyncStatus 获取增量同步状态
func (s *TaskControlService) GetIncrementalSyncStatus(taskID string) (map[string]interface{}, error) {
	if incrementalSync, ok := s.incrementalSyncs.Load(taskID); ok {
		return incrementalSync.(*IncrementalSync).GetStatus(), nil
	}
	return nil, fmt.Errorf("增量同步实例不存在")
}
