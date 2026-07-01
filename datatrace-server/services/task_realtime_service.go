package services

import (
	"datatrace/models"
	"datatrace/utils"
	"encoding/json"
	"fmt"
)

// TaskRealTimeService 任务实时结构服务
type TaskRealTimeService struct {
	dsService *DataSourceService
}

// NewTaskRealTimeService 创建任务实时结构服务
func NewTaskRealTimeService() *TaskRealTimeService {
	return &TaskRealTimeService{
		dsService: NewDataSourceService(),
	}
}

// TaskConfigWithRealTime 包含实时表结构的任务配置
type TaskConfigWithRealTime struct {
	Task         *models.SyncTask    `json:"task"`
	Config       *TaskConfig         `json:"config"`
	RealTimeInfo map[string][]string `json:"real_time_info"` // 数据库名 -> 表名列表
}

// GetConfigWithRealTimeStructure 获取任务配置（包含实时表结构）
func (s *TaskRealTimeService) GetConfigWithRealTimeStructure(taskService *TaskService, id string) (*TaskConfigWithRealTime, error) {
	// 获取任务基本信息
	task, err := taskService.GetByID(id)
	if err != nil {
		return nil, err
	}

	// 解析保存的配置
	var savedConfig TaskConfig
	if task.Config != "" {
		if err := json.Unmarshal([]byte(task.Config), &savedConfig); err != nil {
			return nil, fmt.Errorf("配置解析失败: %w", err)
		}
	}

	// 获取源数据源信息
	sourceDS, err := s.dsService.GetByID(savedConfig.SourceID)
	if err != nil {
		return nil, fmt.Errorf("源数据源不存在")
	}

	// 获取实时表结构
	realTimeStructure, err := s.getRealTimeStructure(sourceDS)
	if err != nil {
		return nil, fmt.Errorf("获取实时表结构失败: %w", err)
	}

	// 合并配置和实时结构
	mergedConfig := s.mergeConfigWithRealTime(&savedConfig, realTimeStructure)

	return &TaskConfigWithRealTime{
		Task:         task,
		Config:       mergedConfig,
		RealTimeInfo: realTimeStructure,
	}, nil
}

// getRealTimeStructure 获取数据源的实时表结构
func (s *TaskRealTimeService) getRealTimeStructure(ds *models.DataSource) (map[string][]string, error) {
	if ds.Type != "mysql" {
		return nil, fmt.Errorf("暂不支持 %s 类型的实时结构查询", ds.Type)
	}

	// 获取连接信息
	username, password, err := s.getDataSourceCredentials(ds)
	if err != nil {
		return nil, err
	}

	// 获取实时表结构
	mysqlService := NewMySQLMetadataService()
	dbWithTables, err := mysqlService.GetDatabasesWithTables(ds.Host, ds.Port, username, password)
	if err != nil {
		return nil, err
	}

	// 转换为 map 格式
	result := make(map[string][]string)
	for _, db := range dbWithTables {
		result[db.Database] = db.Tables
	}

	return result, nil
}

// getDataSourceCredentials 获取数据源的连接凭据
func (s *TaskRealTimeService) getDataSourceCredentials(ds *models.DataSource) (string, string, error) {
	if ds.CredentialID != nil && *ds.CredentialID != "" {
		// 使用凭据
		credService := NewCredentialService()
		cred, err := credService.GetByID(*ds.CredentialID)
		if err != nil {
			return "", "", fmt.Errorf("凭据不存在")
		}
		return cred.Username, cred.Password, nil
	} else {
		// 使用数据源自身的用户名密码
		crypto := utils.NewCryptoService()
		password, err := crypto.Decrypt(ds.Password)
		if err != nil {
			return "", "", fmt.Errorf("密码解密失败: %w", err)
		}
		return ds.Username, password, nil
	}
}

// mergeConfigWithRealTime 合并保存的配置和实时表结构
func (s *TaskRealTimeService) mergeConfigWithRealTime(savedConfig *TaskConfig, realTimeStructure map[string][]string) *TaskConfig {
	if savedConfig.SelectedDatabases == nil {
		return savedConfig
	}

	// 创建新的配置副本
	mergedConfig := *savedConfig
	mergedDatabases := make([]DatabaseSelection, 0, len(savedConfig.SelectedDatabases))

	for _, savedDB := range savedConfig.SelectedDatabases {
		realTimeTables, exists := realTimeStructure[savedDB.SourceDatabase]
		if !exists {
			// 数据库不存在，保留原配置但标记为已删除
			mergedDatabases = append(mergedDatabases, savedDB)
			continue
		}

		// 创建实时表名映射
		realTimeTableMap := make(map[string]bool)
		for _, table := range realTimeTables {
			realTimeTableMap[table] = true
		}

		// 更新表配置
		mergedTables := make([]TableConfig, 0, len(savedDB.Tables))
		for _, savedTable := range savedDB.Tables {
			newTable := savedTable
			// 检查表是否仍然存在
			if _, exists := realTimeTableMap[savedTable.SourceTable]; !exists {
				// 表已被删除，保留原配置但可以添加标记
				// 这里保持原样，让前端显示时处理
			}
			mergedTables = append(mergedTables, newTable)
		}

		mergedDB := savedDB
		mergedDB.Tables = mergedTables
		mergedDatabases = append(mergedDatabases, mergedDB)
	}

	mergedConfig.SelectedDatabases = mergedDatabases
	return &mergedConfig
}
