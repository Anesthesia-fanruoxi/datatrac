package services

import (
	"fmt"
)

// processEvent 处理单个事件
func (c *IncrementalConsumer) processEvent(event *BinlogEvent) error {
	switch event.Type {
	case "INSERT":
		return c.applyInsert(event)
	case "UPDATE":
		return c.applyUpdate(event)
	case "DELETE":
		return c.applyDelete(event)
	default:
		return fmt.Errorf("未知事件类型: %s", event.Type)
	}
}

// applyInsert 应用 INSERT 事件
func (c *IncrementalConsumer) applyInsert(event *BinlogEvent) error {
	// 映射数据库和表名
	targetDB, targetTable, skip := c.mapDatabaseAndTable(event.Database, event.Table)
	if skip {
		// 不在同步范围内，跳过
		return nil
	}

	// 验证数据是否为空
	if len(event.Data) == 0 {
		c.logService.Warning(c.taskID, fmt.Sprintf(
			"INSERT 事件数据为空，跳过: %s.%s -> %s.%s",
			event.Database, event.Table,
			targetDB, targetTable,
		))
		return nil
	}

	// 构建 INSERT 语句
	columns := make([]string, 0, len(event.Data))
	placeholders := make([]string, 0, len(event.Data))
	values := make([]interface{}, 0, len(event.Data))
	updateClauses := make([]string, 0, len(event.Data))

	for col, val := range event.Data {
		columns = append(columns, fmt.Sprintf("`%s`", col))
		placeholders = append(placeholders, "?")
		values = append(values, val)
		// 构建 ON DUPLICATE KEY UPDATE 子句
		updateClauses = append(updateClauses, fmt.Sprintf("`%s`=VALUES(`%s`)", col, col))
	}

	// 使用 INSERT ... ON DUPLICATE KEY UPDATE 处理主键冲突
	query := fmt.Sprintf(
		"INSERT INTO `%s`.`%s` (%s) VALUES (%s) ON DUPLICATE KEY UPDATE %s",
		targetDB,
		targetTable,
		joinStrings(columns, ", "),
		joinStrings(placeholders, ", "),
		joinStrings(updateClauses, ", "),
	)

	// 执行插入
	_, err := c.targetDB.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("执行 INSERT 失败: %v, SQL: %s", err, query)
	}

	return nil
}

// applyUpdate 应用 UPDATE 事件
func (c *IncrementalConsumer) applyUpdate(event *BinlogEvent) error {
	// 映射数据库和表名
	targetDB, targetTable, skip := c.mapDatabaseAndTable(event.Database, event.Table)
	if skip {
		// 不在同步范围内，跳过
		return nil
	}

	// 验证数据是否为空
	if len(event.Data) == 0 {
		c.logService.Warning(c.taskID, fmt.Sprintf(
			"UPDATE 事件新数据为空，跳过: %s.%s -> %s.%s",
			event.Database, event.Table,
			targetDB, targetTable,
		))
		return nil
	}

	// 构建 UPDATE 语句
	setClauses := make([]string, 0, len(event.Data))
	setValues := make([]interface{}, 0, len(event.Data))

	for col, val := range event.Data {
		setClauses = append(setClauses, fmt.Sprintf("`%s` = ?", col))
		setValues = append(setValues, val)
	}

	// 构建 WHERE 条件（使用旧值）
	whereClauses := make([]string, 0, len(event.OldData))
	whereValues := make([]interface{}, 0, len(event.OldData))

	for col, val := range event.OldData {
		if val == nil {
			whereClauses = append(whereClauses, fmt.Sprintf("`%s` IS NULL", col))
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("`%s` = ?", col))
			whereValues = append(whereValues, val)
		}
	}

	// 如果没有 WHERE 条件，记录警告但仍然执行（使用新数据作为条件）
	if len(whereClauses) == 0 {
		c.logService.Warning(c.taskID, fmt.Sprintf(
			"UPDATE 事件缺少旧数据，使用新数据作为 WHERE 条件: %s.%s",
			targetDB, targetTable,
		))
		for col, val := range event.Data {
			if val == nil {
				whereClauses = append(whereClauses, fmt.Sprintf("`%s` IS NULL", col))
			} else {
				whereClauses = append(whereClauses, fmt.Sprintf("`%s` = ?", col))
				whereValues = append(whereValues, val)
			}
		}
	}

	query := fmt.Sprintf(
		"UPDATE `%s`.`%s` SET %s WHERE %s",
		targetDB,
		targetTable,
		joinStrings(setClauses, ", "),
		joinStrings(whereClauses, " AND "),
	)

	// 合并参数
	allValues := append(setValues, whereValues...)

	// 执行更新
	result, err := c.targetDB.Exec(query, allValues...)
	if err != nil {
		return fmt.Errorf("执行 UPDATE 失败: %v, SQL: %s", err, query)
	}

	// 检查是否更新了行（只在调试模式记录）
	affected, _ := result.RowsAffected()
	if affected == 0 {
		// 降低日志级别，这种情况在增量同步中很常见
		// c.logService.Warning(c.taskID, fmt.Sprintf("UPDATE 未影响任何行: %s.%s", targetDB, targetTable))
	}

	return nil
}

// applyDelete 应用 DELETE 事件
func (c *IncrementalConsumer) applyDelete(event *BinlogEvent) error {
	// 映射数据库和表名
	targetDB, targetTable, skip := c.mapDatabaseAndTable(event.Database, event.Table)
	if skip {
		// 不在同步范围内，跳过
		return nil
	}

	// 验证数据是否为空
	if len(event.Data) == 0 {
		c.logService.Warning(c.taskID, fmt.Sprintf(
			"DELETE 事件数据为空，跳过: %s.%s -> %s.%s",
			event.Database, event.Table,
			targetDB, targetTable,
		))
		return nil
	}

	// 构建 WHERE 条件
	whereClauses := make([]string, 0, len(event.Data))
	values := make([]interface{}, 0, len(event.Data))

	for col, val := range event.Data {
		if val == nil {
			whereClauses = append(whereClauses, fmt.Sprintf("`%s` IS NULL", col))
		} else {
			whereClauses = append(whereClauses, fmt.Sprintf("`%s` = ?", col))
			values = append(values, val)
		}
	}

	query := fmt.Sprintf(
		"DELETE FROM `%s`.`%s` WHERE %s",
		targetDB,
		targetTable,
		joinStrings(whereClauses, " AND "),
	)

	// 执行删除
	result, err := c.targetDB.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("执行 DELETE 失败: %v, SQL: %s", err, query)
	}

	// 检查是否删除了行（只在调试模式记录）
	affected, _ := result.RowsAffected()
	if affected == 0 {
		// 降低日志级别
		// c.logService.Warning(c.taskID, fmt.Sprintf("DELETE 未影响任何行: %s.%s", targetDB, targetTable))
	}

	return nil
}
