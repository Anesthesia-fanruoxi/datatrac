package services

import (
	"fmt"
	"regexp"
	"strings"
)

// TableStructure 表结构
type TableStructure struct {
	TableName   string
	Fields      []FieldDefinition
	PrimaryKeys []string
	Indexes     []IndexDefinition
	ForeignKeys []ForeignKeyDefinition
	CreateSQL   string
}

// FieldDefinition 字段定义
type FieldDefinition struct {
	Name       string
	Definition string // 完整的字段定义，如：`id` int(11) NOT NULL AUTO_INCREMENT
	IsPrimary  bool
}

// IndexDefinition 索引定义
type IndexDefinition struct {
	Name       string
	Definition string   // 完整的索引定义
	Columns    []string // 涉及的字段
}

// ForeignKeyDefinition 外键定义
type ForeignKeyDefinition struct {
	Name       string
	Definition string   // 完整的外键定义
	Columns    []string // 涉及的字段
}

// TableStructureParser 表结构解析器
type TableStructureParser struct{}

// NewTableStructureParser 创建表结构解析器
func NewTableStructureParser() *TableStructureParser {
	return &TableStructureParser{}
}

// Parse 解析CREATE TABLE语句
func (p *TableStructureParser) Parse(createSQL string) (*TableStructure, error) {
	structure := &TableStructure{
		CreateSQL: createSQL,
	}

	// 提取表名
	tableNameRegex := regexp.MustCompile(`CREATE TABLE\s+` + "`" + `([^` + "`" + `]+)` + "`")
	matches := tableNameRegex.FindStringSubmatch(createSQL)
	if len(matches) > 1 {
		structure.TableName = matches[1]
	}

	// 提取字段和约束定义（括号内的内容）
	startIdx := strings.Index(createSQL, "(")
	endIdx := strings.LastIndex(createSQL, ")")
	if startIdx == -1 || endIdx == -1 {
		return nil, fmt.Errorf("无效的CREATE TABLE语句")
	}

	content := createSQL[startIdx+1 : endIdx]

	// 分割各个定义（字段、主键、索引等）
	definitions := p.splitDefinitions(content)

	// 解析每个定义
	for _, def := range definitions {
		def = strings.TrimSpace(def)
		if def == "" {
			continue
		}

		// 判断定义类型
		defLower := strings.ToLower(def)

		if strings.HasPrefix(defLower, "primary key") {
			// 主键定义
			p.parsePrimaryKey(def, structure)
		} else if strings.HasPrefix(defLower, "key ") || strings.HasPrefix(defLower, "index ") || strings.HasPrefix(defLower, "unique ") {
			// 索引定义
			p.parseIndex(def, structure)
		} else if strings.HasPrefix(defLower, "constraint") || strings.HasPrefix(defLower, "foreign key") {
			// 外键定义
			p.parseForeignKey(def, structure)
		} else if strings.HasPrefix(def, "`") {
			// 字段定义
			p.parseField(def, structure)
		}
	}

	// 标记主键字段
	for i := range structure.Fields {
		for _, pk := range structure.PrimaryKeys {
			if structure.Fields[i].Name == pk {
				structure.Fields[i].IsPrimary = true
				break
			}
		}
	}

	return structure, nil
}

// splitDefinitions 分割定义（处理逗号分隔，但要考虑括号内的逗号）
func (p *TableStructureParser) splitDefinitions(content string) []string {
	var definitions []string
	var current strings.Builder
	depth := 0

	for _, char := range content {
		if char == '(' {
			depth++
			current.WriteRune(char)
		} else if char == ')' {
			depth--
			current.WriteRune(char)
		} else if char == ',' && depth == 0 {
			// 顶层逗号，分割
			definitions = append(definitions, current.String())
			current.Reset()
		} else {
			current.WriteRune(char)
		}
	}

	// 添加最后一个
	if current.Len() > 0 {
		definitions = append(definitions, current.String())
	}

	return definitions
}

// parseField 解析字段定义
func (p *TableStructureParser) parseField(def string, structure *TableStructure) {
	// 提取字段名（第一个反引号包裹的内容）
	nameRegex := regexp.MustCompile("`" + `([^` + "`" + `]+)` + "`")
	matches := nameRegex.FindStringSubmatch(def)
	if len(matches) > 1 {
		structure.Fields = append(structure.Fields, FieldDefinition{
			Name:       matches[1],
			Definition: def,
			IsPrimary:  false,
		})
	}
}

// parsePrimaryKey 解析主键定义
func (p *TableStructureParser) parsePrimaryKey(def string, structure *TableStructure) {
	// 提取主键字段列表
	// PRIMARY KEY (`id`) 或 PRIMARY KEY (`id`,`name`)
	columnsRegex := regexp.MustCompile(`\((.*?)\)`)
	matches := columnsRegex.FindStringSubmatch(def)
	if len(matches) > 1 {
		columnsPart := matches[1]
		// 分割字段
		columns := strings.Split(columnsPart, ",")
		for _, col := range columns {
			col = strings.TrimSpace(col)
			col = strings.Trim(col, "`")
			if col != "" {
				structure.PrimaryKeys = append(structure.PrimaryKeys, col)
			}
		}
	}
}

// parseIndex 解析索引定义
func (p *TableStructureParser) parseIndex(def string, structure *TableStructure) {
	// 提取索引名和字段列表
	// KEY `idx_name` (`name`)
	// UNIQUE KEY `uk_email` (`email`)

	// 提取索引名
	var indexName string
	nameRegex := regexp.MustCompile("`" + `([^` + "`" + `]+)` + "`")
	matches := nameRegex.FindAllStringSubmatch(def, -1)
	if len(matches) > 0 {
		indexName = matches[0][1]
	}

	// 提取字段列表
	var columns []string
	columnsRegex := regexp.MustCompile(`\((.*?)\)`)
	colMatches := columnsRegex.FindStringSubmatch(def)
	if len(colMatches) > 1 {
		columnsPart := colMatches[1]
		cols := strings.Split(columnsPart, ",")
		for _, col := range cols {
			col = strings.TrimSpace(col)
			col = strings.Trim(col, "`")
			// 去掉长度限制，如 name(10) -> name
			if idx := strings.Index(col, "("); idx > 0 {
				col = col[:idx]
			}
			if col != "" {
				columns = append(columns, col)
			}
		}
	}

	structure.Indexes = append(structure.Indexes, IndexDefinition{
		Name:       indexName,
		Definition: def,
		Columns:    columns,
	})
}

// parseForeignKey 解析外键定义
func (p *TableStructureParser) parseForeignKey(def string, structure *TableStructure) {
	// 提取外键名和字段列表
	// CONSTRAINT `fk_user_id` FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)

	var fkName string
	nameRegex := regexp.MustCompile("`" + `([^` + "`" + `]+)` + "`")
	matches := nameRegex.FindAllStringSubmatch(def, -1)
	if len(matches) > 0 {
		fkName = matches[0][1]
	}

	// 提取字段列表（FOREIGN KEY后面的括号）
	var columns []string
	fkRegex := regexp.MustCompile(`FOREIGN KEY\s*\((.*?)\)`)
	colMatches := fkRegex.FindStringSubmatch(def)
	if len(colMatches) > 1 {
		columnsPart := colMatches[1]
		cols := strings.Split(columnsPart, ",")
		for _, col := range cols {
			col = strings.TrimSpace(col)
			col = strings.Trim(col, "`")
			if col != "" {
				columns = append(columns, col)
			}
		}
	}

	structure.ForeignKeys = append(structure.ForeignKeys, ForeignKeyDefinition{
		Name:       fkName,
		Definition: def,
		Columns:    columns,
	})
}
