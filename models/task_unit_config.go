package models

import (
	"time"
)

// TaskUnitConfig 任务单元配置模型
type TaskUnitConfig struct {
	ID        string    `gorm:"primaryKey;size:36" json:"id"`
	TaskID    string    `gorm:"size:36;not null;uniqueIndex:uk_task_unit;index" json:"task_id"` // 不使用外键
	UnitName  string    `gorm:"size:200;not null;uniqueIndex:uk_task_unit" json:"unit_name"`    // 表名或索引名
	UnitType  string    `gorm:"size:20;not null;index" json:"unit_type"`                        // table/index
	CreatedAt time.Time `json:"created_at"`
}

// TableName 指定表名
func (TaskUnitConfig) TableName() string {
	return "task_unit_configs"
}
