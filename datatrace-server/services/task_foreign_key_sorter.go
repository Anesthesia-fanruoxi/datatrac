package services

import (
	"database/sql"
	"datatrace/database"
	"datatrace/models"
	"datatrace/utils"
	"fmt"
)

// TaskForeignKeySorter 外键排序服务
type TaskForeignKeySorter struct {
	logService *TaskLogService
}

// NewTaskForeignKeySorter 创建外键排序服务
func NewTaskForeignKeySorter() *TaskForeignKeySorter {
	return &TaskForeignKeySorter{
		logService: NewTaskLogService(),
	}
}

// SortUnitsByForeignKeys 按照外键依赖关系对任务单元进行排序
// 返回: 排序后的单元名称列表, 是否有外键依赖, 外键表集合(unitName -> bool), 错误
func (s *TaskForeignKeySorter) SortUnitsByForeignKeys(task *models.SyncTask, unitNames []string, config *TaskConfig) ([]string, bool, map[string]bool, error) {
	fkTableSet := make(map[string]bool) // 记录哪些表有外键依赖

	// 如果只有一个表,直接返回
	if len(unitNames) <= 1 {
		return unitNames, false, fkTableSet, nil
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
	dbGroups := make(map[string][]string)
	for _, unitName := range unitNames {
		parts := splitTableName(unitName)
		if len(parts) >= 1 {
			targetDBName := parts[0]
			dbGroups[targetDBName] = append(dbGroups[targetDBName], unitName)
		}
	}

	s.logService.Info(task.ID, fmt.Sprintf("共 %d 个数据库需要分析外键关系", len(dbGroups)))

	// 对每个数据库的表进行排序
	var sortedUnits []string
	hasForeignKeys := false

	for targetDBName, dbUnitNames := range dbGroups {
		s.logService.Info(task.ID, fmt.Sprintf("正在分析数据库 %s 的 %d 个表...", targetDBName, len(dbUnitNames)))

		// 从配置中找到源数据库名
		var sourceDBName string
		for _, dbSel := range config.SelectedDatabases {
			if dbSel.Database == targetDBName {
				sourceDBName = dbSel.SourceDatabase
				break
			}
		}

		if sourceDBName == "" {
			s.logService.Warning(task.ID, fmt.Sprintf("未找到目标数据库 %s 对应的源数据库, 使用原顺序", targetDBName))
			sortedUnits = append(sortedUnits, dbUnitNames...)
			continue
		}

		// 连接到源数据库
		dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s",
			task.SourceConn.Username, sourcePassword, task.SourceConn.Host, task.SourceConn.Port, sourceDBName)

		db, err := sql.Open("mysql", dsn)
		if err != nil {
			s.logService.Warning(task.ID, fmt.Sprintf("连接源数据库 %s 失败: %v, 使用原顺序", sourceDBName, err))
			sortedUnits = append(sortedUnits, dbUnitNames...)
			continue
		}

		// 测试连接
		if err := db.Ping(); err != nil {
			s.logService.Warning(task.ID, fmt.Sprintf("源数据库 %s Ping 失败: %v, 使用原顺序", sourceDBName, err))
			db.Close()
			sortedUnits = append(sortedUnits, dbUnitNames...)
			continue
		}

		// 提取源表名列表（需要从配置中查找映射）
		sourceTableNames := make([]string, 0, len(dbUnitNames))
		unitMap := make(map[string]string) // sourceTable -> unitName

		for _, unitName := range dbUnitNames {
			parts := splitTableName(unitName)
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
					unitMap[sourceTableName] = unitName
				}
			}
		}

		// 对源表进行排序,并获取外键表列表
		sortedSourceTableNames, dbHasForeignKeys, fkTables, err := SortTablesForDropWithFKList(db, sourceDBName, sourceTableNames)
		db.Close()

		if err != nil {
			s.logService.Warning(task.ID, fmt.Sprintf("源数据库 %s 排序失败: %v, 使用原顺序", sourceDBName, err))
			sortedUnits = append(sortedUnits, dbUnitNames...)
			continue
		}

		// 如果这个数据库有外键,标记整个任务有外键
		if dbHasForeignKeys {
			hasForeignKeys = true
			s.logService.Info(task.ID, fmt.Sprintf("源数据库 %s 检测到 %d 个外键表,已按依赖顺序排序", sourceDBName, len(fkTables)))

			// 记录哪些表有外键(使用完整的 unitName: 目标库.目标表)
			for sourceTableName := range fkTables {
				// 找到对应的 unitName
				if unitName, ok := unitMap[sourceTableName]; ok {
					fkTableSet[unitName] = true
				}
			}
		} else {
			s.logService.Info(task.ID, fmt.Sprintf("源数据库 %s 未检测到外键依赖", sourceDBName))
		}

		// 按照排序后的顺序添加到结果中
		for _, sourceTableName := range sortedSourceTableNames {
			if unitName, ok := unitMap[sourceTableName]; ok {
				sortedUnits = append(sortedUnits, unitName)
			}
		}
	}

	s.logService.Info(task.ID, "外键分析完成")
	return sortedUnits, hasForeignKeys, fkTableSet, nil
}
