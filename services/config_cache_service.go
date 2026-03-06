package services

import (
	"context"
	"datatrace/database"
	"datatrace/models"
	"encoding/json"
	"fmt"
	"time"
)

// ConfigCacheService Redis配置缓存服务
type ConfigCacheService struct{}

// NewConfigCacheService 创建配置缓存服务
func NewConfigCacheService() *ConfigCacheService {
	return &ConfigCacheService{}
}

// LoadTaskConfigToRedis 从MySQL加载任务配置到Redis
func (s *ConfigCacheService) LoadTaskConfigToRedis(taskID string) error {
	if !database.IsRedisEnabled() {
		return fmt.Errorf("Redis未启用")
	}

	// 从MySQL查询任务
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 解析配置
	var config TaskConfig
	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		return fmt.Errorf("解析配置失败: %w", err)
	}

	// 存储到Redis
	ctx := context.Background()
	key := fmt.Sprintf("task:config:%s", taskID)

	configJSON, err := json.Marshal(config)
	if err != nil {
		return fmt.Errorf("序列化配置失败: %w", err)
	}

	if err := database.RedisClient.Set(ctx, key, configJSON, 0).Err(); err != nil {
		return fmt.Errorf("写入Redis失败: %w", err)
	}

	return nil
}

// GetTaskConfigFromRedis 从Redis读取任务配置
func (s *ConfigCacheService) GetTaskConfigFromRedis(taskID string) (*TaskConfig, error) {
	if !database.IsRedisEnabled() {
		return nil, fmt.Errorf("Redis未启用")
	}

	ctx := context.Background()
	key := fmt.Sprintf("task:config:%s", taskID)

	configJSON, err := database.RedisClient.Get(ctx, key).Result()
	if err != nil {
		// 如果Redis中没有，尝试从MySQL加载
		if err := s.LoadTaskConfigToRedis(taskID); err != nil {
			return nil, fmt.Errorf("从MySQL加载配置失败: %w", err)
		}
		// 重新读取
		configJSON, err = database.RedisClient.Get(ctx, key).Result()
		if err != nil {
			return nil, fmt.Errorf("读取Redis失败: %w", err)
		}
	}

	var config TaskConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &config, nil
}

// DeleteTaskConfigFromRedis 删除Redis中的任务配置
func (s *ConfigCacheService) DeleteTaskConfigFromRedis(taskID string) error {
	if !database.IsRedisEnabled() {
		return nil // Redis未启用时不报错
	}

	ctx := context.Background()
	key := fmt.Sprintf("task:config:%s", taskID)

	if err := database.RedisClient.Del(ctx, key).Err(); err != nil {
		return fmt.Errorf("删除Redis配置失败: %w", err)
	}

	return nil
}

// InitAllTaskConfigs 服务启动时加载所有任务配置到Redis
func (s *ConfigCacheService) InitAllTaskConfigs() error {
	if !database.IsRedisEnabled() {
		return nil
	}

	// 查询所有已配置的任务
	var tasks []models.SyncTask
	if err := database.DB.Where("status = ?", "configured").Find(&tasks).Error; err != nil {
		return fmt.Errorf("查询任务失败: %w", err)
	}

	for _, task := range tasks {
		s.LoadTaskConfigToRedis(task.ID)
	}

	return nil
}

// ReloadTaskConfig 重新加载任务配置（用于配置修改后）
func (s *ConfigCacheService) ReloadTaskConfig(taskID string) error {
	// 先删除旧配置
	if err := s.DeleteTaskConfigFromRedis(taskID); err != nil {
		return err
	}

	// 重新加载
	return s.LoadTaskConfigToRedis(taskID)
}

// GetTaskConfigWithFallback 获取任务配置（优先Redis，失败则从MySQL）
func (s *ConfigCacheService) GetTaskConfigWithFallback(taskID string) (*TaskConfig, error) {
	// 尝试从Redis读取
	if database.IsRedisEnabled() {
		config, err := s.GetTaskConfigFromRedis(taskID)
		if err == nil {
			return config, nil
		}
	}

	// 从MySQL读取
	var task models.SyncTask
	if err := database.DB.First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("任务不存在: %w", err)
	}

	var config TaskConfig
	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		return nil, fmt.Errorf("解析配置失败: %w", err)
	}

	return &config, nil
}

// ClearAllTaskConfigs 清除所有任务配置（用于测试或重置）
func (s *ConfigCacheService) ClearAllTaskConfigs() error {
	if !database.IsRedisEnabled() {
		return nil
	}

	ctx := context.Background()
	pattern := "task:config:*"

	// 使用SCAN命令遍历所有匹配的key
	iter := database.RedisClient.Scan(ctx, 0, pattern, 0).Iterator()
	for iter.Next(ctx) {
		database.RedisClient.Del(ctx, iter.Val())
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("扫描Redis失败: %w", err)
	}

	return nil
}

// GetTaskConfigTTL 获取配置的TTL（用于监控）
func (s *ConfigCacheService) GetTaskConfigTTL(taskID string) (time.Duration, error) {
	if !database.IsRedisEnabled() {
		return 0, fmt.Errorf("Redis未启用")
	}

	ctx := context.Background()
	key := fmt.Sprintf("task:config:%s", taskID)

	ttl, err := database.RedisClient.TTL(ctx, key).Result()
	if err != nil {
		return 0, err
	}

	return ttl, nil
}
