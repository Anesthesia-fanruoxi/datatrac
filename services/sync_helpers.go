package services

import (
	"database/sql"
	"datatrace/models"
	"fmt"
	"strings"
)

// failUnit 标记单元失败（内存操作）
func (e *SyncEngine) failUnit(taskID, unitName, errMsg string) error {
	progressManager := GetProgressManager()
	progressManager.UpdateUnitError(taskID, unitName, errMsg)
	e.sseService.BroadcastProgressUpdate(taskID)
	return fmt.Errorf(errMsg)
}

// pauseUnit 标记单元暂停（内存操作）
func (e *SyncEngine) pauseUnit(taskID, unitName, msg string) error {
	progressManager := GetProgressManager()
	progressManager.UpdateUnitStatus(taskID, unitName, "paused")
	e.sseService.BroadcastProgressUpdate(taskID)
	return fmt.Errorf(msg)
}

// parseUnitName 解析单元名称（格式：database.table）
func (e *SyncEngine) parseUnitName(unitName string, config *TaskConfig) (sourceDB, sourceTable, targetDB, targetTable string, err error) {
	parts := strings.Split(unitName, ".")
	if len(parts) != 2 {
		err = fmt.Errorf("无效的单元名称格式: %s", unitName)
		return
	}

	targetDB = parts[0]
	targetTable = parts[1]

	// 从配置中查找源数据库和源表
	for _, dbSel := range config.SelectedDatabases {
		if dbSel.Database == targetDB {
			sourceDB = dbSel.SourceDatabase
			for _, tbl := range dbSel.Tables {
				if tbl.TargetTable == targetTable {
					sourceTable = tbl.SourceTable
					return
				}
			}
		}
	}

	err = fmt.Errorf("未找到表 %s 的配置", unitName)
	return
}

// safePercent 安全计算百分比
func safePercent(processed, total int64) float64 {
	if total == 0 {
		return 0
	}
	return float64(processed) / float64(total) * 100
}

// splitTableName 分割表名（格式：database.table）
func splitTableName(unitName string) []string {
	return strings.Split(unitName, ".")
}

// getSelectedFields 获取表的选中字段列表
func (e *SyncEngine) getSelectedFields(config *TaskConfig, sourceDB, sourceTable string) []string {
	if config == nil {
		return nil
	}

	// 遍历配置，查找匹配的数据库和表
	for _, dbSel := range config.SelectedDatabases {
		if dbSel.SourceDatabase == sourceDB {
			for _, tbl := range dbSel.Tables {
				if tbl.SourceTable == sourceTable {
					return tbl.SelectedFields
				}
			}
		}
	}

	return nil
}

// calculateAdaptiveBatchSize 计算自适应批次大小
func (e *SyncEngine) calculateAdaptiveBatchSize(sourceConn *models.DataSource, database, table, password string) int {
	calculator := NewAdaptiveConfigCalculator()

	// 尝试连接数据库获取表统计信息
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		sourceConn.Username, password, sourceConn.Host, sourceConn.Port, database)

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		// 连接失败，使用默认配置
		e.logService.Warning("", fmt.Sprintf("无法连接数据库获取统计信息，使用默认批次大小: %v", err))
		return calculator.GetDefaultConfig().BatchSize
	}
	defer db.Close()

	// 获取表统计信息
	stats, err := calculator.GetTableStats(db, database, table)
	if err != nil {
		// 获取统计信息失败，使用默认配置
		e.logService.Warning("", fmt.Sprintf("无法获取表统计信息，使用默认批次大小: %v", err))
		return calculator.GetDefaultConfig().BatchSize
	}

	// 计算批次大小
	batchSize := calculator.CalculateForTable(stats)

	return batchSize
}
