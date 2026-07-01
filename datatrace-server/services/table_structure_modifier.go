package services

import (
	"fmt"
	"strings"
)

// TableStructureModifier 表结构修改器
type TableStructureModifier struct {
	parser *TableStructureParser
}

// NewTableStructureModifier 创建表结构修改器
func NewTableStructureModifier() *TableStructureModifier {
	return &TableStructureModifier{
		parser: NewTableStructureParser(),
	}
}

// FilterFieldsAndRebuild 过滤字段并重建CREATE TABLE语句
func (m *TableStructureModifier) FilterFieldsAndRebuild(createSQL string, selectedFields []string, newTableName string) (string, error) {
	// 解析原始CREATE TABLE语句
	structure, err := m.parser.Parse(createSQL)
	if err != nil {
		return "", fmt.Errorf("解析CREATE TABLE失败: %w", err)
	}

	// 如果没有选择字段，返回所有字段
	if len(selectedFields) == 0 {
		// 只替换表名
		return m.replaceTableName(createSQL, structure.TableName, newTableName), nil
	}

	// 创建字段集合（用于快速查找）
	fieldSet := make(map[string]bool)
	for _, field := range selectedFields {
		fieldSet[field] = true
	}

	// 主键必须包含
	for _, pk := range structure.PrimaryKeys {
		fieldSet[pk] = true
	}

	// 过滤字段
	var filteredFields []FieldDefinition
	for _, field := range structure.Fields {
		if fieldSet[field.Name] {
			filteredFields = append(filteredFields, field)
		}
	}

	// 过滤索引（只保留所有字段都被选中的索引）
	var filteredIndexes []IndexDefinition
	for _, index := range structure.Indexes {
		allSelected := true
		for _, col := range index.Columns {
			if !fieldSet[col] {
				allSelected = false
				break
			}
		}
		if allSelected {
			filteredIndexes = append(filteredIndexes, index)
		}
	}

	// 过滤外键（只保留所有字段都被选中的外键）
	var filteredForeignKeys []ForeignKeyDefinition
	for _, fk := range structure.ForeignKeys {
		allSelected := true
		for _, col := range fk.Columns {
			if !fieldSet[col] {
				allSelected = false
				break
			}
		}
		if allSelected {
			filteredForeignKeys = append(filteredForeignKeys, fk)
		}
	}

	// 重建CREATE TABLE语句
	return m.buildCreateSQL(newTableName, filteredFields, structure.PrimaryKeys, filteredIndexes, filteredForeignKeys), nil
}

// buildCreateSQL 构建CREATE TABLE语句
func (m *TableStructureModifier) buildCreateSQL(tableName string, fields []FieldDefinition, primaryKeys []string, indexes []IndexDefinition, foreignKeys []ForeignKeyDefinition) string {
	var parts []string

	// 添加字段定义
	for _, field := range fields {
		parts = append(parts, "  "+field.Definition)
	}

	// 添加主键定义（如果有且不在字段定义中）
	if len(primaryKeys) > 0 {
		// 检查是否已经在字段定义中包含了PRIMARY KEY
		hasPKInField := false
		for _, field := range fields {
			if strings.Contains(strings.ToUpper(field.Definition), "PRIMARY KEY") {
				hasPKInField = true
				break
			}
		}

		if !hasPKInField {
			pkCols := make([]string, len(primaryKeys))
			for i, pk := range primaryKeys {
				pkCols[i] = fmt.Sprintf("`%s`", pk)
			}
			parts = append(parts, fmt.Sprintf("  PRIMARY KEY (%s)", strings.Join(pkCols, ",")))
		}
	}

	// 添加索引定义
	for _, index := range indexes {
		parts = append(parts, "  "+index.Definition)
	}

	// 添加外键定义
	for _, fk := range foreignKeys {
		parts = append(parts, "  "+fk.Definition)
	}

	// 组装完整的CREATE TABLE语句
	sql := fmt.Sprintf("CREATE TABLE `%s` (\n%s\n) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci",
		tableName,
		strings.Join(parts, ",\n"))

	return sql
}

// replaceTableName 替换表名
func (m *TableStructureModifier) replaceTableName(createSQL, oldTableName, newTableName string) string {
	oldTableDef := fmt.Sprintf("CREATE TABLE `%s`", oldTableName)
	newTableDef := fmt.Sprintf("CREATE TABLE `%s`", newTableName)
	return strings.Replace(createSQL, oldTableDef, newTableDef, 1)
}

// GenerateAlterSQL 生成ALTER TABLE语句（用于配置变更时增删字段）
func (m *TableStructureModifier) GenerateAlterSQL(tableName string, oldFields, newFields []string, sourceCreateSQL string) ([]string, error) {
	var alterSQLs []string

	// 解析源表结构（用于获取字段定义）
	structure, err := m.parser.Parse(sourceCreateSQL)
	if err != nil {
		return nil, fmt.Errorf("解析CREATE TABLE失败: %w", err)
	}

	// 创建字段映射
	fieldMap := make(map[string]FieldDefinition)
	for _, field := range structure.Fields {
		fieldMap[field.Name] = field
	}

	// 创建旧字段集合
	oldFieldSet := make(map[string]bool)
	for _, field := range oldFields {
		oldFieldSet[field] = true
	}

	// 创建新字段集合
	newFieldSet := make(map[string]bool)
	for _, field := range newFields {
		newFieldSet[field] = true
	}

	// 找出需要添加的字段
	for _, field := range newFields {
		if !oldFieldSet[field] {
			// 需要添加
			if fieldDef, ok := fieldMap[field]; ok {
				alterSQL := fmt.Sprintf("ALTER TABLE `%s` ADD COLUMN %s", tableName, fieldDef.Definition)
				alterSQLs = append(alterSQLs, alterSQL)
			}
		}
	}

	// 找出需要删除的字段
	for _, field := range oldFields {
		if !newFieldSet[field] {
			// 需要删除（但不删除主键）
			if fieldDef, ok := fieldMap[field]; ok && !fieldDef.IsPrimary {
				alterSQL := fmt.Sprintf("ALTER TABLE `%s` DROP COLUMN `%s`", tableName, field)
				alterSQLs = append(alterSQLs, alterSQL)
			}
		}
	}

	return alterSQLs, nil
}
