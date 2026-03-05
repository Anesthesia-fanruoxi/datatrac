package services

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/fsnotify/fsnotify"
)

// LogFileWatcher 日志文件监听器
type LogFileWatcher struct {
	taskID   string
	category string
	filePath string
	watcher  *fsnotify.Watcher
	client   chan SSEMessage
	done     <-chan struct{}
	offset   int64 // 当前读取位置
}

// NewLogFileWatcher 创建日志文件监听器
func NewLogFileWatcher(taskID string, category string, client chan SSEMessage, done <-chan struct{}) (*LogFileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("创建文件监听器失败: %v", err)
	}

	filePath := filepath.Join("logs", taskID, category+".log")

	return &LogFileWatcher{
		taskID:   taskID,
		category: category,
		filePath: filePath,
		watcher:  watcher,
		client:   client,
		done:     done,
		offset:   0,
	}, nil
}

// Start 启动监听
func (w *LogFileWatcher) Start() error {
	// 1. 读取文件最后1000行（初始数据）
	if err := w.sendInitialLogs(); err != nil {
		fmt.Printf("发送初始日志失败: %v\n", err)
	}

	// 2. 确保文件存在
	if err := w.ensureFileExists(); err != nil {
		return fmt.Errorf("确保文件存在失败: %v", err)
	}

	// 3. 添加文件到监听器
	if err := w.watcher.Add(w.filePath); err != nil {
		return fmt.Errorf("添加文件监听失败: %v", err)
	}

	// 4. 启动监听循环
	go w.watchLoop()

	return nil
}

// sendInitialLogs 发送初始日志（最后1000行）
func (w *LogFileWatcher) sendInitialLogs() error {
	logService := NewTaskLogService()
	logs, err := logService.GetTaskLogs(w.taskID, w.category, 1000)
	if err != nil {
		return err
	}

	if len(logs) > 0 {
		// 发送初始日志
		select {
		case w.client <- SSEMessage{
			Event: "log",
			Data:  logs,
		}:
		case <-w.done:
			return nil
		}

		// 更新偏移量到文件末尾
		if file, err := os.Stat(w.filePath); err == nil {
			w.offset = file.Size()
		}
	}

	return nil
}

// ensureFileExists 确保文件存在
func (w *LogFileWatcher) ensureFileExists() error {
	// 确保目录存在
	dir := filepath.Dir(w.filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	// 如果文件不存在，创建空文件
	if _, err := os.Stat(w.filePath); os.IsNotExist(err) {
		file, err := os.Create(w.filePath)
		if err != nil {
			return err
		}
		file.Close()
	}

	return nil
}

// watchLoop 监听循环
func (w *LogFileWatcher) watchLoop() {
	defer w.watcher.Close()

	for {
		select {
		case <-w.done:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// 只处理写入事件
			if event.Op&fsnotify.Write == fsnotify.Write {
				w.readNewContent()
			}

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			fmt.Printf("文件监听错误: %v\n", err)
		}
	}
}

// readNewContent 读取新内容
func (w *LogFileWatcher) readNewContent() {
	file, err := os.Open(w.filePath)
	if err != nil {
		fmt.Printf("打开文件失败: %v\n", err)
		return
	}
	defer file.Close()

	// 跳到上次读取的位置
	if _, err := file.Seek(w.offset, 0); err != nil {
		fmt.Printf("文件seek失败: %v\n", err)
		return
	}

	// 读取新行
	scanner := bufio.NewScanner(file)
	logs := make([]TaskLog, 0)

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		// 解析日志行（使用task_log_service.go中的parseLogLine）
		if log := parseLogLine(line); log != nil {
			logs = append(logs, *log)
		}

		// 更新偏移量
		w.offset += int64(len(line)) + 1 // +1 for newline
	}

	if err := scanner.Err(); err != nil {
		fmt.Printf("读取文件失败: %v\n", err)
		return
	}

	// 发送新日志
	if len(logs) > 0 {
		select {
		case w.client <- SSEMessage{
			Event: "log",
			Data:  logs,
		}:
		case <-w.done:
			return
		case <-time.After(1 * time.Second):
			// 超时，跳过
		}
	}
}
