package services

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// MySQLWriter MySQL数据写入器
type MySQLWriter struct {
	db        *sql.DB
	tableName string
}

// NewMySQLWriter 创建MySQL写入器
func NewMySQLWriter(host string, port int, username, password, database, tableName string) (*MySQLWriter, error) {
	// 校验表名
	if err := ValidateTableName(tableName); err != nil {
		return nil, fmt.Errorf("表名校验失败: %w", err)
	}

	// 构建连接字符串（连接到指定数据库）
	// 添加超时参数
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local&timeout=10s&readTimeout=30s&writeTimeout=30s",
		username, password, host, port, database)

	// 连接数据库
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("连接数据库失败: %w", err)
	}

	// 优化连接池参数,避免连接数过多
	db.SetMaxOpenConns(2)                   // 每个Writer最多2个连接
	db.SetMaxIdleConns(1)                   // 最多1个空闲连接
	db.SetConnMaxLifetime(1 * time.Minute)  // 连接1分钟后回收
	db.SetConnMaxIdleTime(30 * time.Second) // 空闲30秒后关闭

	// 测试连接
	ctx1, cancel1 := context.WithTimeout(context.Background(), 5*time.Second)
	err = db.PingContext(ctx1)
	cancel1()
	if err != nil {
		db.Close()
		return nil, fmt.Errorf("数据库连接测试失败: %w", err)
	}

	writer := &MySQLWriter{
		db:        db,
		tableName: tableName,
	}

	// 设置会话级别的锁等待超时(不再禁用外键检查)
	ctx2, cancel2 := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel2()

	_, err = db.ExecContext(ctx2, "SET SESSION lock_wait_timeout = 10")
	if err != nil {
		// 如果设置失败,继续使用默认值
	}

	return writer, nil
}

// CreateDatabaseIfNotExists 创建数据库（如果不存在）
func CreateDatabaseIfNotExists(host string, port int, username, password, database, sourceCharset, sourceCollation string) (bool, error) {
	// 校验数据库名
	if err := ValidateDatabaseName(database); err != nil {
		return false, fmt.Errorf("数据库名校验失败: %w", err)
	}

	// 连接到MySQL服务器（不指定数据库）
	dsnWithoutDB := fmt.Sprintf("%s:%s@tcp(%s:%d)/?charset=utf8mb4&parseTime=True&loc=Local",
		username, password, host, port)

	db, err := sql.Open("mysql", dsnWithoutDB)
	if err != nil {
		return false, fmt.Errorf("连接MySQL服务器失败: %w", err)
	}
	defer db.Close()

	// 测试连接
	if err := db.Ping(); err != nil {
		return false, fmt.Errorf("MySQL服务器连接测试失败: %w", err)
	}

	// 检查数据库是否存在
	var exists bool
	checkQuery := fmt.Sprintf("SELECT COUNT(*) FROM information_schema.schemata WHERE schema_name = '%s'", database)
	err = db.QueryRow(checkQuery).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("检查数据库是否存在失败: %w", err)
	}

	// 如果数据库已存在，直接返回
	if exists {
		return false, nil
	}

	// 使用源数据库的字符集和排序规则创建目标数据库
	createDBQuery := fmt.Sprintf("CREATE DATABASE `%s` CHARACTER SET %s COLLATE %s",
		database, sourceCharset, sourceCollation)

	_, err = db.Exec(createDBQuery)
	if err != nil {
		return false, fmt.Errorf("创建数据库失败: %w", err)
	}

	return true, nil
}

// WriteBatch 批量写入数据
func (w *MySQLWriter) WriteBatch(records []map[string]interface{}) error {
	if len(records) == 0 {
		return nil
	}

	// 获取列名（从第一条记录）
	var columns []string
	for col := range records[0] {
		columns = append(columns, col)
	}

	columnCount := len(columns)

	// MySQL 的占位符限制是 65535
	// 计算每批次最多能插入多少条记录
	maxPlaceholders := 65535
	maxRecordsPerBatch := maxPlaceholders / columnCount
	if maxRecordsPerBatch == 0 {
		maxRecordsPerBatch = 1
	}

	// 如果列数太多导致单条记录就超限,强制设为1
	if columnCount > maxPlaceholders {
		return fmt.Errorf("表列数 %d 超过 MySQL 占位符限制 %d", columnCount, maxPlaceholders)
	}

	// 分批插入
	for i := 0; i < len(records); i += maxRecordsPerBatch {
		end := i + maxRecordsPerBatch
		if end > len(records) {
			end = len(records)
		}
		if err := w.writeBatchInternal(records[i:end], columns); err != nil {
			return err
		}
	}

	return nil
}

// writeBatchInternal 内部批量写入方法
func (w *MySQLWriter) writeBatchInternal(records []map[string]interface{}, columns []string) error {
	// 构建INSERT语句
	placeholders := make([]string, len(records))
	values := make([]interface{}, 0, len(records)*len(columns))

	for i, record := range records {
		// 构建单行占位符 (?, ?, ?)
		rowPlaceholders := make([]string, len(columns))
		for j := range columns {
			rowPlaceholders[j] = "?"
		}
		placeholders[i] = fmt.Sprintf("(%s)", strings.Join(rowPlaceholders, ", "))

		// 收集值
		for _, col := range columns {
			values = append(values, record[col])
		}
	}

	// 构建完整SQL
	columnNames := make([]string, len(columns))
	for i, col := range columns {
		columnNames[i] = fmt.Sprintf("`%s`", col)
	}

	query := fmt.Sprintf("INSERT INTO `%s` (%s) VALUES %s",
		w.tableName,
		strings.Join(columnNames, ", "),
		strings.Join(placeholders, ", "))

	// 执行插入
	_, err := w.db.Exec(query, values...)
	if err != nil {
		return fmt.Errorf("批量插入失败: %w", err)
	}

	return nil
}

// TruncateTable 清空表
func (w *MySQLWriter) TruncateTable() error {
	query := fmt.Sprintf("TRUNCATE TABLE `%s`", w.tableName)

	// 设置5秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := w.db.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("清空表失败: %w", err)
	}
	return nil
}

// DropTable 删除表
func (w *MySQLWriter) DropTable() error {
	// 由于在连接时已经设置了 FOREIGN_KEY_CHECKS = 0,这里直接删除
	query := fmt.Sprintf("DROP TABLE IF EXISTS `%s`", w.tableName)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	startTime := time.Now()
	_, err := w.db.ExecContext(ctx, query)
	elapsed := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("删除表失败(耗时%.2fs): %w", elapsed.Seconds(), err)
	}

	return nil
}

// CreateTableLike 根据源表结构创建表
func (w *MySQLWriter) CreateTableLike(sourceDB *sql.DB, sourceTable string) error {
	return w.CreateTableLikeWithFields(sourceDB, sourceTable, nil)
}

// CreateTableLikeWithFields 根据源表结构创建表（支持字段过滤）
func (w *MySQLWriter) CreateTableLikeWithFields(sourceDB *sql.DB, sourceTable string, selectedFields []string) error {
	// 获取源表的CREATE TABLE语句
	var tableName, createSQL string
	query := fmt.Sprintf("SHOW CREATE TABLE `%s`", sourceTable)

	// 设置5秒超时
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := sourceDB.QueryRowContext(ctx, query).Scan(&tableName, &createSQL)
	if err != nil {
		return fmt.Errorf("获取源表结构失败: %w", err)
	}

	// 如果指定了字段列表，过滤字段并重建CREATE TABLE语句
	if len(selectedFields) > 0 {
		modifier := NewTableStructureModifier()
		createSQL, err = modifier.FilterFieldsAndRebuild(createSQL, selectedFields, w.tableName)
		if err != nil {
			return fmt.Errorf("过滤字段失败: %w", err)
		}
	} else {
		// 没有指定字段，只替换表名
		oldTableDef := fmt.Sprintf("CREATE TABLE `%s`", sourceTable)
		newTableDef := fmt.Sprintf("CREATE TABLE `%s`", w.tableName)
		createSQL = strings.Replace(createSQL, oldTableDef, newTableDef, 1)
	}

	// 执行创建表
	ctx2, cancel2 := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel2()

	_, err = w.db.ExecContext(ctx2, createSQL)
	if err != nil {
		return fmt.Errorf("创建表失败: %w", err)
	}

	return nil
}

// Close 关闭连接
func (w *MySQLWriter) Close() error {
	if w.db != nil {
		return w.db.Close()
	}
	return nil
}
