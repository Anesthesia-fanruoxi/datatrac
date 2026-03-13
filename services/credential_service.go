package services

import (
	"datatrace/database"
	"datatrace/models"
	"datatrace/utils"
	"fmt"

	"github.com/google/uuid"
)

// CredentialService 凭据服务
type CredentialService struct {
	crypto *utils.CryptoService
}

// NewCredentialService 创建凭据服务
func NewCredentialService() *CredentialService {
	return &CredentialService{
		crypto: utils.NewCryptoService(),
	}
}

// CreateCredentialRequest 创建凭据请求
type CreateCredentialRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
}

// Create 创建凭据
func (s *CredentialService) Create(req *CreateCredentialRequest) (*models.Credential, error) {
	// 验证
	if err := s.validate(req); err != nil {
		return nil, err
	}

	// 检查名称唯一性
	var count int64
	database.DB.Model(&models.Credential{}).Where("name = ?", req.Name).Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("凭据名称已存在")
	}

	// 加密密码
	encryptedPassword, err := s.crypto.Encrypt(req.Password)
	if err != nil {
		return nil, fmt.Errorf("密码加密失败: %w", err)
	}

	// 创建凭据
	credential := &models.Credential{
		ID:          uuid.New().String(),
		Name:        req.Name,
		Description: req.Description,
		Username:    req.Username,
		Password:    encryptedPassword,
	}

	if err := database.DB.Create(credential).Error; err != nil {
		return nil, fmt.Errorf("创建失败: %w", err)
	}

	return credential, nil
}

// List 获取凭据列表
func (s *CredentialService) List() ([]models.Credential, error) {
	var list []models.Credential
	if err := database.DB.Order("created_at DESC").Find(&list).Error; err != nil {
		return nil, err
	}
	return list, nil
}

// GetByID 根据ID获取凭据
func (s *CredentialService) GetByID(id string) (*models.Credential, error) {
	var credential models.Credential
	if err := database.DB.First(&credential, "id = ?", id).Error; err != nil {
		return nil, err
	}
	return &credential, nil
}

// Update 更新凭据
func (s *CredentialService) Update(id string, req *CreateCredentialRequest) (*models.Credential, error) {
	// 查询凭据
	credential, err := s.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("凭据不存在")
	}

	// 验证
	if err := s.validate(req); err != nil {
		return nil, err
	}

	// 检查名称唯一性（排除自身）
	var count int64
	database.DB.Model(&models.Credential{}).
		Where("name = ? AND id != ?", req.Name, id).
		Count(&count)
	if count > 0 {
		return nil, fmt.Errorf("凭据名称已存在")
	}

	// 更新字段
	credential.Name = req.Name
	credential.Description = req.Description
	credential.Username = req.Username

	// 如果提供了新密码，重新加密
	if req.Password != "" {
		encryptedPassword, err := s.crypto.Encrypt(req.Password)
		if err != nil {
			return nil, fmt.Errorf("密码加密失败: %w", err)
		}
		credential.Password = encryptedPassword
	}

	if err := database.DB.Save(credential).Error; err != nil {
		return nil, fmt.Errorf("更新失败: %w", err)
	}

	return credential, nil
}

// Delete 删除凭据
func (s *CredentialService) Delete(id string) error {
	// 检查是否被数据源使用
	var count int64
	database.DB.Model(&models.DataSource{}).
		Where("credential_id = ?", id).
		Count(&count)
	if count > 0 {
		return fmt.Errorf("凭据正在被 %d 个数据源使用，无法删除", count)
	}

	if err := database.DB.Delete(&models.Credential{}, "id = ?", id).Error; err != nil {
		return fmt.Errorf("删除失败: %w", err)
	}

	return nil
}

// validate 验证请求
func (s *CredentialService) validate(req *CreateCredentialRequest) error {
	if req.Name == "" {
		return fmt.Errorf("凭据名称不能为空")
	}
	if req.Username == "" {
		return fmt.Errorf("用户名不能为空")
	}
	return nil
}

// GetDecryptedPassword 获取解密后的密码（内部使用）
func (s *CredentialService) GetDecryptedPassword(id string) (string, error) {
	credential, err := s.GetByID(id)
	if err != nil {
		return "", err
	}

	password, err := s.crypto.Decrypt(credential.Password)
	if err != nil {
		return "", fmt.Errorf("密码解密失败: %w", err)
	}

	return password, nil
}
