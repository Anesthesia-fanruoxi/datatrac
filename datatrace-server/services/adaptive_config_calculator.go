package services

import (
	"database/sql"
	"fmt"
)

// AdaptiveConfigCalculator 自适应配置计算器
type AdaptiveConfigCalculator struct {
	resourceDetector *SystemResourceDetector
}

// NewAdaptiveConfigCalculator 创建自适应配置计算器
func NewAdaptiveConfigCalculator() *AdaptiveConfigCalculator {
	return &AdaptiveConfigCalculator{
		resourceDetector: NewSystemResourceDetector(),
	}
}

// TableStats 表统计信息
type TableStats struct {
	TableName    string
	RowCount     int64
	AvgRowLength int64 // 平均行长度（字节）
	DataLength   int64 // 数据大小（字节）
	ColumnCount  int   // 字段数量
}

// AdaptiveConfig 自适应配置
type AdaptiveConfig struct {
	ThreadCount int // 线程数
	BatchSize   int // 批次大小
}

// CalculateForTask 为任务计算自适应配置
func (c *AdaptiveConfigCalculator) CalculateForTask(tableCount int, avgTableSize int64) *AdaptiveConfig {
	resources := c.resourceDetector.DetectResources()

	// 计算线程数：基于CPU核心数和表数量
	threadCount := resources.RecommendedThreads
	if tableCount < threadCount {
		threadCount = tableCount
	}
	if threadCount < 1 {
		threadCount = 1
	}

	// 计算批次大小：基于表大小
	batchSize := c.calculateBatchSize(avgTableSize)

	return &AdaptiveConfig{
		ThreadCount: threadCount,
		BatchSize:   batchSize,
	}
}

// CalculateForTable 为单个表计算批次大小
func (c *AdaptiveConfigCalculator) CalculateForTable(stats *TableStats) int {
	// 基于行数和行长度计算批次大小
	if stats.RowCount == 0 {
		return 1000 // 默认值
	}

	// 策略：
	// 1. 小表（<1万行）：批次1000
	// 2. 中表（1万-100万行）：批次5000
	// 3. 大表（>100万行）：批次10000
	// 4. 如果行很长（>1KB），批次减半

	batchSize := 1000

	if stats.RowCount > 1000000 {
		batchSize = 10000
	} else if stats.RowCount > 10000 {
		batchSize = 5000
	}

	// 如果平均行长度超过1KB，批次减半
	if stats.AvgRowLength > 1024 {
		batchSize = batchSize / 2
	}

	// 如果字段很多（>50个），批次减半
	if stats.ColumnCount > 50 {
		batchSize = batchSize / 2
	}

	// 最小批次100，最大批次100000
	if batchSize < 100 {
		batchSize = 100
	}
	if batchSize > 100000 {
		batchSize = 100000
	}

	return batchSize
}

// calculateBatchSize 计算批次大小（基于平均表大小）
func (c *AdaptiveConfigCalculator) calculateBatchSize(avgTableSize int64) int {
	// avgTableSize 单位：字节
	// 转换为MB
	avgTableSizeMB := avgTableSize / 1024 / 1024

	// 策略：
	// 小表（<10MB）：批次1000
	// 中表（10MB-1GB）：批次5000
	// 大表（>1GB）：批次10000

	if avgTableSizeMB > 1024 {
		return 10000
	} else if avgTableSizeMB > 10 {
		return 5000
	}
	return 1000
}

// GetTableStats 获取表的统计信息
func (c *AdaptiveConfigCalculator) GetTableStats(db *sql.DB, database, table string) (*TableStats, error) {
	query := `
		SELECT 
			TABLE_NAME,
			IFNULL(TABLE_ROWS, 0) as row_count,
			IFNULL(AVG_ROW_LENGTH, 0) as avg_row_length,
			IFNULL(DATA_LENGTH, 0) as data_length
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
	`

	stats := &TableStats{}
	err := db.QueryRow(query, database, table).Scan(
		&stats.TableName,
		&stats.RowCount,
		&stats.AvgRowLength,
		&stats.DataLength,
	)

	if err != nil {
		return nil, fmt.Errorf("查询表统计信息失败: %w", err)
	}

	// 获取字段数量
	columnQuery := `
		SELECT COUNT(*) 
		FROM information_schema.COLUMNS
		WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ?
	`

	err = db.QueryRow(columnQuery, database, table).Scan(&stats.ColumnCount)
	if err != nil {
		stats.ColumnCount = 0 // 默认值
	}

	return stats, nil
}

// GetDefaultConfig 获取默认配置（当无法获取统计信息时使用）
func (c *AdaptiveConfigCalculator) GetDefaultConfig() *AdaptiveConfig {
	resources := c.resourceDetector.DetectResources()

	return &AdaptiveConfig{
		ThreadCount: resources.RecommendedThreads,
		BatchSize:   5000, // 默认批次大小
	}
}
