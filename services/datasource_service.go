package services

import (
	"datatrace/database"
	"datatrace/models"
	"datatrace/utils"
	"fmt"

	_ "github.com/go-sql-driver/mysql" // MySQL 驱动
	"github.com/google/uuid"
)

// DataSourceService 数据源服务
type DataSourceService struct {
	crypto *utils.CryptoService
}

// NewDataSourceService 创建数据源服务
func NewDataSourceService() *DataSourceService {
	return &DataSourceService{
		crypto: utils.NewCryptoService(),
	}
}

// CreateDataSourceRequest 创建数据源请求
type CreateDataSourceRequest struct {
	Name         string  `json:"name" binding:"required"`
	Type         string  `json:"type" binding:"required"`
	Host         string  `json:"host" binding:"required"`
	Port         int     `json:"port" binding:"required"`
	CredentialID *string `json:"credential_id"` // 凭据ID（可选）
	Username     string  `json:"username"`      // 用户名（凭据为空时必填）
	Password     string  `json:"password"`      // 密码（凭据为空时必填）
	DatabaseName string  `json:"database_name"`
}

// Create 创建数据源
func (s *DataSourceService) Create(req *CreateDataSourceRequest) (*models.DataSource, error) {
	// 验证
	if err := s.validate(req); err != nil {
		return nil, err
	}

	// 检查名称唯一性
	var count int64
	database.DB.Model(&models.DataSource{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("数据源名称已存在")
	}

	// 创建数据源
	ds := &models.DataSource{
		ID:           uuid.New().String(),
		Name:         req.Name,
		Type:         req.Type,
		Host:         req.Host,
		Port:         req.Port,
		CredentialID: req.CredentialID,
		DatabaseName: req.DatabaseName,
	}

	// 如果使用凭据，验证凭据是否存在
	if req.CredentialID != nil && *req.CredentialID != "" {
		var credential models.Credential
		if err := database.DB.First(&credential, "id = ?", *req.CredentialID).Error; err != nil {
			return nil, fmt.Errorf("凭据不存在")
		}
		// 使用凭据时，不需要单独的用户名和密码
		ds.Username = ""
		ds.Password = ""
	} else {
		// 不使用凭据时，加密密码
		encryptedPassword, err := s.crypto.Encrypt(req.Password)
		if err != nil {
			return nil, fmt.Errorf("密码加密失败: %w", err)
		}
		ds.Username = req.Username
		ds.Password = encryptedPassword
	}

	if err := database.DB.Create(ds).Error; err != nil {
		return nil, fmt.Errorf("创建失败: %w", err)
	}

	return ds, nil
}

// List 获取数据源列表
func (s *DataSourceService) List() ([]models.DataSource, error) {
	var list []models.DataSource
	if err := database.DB.Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetByID 根据ID获取数据源
func (s *DataSourceService) GetByID(id string) (*models.DataSource, error) {
	var ds models.DataSource
	if err := database.DB.First(&ds, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &ds, nil
}

// Update 更新数据源
func (s *DataSourceService) Update(id string, req *CreateDataSourceRequest) (*models.DataSource, error) {
	// 查询数据源
	ds, err := s.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("数据源不存在")
	}

	// 验证
	if err := s.validate(req); err != nil {
		return nil, err
	}

	// 检查名称唯一性（排除自身）
	var count int64
	database.DB.Model(&models.DataSource{}).
		Where("name = ? AND id != ?", req.Name, id).
		Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("数据源名称已存在")
	}

	// 更新字段
	ds.Name = req.Name
	ds.Type = req.Type
	ds.Host = req.Host
	ds.Port = req.Port
	ds.CredentialID = req.CredentialID
	ds.DatabaseName = req.DatabaseName

	// 如果使用凭据，验证凭据是否存在
	if req.CredentialID != nil && *req.CredentialID != "" {
		var credential models.Credential
		if err := database.DB.First(&credential, "id = ?", *req.CredentialID).Error; err != nil {
			return nil, fmt.Errorf("凭据不存在")
		}
		// 使用凭据时，清空用户名和密码
		ds.Username = ""
		ds.Password = ""
	} else {
		// 不使用凭据时，更新用户名和密码
		ds.Username = req.Username
		if req.Password != "" {
			encryptedPassword, err := s.crypto.Encrypt(req.Password)
			if err != nil {
				return nil, fmt.Errorf("密码加密失败: %w", err)
			}
			ds.Password = encryptedPassword
		}
	}

	if err := database.DB.Save(ds).Error; err != nil {
		return nil, fmt.Errorf("更新失败: %w", err)
	}

	return ds, nil
}

// Delete 删除数据源
func (s *DataSourceService) Delete(id string) error {
	// 检查是否被任务使用
	var count int64
	database.DB.Model(&models.SyncTask{}).
		Where("source_id = ? OR target_id = ?", id, id).
		Count(&count)
	if count > 0 {
		return fmt.Errorf("数据源正在被 %d 个任务使用，无法删除", count)
	}

	if err := database.DB.Delete(&models.DataSource{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}

	return nil
}

// validate 验证请求
func (s *DataSourceService) validate(req *CreateDataSourceRequest) error {
	if req.Name == "" {
		return fmt.Errorf("数据源名称不能为空")
	}
	if req.Type != "mysql" && req.Type != "elasticsearch" {
		return fmt.Errorf("数据源类型无效")
	}
	if req.Host == "" {
		return fmt.Errorf("主机地址不能为空")
	}
	if req.Port <= 0 || req.Port > 65535 {
		return fmt.Errorf("端口号无效")
	}

	// 如果没有使用凭据，则用户名和密码必填
	if req.CredentialID == nil || *req.CredentialID == "" {
		if req.Username == "" {
			return fmt.Errorf("用户名不能为空")
		}
		if req.Password == "" {
			return fmt.Errorf("密码不能为空")
		}
	}

	return nil
}

// 连接测试相关方法已拆分到 datasource_test_service.go
