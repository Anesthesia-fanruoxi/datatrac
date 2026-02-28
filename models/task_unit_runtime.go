package models

import (
	"time"
)

// TaskUnitRuntime 任务单元运行记录模型
type TaskUnitRuntime struct {
	ID                 string     `gorm:"primaryKey;size:36" json:"id"`
	TaskID             string     `gorm:"size:36;not null;uniqueIndex:uk_task_unit;index:idx_task_status" json:"task_id"`
	UnitName           string     `gorm:"size:200;not null;uniqueIndex:uk_task_unit" json:"unit_name"`
	Status             string     `gorm:"size:20;not null;index;index:idx_task_status" json:"status"` // pending/running/completed/failed/paused
	TotalRecords       int64      `gorm:"default:0" json:"total_records"`
	ProcessedRecords   int64      `gorm:"default:0" json:"processed_records"`
	ErrorMessage       string     `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt          *time.Time `json:"started_at,omitempty"`
	UpdatedAt          time.Time  `json:"updated_at"`
	LastProcessedBatch *int       `json:"last_processed_batch,omitempty"` // 预留字段
}

// TableName 指定表名
func (TaskUnitRuntime) TableName() string {
	return "task_unit_runtimes"
}
