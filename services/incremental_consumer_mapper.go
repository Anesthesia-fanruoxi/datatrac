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
