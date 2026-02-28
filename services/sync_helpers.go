package services

import (
	"database/sql"
	"fmt"
	"time"

	"datatrace/database"
	"datatrace/models"
)

// safePercent 安全计算百分比，避免除零
func safePercent(processed, total int64) float64 {
	if total <= 0 {
		return 0
	}
	return float64(processed) / float64(total) * 100
}

// splitTableName 分割表名
func splitTableName(fullName string) []string {
	result := []string{}
	parts := []rune(fullName)
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

// handleTableExistsStrategy 处理表存在策略
func (e *SyncEngine) handleTableExistsStrategy(writer *MySQLWriter, sourceDB *sql.DB, sourceTable, strategy, taskID string) error {
	switch strategy {
	case "drop":
		if err := writer.DropTable(); err != nil {
			return err
		}
		if err := writer.CreateTableLike(sourceDB, sourceTable); err != nil {
			return err
		}
		// 记录创建表的日志
		e.logService.AddLog(taskID, "info", fmt.Sprintf("创建表: %s", writer.tableName), "create")
		// 发送创建表的日志
		createLog := TaskLog{
			Time:     formatLogTime(time.Now()),
			Level:    "info",
			Message:  fmt.Sprintf("创建表: %s", writer.tableName),
			Category: "create",
		}
		e.sseService.BroadcastLogUpdate(taskID, []TaskLog{createLog})
		return nil
	case "truncate":
		return writer.TruncateTable()
	case "append":
		return nil
	default:
		return fmt.Errorf("未知的表存在策略: %s", strategy)
	}
}

// failUnit 标记单元失败
func (e *SyncEngine) failUnit(unit *models.TaskUnitRuntime, errMsg string) error {
	unit.Status = "failed"
	unit.ErrorMessage = errMsg
	database.DB.Save(unit)
	return fmt.Errorf(errMsg)
}

// pauseUnit 标记单元暂停
func (e *SyncEngine) pauseUnit(unit *models.TaskUnitRuntime, msg string) error {
	unit.Status = "paused"
	database.DB.Save(unit)
	return fmt.Errorf(msg)
}

// parseUnitName 解析单元名称
func (e *SyncEngine) parseUnitName(unitName string, config *TaskConfig) (sourceDB, sourceTable, targetDB, targetTable string, err error) {
	parts := splitTableName(unitName)
	if len(parts) != 2 {
		return "", "", "", "", fmt.Errorf("无效的表名格式: %s", unitName)
	}

	targetDB = parts[0]
	targetTable = parts[1]

	for _, dbSel := range config.SelectedDatabases {
		if dbSel.Database == targetDB {
			sourceDB = dbSel.SourceDatabase
			for _, tbl := range dbSel.Tables {
				if tbl.TargetTable == targetTable {
					sourceTable = tbl.SourceTable
					return sourceDB, sourceTable, targetDB, targetTable, nil
				}
			}
			return "", "", "", "", fmt.Errorf("未找到表映射: %s (数据库 %s 中没有表 %s)", unitName, targetDB, targetTable)
		}
	}

	return "", "", "", "", fmt.Errorf("未找到数据库映射: %s (目标数据库 %s 不在配置中)", unitName, targetDB)
}
