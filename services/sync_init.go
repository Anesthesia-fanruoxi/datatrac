package services

import (
	"context"
	"database/sql"
	"datatrace/database"
	"datatrace/models"
	"fmt"
)

// InitializeWorker 初始化Worker执行逻辑（串行处理所有表的初始化）
func (e *SyncEngine) InitializeWorker(ctx context.Context, taskID string, unitNames []string) error {
	progressManager := GetProgressManager()

	// 记录初始化开始日志
	e.logService.AddLog(taskID, "info", "========== 初始化阶段开始 ==========", "initialize")

	// 更新任务步骤
	database.DB.Model(&models.SyncTask{}).
		Where("id = ?", taskID).
		Update("current_step", "initialize")
	progressManager.UpdateTaskStep(taskID, "initialize")

	// 1. 查询任务配置
	var task models.SyncTask
	if err := database.DB.Preload("SourceConn").Preload("TargetConn").First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("查询任务失败: %v", err)
	}

	// 2. 从Redis获取配置
	configCache := NewConfigCacheService()
	config, err := configCache.GetTaskConfigWithFallback(taskID)
	if err != nil {
		return fmt.Errorf("获取配置失败: %v", err)
	}

	// 3. 解密密码
	sourcePassword, err := e.dsService.crypto.Decrypt(task.SourceConn.Password)
	if err != nil {
		return fmt.Errorf("解密源数据库密码失败: %v", err)
	}

	targetPassword, err := e.dsService.crypto.Decrypt(task.TargetConn.Password)
	if err != nil {
		return fmt.Errorf("解密目标数据库密码失败: %v", err)
	}

	// 4. 按数据库分组
	dbGroups := make(map[string]struct {
		sourceDB  string
		targetDB  string
		unitNames []string
	})

	for _, unitName := range unitNames {
		sourceDB, _, targetDB, _, err := e.parseUnitName(unitName, config)
		if err != nil {
			e.logService.Error(taskID, fmt.Sprintf("解析表名失败: %v", err))
			continue
		}

		key := targetDB
		if group, ok := dbGroups[key]; ok {
			group.unitNames = append(group.unitNames, unitName)
			dbGroups[key] = group
		} else {
			dbGroups[key] = struct {
				sourceDB  string
				targetDB  string
				unitNames []string
			}{
				sourceDB:  sourceDB,
				targetDB:  targetDB,
				unitNames: []string{unitName},
			}
		}
	}

	// 记录数据库初始化信息
	msg := fmt.Sprintf("需要初始化 %d 个数据库", len(dbGroups))
	e.logService.Info(taskID, msg)

	// 5. 创建所有数据库
	for _, group := range dbGroups {
		// 检查context是否被取消
		select {
		case <-ctx.Done():
			return fmt.Errorf("初始化被取消: %v", ctx.Err())
		default:
		}

		// 连接源数据库获取字符集
		sourceDSN := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
			task.SourceConn.Username, sourcePassword, task.SourceConn.Host, task.SourceConn.Port, group.sourceDB)

		sourceDB, err := sql.Open("mysql", sourceDSN)
		if err != nil {
			e.logService.Error(taskID, fmt.Sprintf("连接源数据库 %s 失败: %v", group.sourceDB, err))
			continue
		}

		// 获取字符集
		var charset, collation string
		err = sourceDB.QueryRow("SELECT @@character_set_database, @@collation_database").Scan(&charset, &collation)
		sourceDB.Close()

		if err != nil {
			e.logService.Warning(taskID, fmt.Sprintf("获取数据库 %s 字符集失败，使用默认值: %v", group.sourceDB, err))
			charset = "utf8mb4"
			collation = "utf8mb4_general_ci"
		}

		// 创建目标数据库
		created, err := CreateDatabaseIfNotExists(
			task.TargetConn.Host,
			task.TargetConn.Port,
			task.TargetConn.Username,
			targetPassword,
			group.targetDB,
			charset,
			collation,
		)

		if err != nil {
			e.logService.Error(taskID, fmt.Sprintf("创建数据库 %s 失败: %v", group.targetDB, err))
			return fmt.Errorf("创建数据库失败: %v", err)
		}

		if created {
			e.logService.Info(taskID, fmt.Sprintf("创建数据库: %s (字符集: %s, 排序规则: %s)", group.targetDB, charset, collation))
		} else {
			e.logService.Info(taskID, fmt.Sprintf("数据库已存在: %s", group.targetDB))
		}
	}

	// 6. 按顺序初始化所有表
	e.logService.Info(taskID, fmt.Sprintf("开始初始化 %d 个表的结构", len(unitNames)))

	strategy := config.SyncConfig.TableExistsStrategy

	// 如果是 drop 策略，需要分两个阶段：先删除所有表，再创建所有表
	if strategy == "drop" {
		// 阶段1：按删除顺序删除所有表（子表在前，父表在后）
		e.logService.Info(taskID, "阶段1: 删除所有表")

		for _, unitName := range unitNames {
			if err := e.dropTable(ctx, taskID, unitName, &task, config, sourcePassword, targetPassword); err != nil {
				e.logService.Error(taskID, fmt.Sprintf("删除表 %s 失败: %v", unitName, err))
				return fmt.Errorf("删除表失败: %v", err)
			}
		}

		// 阶段2：按创建顺序创建所有表（父表在前，子表在后，需要反转）
		e.logService.Info(taskID, "阶段2: 创建所有表")

		for i := len(unitNames) - 1; i >= 0; i-- {
			unitName := unitNames[i]
			if err := e.createTable(ctx, taskID, unitName, &task, config, sourcePassword, targetPassword); err != nil {
				e.logService.Error(taskID, fmt.Sprintf("创建表 %s 失败: %v", unitName, err))
				return fmt.Errorf("创建表失败: %v", err)
			}
		}
	} else {
		// truncate 或 append 策略，直接按顺序处理
		for _, unitName := range unitNames {
			select {
			case <-ctx.Done():
				return fmt.Errorf("初始化被取消: %v", ctx.Err())
			default:
			}

			if err := e.InitializeTable(ctx, taskID, unitName, &task, config, sourcePassword, targetPassword); err != nil {
				e.logService.Error(taskID, fmt.Sprintf("初始化表 %s 失败: %v", unitName, err))
				return fmt.Errorf("初始化表失败: %v", err)
			}
		}
	}

	// 记录初始化完成日志
	msg = "========== 初始化阶段完成 =========="
	e.logService.AddLog(taskID, "success", msg, "initialize")

	return nil
}

// InitializeTable 初始化单个表（只处理表结构，不同步数据）
func (e *SyncEngine) InitializeTable(ctx context.Context, taskID string, unitName string,
	task *models.SyncTask, config *TaskConfig, sourcePassword, targetPassword string) error {

	progressManager := GetProgressManager()

	// 1. 解析表名
	sourceDB, sourceTable, targetDB, targetTable, err := e.parseUnitName(unitName, config)
	if err != nil {
		return fmt.Errorf("解析表名失败: %v", err)
	}

	// 2. 创建Reader（用于获取表结构和记录数）
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
		return fmt.Errorf("创建Reader失败: %v", err)
	}
	defer reader.Close()

	// 3. 创建Writer
	writer, err := NewMySQLWriter(
		task.TargetConn.Host,
		task.TargetConn.Port,
		task.TargetConn.Username,
		targetPassword,
		targetDB,
		targetTable,
	)
	if err != nil {
		return fmt.Errorf("创建Writer失败: %v", err)
	}
	defer writer.Close()

	// 4. 处理表存在策略
	strategy := config.SyncConfig.TableExistsStrategy
	switch strategy {
	case "drop":
		// 删除表
		if err := writer.DropTable(); err != nil {
			return fmt.Errorf("删除表失败: %v", err)
		}

		// 创建表结构（包含外键）
		if err := writer.CreateTableLike(reader.GetDB(), sourceTable); err != nil {
			return fmt.Errorf("创建表结构失败: %v", err)
		}

	case "truncate":
		// 清空表
		if err := writer.TruncateTable(); err != nil {
			return fmt.Errorf("清空表失败: %v", err)
		}

	case "append":
		// 检查表是否存在，不存在则创建
		err := writer.CreateTableLike(reader.GetDB(), sourceTable)
		if err != nil {
			// 表可能已存在，忽略错误
		}
	}

	// 5. 更新总记录数到内存（为后续同步做准备）
	totalRecords := reader.GetTotalCount()
	progressManager.UpdateUnitProgress(taskID, unitName, totalRecords, 0)
	progressManager.UpdateUnitStatus(taskID, unitName, "initialized")

	return nil
}

// dropTable 只删除表（用于 drop 策略的第一阶段）
func (e *SyncEngine) dropTable(ctx context.Context, taskID string, unitName string,
	task *models.SyncTask, config *TaskConfig, sourcePassword, targetPassword string) error {

	// 解析表名
	_, _, targetDB, targetTable, err := e.parseUnitName(unitName, config)
	if err != nil {
		return fmt.Errorf("解析表名失败: %v", err)
	}

	// 创建Writer
	writer, err := NewMySQLWriter(
		task.TargetConn.Host,
		task.TargetConn.Port,
		task.TargetConn.Username,
		targetPassword,
		targetDB,
		targetTable,
	)
	if err != nil {
		return fmt.Errorf("创建Writer失败: %v", err)
	}
	defer writer.Close()

	// 删除表
	if err := writer.DropTable(); err != nil {
		return fmt.Errorf("删除表失败: %v", err)
	}

	return nil
}

// createTable 只创建表（用于 drop 策略的第二阶段）
func (e *SyncEngine) createTable(ctx context.Context, taskID string, unitName string,
	task *models.SyncTask, config *TaskConfig, sourcePassword, targetPassword string) error {

	progressManager := GetProgressManager()

	// 解析表名
	sourceDB, sourceTable, targetDB, targetTable, err := e.parseUnitName(unitName, config)
	if err != nil {
		return fmt.Errorf("解析表名失败: %v", err)
	}

	// 创建Reader（用于获取表结构和记录数）
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
		return fmt.Errorf("创建Reader失败: %v", err)
	}
	defer reader.Close()

	// 创建Writer
	writer, err := NewMySQLWriter(
		task.TargetConn.Host,
		task.TargetConn.Port,
		task.TargetConn.Username,
		targetPassword,
		targetDB,
		targetTable,
	)
	if err != nil {
		return fmt.Errorf("创建Writer失败: %v", err)
	}
	defer writer.Close()

	// 创建表结构（包含外键）
	if err := writer.CreateTableLike(reader.GetDB(), sourceTable); err != nil {
		return fmt.Errorf("创建表结构失败: %v", err)
	}

	// 更新总记录数到内存（为后续同步做准备）
	totalRecords := reader.GetTotalCount()
	progressManager.UpdateUnitProgress(taskID, unitName, totalRecords, 0)
	progressManager.UpdateUnitStatus(taskID, unitName, "initialized")

	return nil
}
