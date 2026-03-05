package services

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLMetadataService MySQL 元数据查询服务
type MySQLMetadataService struct{}

// NewMySQLMetadataService 创建 MySQL 元数据服务
func NewMySQLMetadataService() *MySQLMetadataService {
	return &MySQLMetadataService{}
}

// DatabaseInfo 数据库信息
type DatabaseInfo struct {
	Name       string `json:"name"`
	TableCount int    `json:"table_count"`
}

// TableInfo 表信息
type TableInfo struct {
	Name    string `json:"name"`
	Comment string `json:"comment"`
}

// DatabaseWithTables 数据库及其表列表
type DatabaseWithTables struct {
	Database string   `json:"database"`
	Tables   []string `json:"tables"`
}

// GetDatabases 获取数据库列表
func (s *MySQLMetadataService) GetDatabases(host string, port int, username, password string) ([]DatabaseInfo, error) {
	// 构建连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, host, port)

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	defer db.Close()

	// 查询数据库列表（排除系统数据库）
	query := `
		SELECT 
			SCHEMA_NAME as name,
			(SELECT COUNT(*) FROM information_schema.TABLES 
			 WHERE TABLE_SCHEMA = SCHEMA_NAME AND TABLE_TYPE = 'BASE TABLE') as table_count
		FROM information_schema.SCHEMATA
		WHERE SCHEMA_NAME NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY SCHEMA_NAME
	`

	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	var databases []DatabaseInfo
	for rows.Next() {
		var db DatabaseInfo
		if err := rows.Scan(&db.Name, &db.TableCount); err != nil {
			return nil, err
		}
		databases = append(databases, db)
	}

	return databases, nil
}

// GetTables 获取指定数据库的表列表
func (s *MySQLMetadataService) GetTables(host string, port int, username, password, database string) ([]TableInfo, error) {
	// 构建连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, host, port, database)

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	defer db.Close()

	// 查询表列表（只查询表名，不查询行数，提高速度）
	query := "SELECT TABLE_NAME as name, IFNULL(TABLE_COMMENT, '') as comment FROM information_schema.TABLES WHERE TABLE_SCHEMA = ? AND TABLE_TYPE = 'BASE TABLE' ORDER BY TABLE_NAME"

	rows, err := db.Query(query, database)
	if err != nil {
		return nil, fmt.Errorf("查询失败: %w", err)
	}
	defer rows.Close()

	var tables []TableInfo
	for rows.Next() {
		var table TableInfo
		if err := rows.Scan(&table.Name, &table.Comment); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, nil
}

// GetDatabasesWithTables 获取所有数据库及其表列表（树形结构）
func (s *MySQLMetadataService) GetDatabasesWithTables(host string, port int, username, password string) ([]DatabaseWithTables, error) {
	// 构建连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, host, port)

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接失败: %w", err)
	}
	defer db.Close()

	// 1. 获取数据库列表（排除系统数据库）
	dbQuery := `
		SELECT SCHEMA_NAME
		FROM information_schema.SCHEMATA
		WHERE SCHEMA_NAME NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		ORDER BY SCHEMA_NAME
	`

	dbRows, err := db.Query(dbQuery)
	if err != nil {
		return nil, fmt.Errorf("查询数据库列表失败: %w", err)
	}
	defer dbRows.Close()

	var databaseNames []string
	for dbRows.Next() {
		var dbName string
		if err := dbRows.Scan(&dbName); err != nil {
			return nil, err
		}
		databaseNames = append(databaseNames, dbName)
	}

	// 2. 批量查询所有数据库的表
	tableQuery := `
		SELECT TABLE_SCHEMA, TABLE_NAME
		FROM information_schema.TABLES
		WHERE TABLE_SCHEMA NOT IN ('information_schema', 'mysql', 'performance_schema', 'sys')
		  AND TABLE_TYPE = 'BASE TABLE'
		ORDER BY TABLE_SCHEMA, TABLE_NAME
	`

	tableRows, err := db.Query(tableQuery)
	if err != nil {
		return nil, fmt.Errorf("查询表列表失败: %w", err)
	}
	defer tableRows.Close()

	// 3. 构建数据库和表的映射
	dbTablesMap := make(map[string][]string)
	for tableRows.Next() {
		var dbName, tableName string
		if err := tableRows.Scan(&dbName, &tableName); err != nil {
			return nil, err
		}
		dbTablesMap[dbName] = append(dbTablesMap[dbName], tableName)
	}

	// 4. 构建返回结果
	var result []DatabaseWithTables
	for _, dbName := range databaseNames {
		tables := dbTablesMap[dbName]
		if tables == nil {
			tables = []string{} // 确保返回空数组而不是 null
		}
		result = append(result, DatabaseWithTables{
			Database: dbName,
			Tables:   tables,
		})
	}

	return result, nil
}
