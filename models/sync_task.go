package models

import (
	"time"
)

// SyncTask 同步任务模型
type SyncTask struct {
	ID         string    `gorm:"primaryKey;size:36" json:"id"`
	Name       string    `gorm:"size:100;not null" json:"name"`
	SourceID   string    `gorm:"size:36;index" json:"source_id"`                    // 可空，不使用外键
	TargetID   string    `gorm:"size:36;index" json:"target_id"`                    // 可空，不使用外键
	SourceType string    `gorm:"size:20;not null" json:"source_type"`               // mysql/elasticsearch
	TargetType string    `gorm:"size:20;not null" json:"target_type"`               // mysql/elasticsearch
	Config     string    `gorm:"type:text;not null" json:"config"`                  // JSON格式配置
	Status     string    `gorm:"size:20;not null;default:idle;index" json:"status"` // idle/configured（配置状态）
	IsRunning  bool      `gorm:"not null;default:false;index" json:"is_running"`    // 是否正在运行
	SyncMode   string    `gorm:"size:20;not null;default:full" json:"sync_mode"`    // full/incremental
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`

	// 关联（不存储到数据库，禁用外键约束）
	SourceConn *DataSource `gorm:"foreignKey:SourceID;references:ID;constraint:-" json:"source_conn,omitempty"`
	TargetConn *DataSource `gorm:"foreignKey:TargetID;references:ID;constraint:-" json:"target_conn,omitempty"`
}

// TableName 指定表名
func (SyncTask) TableName() string {
	return "sync_tasks"
}
