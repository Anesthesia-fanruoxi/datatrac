package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ForeignKeyRelation 外键关系
type ForeignKeyRelation struct {
	ChildTable  string // 子表(包含外键的表)
	ParentTable string // 父表(被引用的表)
}

// TableSorter 表排序器
type TableSorter struct {
	db *sql.DB
}

// AnalyzeForeignKeys 分析数据库中的外键关系
func (ts *TableSorter) AnalyzeForeignKeys(database string, tables []string) ([]ForeignKeyRelation, error) {
	// 构建表名列表的 IN 子句
	if len(tables) == 0 {
		return []ForeignKeyRelation{}, nil
	}

	// 查询外键关系
	query := `
		SELECT 
			kcu.TABLE_NAME as child_table,
			kcu.REFERENCED_TABLE_NAME as parent_table
		FROM information_schema.KEY_COLUMN_USAGE kcu
		WHERE kcu.TABLE_SCHEMA = ?
			AND kcu.REFERENCED_TABLE_NAME IS NOT NULL
			AND kcu.TABLE_NAME != kcu.REFERENCED_TABLE_NAME
		ORDER BY kcu.TABLE_NAME, kcu.CONSTRAINT_NAME
	`

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	fmt.Printf("[DEBUG] 数据库 %s: 执行外键查询 SQL\n", database)

	rows, err := ts.db.QueryContext(ctx, query, database)
	if err != nil {
		fmt.Printf("[DEBUG] 数据库 %s: 查询失败: %v\n", database, err)
		return nil, fmt.Errorf("查询外键关系失败: %w", err)
	}
	defer rows.Close()

	var relations []ForeignKeyRelation
	tableSet := make(map[string]bool)
	for _, t := range tables {
		tableSet[t] = true
	}

	allRelations := 0
	keptRelations := 0

	fmt.Printf("[DEBUG] 数据库 %s: 开始查询外键关系, 同步表列表: %v\n", database, tables)

	for rows.Next() {
		var rel ForeignKeyRelation
		if err := rows.Scan(&rel.ChildTable, &rel.ParentTable); err != nil {
			fmt.Printf("[DEBUG] 数据库 %s: 扫描失败: %v\n", database, err)
			return nil, fmt.Errorf("扫描外键关系失败: %w", err)
		}

		allRelations++
		fmt.Printf("[DEBUG] 数据库 %s: 发现外键 %s -> %s\n", database, rel.ChildTable, rel.ParentTable)

		// 只要子表在同步列表中,就保留这个外键关系
		// 因为删除子表时需要知道它依赖哪些父表
		if tableSet[rel.ChildTable] {
			relations = append(relations, rel)
			keptRelations++
			fmt.Printf("[DEBUG] 数据库 %s: 保留外键 %s -> %s (子表在同步列表中)\n", database, rel.ChildTable, rel.ParentTable)
		} else {
			fmt.Printf("[DEBUG] 数据库 %s: 跳过外键 %s -> %s (子表不在同步列表中)\n", database, rel.ChildTable, rel.ParentTable)
		}
	}

	// 检查是否有扫描错误
	if err := rows.Err(); err != nil {
		fmt.Printf("[DEBUG] 数据库 %s: 行扫描错误: %v\n", database, err)
		return nil, fmt.Errorf("行扫描错误: %w", err)
	}

	// 调试日志:显示最终结果
	fmt.Printf("[DEBUG] 数据库 %s: 查询到 %d 个外键关系, 保留 %d 个\n", database, allRelations, keptRelations)

	return relations, nil
}

// TopologicalSort 拓扑排序,返回删除顺序(子表在前,父表在后)
func (ts *TableSorter) TopologicalSort(tables []string, relations []ForeignKeyRelation) ([]string, error) {
	// 构建邻接表和入度表
	graph := make(map[string][]string) // parent -> []children
	inDegree := make(map[string]int)   // 入度(有多少个父表)
	allTables := make(map[string]bool)

	// 初始化所有表
	for _, table := range tables {
		allTables[table] = true
		inDegree[table] = 0
		graph[table] = []string{}
	}

	// 构建图
	for _, rel := range relations {
		// parent -> child 的边
		graph[rel.ParentTable] = append(graph[rel.ParentTable], rel.ChildTable)
		inDegree[rel.ChildTable]++
	}

	// 拓扑排序(Kahn算法)
	var queue []string
	var result []string

	// 找到所有入度为0的节点(没有依赖的表,或者是父表)
	for table := range allTables {
		if inDegree[table] == 0 {
			queue = append(queue, table)
		}
	}

	// BFS遍历
	for len(queue) > 0 {
		// 取出队首
		current := queue[0]
		queue = queue[1:]
		result = append(result, current)

		// 遍历所有子节点
		for _, child := range graph[current] {
			inDegree[child]--
			if inDegree[child] == 0 {
				queue = append(queue, child)
			}
		}
	}

	// 检查是否有环
	if len(result) != len(tables) {
		return nil, fmt.Errorf("检测到循环外键依赖,无法确定删除顺序")
	}

	// 反转结果,得到删除顺序(子表在前,父表在后)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}

// SortTablesForDrop 对表进行排序,返回删除顺序
func SortTablesForDrop(db *sql.DB, database string, tables []string) ([]string, error) {
	sorted, _, err := SortTablesForDropWithCheck(db, database, tables)
	return sorted, err
}

// SortTablesForDropWithCheck 对表进行排序,返回删除顺序和是否有外键依赖
func SortTablesForDropWithCheck(db *sql.DB, database string, tables []string) ([]string, bool, error) {
	sorter := &TableSorter{db: db}

	// 分析外键关系
	relations, err := sorter.AnalyzeForeignKeys(database, tables)
	if err != nil {
		// 分析失败,返回原顺序
		return tables, false, nil
	}

	// 如果没有外键关系,直接返回原顺序
	if len(relations) == 0 {
		return tables, false, nil
	}

	// 拓扑排序
	sorted, err := sorter.TopologicalSort(tables, relations)
	if err != nil {
		// 排序失败(可能有循环依赖),返回原顺序
		return tables, false, nil
	}

	return sorted, true, nil
}

// SortTablesForDropWithFKList 对表进行排序,返回删除顺序、是否有外键依赖、外键表列表
func SortTablesForDropWithFKList(db *sql.DB, database string, tables []string) ([]string, bool, map[string]bool, error) {
	sorter := &TableSorter{db: db}
	fkTables := make(map[string]bool)

	// 分析外键关系
	relations, err := sorter.AnalyzeForeignKeys(database, tables)
	if err != nil {
		// 分析失败,返回原顺序
		return tables, false, fkTables, nil
	}

	// 如果没有外键关系,直接返回原顺序
	if len(relations) == 0 {
		return tables, false, fkTables, nil
	}

	// 记录哪些表有外键关系(子表或父表)
	for _, rel := range relations {
		fkTables[rel.ChildTable] = true
		fkTables[rel.ParentTable] = true
	}

	// 拓扑排序
	sorted, err := sorter.TopologicalSort(tables, relations)
	if err != nil {
		// 排序失败(可能有循环依赖),返回原顺序
		return tables, false, fkTables, nil
	}

	return sorted, true, fkTables, nil
}
