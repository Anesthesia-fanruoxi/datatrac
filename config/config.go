package config

import (
	"fmt"
	"github.com/spf13/viper"
)

// Config 全局配置结构
type Config struct {
	Database DatabaseConfig `mapstructure:"database"`
	Server   ServerConfig   `mapstructure:"server"`
	Security SecurityConfig `mapstructure:"security"`
}

// DatabaseConfig 数据库配置
type DatabaseConfig struct {
	Host         string `mapstructure:"host"`
	Port         int    `mapstructure:"port"`
	Username     string `mapstructure:"username"`
	Password     string `mapstructure:"password"`
	Database     string `mapstructure:"database"`
	MaxOpenConns int    `mapstructure:"max_open_conns"`
	MaxIdleConns int    `mapstructure:"max_idle_conns"`
}

// ServerConfig 服务器配置
type ServerConfig struct {
	Port int    `mapstructure:"port"`
	Mode string `mapstructure:"mode"`
}

// SecurityConfig 安全配置
type SecurityConfig struct {
	EncryptionKey string `mapstructure:"encryption_key"`
}

var GlobalConfig *Config

// LoadConfig 加载配置文件
func LoadConfig(configPath string) error {
	viper.SetConfigFile(configPath)
	viper.SetConfigType("yaml")

	// 读取配置文件
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("读取配置文件失败: %w", err)
	}

	// 解析配置
	if err := viper.Unmarshal(&GlobalConfig); err != nil {
		return fmt.Errorf("解析配置文件失败: %w", err)
	}

	// 验证配置
	if err := validateConfig(); err != nil {
		return fmt.Errorf("配置验证失败: %w", err)
	}

	return nil
}

// validateConfig 验证配置
func validateConfig() error {
	if GlobalConfig.Database.Host == "" {
		return fmt.Errorf("数据库主机地址不能为空")
	}
	if GlobalConfig.Database.Port == 0 {
		return fmt.Errorf("数据库端口不能为空")
	}
	if GlobalConfig.Database.Username == "" {
		return fmt.Errorf("数据库用户名不能为空")
	}
	if GlobalConfig.Database.Database == "" {
		return fmt.Errorf("数据库名称不能为空")
	}
	if len(GlobalConfig.Security.EncryptionKey) != 32 {
		return fmt.Errorf("加密密钥必须是32字节")
	}
	return nil
}

// GetDSN 获取数据库连接字符串
func (c *DatabaseConfig) GetDSN() string {
	return fmt.Sprintf("%s:%s@tcp(%s:%d)/%s?charset=utf8mb4&parseTime=True&loc=Local",
		c.Username,
		c.Password,
		c.Host,
		c.Port,
		c.Database,
	)
}
