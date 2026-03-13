package services

import (
	"database/sql"
	"fmt"
	"strings"
)

// TableStructureAlterService 表结构对比和ALTER服务
type TableStructureAlterService struct {
	parser *TableStructureParser
}

// NewTableStructureAlterService 创建表结构对比服务
func NewTableStructureAlterService() *TableStructureAlterService {
	return &TableStructureAlterService{
		parser: NewTableStructureParser(),
	}
}

// AlterResult ALTER操作结果
type AlterResult struct {
	TableName      string
	Action         string // create/update/skip
	SourceFields   []string
	TargetFields   []string
	AddedFields    []string    // 目标新增的字段
	RemovedFields  []string    // 目标移除的字段
	ModifiedFields []FieldDiff // 目标修改的字段
	SourceIndexes  []string
	TargetIndexes  []string
	AddedIndexes   []string
	RemovedIndexes []string
	SQLs           []string
	Error          error
}

// FieldDiff 字段差异
type FieldDiff struct {
	FieldName string
	SourceDef string
	TargetDef string
}

// CompareAndAlter 对比并执行ALTER（表存在则对比+ALTER，表不存在则创建）
func (s *TableStructureAlterService) CompareAndAlter(
	sourceDB *sql.DB,
	targetDB *sql.DB,
	sourceTable string,
	targetTable string,
	selectedFields []string,
) (*AlterResult, error) {
	result := &AlterResult{
		TableName: targetTable,
	}

	// 1. 获取源表结构
	sourceCreateSQL, err := s.getCreateTableSQL(sourceDB, sourceTable)
	if err != nil {
		return nil, fmt.Errorf("获取源表结构失败: %w", err)
	}

	sourceStructure, err := s.parser.Parse(sourceCreateSQL)
	if err != nil {
		return nil, fmt.Errorf("解析源表结构失败: %w", err)
	}

	// 过滤字段（如果配置了字段选择）
	if len(selectedFields) > 0 {
		var filteredFields []FieldDefinition
		for _, field := range sourceStructure.Fields {
			if contains(selectedFields, field.Name) {
				filteredFields = append(filteredFields, field)
			}
		}
		sourceStructure.Fields = filteredFields
	}

	// 2. 检查目标表是否存在
	exists, err := s.tableExists(targetDB, targetTable)
	if err != nil {
		return nil, fmt.Errorf("检查目标表是否存在失败: %w", err)
	}

	if !exists {
		// 表不存在，直接创建
		result.Action = "create"
		result.SourceFields = s.extractFieldNames(sourceStructure.Fields)
		result.SourceIndexes = s.extractIndexNames(sourceStructure.Indexes)
		return result, nil
	}

	// 3. 表存在，对比结构
	result.Action = "update"

	// 获取目标表结构
	targetCreateSQL, err := s.getCreateTableSQL(targetDB, targetTable)
	if err != nil {
		return nil, fmt.Errorf("获取目标表结构失败: %w", err)
	}

	targetStructure, err := s.parser.Parse(targetCreateSQL)
	if err != nil {
		return nil, fmt.Errorf("解析目标表结构失败: %w", err)
	}

	// 4. 对比字段
	sourceFieldMap := make(map[string]FieldDefinition)
	for _, f := range sourceStructure.Fields {
		sourceFieldMap[f.Name] = f
	}

	targetFieldMap := make(map[string]FieldDefinition)
	for _, f := range targetStructure.Fields {
		targetFieldMap[f.Name] = f
	}

	result.SourceFields = s.extractFieldNames(sourceStructure.Fields)
	result.TargetFields = s.extractFieldNames(targetStructure.Fields)

	// 找出新增和移除的字段
	for _, sourceField := range sourceStructure.Fields {
		if _, ok := targetFieldMap[sourceField.Name]; !ok {
			result.AddedFields = append(result.AddedFields, sourceField.Name)
		}
	}

	for _, targetField := range targetStructure.Fields {
		if _, ok := sourceFieldMap[targetField.Name]; !ok {
			result.RemovedFields = append(result.RemovedFields, targetField.Name)
		}
	}

	// 找出修改的字段（字段名相同但定义不同）
	for _, sourceField := range sourceStructure.Fields {
		if targetField, ok := targetFieldMap[sourceField.Name]; ok {
			// 标准化定义后比较
			sourceDef := s.normalizeFieldDef(sourceField.Definition)
			targetDef := s.normalizeFieldDef(targetField.Definition)
			if sourceDef != targetDef {
				result.ModifiedFields = append(result.ModifiedFields, FieldDiff{
					FieldName: sourceField.Name,
					SourceDef: sourceField.Definition,
					TargetDef: targetField.Definition,
				})
			}
		}
	}

	// 5. 对比索引
	result.SourceIndexes = s.extractIndexNames(sourceStructure.Indexes)
	result.TargetIndexes = s.extractIndexNames(targetStructure.Indexes)

	sourceIndexMap := make(map[string]bool)
	targetIndexMap := make(map[string]bool)

	for _, idx := range sourceStructure.Indexes {
		sourceIndexMap[idx.Name] = true
	}
	for _, idx := range targetStructure.Indexes {
		targetIndexMap[idx.Name] = true
	}

	for _, idx := range sourceStructure.Indexes {
		if !targetIndexMap[idx.Name] {
			result.AddedIndexes = append(result.AddedIndexes, idx.Name)
		}
	}

	// 6. 生成ALTER SQL
	result.SQLs = s.generateAlterSQLs(sourceStructure, targetStructure, result)

	// 如果没有差异，设置action为skip
	if len(result.AddedFields) == 0 && len(result.RemovedFields) == 0 &&
		len(result.ModifiedFields) == 0 && len(result.AddedIndexes) == 0 {
		result.Action = "skip"
	}

	return result, nil
}

// getCreateTableSQL 获取表的CREATE TABLE语句
func (s *TableStructureAlterService) getCreateTableSQL(db *sql.DB, tableName string) (string, error) {
	var createSQL string
	err := db.QueryRow(fmt.Sprintf("SHOW CREATE TABLE `%s`", tableName)).Scan(&tableName, &createSQL)
	if err != nil {
		return "", err
	}
	return createSQL, nil
}

// tableExists 检查表是否存在
func (s *TableStructureAlterService) tableExists(db *sql.DB, tableName string) (bool, error) {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM information_schema.TABLES WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = ?", tableName).Scan(&count)
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// extractFieldNames 提取字段名列表
func (s *TableStructureAlterService) extractFieldNames(fields []FieldDefinition) []string {
	var names []string
	for _, f := range fields {
		names = append(names, f.Name)
	}
	return names
}

// extractIndexNames 提取索引名列表
func (s *TableStructureAlterService) extractIndexNames(indexes []IndexDefinition) []string {
	var names []string
	for _, idx := range indexes {
		names = append(names, idx.Name)
	}
	return names
}

// normalizeFieldDef 标准化字段定义（去除多余空格）
func (s *TableStructureAlterService) normalizeFieldDef(def string) string {
	// 移除多余空格
	def = strings.Join(strings.Fields(def), " ")
	// 移除大小写差异（统一转小写比较类型部分）
	return def
}

// generateAlterSQLs 生成ALTER SQL语句
func (s *TableStructureAlterService) generateAlterSQLs(
	sourceStructure *TableStructure,
	targetStructure *TableStructure,
	result *AlterResult,
) []string {
	var sqls []string

	// 1. 添加新字段
	for _, fieldName := range result.AddedFields {
		if field, ok := findField(sourceStructure.Fields, fieldName); ok {
			// 提取字段定义部分
			fieldDef := extractFieldDefinition(field.Definition)
			sqls = append(sqls, fmt.Sprintf("ADD COLUMN %s", fieldDef))
		}
	}

	// 2. 修改字段（类型、长度等）
	for _, diff := range result.ModifiedFields {
		if field, ok := findField(sourceStructure.Fields, diff.FieldName); ok {
			fieldDef := extractFieldDefinition(field.Definition)
			sqls = append(sqls, fmt.Sprintf("MODIFY COLUMN %s", fieldDef))
		}
	}

	// 3. 删除字段（谨慎处理）
	// 通常不自动删除字段，以免丢失数据
	// 如果确实需要删除，可以取消注释下面代码
	// for _, fieldName := range result.RemovedFields {
	// 	sqls = append(sqls, fmt.Sprintf("DROP COLUMN `%s`", fieldName))
	// }

	// 4. 添加索引
	for _, idxName := range result.AddedIndexes {
		if idx, ok := findIndex(sourceStructure.Indexes, idxName); ok {
			idxDef := extractIndexDefinition(idx.Definition)
			sqls = append(sqls, idxDef)
		}
	}

	return sqls
}

// findField 查找字段
func findField(fields []FieldDefinition, name string) (FieldDefinition, bool) {
	for _, f := range fields {
		if f.Name == name {
			return f, true
		}
	}
	return FieldDefinition{}, false
}

// findIndex 查找索引
func findIndex(indexes []IndexDefinition, name string) (IndexDefinition, bool) {
	for _, idx := range indexes {
		if idx.Name == name {
			return idx, true
		}
	}
	return IndexDefinition{}, false
}

// extractFieldDefinition 从完整定义中提取字段定义（不含反引号包裹的字段名）
func extractFieldDefinition(fullDef string) string {
	// 格式：`field_name` definition -> field_name definition
	// 或者：field_name definition
	parts := strings.SplitN(fullDef, "`", 3)
	if len(parts) >= 3 {
		return parts[2]
	}
	parts = strings.SplitN(fullDef, " ", 2)
	if len(parts) >= 2 {
		return parts[1]
	}
	return fullDef
}

// extractIndexDefinition 从索引定义中提取索引定义部分
func extractIndexDefinition(fullDef string) string {
	// 格式：KEY `idx_name` (`col1`,`col2`) -> KEY `idx_name` (`col1`,`col2`)
	// 格式：UNIQUE KEY `uk_name` (`col1`) -> UNIQUE KEY `uk_name` (`col1`)
	return fullDef
}

// contains 检查切片是否包含元素
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
