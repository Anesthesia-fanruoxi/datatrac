package services

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// BinlogEvent Binlog 事件结构
type BinlogEvent struct {
	ID        string                 `json:"id"`        // 事件ID
	Type      string                 `json:"type"`      // INSERT/UPDATE/DELETE
	Database  string                 `json:"database"`  // 数据库名
	Table     string                 `json:"table"`     // 表名
	Timestamp uint32                 `json:"timestamp"` // 事件时间戳
	Data      map[string]interface{} `json:"data"`      // 新数据
	OldData   map[string]interface{} `json:"old_data"`  // 旧数据(UPDATE时使用)

	// Binlog 位置信息
	BinlogFile string `json:"binlog_file"` // Binlog 文件名
	BinlogPos  uint32 `json:"binlog_pos"`  // Binlog 位置
}

// BinlogQueue 队列接口
type BinlogQueue interface {
	// Push 推送事件到队列
	Push(event *BinlogEvent) error

	// Pop 从队列获取事件（阻塞）
	Pop(ctx context.Context) (*BinlogEvent, error)

	// Ack 确认事件已处理
	Ack(eventID string) error

	// Close 关闭队列
	Close() error

	// Size 获取队列大小
	Size() (int64, error)
}

// MemoryQueue 内存队列实现（基于 channel）
type MemoryQueue struct {
	ch     chan *BinlogEvent
	closed bool
	mu     sync.RWMutex
}

// NewMemoryQueue 创建内存队列
func NewMemoryQueue(bufferSize int) *MemoryQueue {
	if bufferSize <= 0 {
		bufferSize = 1000 // 默认缓冲区大小
	}
	return &MemoryQueue{
		ch:     make(chan *BinlogEvent, bufferSize),
		closed: false,
	}
}

// Push 推送事件
func (q *MemoryQueue) Push(event *BinlogEvent) error {
	q.mu.RLock()
	defer q.mu.RUnlock()

	if q.closed {
		return fmt.Errorf("队列已关闭")
	}

	select {
	case q.ch <- event:
		return nil
	default:
		return fmt.Errorf("队列已满")
	}
}

// Pop 获取事件
func (q *MemoryQueue) Pop(ctx context.Context) (*BinlogEvent, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case event, ok := <-q.ch:
		if !ok {
			return nil, fmt.Errorf("队列已关闭")
		}
		return event, nil
	}
}

// Ack 确认事件（内存队列无需确认）
func (q *MemoryQueue) Ack(eventID string) error {
	return nil
}

// Close 关闭队列
func (q *MemoryQueue) Close() error {
	q.mu.Lock()
	defer q.mu.Unlock()

	if !q.closed {
		q.closed = true
		close(q.ch)
	}
	return nil
}

// Size 获取队列大小
func (q *MemoryQueue) Size() (int64, error) {
	return int64(len(q.ch)), nil
}

// RedisQueue Redis Stream 队列实现
type RedisQueue struct {
	client       *redis.Client
	streamKey    string // Stream 键名: binlog:{taskID}
	groupName    string // 消费组名: sync-group-{taskID}
	consumerName string // 消费者名: worker-1
	ctx          context.Context
}

// NewRedisQueue 创建 Redis 队列
func NewRedisQueue(client *redis.Client, taskID string, consumerName string) (*RedisQueue, error) {
	if client == nil {
		return nil, fmt.Errorf("Redis 客户端不能为空")
	}

	streamKey := fmt.Sprintf("binlog:%s", taskID)
	groupName := fmt.Sprintf("sync-group-%s", taskID)

	if consumerName == "" {
		consumerName = "worker-1"
	}

	q := &RedisQueue{
		client:       client,
		streamKey:    streamKey,
		groupName:    groupName,
		consumerName: consumerName,
		ctx:          context.Background(),
	}

	// 创建消费组（如果不存在）
	err := q.createConsumerGroup()
	if err != nil {
		return nil, fmt.Errorf("创建消费组失败: %v", err)
	}

	return q, nil
}

// createConsumerGroup 创建消费组
func (q *RedisQueue) createConsumerGroup() error {
	// 尝试创建消费组，如果已存在则忽略错误
	err := q.client.XGroupCreateMkStream(q.ctx, q.streamKey, q.groupName, "0").Err()
	if err != nil && err.Error() != "BUSYGROUP Consumer Group name already exists" {
		return err
	}
	return nil
}

// Push 推送事件到 Redis Stream
func (q *RedisQueue) Push(event *BinlogEvent) error {
	// 序列化事件
	data, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("序列化事件失败: %v", err)
	}

	// 推送到 Stream
	values := map[string]interface{}{
		"event": string(data),
	}

	_, err = q.client.XAdd(q.ctx, &redis.XAddArgs{
		Stream: q.streamKey,
		Values: values,
	}).Result()

	if err != nil {
		return fmt.Errorf("推送到 Redis Stream 失败: %v", err)
	}

	return nil
}

// Pop 从 Redis Stream 获取事件
func (q *RedisQueue) Pop(ctx context.Context) (*BinlogEvent, error) {
	// 从消费组读取消息（阻塞模式，超时 5 秒）
	streams, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.groupName,
		Consumer: q.consumerName,
		Streams:  []string{q.streamKey, ">"},
		Count:    1,
		Block:    5 * time.Second,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			// 没有新消息，返回 nil
			return nil, nil
		}
		return nil, fmt.Errorf("读取 Redis Stream 失败: %v", err)
	}

	if len(streams) == 0 || len(streams[0].Messages) == 0 {
		return nil, nil
	}

	// 解析消息
	msg := streams[0].Messages[0]
	eventData, ok := msg.Values["event"].(string)
	if !ok {
		return nil, fmt.Errorf("消息格式错误")
	}

	var event BinlogEvent
	if err := json.Unmarshal([]byte(eventData), &event); err != nil {
		return nil, fmt.Errorf("反序列化事件失败: %v", err)
	}

	// 保存消息 ID 用于 ACK
	event.ID = msg.ID

	return &event, nil
}

// Ack 确认事件已处理
func (q *RedisQueue) Ack(eventID string) error {
	_, err := q.client.XAck(q.ctx, q.streamKey, q.groupName, eventID).Result()
	if err != nil {
		return fmt.Errorf("ACK 失败: %v", err)
	}
	return nil
}

// Close 关闭队列（清理资源）
func (q *RedisQueue) Close() error {
	// Redis 队列不需要特殊关闭操作
	return nil
}

// Size 获取队列大小
func (q *RedisQueue) Size() (int64, error) {
	length, err := q.client.XLen(q.ctx, q.streamKey).Result()
	if err != nil {
		return 0, fmt.Errorf("获取队列大小失败: %v", err)
	}
	return length, nil
}

// GetPendingCount 获取待处理消息数量
func (q *RedisQueue) GetPendingCount() (int64, error) {
	pending, err := q.client.XPending(q.ctx, q.streamKey, q.groupName).Result()
	if err != nil {
		return 0, fmt.Errorf("获取待处理消息数量失败: %v", err)
	}
	return pending.Count, nil
}

// ClaimPendingMessages 认领超时的待处理消息
func (q *RedisQueue) ClaimPendingMessages(idleTime time.Duration) ([]*BinlogEvent, error) {
	// 获取待处理消息列表
	pending, err := q.client.XPendingExt(q.ctx, &redis.XPendingExtArgs{
		Stream: q.streamKey,
		Group:  q.groupName,
		Start:  "-",
		End:    "+",
		Count:  10,
	}).Result()

	if err != nil {
		return nil, fmt.Errorf("获取待处理消息列表失败: %v", err)
	}

	var events []*BinlogEvent

	for _, p := range pending {
		// 只认领超过指定时间的消息
		if p.Idle < idleTime {
			continue
		}

		// 认领消息
		msgs, err := q.client.XClaim(q.ctx, &redis.XClaimArgs{
			Stream:   q.streamKey,
			Group:    q.groupName,
			Consumer: q.consumerName,
			MinIdle:  idleTime,
			Messages: []string{p.ID},
		}).Result()

		if err != nil {
			continue
		}

		for _, msg := range msgs {
			eventData, ok := msg.Values["event"].(string)
			if !ok {
				continue
			}

			var event BinlogEvent
			if err := json.Unmarshal([]byte(eventData), &event); err != nil {
				continue
			}

			event.ID = msg.ID
			events = append(events, &event)
		}
	}

	return events, nil
}
