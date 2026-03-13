package services

import (
	"context"
	"datatrace/database"
	"datatrace/models"
	"encoding/json"
	"fmt"
	"time"
)

// IncrementalTableStats 增量同步表统计
type IncrementalTableStats struct {
	Database              string    `json:"database"`                // 数据库名
	Table                 string    `json:"table"`                   // 表名
	InsertCount           int64     `json:"insert_count"`            // INSERT 数量
	UpdateCount           int64     `json:"update_count"`            // UPDATE 数量
	DeleteCount           int64     `json:"delete_count"`            // DELETE 数量
	TotalCount            int64     `json:"total_count"`             // 总事件数（历史增量）
	TodayCount            int64     `json:"today_count"`             // 今日处理数量
	LastEventTime         time.Time `json:"last_event_time"`         // 最新事件时间
	LastEventType         string    `json:"last_event_type"`         // 最新事件类型
	ReplicationLagSeconds int       `json:"replication_lag_seconds"` // 复制延迟(秒)

	// 全量同步信息
	FullSyncTotalRecords     int64   `json:"full_sync_total_records"`     // 全量同步总记录数
	FullSyncProcessedRecords int64   `json:"full_sync_processed_records"` // 全量同步已处理记录数
	FullSyncProgress         float64 `json:"full_sync_progress"`          // 全量同步进度百分比
}

// IncrementalDatabaseStats 增量同步数据库级别统计（合计）
type IncrementalDatabaseStats struct {
	Database                 string  `json:"database"`                    // 数据库名
	TotalInsertCount         int64   `json:"total_insert_count"`          // 总 INSERT 数量
	TotalUpdateCount         int64   `json:"total_update_count"`          // 总 UPDATE 数量
	TotalDeleteCount         int64   `json:"total_delete_count"`          // 总 DELETE 数量
	TotalCount               int64   `json:"total_count"`                 // 总事件数
	TodayCount               int64   `json:"today_count"`                 // 今日处理数量
	TableCount               int     `json:"table_count"`                 // 表数量
	FullSyncTotalRecords     int64   `json:"full_sync_total_records"`     // 全量同步总记录数
	FullSyncProcessedRecords int64   `json:"full_sync_processed_records"` // 全量同步已处理记录数
	FullSyncProgress         float64 `json:"full_sync_progress"`          // 全量同步进度百分比
}

// IncrementalTaskStats 增量同步任务统计
type IncrementalTaskStats struct {
	TaskID                string    `json:"task_id"`                 // 任务ID
	TotalEvents           int64     `json:"total_events"`            // 历史总事件数
	TodayEvents           int64     `json:"today_events"`            // 今日总事件数
	InsertCount           int64     `json:"insert_count"`            // 总 INSERT 数
	UpdateCount           int64     `json:"update_count"`            // 总 UPDATE 数
	DeleteCount           int64     `json:"delete_count"`            // 总 DELETE 数
	FailedCount           int64     `json:"failed_count"`            // 失败数
	CurrentBinlogFile     string    `json:"current_binlog_file"`     // 当前 Binlog 文件
	CurrentBinlogPos      uint32    `json:"current_binlog_pos"`      // 当前 Binlog 位置
	LastEventTime         time.Time `json:"last_event_time"`         // 最新事件时间
	ReplicationLagSeconds int       `json:"replication_lag_seconds"` // 复制延迟(秒)
	TodayResetAt          time.Time `json:"today_reset_at"`          // 今日计数重置时间
}

// IncrementalStatsService 增量同步统计服务
type IncrementalStatsService struct {
	ctx context.Context
}

// NewIncrementalStatsService 创建统计服务
func NewIncrementalStatsService() *IncrementalStatsService {
	return &IncrementalStatsService{
		ctx: context.Background(),
	}
}

// getTaskStatsKey 获取任务统计的 Redis key
func (s *IncrementalStatsService) getTaskStatsKey(taskID string) string {
	return fmt.Sprintf("incremental:task:%s:stats", taskID)
}

// getTableStatsKey 获取表统计的 Redis key
func (s *IncrementalStatsService) getTableStatsKey(taskID, database, table string) string {
	return fmt.Sprintf("incremental:task:%s:table:%s.%s", taskID, database, table)
}

// getTableListKey 获取表列表的 Redis key
func (s *IncrementalStatsService) getTableListKey(taskID string) string {
	return fmt.Sprintf("incremental:task:%s:tables", taskID)
}

// InitTaskStats 初始化任务统计
func (s *IncrementalStatsService) InitTaskStats(taskID string) error {
	if database.RedisClient == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}

	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	stats := &IncrementalTaskStats{
		TaskID:       taskID,
		TodayResetAt: todayStart,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("序列化统计数据失败: %v", err)
	}

	key := s.getTaskStatsKey(taskID)
	if err := database.RedisClient.Set(s.ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("保存到 Redis 失败: %v", err)
	}

	return nil
}

// InitTableStats 初始化表统计（任务启动时调用）
func (s *IncrementalStatsService) InitTableStats(taskID, dbName, tableName string) error {
	if database.RedisClient == nil {
		return fmt.Errorf("Redis 客户端未初始化")
	}

	key := s.getTableStatsKey(taskID, dbName, tableName)

	// 检查是否已存在
	exists, err := database.RedisClient.Exists(s.ctx, key).Result()
	if err == nil && exists > 0 {
		// 已存在，不覆盖
		return nil
	}

	// 创建初始统计
	stats := &IncrementalTableStats{
		Database:    dbName,
		Table:       tableName,
		InsertCount: 0,
		UpdateCount: 0,
		DeleteCount: 0,
		TotalCount:  0,
		TodayCount:  0,
	}

	data, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("序列化表统计失败: %v", err)
	}

	// 保存到 Redis，永久有效
	if err := database.RedisClient.Set(s.ctx, key, data, 0).Err(); err != nil {
		return fmt.Errorf("保存表统计到 Redis 失败: %v", err)
	}

	// 添加到表列表
	listKey := s.getTableListKey(taskID)
	tableKey := fmt.Sprintf("%s.%s", dbName, tableName)
	database.RedisClient.SAdd(s.ctx, listKey, tableKey)
	// 表列表永久有效
	database.RedisClient.Persist(s.ctx, listKey)

	return nil
}

// UpdateTableStats 更新表统计
func (s *IncrementalStatsService) UpdateTableStats(taskID, dbName, tableName, eventType string, eventTime time.Time) error {
	if database.RedisClient == nil {
		return nil // Redis 未配置时静默失败
	}

	key := s.getTableStatsKey(taskID, dbName, tableName)

	// 获取现有统计
	var stats IncrementalTableStats
	data, err := database.RedisClient.Get(s.ctx, key).Result()
	if err == nil {
		json.Unmarshal([]byte(data), &stats)
	} else {
		// 初始化
		stats.Database = dbName
		stats.Table = tableName
	}

	// 检查是否需要重置今日计数
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if stats.LastEventTime.Before(todayStart) {
		stats.TodayCount = 0
	}

	// 更新统计
	stats.TotalCount++
	stats.TodayCount++
	stats.LastEventTime = eventTime
	stats.LastEventType = eventType
	stats.ReplicationLagSeconds = int(now.Sub(eventTime).Seconds())
	if stats.ReplicationLagSeconds < 0 {
		stats.ReplicationLagSeconds = 0
	}

	switch eventType {
	case "INSERT":
		stats.InsertCount++
	case "UPDATE":
		stats.UpdateCount++
	case "DELETE":
		stats.DeleteCount++
	}

	// 保存统计
	newData, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	// 保存到 Redis，永久有效
	if err := database.RedisClient.Set(s.ctx, key, newData, 0).Err(); err != nil {
		return err
	}

	// 添加到表列表
	listKey := s.getTableListKey(taskID)
	tableKey := fmt.Sprintf("%s.%s", dbName, tableName)
	database.RedisClient.SAdd(s.ctx, listKey, tableKey)
	// 表列表永久有效
	database.RedisClient.Persist(s.ctx, listKey)

	return nil
}

// UpdateTaskStats 更新任务统计
func (s *IncrementalStatsService) UpdateTaskStats(taskID, eventType string, binlogFile string, binlogPos uint32, eventTime time.Time, failed bool) error {
	if database.RedisClient == nil {
		return nil
	}

	key := s.getTaskStatsKey(taskID)

	// 获取现有统计
	var stats IncrementalTaskStats
	data, err := database.RedisClient.Get(s.ctx, key).Result()
	if err == nil {
		json.Unmarshal([]byte(data), &stats)
	} else {
		stats.TaskID = taskID
		now := time.Now()
		stats.TodayResetAt = time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	}

	// 检查是否需要重置今日计数
	now := time.Now()
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	if stats.TodayResetAt.Before(todayStart) {
		stats.TodayEvents = 0
		stats.TodayResetAt = todayStart
	}

	// 更新统计
	if failed {
		stats.FailedCount++
	} else {
		stats.TotalEvents++
		stats.TodayEvents++

		switch eventType {
		case "INSERT":
			stats.InsertCount++
		case "UPDATE":
			stats.UpdateCount++
		case "DELETE":
			stats.DeleteCount++
		}
	}

	stats.CurrentBinlogFile = binlogFile
	stats.CurrentBinlogPos = binlogPos
	stats.LastEventTime = eventTime
	stats.ReplicationLagSeconds = int(now.Sub(eventTime).Seconds())
	if stats.ReplicationLagSeconds < 0 {
		stats.ReplicationLagSeconds = 0
	}

	// 保存统计
	newData, err := json.Marshal(stats)
	if err != nil {
		return err
	}

	return database.RedisClient.Set(s.ctx, key, newData, 24*time.Hour).Err()
}

// GetTaskStats 获取任务统计（失败时返回默认值）
func (s *IncrementalStatsService) GetTaskStats(taskID string) (*IncrementalTaskStats, error) {
	// 默认统计对象
	defaultStats := &IncrementalTaskStats{
		TaskID: taskID,
	}

	if database.RedisClient == nil {
		return defaultStats, nil // 返回默认值而不是错误
	}

	key := s.getTaskStatsKey(taskID)
	data, err := database.RedisClient.Get(s.ctx, key).Result()
	if err != nil {
		return defaultStats, nil // Redis 读取失败时返回默认值
	}

	var stats IncrementalTaskStats
	if err := json.Unmarshal([]byte(data), &stats); err != nil {
		return defaultStats, nil // 解析失败时返回默认值
	}

	return &stats, nil
}

// GetTableStatsList 获取所有表的统计列表（失败时返回空列表）
func (s *IncrementalStatsService) GetTableStatsList(taskID string) ([]*IncrementalTableStats, error) {
	if database.RedisClient == nil {
		return []*IncrementalTableStats{}, nil // 返回空列表而不是错误
	}

	// 获取表列表
	listKey := s.getTableListKey(taskID)
	tables, err := database.RedisClient.SMembers(s.ctx, listKey).Result()
	if err != nil {
		return []*IncrementalTableStats{}, nil // 返回空列表
	}

	var statsList []*IncrementalTableStats
	for _, tableKey := range tables {
		// 解析 database.table
		key := s.getTableStatsKey(taskID, "", "")
		key = fmt.Sprintf("incremental:task:%s:table:%s", taskID, tableKey)

		data, err := database.RedisClient.Get(s.ctx, key).Result()
		if err != nil {
			continue
		}

		var stats IncrementalTableStats
		if err := json.Unmarshal([]byte(data), &stats); err != nil {
			continue
		}

		statsList = append(statsList, &stats)
	}

	return statsList, nil
}

// InitTableStatsFromConfig 从任务配置初始化表统计（用于任务未启动时）
func (s *IncrementalStatsService) InitTableStatsFromConfig(taskID string, task *models.SyncTask) []*IncrementalTableStats {
	// 解析任务配置
	var config struct {
		SelectedDatabases []struct {
			Database       string `json:"database"`
			SourceDatabase string `json:"source_database"`
			Tables         []struct {
				SourceTable string `json:"source_table"`
				TargetTable string `json:"target_table"`
			} `json:"tables"`
		} `json:"selected_databases"`
	}

	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		return []*IncrementalTableStats{}
	}

	var statsList []*IncrementalTableStats
	for _, dbSel := range config.SelectedDatabases {
		// 使用目标数据库名(与UnitName格式一致)
		targetDB := dbSel.Database

		for _, tbl := range dbSel.Tables {
			// 使用目标表名(与UnitName格式一致)
			targetTable := tbl.TargetTable
			if targetTable == "" {
				targetTable = tbl.SourceTable
			}

			stats := &IncrementalTableStats{
				Database:    targetDB,
				Table:       targetTable,
				InsertCount: 0,
				UpdateCount: 0,
				DeleteCount: 0,
				TotalCount:  0,
				TodayCount:  0,
			}
			statsList = append(statsList, stats)
		}
	}

	return statsList
}

// InitDatabaseStatsFromConfig 从任务配置初始化数据库统计（用于任务未启动时）
func (s *IncrementalStatsService) InitDatabaseStatsFromConfig(taskID string, task *models.SyncTask) []*IncrementalDatabaseStats {
	// 解析任务配置
	var config struct {
		SelectedDatabases []struct {
			Database       string `json:"database"`
			SourceDatabase string `json:"source_database"`
			Tables         []struct {
				SourceTable string `json:"source_table"`
				TargetTable string `json:"target_table"`
			} `json:"tables"`
		} `json:"selected_databases"`
	}

	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		return []*IncrementalDatabaseStats{}
	}

	var statsList []*IncrementalDatabaseStats
	for _, dbSel := range config.SelectedDatabases {
		// 使用目标数据库名
		targetDB := dbSel.Database

		stats := &IncrementalDatabaseStats{
			Database:   targetDB,
			TableCount: len(dbSel.Tables),
		}
		statsList = append(statsList, stats)
	}

	return statsList
}

// ClearTaskStats 清理任务统计（任务停止时调用）
func (s *IncrementalStatsService) ClearTaskStats(taskID string) error {
	if database.RedisClient == nil {
		return nil
	}

	// 删除任务统计
	taskKey := s.getTaskStatsKey(taskID)
	database.RedisClient.Del(s.ctx, taskKey)

	// 获取并删除所有表统计
	listKey := s.getTableListKey(taskID)
	tables, err := database.RedisClient.SMembers(s.ctx, listKey).Result()
	if err == nil {
		for _, tableKey := range tables {
			key := fmt.Sprintf("incremental:task:%s:table:%s", taskID, tableKey)
			database.RedisClient.Del(s.ctx, key)
		}
	}

	// 删除表列表
	database.RedisClient.Del(s.ctx, listKey)

	return nil
}

// GetDatabaseStatsList 获取数据库级别的统计列表（合计）
func (s *IncrementalStatsService) GetDatabaseStatsList(taskID string) ([]*IncrementalDatabaseStats, error) {
	// 先获取所有表的统计
	tableStats, err := s.GetTableStatsList(taskID)
	if err != nil {
		return []*IncrementalDatabaseStats{}, err
	}

	// 按数据库分组合计
	dbStatsMap := make(map[string]*IncrementalDatabaseStats)

	for _, tableStat := range tableStats {
		dbName := tableStat.Database

		if _, exists := dbStatsMap[dbName]; !exists {
			dbStatsMap[dbName] = &IncrementalDatabaseStats{
				Database: dbName,
			}
		}

		dbStats := dbStatsMap[dbName]
		dbStats.TotalInsertCount += tableStat.InsertCount
		dbStats.TotalUpdateCount += tableStat.UpdateCount
		dbStats.TotalDeleteCount += tableStat.DeleteCount
		dbStats.TotalCount += tableStat.TotalCount
		dbStats.TodayCount += tableStat.TodayCount
		dbStats.TableCount++
		dbStats.FullSyncTotalRecords += tableStat.FullSyncTotalRecords
		dbStats.FullSyncProcessedRecords += tableStat.FullSyncProcessedRecords
	}

	// 计算每个数据库的全量同步进度
	var result []*IncrementalDatabaseStats
	for _, dbStats := range dbStatsMap {
		if dbStats.FullSyncTotalRecords > 0 {
			dbStats.FullSyncProgress = float64(dbStats.FullSyncProcessedRecords) / float64(dbStats.FullSyncTotalRecords) * 100
		}
		result = append(result, dbStats)
	}

	return result, nil
}

// GetTableStatsListByDatabase 获取指定数据库下的表统计列表
func (s *IncrementalStatsService) GetTableStatsListByDatabase(taskID, database string) ([]*IncrementalTableStats, error) {
	// 获取所有表统计
	allTableStats, err := s.GetTableStatsList(taskID)
	if err != nil {
		return []*IncrementalTableStats{}, err
	}

	// 过滤出指定数据库的表
	var result []*IncrementalTableStats
	for _, tableStat := range allTableStats {
		if tableStat.Database == database {
			result = append(result, tableStat)
		}
	}

	return result, nil
}
