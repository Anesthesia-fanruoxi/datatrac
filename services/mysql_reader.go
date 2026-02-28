package services

import (
	"database/sql"
	"fmt"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLReader MySQL数据读取器
type MySQLReader struct {
	db         *sql.DB
	tableName  string
	batchSize  int
	offset     int64
	totalCount int64
}

// NewMySQLReader 创建MySQL读取器
func NewMySQLReader(host string, port int, username, password, database, tableName string, batchSize int) (*MySQLReader, error) {
	// 校验表名
	if err := ValidateTableName(tableName); err != nil {
		return nil, fmt.Errorf("表名校验失败: %w", err)
	}

	// 构建连接字符串
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, host, port, database)

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 测试连接
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	reader := &MySQLReader{
		db:        db,
		tableName: tableName,
		batchSize: batchSize,
		offset:    0,
	}

	// 查询总记录数
	if err := reader.queryTotalCount(); err != nil {
		return nil, err
	}

	return reader, nil
}

// queryTotalCount 查询总记录数
func (r *MySQLReader) queryTotalCount() error {
	query := fmt.Sprintf("SELECT COUNT(*) FROM `%s`", r.tableName)
	err := r.db.QueryRow(query).Scan(&r.totalCount)
	if err != nil {
		return fmt.Errorf("查询总记录数失败: %w", err)
	}
	return nil
}

// GetTotalCount 获取总记录数
func (r *MySQLReader) GetTotalCount() int64 {
	return r.totalCount
}

// ReadBatch 读取一批数据
func (r *MySQLReader) ReadBatch() ([]map[string]interface{}, error) {
	// 构建查询语句
	query := fmt.Sprintf("SELECT * FROM `%s` LIMIT %d OFFSET %d",
		r.tableName, r.batchSize, r.offset)

	// 执行查询
	rows, err := r.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("查询数据失败: %w", err)
	}
	defer rows.Close()

	// 获取列名
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("获取列名失败: %w", err)
	}

	// 读取数据
	var results []map[string]interface{}
	for rows.Next() {
		// 创建列值容器
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// 扫描行数据
		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, fmt.Errorf("扫描行数据失败: %w", err)
		}

		// 构建结果map
		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// 处理[]byte类型
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		results = append(results, row)
	}

	// 更新偏移量
	r.offset += int64(len(results))

	return results, nil
}

// HasMore 是否还有更多数据
func (r *MySQLReader) HasMore() bool {
	return r.offset < r.totalCount
}

// GetDB 获取数据库连接（用于创建表结构）
func (r *MySQLReader) GetDB() *sql.DB {
	return r.db
}

// GetDatabaseCharset 获取数据库字符集和排序规则
func (r *MySQLReader) GetDatabaseCharset(database string) (charset string, collation string, err error) {
	query := `SELECT DEFAULT_CHARACTER_SET_NAME, DEFAULT_COLLATION_NAME 
	          FROM information_schema.SCHEMATA 
	          WHERE SCHEMA_NAME = ?`

	err = r.db.QueryRow(query, database).Scan(&charset, &collation)
	if err != nil {
		// 如果查询失败，使用默认值
		charset = "utf8mb4"
		collation = "utf8mb4_unicode_ci"
		err = nil
	}
	return
}

// Close 关闭连接
func (r *MySQLReader) Close() error {
	if r.db != nil {
		return r.db.Close()
	}
	return nil
}
