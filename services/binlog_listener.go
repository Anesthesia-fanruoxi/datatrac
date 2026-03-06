package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/go-mysql-org/go-mysql/mysql"
	"github.com/go-mysql-org/go-mysql/replication"
)

// BinlogListener Binlog 监听器
type BinlogListener struct {
	// 连接配置
	host     string
	port     int
	username string
	password string
	serverID uint32

	// 监听配置
	startFile string                     // 起始 Binlog 文件
	startPos  uint32                     // 起始 Binlog 位置
	tables    map[string]map[string]bool // database -> table -> true

	// 队列
	queue BinlogQueue

	// 同步器
	syncer *replication.BinlogSyncer

	// 上下文
	ctx    context.Context
	cancel context.CancelFunc

	// 日志服务
	logService *TaskLogService
	taskID     string

	// 表结构缓存: "database.table" -> []columnName
	tableSchemaCache map[string][]string
	sourceDB         *sql.DB // 源数据库连接，用于查询表结构
}

// BinlogListenerConfig 监听器配置
type BinlogListenerConfig struct {
	Host      string
	Port      int
	Username  string
	Password  string
	ServerID  uint32
	StartFile string
	StartPos  uint32
	Tables    map[string][]string // database -> []table
	Queue     BinlogQueue
	TaskID    string
	SourceDB  *sql.DB // 源数据库连接
}

// NewBinlogListener 创建 Binlog 监听器
func NewBinlogListener(config *BinlogListenerConfig) (*BinlogListener, error) {
	if config.Queue == nil {
		return nil, fmt.Errorf("队列不能为空")
	}

	if config.ServerID == 0 {
		config.ServerID = 100 // 默认 Server ID
	}

	// 转换表过滤配置
	tables := make(map[string]map[string]bool)
	for db, tbls := range config.Tables {
		if tables[db] == nil {
			tables[db] = make(map[string]bool)
		}
		for _, tbl := range tbls {
			tables[db][tbl] = true
		}
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &BinlogListener{
		host:             config.Host,
		port:             config.Port,
		username:         config.Username,
		password:         config.Password,
		serverID:         config.ServerID,
		startFile:        config.StartFile,
		startPos:         config.StartPos,
		tables:           tables,
		queue:            config.Queue,
		ctx:              ctx,
		cancel:           cancel,
		logService:       NewTaskLogService(),
		taskID:           config.TaskID,
		tableSchemaCache: make(map[string][]string),
		sourceDB:         config.SourceDB,
	}, nil
}

// Start 启动监听
func (l *BinlogListener) Start() error {
	// 创建 Binlog 同步器配置
	cfg := replication.BinlogSyncerConfig{
		ServerID: l.serverID,
		Flavor:   "mysql",
		Host:     l.host,
		Port:     uint16(l.port),
		User:     l.username,
		Password: l.password,
	}

	// 创建同步器
	l.syncer = replication.NewBinlogSyncer(cfg)

	// 开始同步
	pos := mysql.Position{
		Name: l.startFile,
		Pos:  l.startPos,
	}

	l.logService.Info(l.taskID, fmt.Sprintf("Binlog 监听器启动，从位置 %s:%d 开始", pos.Name, pos.Pos))

	streamer, err := l.syncer.StartSync(pos)
	if err != nil {
		return fmt.Errorf("启动 Binlog 同步失败: %v", err)
	}

	// 启动事件处理循环
	go l.eventLoop(streamer)

	return nil
}

// eventLoop 事件处理循环
func (l *BinlogListener) eventLoop(streamer *replication.BinlogStreamer) {
	defer func() {
		if r := recover(); r != nil {
			l.logService.Error(l.taskID, fmt.Sprintf("Binlog 监听器 panic: %v", r))
		}
	}()

	for {
		select {
		case <-l.ctx.Done():
			l.logService.Info(l.taskID, "Binlog 监听器收到停止信号")
			return

		default:
			// 获取事件（超时 1 秒）
			ctx, cancel := context.WithTimeout(l.ctx, 1*time.Second)
			ev, err := streamer.GetEvent(ctx)
			cancel()

			if err != nil {
				if err == context.DeadlineExceeded {
					// 超时，继续下一次循环
					continue
				}
				l.logService.Error(l.taskID, fmt.Sprintf("获取 Binlog 事件失败: %v", err))
				time.Sleep(1 * time.Second)
				continue
			}

			// 处理事件
			if err := l.handleEvent(ev); err != nil {
				l.logService.Error(l.taskID, fmt.Sprintf("处理 Binlog 事件失败: %v", err))
			}
		}
	}
}

// handleEvent 处理单个事件
func (l *BinlogListener) handleEvent(ev *replication.BinlogEvent) error {
	switch ev.Header.EventType {
	case replication.WRITE_ROWS_EVENTv1, replication.WRITE_ROWS_EVENTv2:
		return l.handleInsert(ev)

	case replication.UPDATE_ROWS_EVENTv1, replication.UPDATE_ROWS_EVENTv2:
		return l.handleUpdate(ev)

	case replication.DELETE_ROWS_EVENTv1, replication.DELETE_ROWS_EVENTv2:
		return l.handleDelete(ev)

	case replication.ROTATE_EVENT:
		// Binlog 文件轮转
		rotateEvent := ev.Event.(*replication.RotateEvent)
		l.startFile = string(rotateEvent.NextLogName)
		l.startPos = uint32(rotateEvent.Position)
		l.logService.Info(l.taskID, fmt.Sprintf("Binlog 文件轮转: %s:%d", l.startFile, l.startPos))
	}

	return nil
}

// handleInsert 处理 INSERT 事件
func (l *BinlogListener) handleInsert(ev *replication.BinlogEvent) error {
	rowsEvent := ev.Event.(*replication.RowsEvent)

	// 获取数据库和表名
	database := string(rowsEvent.Table.Schema)
	table := string(rowsEvent.Table.Table)

	// 检查是否需要过滤
	if !l.shouldProcess(database, table) {
		return nil
	}

	// 处理每一行
	for _, row := range rowsEvent.Rows {
		data := l.rowToMap(database, table, rowsEvent.Table.ColumnName, row)

		// 验证数据是否为空
		if len(data) == 0 {
			l.logService.Warning(l.taskID, fmt.Sprintf(
				"INSERT 事件数据为空，跳过: %s.%s, 列数=%d, 行数据长度=%d",
				database, table,
				len(rowsEvent.Table.ColumnName),
				len(row),
			))
			continue
		}

		event := &BinlogEvent{
			Type:       "INSERT",
			Database:   database,
			Table:      table,
			Timestamp:  ev.Header.Timestamp,
			Data:       data,
			BinlogFile: l.startFile,
			BinlogPos:  ev.Header.LogPos,
		}

		// 推送到队列
		if err := l.queue.Push(event); err != nil {
			return fmt.Errorf("推送 INSERT 事件到队列失败: %v", err)
		}
	}

	return nil
}

// handleUpdate 处理 UPDATE 事件
func (l *BinlogListener) handleUpdate(ev *replication.BinlogEvent) error {
	rowsEvent := ev.Event.(*replication.RowsEvent)

	database := string(rowsEvent.Table.Schema)
	table := string(rowsEvent.Table.Table)

	if !l.shouldProcess(database, table) {
		return nil
	}

	// UPDATE 事件的 Rows 是成对出现的：[旧值, 新值, 旧值, 新值, ...]
	for i := 0; i < len(rowsEvent.Rows); i += 2 {
		if i+1 >= len(rowsEvent.Rows) {
			l.logService.Warning(l.taskID, fmt.Sprintf(
				"UPDATE 事件行数据不完整，跳过: %s.%s",
				database, table,
			))
			break
		}

		oldRow := rowsEvent.Rows[i]
		newRow := rowsEvent.Rows[i+1]

		oldData := l.rowToMap(database, table, rowsEvent.Table.ColumnName, oldRow)
		newData := l.rowToMap(database, table, rowsEvent.Table.ColumnName, newRow)

		// 验证数据是否为空
		if len(newData) == 0 {
			l.logService.Warning(l.taskID, fmt.Sprintf(
				"UPDATE 事件新数据为空，跳过: %s.%s",
				database, table,
			))
			continue
		}

		event := &BinlogEvent{
			Type:       "UPDATE",
			Database:   database,
			Table:      table,
			Timestamp:  ev.Header.Timestamp,
			Data:       newData,
			OldData:    oldData,
			BinlogFile: l.startFile,
			BinlogPos:  ev.Header.LogPos,
		}

		if err := l.queue.Push(event); err != nil {
			return fmt.Errorf("推送 UPDATE 事件到队列失败: %v", err)
		}
	}

	return nil
}

// handleDelete 处理 DELETE 事件
func (l *BinlogListener) handleDelete(ev *replication.BinlogEvent) error {
	rowsEvent := ev.Event.(*replication.RowsEvent)

	database := string(rowsEvent.Table.Schema)
	table := string(rowsEvent.Table.Table)

	if !l.shouldProcess(database, table) {
		return nil
	}

	for _, row := range rowsEvent.Rows {
		data := l.rowToMap(database, table, rowsEvent.Table.ColumnName, row)

		// 验证数据是否为空
		if len(data) == 0 {
			l.logService.Warning(l.taskID, fmt.Sprintf(
				"DELETE 事件数据为空，跳过: %s.%s",
				database, table,
			))
			continue
		}

		event := &BinlogEvent{
			Type:       "DELETE",
			Database:   database,
			Table:      table,
			Timestamp:  ev.Header.Timestamp,
			Data:       data,
			BinlogFile: l.startFile,
			BinlogPos:  ev.Header.LogPos,
		}

		if err := l.queue.Push(event); err != nil {
			return fmt.Errorf("推送 DELETE 事件到队列失败: %v", err)
		}
	}

	return nil
}

// shouldProcess 判断是否需要处理该表
func (l *BinlogListener) shouldProcess(database, table string) bool {
	// 如果没有配置过滤，则处理所有表
	if len(l.tables) == 0 {
		return true
	}

	// 检查是否在过滤列表中
	if dbTables, ok := l.tables[database]; ok {
		if len(dbTables) == 0 {
			// 处理该数据库的所有表
			return true
		}
		return dbTables[table]
	}

	return false
}

// rowToMap 将行数据转换为 map
func (l *BinlogListener) rowToMap(database, table string, columns [][]byte, row []interface{}) map[string]interface{} {
	data := make(map[string]interface{})

	// 如果 Binlog 事件中没有列名，从数据库查询表结构
	if len(columns) == 0 && len(row) > 0 {
		columns = l.getTableColumns(database, table)
	}

	// 验证数据
	if len(columns) == 0 {
		l.logService.Warning(l.taskID, fmt.Sprintf(
			"无法获取列名: %s.%s",
			database, table,
		))
		return data
	}

	if len(row) == 0 {
		l.logService.Warning(l.taskID, "行数据为空，无法转换")
		return data
	}

	// 记录列数和行数不匹配的情况
	if len(columns) != len(row) {
		l.logService.Warning(l.taskID, fmt.Sprintf(
			"列数(%d)和行数据长度(%d)不匹配: %s.%s",
			len(columns), len(row),
			database, table,
		))
	}

	for i, col := range columns {
		if i < len(row) {
			colName := string(col)
			data[colName] = row[i]
		}
	}

	return data
}

// getTableColumns 获取表的列名（带缓存）
func (l *BinlogListener) getTableColumns(database, table string) [][]byte {
	cacheKey := fmt.Sprintf("%s.%s", database, table)

	// 检查缓存
	if columns, ok := l.tableSchemaCache[cacheKey]; ok {
		result := make([][]byte, len(columns))
		for i, col := range columns {
			result[i] = []byte(col)
		}
		return result
	}

	// 从数据库查询
	if l.sourceDB == nil {
		l.logService.Error(l.taskID, "源数据库连接为空，无法查询表结构")
		return nil
	}

	query := fmt.Sprintf(
		"SELECT COLUMN_NAME FROM INFORMATION_SCHEMA.COLUMNS WHERE TABLE_SCHEMA = ? AND TABLE_NAME = ? ORDER BY ORDINAL_POSITION",
	)

	rows, err := l.sourceDB.Query(query, database, table)
	if err != nil {
		l.logService.Error(l.taskID, fmt.Sprintf(
			"查询表结构失败: %s.%s, 错误: %v",
			database, table, err,
		))
		return nil
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var colName string
		if err := rows.Scan(&colName); err != nil {
			continue
		}
		columns = append(columns, colName)
	}

	if len(columns) == 0 {
		l.logService.Error(l.taskID, fmt.Sprintf(
			"未找到表结构: %s.%s",
			database, table,
		))
		return nil
	}

	// 缓存结果
	l.tableSchemaCache[cacheKey] = columns

	l.logService.Info(l.taskID, fmt.Sprintf(
		"已缓存表结构: %s.%s (%d列)",
		database, table, len(columns),
	))

	result := make([][]byte, len(columns))
	for i, col := range columns {
		result[i] = []byte(col)
	}
	return result
}

// Stop 停止监听
func (l *BinlogListener) Stop() error {
	l.logService.Info(l.taskID, "正在停止 Binlog 监听器...")

	// 取消上下文
	l.cancel()

	// 关闭同步器
	if l.syncer != nil {
		l.syncer.Close()
	}

	l.logService.Info(l.taskID, "Binlog 监听器已停止")
	return nil
}

// GetCurrentPosition 获取当前 Binlog 位置
func (l *BinlogListener) GetCurrentPosition() (string, uint32) {
	return l.startFile, l.startPos
}
