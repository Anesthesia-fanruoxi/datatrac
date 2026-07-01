package services

import (
	"fmt"
	"regexp"
)

// validNameRegex 合法的数据库/表名正则：只允许字母、数字、下划线、中划线
var validNameRegex = regexp.MustCompile(`^[a-zA-Z0-9_\-]+$`)

// ValidateTableName 校验表名是否合法，防止SQL注入
func ValidateTableName(name string) error {
	if name == "" {
		return fmt.Errorf("表名不能为空")
	}
	if len(name) > 64 {
		return fmt.Errorf("表名长度不能超过64个字符")
	}
	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("表名包含非法字符: %s", name)
	}
	return nil
}

// ValidateDatabaseName 校验数据库名是否合法，防止SQL注入
func ValidateDatabaseName(name string) error {
	if name == "" {
		return fmt.Errorf("数据库名不能为空")
	}
	if len(name) > 64 {
		return fmt.Errorf("数据库名长度不能超过64个字符")
	}
	if !validNameRegex.MatchString(name) {
		return fmt.Errorf("数据库名包含非法字符: %s", name)
	}
	return nil
}
