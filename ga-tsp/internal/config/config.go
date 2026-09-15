// Package config loads local configuration without logging credentials.
package config

import (
	"bufio"
	"fmt"
	"os"
	"regexp"
	"strings"
)

type Config struct {
	MySQLAddr      string
	MySQLUser      string
	MySQLPassword  string
	Database       string
	CreateDatabase bool
	RedisAddr      string
	RedisPassword  string
	CookieSecure   bool
}

// LoadEnv does not expand variables and never overrides existing environment.
func LoadEnv(path string) error {
	file, err := os.Open(path)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("无法读取本地环境配置")
	}
	defer file.Close()
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		key = strings.TrimSpace(key)
		if !ok || !regexp.MustCompile(`^[A-Z][A-Z0-9_]*$`).MatchString(key) {
			return fmt.Errorf("环境配置格式不正确")
		}
		value = strings.TrimSpace(value)
		if len(value) >= 2 && ((value[0] == '"' && value[len(value)-1] == '"') || (value[0] == '\'' && value[len(value)-1] == '\'')) {
			value = value[1 : len(value)-1]
		}
		if _, set := os.LookupEnv(key); !set {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("无法加载环境变量")
			}
		}
	}
	if scanner.Err() != nil {
		return fmt.Errorf("环境配置读取失败")
	}
	return nil
}

func Read() (Config, error) {
	c := Config{
		MySQLAddr: os.Getenv("QIJI_MYSQL_ADDR"), MySQLUser: os.Getenv("QIJI_MYSQL_USER"),
		MySQLPassword: os.Getenv("QIJI_MYSQL_PASSWORD"), Database: os.Getenv("QIJI_MYSQL_DATABASE"),
		RedisAddr: os.Getenv("QIJI_REDIS_ADDR"), RedisPassword: os.Getenv("QIJI_REDIS_PASSWORD"),
		CreateDatabase: os.Getenv("QIJI_CREATE_DATABASE") == "true", CookieSecure: os.Getenv("QIJI_COOKIE_SECURE") == "true",
	}
	if c.MySQLAddr == "" || c.MySQLUser == "" || c.RedisAddr == "" {
		return c, fmt.Errorf("请配置 .env 中的 MySQL 与 Redis 连接信息")
	}
	if !regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]{0,63}$`).MatchString(c.Database) {
		return c, fmt.Errorf("专用数据库名称格式不正确")
	}
	return c, nil
}
