package services

import (
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
