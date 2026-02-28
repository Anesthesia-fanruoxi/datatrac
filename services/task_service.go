package services

import (
	"datatrace/database"
	"datatrace/models"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
)

// TaskService 任务服务
type TaskService struct {
	dsService *DataSourceService
}

// NewTaskService 创建任务服务
func NewTaskService() *TaskService {
	return &TaskService{
		dsService: NewDataSourceService(),
	}
}

// CreateTaskRequest 创建任务请求
type CreateTaskRequest struct {
	Name       string `json:"name" binding:"required"`
	SourceType string `json:"source_type" binding:"required"`
	TargetType string `json:"target_type" binding:"required"`
	Remark     string `json:"remark"`
}

// UpdateTaskConfigRequest 更新任务配置请求
type UpdateTaskConfigRequest struct {
	SourceID          string              `json:"source_id" binding:"required"`
	TargetID          string              `json:"target_id" binding:"required"`
	SelectedDatabases []DatabaseSelection `json:"selected_databases"`
	SyncConfig        SyncConfigParams    `json:"sync_config"`
}

// DatabaseSelection 数据库选择
type DatabaseSelection struct {
	Database           string        `json:"database"`
	SourceDatabase     string        `json:"source_database"`
	IsDatabaseModified bool          `json:"is_database_modified"`
	Tables             []TableConfig `json:"tables"`
}

// TableConfig 表配置
type TableConfig struct {
	SourceTable string `json:"source_table"`
	TargetTable string `json:"target_table"`
	IsModified  bool   `json:"is_modified"`
}

// SyncConfigParams 同步配置参数
type SyncConfigParams struct {
	BatchSize           int    `json:"batch_size"`
	ThreadCount         int    `json:"thread_count"`
	SyncMode            string `json:"sync_mode"` // full/incremental
	ErrorStrategy       string `json:"error_strategy"`
	TableExistsStrategy string `json:"table_exists_strategy"`
}

// TaskConfig 任务配置（存储在config字段的JSON）
type TaskConfig struct {
	SourceID          string              `json:"source_id"`
	TargetID          string              `json:"target_id"`
	SelectedDatabases []DatabaseSelection `json:"selected_databases"`
	SyncConfig        SyncConfigParams    `json:"sync_config"`
}

// Create 创建任务
func (s *TaskService) Create(req *CreateTaskRequest) (*models.SyncTask, error) {
	// 验证
	if err := s.validateCreateRequest(req); err != nil {
		return nil, err
	}

	// 检查名称唯一性
	var count int64
	database.DB.Model(&models.SyncTask{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("任务名称已存在")
	}

	// 创建任务（初始配置为空，数据源ID稍后配置时设置）
	task := &models.SyncTask{
		ID:         uuid.New().String(),
		Name:       req.Name,
		SourceType: req.SourceType,
		TargetType: req.TargetType,
		SourceID:   "", // 初始为空
		TargetID:   "", // 初始为空
		Config:     "{}",
		Status:     "idle",
		SyncMode:   "auto",
	}

	if err := database.DB.Create(task).Error; err != nil {
		return nil, fmt.Errorf("创建失败: %w", err)
	}

	return task, nil
}

// List 获取任务列表
func (s *TaskService) List() ([]models.SyncTask, error) {
	var list []models.SyncTask
	if err := database.DB.Preload("SourceConn").Preload("TargetConn").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetByID 根据ID获取任务
func (s *TaskService) GetByID(id string) (*models.SyncTask, error) {
	var task models.SyncTask
	if err := database.DB.Preload("SourceConn").Preload("TargetConn").First(&task, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &task, nil
}

// UpdateConfig 更新任务配置
func (s *TaskService) UpdateConfig(id string, req *UpdateTaskConfigRequest) (*models.SyncTask, error) {
	// 查询任务
	task, err := s.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("任务不存在")
	}

	// 验证数据源
	if err := s.validateDataSources(task, req.SourceID, req.TargetID); err != nil {
		return nil, err
	}

	// 构建配置
	config := TaskConfig{
		SourceID:          req.SourceID,
		TargetID:          req.TargetID,
		SelectedDatabases: req.SelectedDatabases,
		SyncConfig:        req.SyncConfig,
	}

	configJSON, err := json.Marshal(config)
	if err != nil {
		return nil, fmt.Errorf("配置序列化失败: %w", err)
	}

	// 更新任务
	task.SourceID = req.SourceID
	task.TargetID = req.TargetID
	task.Config = string(configJSON)
	task.Status = "configured" // 更新状态为已配置

	// 从sync_config中提取sync_mode并更新到任务字段
	if req.SyncConfig.SyncMode != "" {
		task.SyncMode = req.SyncConfig.SyncMode
	}

	if err := database.DB.Save(task).Error; err != nil {
		return nil, fmt.Errorf("更新失败: %w", err)
	}

	// 生成任务单元配置
	if err := s.generateTaskUnits(task); err != nil {
		return nil, fmt.Errorf("生成任务单元失败: %w", err)
	}

	return task, nil
}

// Delete 删除任务
func (s *TaskService) Delete(id string) error {
	// 检查任务状态
	task, err := s.GetByID(id)
	if err != nil {
		return fmt.Errorf("任务不存在")
	}

	if task.IsRunning {
		return fmt.Errorf("任务正在运行，无法删除")
	}

	// 级联删除任务单元运行记录
	database.DB.Where("task_id = ?", id).Delete(&models.TaskUnitRuntime{})

	// 级联删除任务单元配置
	database.DB.Where("task_id = ?", id).Delete(&models.TaskUnitConfig{})

	// 删除任务本身
	if err := database.DB.Delete(&models.SyncTask{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}

	return nil
}

// generateTaskUnits 生成任务单元配置
func (s *TaskService) generateTaskUnits(task *models.SyncTask) error {
	// 解析配置
	var config TaskConfig
	if err := json.Unmarshal([]byte(task.Config), &config); err != nil {
		return err
	}

	// 删除旧的任务单元配置
	database.DB.Where("task_id = ?", task.ID).Delete(&models.TaskUnitConfig{})

	// 生成新的任务单元配置
	var units []models.TaskUnitConfig
	for _, db := range config.SelectedDatabases {
		for _, tableConfig := range db.Tables {
			// 使用目标表名作为单元名称
			targetDatabase := db.Database
			targetTable := tableConfig.TargetTable
			if targetTable == "" {
				targetTable = tableConfig.SourceTable
			}

			unit := models.TaskUnitConfig{
				ID:       uuid.New().String(),
				TaskID:   task.ID,
				UnitName: fmt.Sprintf("%s.%s", targetDatabase, targetTable),
				UnitType: "table",
			}
			units = append(units, unit)
		}
	}

	// 批量插入
	if len(units) > 0 {
		if err := database.DB.Create(&units).Error; err != nil {
			return err
		}
	}

	return nil
}

// validateCreateRequest 验证创建请求
func (s *TaskService) validateCreateRequest(req *CreateTaskRequest) error {
	if req.Name == "" {
		return fmt.Errorf("任务名称不能为空")
	}
	if req.SourceType != "mysql" && req.SourceType != "elasticsearch" {
		return fmt.Errorf("源类型无效")
	}
	if req.TargetType != "mysql" && req.TargetType != "elasticsearch" {
		return fmt.Errorf("目标类型无效")
	}
	return nil
}

// validateDataSources 验证数据源
func (s *TaskService) validateDataSources(task *models.SyncTask, sourceID, targetID string) error {
	// 获取数据源
	source, err := s.dsService.GetByID(sourceID)
	if err != nil {
		return fmt.Errorf("源数据源不存在")
	}

	target, err := s.dsService.GetByID(targetID)
	if err != nil {
		return fmt.Errorf("目标数据源不存在")
	}

	// 验证类型匹配
	if source.Type != task.SourceType {
		return fmt.Errorf("源数据源类型不匹配")
	}
	if target.Type != task.TargetType {
		return fmt.Errorf("目标数据源类型不匹配")
	}

	// 验证不能是同一个数据源
	if sourceID == targetID {
		return fmt.Errorf("源和目标不能是同一个数据源")
	}

	return nil
}
