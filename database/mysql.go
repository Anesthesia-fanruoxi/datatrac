package database

import (
	"datatrace/config"
	"datatrace/models"
	"fmt"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"time"
)

var DB *gorm.DB

// InitDB 初始化数据库连接
func InitDB() error {
	dbConfig := config.GlobalConfig.Database

	// 配置 GORM（禁用日志）
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	// 连接数据库
	var err error
	DB, err = gorm.Open(mysql.Open(dbConfig.GetDSN()), gormConfig)
	if err != nil {
		return fmt.Errorf("连接数据库失败: %w", err)
	}

	// 获取底层 sql.DB
	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("获取数据库实例失败: %w", err)
	}

	// 设置连接池
	sqlDB.SetMaxOpenConns(dbConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(dbConfig.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(time.Hour)

	// 测试连接
	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("数据库连接测试失败: %w", err)
	}

	// 自动迁移表结构
	if err := autoMigrate(); err != nil {
		return fmt.Errorf("数据库迁移失败: %w", err)
	}

	return nil
}

// autoMigrate 自动迁移表结构
func autoMigrate() error {
	err := DB.AutoMigrate(
		&models.DataSource{},
		&models.SyncTask{},
		&models.TaskUnitConfig{},
		&models.TaskUnitRuntime{},
	)

	if err != nil {
		return err
	}

	return nil
}

// CloseDB 关闭数据库连接
func CloseDB() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}
