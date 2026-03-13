package services

// mapDatabaseAndTable 映射源数据库和表名到目标数据库和表名
// 返回: targetDB, targetTable, skip
// skip=true 表示该表不在同步范围内，应该跳过
func (c *IncrementalConsumer) mapDatabaseAndTable(sourceDB, sourceTable string) (string, string, bool) {
	if c.config == nil {
		// 没有配置，直接使用原名
		return sourceDB, sourceTable, false
	}

	// 遍历配置，查找匹配的数据库和表
	for _, dbSel := range c.config.SelectedDatabases {
		// 检查源数据库是否匹配
		if dbSel.SourceDatabase != sourceDB {
			continue
		}

		// 找到匹配的数据库，获取目标数据库名
		targetDB := dbSel.Database

		// 查找匹配的表
		for _, tbl := range dbSel.Tables {
			if tbl.SourceTable == sourceTable {
				// 找到匹配的表
				targetTable := tbl.TargetTable
				if targetTable == "" {
					targetTable = sourceTable // 如果没有指定目标表名，使用源表名
				}
				return targetDB, targetTable, false
			}
		}

		// 数据库匹配但表不匹配，跳过
		return "", "", true
	}

	// 数据库不在同步范围内，跳过
	return "", "", true
}

// GetStatistics 获取统计信息
func (c *IncrementalConsumer) GetStatistics() map[string]interface{} {
	return map[string]interface{}{
		"events_processed": c.eventsProcessed,
		"events_failed":    c.eventsFailed,
		"last_save_time":   c.lastSaveTime,
	}
}

// joinStrings 连接字符串数组
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}

	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}

	return result
}

// getSelectedFields 获取表的选中字段列表
func (c *IncrementalConsumer) getSelectedFields(sourceDB, sourceTable string) []string {
	if c.config == nil {
		return nil
	}

	// 遍历配置，查找匹配的数据库和表
	for _, dbSel := range c.config.SelectedDatabases {
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

// filterEventFields 根据配置过滤事件中的字段
func (c *IncrementalConsumer) filterEventFields(event *BinlogEvent) {
	selectedFields := c.getSelectedFields(event.Database, event.Table)

	// 如果没有配置字段或字段列表为空，不过滤
	if len(selectedFields) == 0 {
		return
	}

	// 创建字段集合用于快速查找
	fieldSet := make(map[string]bool)
	for _, field := range selectedFields {
		fieldSet[field] = true
	}

	// 过滤 Data 字段
	if event.Data != nil {
		filteredData := make(map[string]interface{})
		for col, val := range event.Data {
			if fieldSet[col] {
				filteredData[col] = val
			}
		}
		event.Data = filteredData
	}

	// 过滤 OldData 字段
	if event.OldData != nil {
		filteredOldData := make(map[string]interface{})
		for col, val := range event.OldData {
			if fieldSet[col] {
				filteredOldData[col] = val
			}
		}
		event.OldData = filteredOldData
	}
}
