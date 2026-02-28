package services

import (
	"context"
	"crypto/tls"
	"database/sql"
	"fmt"
	"net/http"
	"time"
)

// TestConnectionRequest 测试连接请求
type TestConnectionRequest struct {
	Type         string `json:"type" binding:"required"`
	Host         string `json:"host" binding:"required"`
	Port         int    `json:"port" binding:"required"`
	Username     string `json:"username" binding:"required"`
	Password     string `json:"password" binding:"required"`
	DatabaseName string `json:"database_name"`
	UseSSL       bool   `json:"use_ssl"` // 是否使用SSL/HTTPS
}

// TestConnectionResponse 测试连接响应
type TestConnectionResponse struct {
	Success bool   `json:"success"`
	Version string `json:"version"`
	Message string `json:"message"`
}

// TestConnection 测试数据源连接
func (s *DataSourceService) TestConnection(req *TestConnectionRequest) (*TestConnectionResponse, error) {
	switch req.Type {
	case "mysql":
		return s.testMySQLConnection(req)
	case "elasticsearch":
		return s.testElasticsearchConnection(req)
	default:
		return nil, fmt.Errorf("不支持的数据源类型: %s", req.Type)
	}
}

// TestConnectionByID 根据ID测试数据源连接
func (s *DataSourceService) TestConnectionByID(id string) (*TestConnectionResponse, error) {
	ds, err := s.GetByID(id)
	if err != nil {
		return nil, fmt.Errorf("数据源不存在")
	}

	password, err := s.crypto.Decrypt(ds.Password)
	if err != nil {
		return &TestConnectionResponse{
			Success: false,
			Message: "密码解密失败",
		}, nil
	}

	req := &TestConnectionRequest{
		Type:         ds.Type,
		Host:         ds.Host,
		Port:         ds.Port,
		Username:     ds.Username,
		Password:     password,
		DatabaseName: ds.DatabaseName,
	}

	return s.TestConnection(req)
}

// testMySQLConnection 测试 MySQL 连接
func (s *DataSourceService) testMySQLConnection(req *TestConnectionRequest) (*TestConnectionResponse, error) {
	dsn := fmt.Sprintf("%s:%s@tcp(%s:%d)/", req.Username, req.Password, req.Host, req.Port)
	if req.DatabaseName != "" {
		dsn += req.DatabaseName
	}
	dsn += "?charset=utf8mb4&parseTime=True&loc=Local&timeout=5s"

	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return &TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("连接失败: %v", err),
		}, nil
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return &TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("连接失败: %v", err),
		}, nil
	}

	var version string
	err = db.QueryRowContext(ctx, "SELECT VERSION()").Scan(&version)
	if err != nil {
		version = "未知"
	}

	return &TestConnectionResponse{
		Success: true,
		Version: version,
		Message: "连接成功",
	}, nil
}

// testElasticsearchConnection 测试 Elasticsearch 连接
func (s *DataSourceService) testElasticsearchConnection(req *TestConnectionRequest) (*TestConnectionResponse, error) {
	// 根据 UseSSL 选择协议
	scheme := "http"
	if req.UseSSL {
		scheme = "https"
	}
	url := fmt.Sprintf("%s://%s:%d", scheme, req.Host, req.Port)

	// 创建 HTTP 客户端（支持自签名证书）
	client := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: req.UseSSL},
		},
	}

	httpReq, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return &TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("创建请求失败: %v", err),
		}, nil
	}

	httpReq.SetBasicAuth(req.Username, req.Password)

	resp, err := client.Do(httpReq)
	if err != nil {
		return &TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("连接失败: %v", err),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &TestConnectionResponse{
			Success: false,
			Message: fmt.Sprintf("连接失败: HTTP %d", resp.StatusCode),
		}, nil
	}

	return &TestConnectionResponse{
		Success: true,
		Version: "已连接",
		Message: "连接成功",
	}, nil
}
