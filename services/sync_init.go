package services

import (
	"context"
	"database/sql"
	"datatrace/database"
	"datatrace/models"
	"fmt"
	"strings"
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

	// 初始化阶段也推送 SSE，便于前端展示进度并创建 sse 日志文件
	sseService := NewTaskSSEService()
	sseService.BroadcastProgressUpdate(taskID)

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

	// 获取多目标源ID列表
	targetIDs := config.TargetIDs
	if len(targetIDs) == 0 {
		// 兼容旧配置
		targetIDs = []string{config.TargetID}
	}

	// 5. 创建所有数据库（为每个目标源创建）
	for _, targetID := range targetIDs {
		// 查询目标源
		var targetDS models.DataSource
		if err := database.DB.First(&targetDS, "id = ?", targetID).Error; err != nil {
			e.logService.Error(taskID, fmt.Sprintf("查询目标源 %s 失败: %v", targetID, err))
			continue
		}

		targetPwd, err := e.dsService.crypto.Decrypt(targetDS.Password)
		if err != nil {
			e.logService.Error(taskID, fmt.Sprintf("解密目标源 %s 密码失败: %v", targetID, err))
			continue
		}

		e.logService.Info(taskID, fmt.Sprintf("初始化目标源: %s", targetDS.Name))

		// 为该目标源创建所有数据库
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
				targetDS.Host,
				targetDS.Port,
				targetDS.Username,
				targetPwd,
				group.targetDB,
				charset,
				collation,
			)

			if err != nil {
				e.logService.Error(taskID, fmt.Sprintf("目标 %s 创建数据库 %s 失败: %v", targetDS.Name, group.targetDB, err))
				return fmt.Errorf("创建数据库失败: %v", err)
			}

			if created {
				e.logService.Info(taskID, fmt.Sprintf("目标 %s 创建数据库: %s (字符集: %s, 排序规则: %s)", targetDS.Name, group.targetDB, charset, collation))
			} else {
				e.logService.Info(taskID, fmt.Sprintf("目标 %s 数据库已存在: %s", targetDS.Name, group.targetDB))
			}
		}
	}

	// 6. 初始化阶段：获取源表近似行数
	e.logService.Info(taskID, "获取源表数据量...")
	if err := e.fetchSourceTableRows(ctx, taskID, &task, config, sourcePassword); err != nil {
		e.logService.Warning(taskID, fmt.Sprintf("获取源表数据量失败: %v，将使用实际同步量统计", err))
	}

	// 7. 按顺序初始化所有表
	e.logService.Info(taskID, fmt.Sprintf("开始初始化 %d 个表的结构", len(unitNames)))

	strategy := config.SyncConfig.TableExistsStrategy

	// 如果是 drop 策略，需要分两个阶段：先删除所有表，再创建所有表
	if strategy == "drop" {
		// 阶段1：按删除顺序删除所有表（子表在前，父表在后）
		e.logService.Info(taskID, "阶段1: 删除所有表")

		for _, unitName := range unitNames {
			if err := e.dropTable(ctx, taskID, unitName, &task, config, targetIDs); err != nil {
				e.logService.Error(taskID, fmt.Sprintf("删除表 %s 失败: %v", unitName, err))
				return fmt.Errorf("删除表失败: %v", err)
			}
		}

		// 阶段2：按创建顺序创建所有表（父表在前，子表在后，需要反转）
		e.logService.Info(taskID, "阶段2: 创建所有表")

		for i := len(unitNames) - 1; i >= 0; i-- {
			unitName := unitNames[i]
			if err := e.createTable(ctx, taskID, unitName, &task, config, sourcePassword, targetIDs); err != nil {
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
// 支持多目标源和ALTER对比
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

	// 获取字段配置
	selectedFields := e.getSelectedFields(config, sourceDB, sourceTable)

	// 获取多目标源ID列表
	targetIDs := config.TargetIDs
	if len(targetIDs) == 0 {
		// 兼容旧配置
		targetIDs = []string{config.TargetID}
	}

	// 3. 对每个目标源进行初始化
	for targetIdx, targetID := range targetIDs {
		targetIdxStr := fmt.Sprintf("%d", targetIdx+1)

		// 查询目标源
		var targetDS models.DataSource
		if err := database.DB.First(&targetDS, "id = ?", targetID).Error; err != nil {
			return fmt.Errorf("查询目标源 %s 失败: %w", targetID, err)
		}

		targetPwd, err := e.dsService.crypto.Decrypt(targetDS.Password)
		if err != nil {
			return fmt.Errorf("解密目标源 %s 密码失败: %w", targetID, err)
		}

		e.logService.Info(taskID, fmt.Sprintf("初始化目标 %s/%s: %s", targetIdxStr, len(targetIDs), targetDS.Name))

		// 创建Writer
		writer, err := NewMySQLWriter(
			targetDS.Host,
			targetDS.Port,
			targetDS.Username,
			targetPwd,
			targetDB,
			targetTable,
		)
		if err != nil {
			return fmt.Errorf("目标 %s 创建Writer失败: %w", targetDS.Name, err)
		}
		defer writer.Close()

		// 4. 处理表存在策略
		strategy := config.SyncConfig.TableExistsStrategy

		// 检查是否是"只同步表结构"模式
		if config.SyncConfig.SyncStructureOnly {
			// 使用ALTER对比逻辑
			alterService := NewTableStructureAlterService()
			alterResult, err := alterService.CompareAndAlter(
				reader.GetDB(),
				writer.GetDB(),
				sourceTable,
				targetTable,
				selectedFields,
			)
			if err != nil {
				return fmt.Errorf("目标 %s 表结构对比失败: %w", targetDS.Name, err)
			}

			switch alterResult.Action {
			case "create":
				e.logService.Info(taskID, fmt.Sprintf("目标 %s: 表 %s 不存在，创建表结构", targetDS.Name, unitName))
				if err := writer.CreateTableLikeWithFields(reader.GetDB(), sourceTable, selectedFields); err != nil {
					return fmt.Errorf("目标 %s 创建表结构失败: %w", targetDS.Name, err)
				}

			case "update":
				e.logService.Info(taskID, fmt.Sprintf("目标 %s: 表 %s 存在，对比结构差异", targetDS.Name, unitName))
				if len(alterResult.SQLs) > 0 {
					// 执行ALTER
					alterSQL := "ALTER TABLE `" + targetTable + "` " + strings.Join(alterResult.SQLs, ", ")
					e.logService.Info(taskID, fmt.Sprintf("目标 %s: 执行ALTER: %s", targetDS.Name, alterSQL))
					if _, err := writer.GetDB().Exec(alterSQL); err != nil {
						return fmt.Errorf("目标 %s 执行ALTER失败: %w", targetDS.Name, err)
					}
					e.logService.Info(taskID, fmt.Sprintf("目标 %s: ALTER完成", targetDS.Name))
				} else {
					e.logService.Info(taskID, fmt.Sprintf("目标 %s: 表结构无需修改", targetDS.Name))
				}

			case "skip":
				e.logService.Info(taskID, fmt.Sprintf("目标 %s: 表 %s 结构一致，跳过", targetDS.Name, unitName))
			}
		} else {
			// 原有逻辑：全量/增量同步
			switch strategy {
			case "drop":
				// 删除表
				if err := writer.DropTable(); err != nil {
					return fmt.Errorf("目标 %s 删除表失败: %w", targetDS.Name, err)
				}

				// 创建表结构（支持字段过滤）
				if err := writer.CreateTableLikeWithFields(reader.GetDB(), sourceTable, selectedFields); err != nil {
					return fmt.Errorf("目标 %s 创建表结构失败: %w", targetDS.Name, err)
				}

			case "truncate":
				// 清空表
				if err := writer.TruncateTable(); err != nil {
					return fmt.Errorf("目标 %s 清空表失败: %w", targetDS.Name, err)
				}

			case "append":
				// 使用ALTER对比逻辑
				alterService := NewTableStructureAlterService()
				alterResult, err := alterService.CompareAndAlter(
					reader.GetDB(),
					writer.GetDB(),
					sourceTable,
					targetTable,
					selectedFields,
				)
				if err != nil {
					return fmt.Errorf("目标 %s 表结构对比失败: %w", targetDS.Name, err)
				}

				switch alterResult.Action {
				case "create":
					e.logService.Info(taskID, fmt.Sprintf("目标 %s: 表 %s 不存在，创建表结构", targetDS.Name, unitName))
					if err := writer.CreateTableLikeWithFields(reader.GetDB(), sourceTable, selectedFields); err != nil {
						return fmt.Errorf("目标 %s 创建表结构失败: %w", targetDS.Name, err)
					}

				case "update":
					e.logService.Info(taskID, fmt.Sprintf("目标 %s: 表 %s 存在，对比结构差异", targetDS.Name, unitName))
					if len(alterResult.SQLs) > 0 {
						alterSQL := "ALTER TABLE `" + targetTable + "` " + strings.Join(alterResult.SQLs, ", ")
						e.logService.Info(taskID, fmt.Sprintf("目标 %s: 执行ALTER: %s", targetDS.Name, alterSQL))
						if _, err := writer.GetDB().Exec(alterSQL); err != nil {
							return fmt.Errorf("目标 %s 执行ALTER失败: %w", targetDS.Name, err)
						}
					}

				case "skip":
					e.logService.Info(taskID, fmt.Sprintf("目标 %s: 表 %s 结构一致，跳过", targetDS.Name, unitName))
				}
			}
		}

		e.logService.Info(taskID, fmt.Sprintf("目标 %s 初始化完成: %s", targetDS.Name, unitName))
	}

	// 5. 更新总记录数到内存（为后续同步做准备）
	// 只有在非"只同步表结构"模式下才更新记录数
	if !config.SyncConfig.SyncStructureOnly {
		totalRecords := reader.GetTotalCount()
		progressManager.UpdateUnitProgress(taskID, unitName, totalRecords, 0)
	}
	progressManager.UpdateUnitStatus(taskID, unitName, "initialized")

	// 同时更新所有目标端的状态
	for _, targetID := range targetIDs {
		progressManager.UpdateTargetUnitStatus(taskID, targetID, unitName, "initialized")
	}

	// 每张表初始化后推送 SSE，便于前端与 sse 日志看到初始化进度
	NewTaskSSEService().BroadcastProgressUpdate(taskID)

	return nil
}

// dropTable 只删除表（用于 drop 策略的第一阶段）- 支持多目标源
func (e *SyncEngine) dropTable(ctx context.Context, taskID string, unitName string,
	task *models.SyncTask, config *TaskConfig, targetIDs []string) error {

	// 解析表名
	_, _, targetDB, targetTable, err := e.parseUnitName(unitName, config)
	if err != nil {
		return fmt.Errorf("解析表名失败: %v", err)
	}

	// 为每个目标源删除表
	for _, targetID := range targetIDs {
		// 查询目标源
		var targetDS models.DataSource
		if err := database.DB.First(&targetDS, "id = ?", targetID).Error; err != nil {
			return fmt.Errorf("查询目标源 %s 失败: %w", targetID, err)
		}

		targetPwd, err := e.dsService.crypto.Decrypt(targetDS.Password)
		if err != nil {
			return fmt.Errorf("解密目标源 %s 密码失败: %w", targetID, err)
		}

		// 创建Writer
		writer, err := NewMySQLWriter(
			targetDS.Host,
			targetDS.Port,
			targetDS.Username,
			targetPwd,
			targetDB,
			targetTable,
		)
		if err != nil {
			return fmt.Errorf("目标 %s 创建Writer失败: %w", targetDS.Name, err)
		}

		// 删除表
		if err := writer.DropTable(); err != nil {
			writer.Close()
			return fmt.Errorf("目标 %s 删除表失败: %w", targetDS.Name, err)
		}
		writer.Close()

		e.logService.Info(taskID, fmt.Sprintf("目标 %s 删除表: %s.%s", targetDS.Name, targetDB, targetTable))
	}

	return nil
}

// createTable 只创建表（用于 drop 策略的第二阶段）- 支持多目标源
func (e *SyncEngine) createTable(ctx context.Context, taskID string, unitName string,
	task *models.SyncTask, config *TaskConfig, sourcePassword string, targetIDs []string) error {

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

	// 获取字段配置
	selectedFields := e.getSelectedFields(config, sourceDB, sourceTable)

	// 为每个目标源创建表
	for _, targetID := range targetIDs {
		// 查询目标源
		var targetDS models.DataSource
		if err := database.DB.First(&targetDS, "id = ?", targetID).Error; err != nil {
			return fmt.Errorf("查询目标源 %s 失败: %w", targetID, err)
		}

		targetPwd, err := e.dsService.crypto.Decrypt(targetDS.Password)
		if err != nil {
			return fmt.Errorf("解密目标源 %s 密码失败: %w", targetID, err)
		}

		// 创建Writer
		writer, err := NewMySQLWriter(
			targetDS.Host,
			targetDS.Port,
			targetDS.Username,
			targetPwd,
			targetDB,
			targetTable,
		)
		if err != nil {
			return fmt.Errorf("目标 %s 创建Writer失败: %w", targetDS.Name, err)
		}

		// 创建表结构（支持字段过滤）
		if err := writer.CreateTableLikeWithFields(reader.GetDB(), sourceTable, selectedFields); err != nil {
			writer.Close()
			return fmt.Errorf("目标 %s 创建表结构失败: %w", targetDS.Name, err)
		}
		writer.Close()

		e.logService.Info(taskID, fmt.Sprintf("目标 %s 创建表: %s.%s", targetDS.Name, targetDB, targetTable))
	}

	// 更新总记录数到内存（为后续同步做准备）- 只更新一次
	totalRecords := reader.GetTotalCount()
	progressManager.UpdateUnitProgress(taskID, unitName, totalRecords, 0)
	progressManager.UpdateUnitStatus(taskID, unitName, "initialized")

	// 同时更新所有目标端的状态
	for _, targetID := range targetIDs {
		progressManager.UpdateTargetUnitStatus(taskID, targetID, unitName, "initialized")
	}

	// 每张表初始化后推送 SSE（drop 策略创建表阶段）
	NewTaskSSEService().BroadcastProgressUpdate(taskID)

	return nil
}

// fetchSourceTableRows 获取源表近似行数（从 SHOW TABLE STATUS）
func (e *SyncEngine) fetchSourceTableRows(ctx context.Context, taskID string, task *models.SyncTask, config *TaskConfig, sourcePassword string) error {
	progressManager := GetProgressManager()

	// 构建 targetDB.targetTable -> (sourceDB, sourceTable) 的映射
	tableMapping := make(map[string][2]string) // key: targetDB.targetTable, value: [0]=sourceDB, [1]=sourceTable

	for _, db := range config.SelectedDatabases {
		sourceDB := db.SourceDatabase
		if sourceDB == "" {
			sourceDB = db.Database
		}
		for _, tbl := range db.Tables {
			key := db.Database + "." + tbl.TargetTable
			tableMapping[key] = [2]string{sourceDB, tbl.SourceTable}
		}
	}

	// 按源数据库分组
	sourceDBs := make(map[string][]string) // sourceDB -> list of targetTable

	for key, mapping := range tableMapping {
		sourceDB := mapping[0]
		sourceDBs[sourceDB] = append(sourceDBs[sourceDB], key)
	}

	// 连接每个源数据库并获取表状态
	for dbName, tables := range sourceDBs {
		select {
		case <-ctx.Done():
			return fmt.Errorf("获取源表数据量被取消")
		default:
		}

		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=30s",
			task.SourceConn.Username, sourcePassword, task.SourceConn.Host, task.SourceConn.Port, dbName)

		db, err := sql.Open("mysql", dsn)
		if err != nil {
			return fmt.Errorf("连接源数据库 %s 失败: %w", dbName, err)
		}
		defer db.Close()

		// 执行 SHOW TABLE STATUS
		rows, err := db.Query("SHOW TABLE STATUS")
		if err != nil {
			return fmt.Errorf("获取表状态失败: %w", err)
		}
		defer rows.Close()

		// 创建需要过滤的表集合
		neededTables := make(map[string]bool)
		for _, t := range tables {
			mapping := tableMapping[t]
			neededTables[mapping[1]] = true // 使用源表名
		}

		for rows.Next() {
			var name sql.NullString
			var rowsCount sql.NullInt64
			// 只关心 Name 和 Rows 字段
			err := rows.Scan(&name, &rowsCount)
			if err != nil {
				continue
			}
			if name.Valid {
				tableName := name.String
				// 只处理配置的表
				if !neededTables[tableName] {
					continue
				}

				approxRows := int64(0)
				if rowsCount.Valid {
					approxRows = rowsCount.Int64
				}

				// 找到对应的 targetDB.targetTable
				for targetKey, mapping := range tableMapping {
					if mapping[1] == tableName && mapping[0] == dbName {
						// 存储到进度管理器（使用 targetDB.targetTable 格式）
						progressManager.SetApproxRows(taskID, targetKey, approxRows)
						break
					}
				}
			}
		}
		rows.Close()
	}

	return nil
}
