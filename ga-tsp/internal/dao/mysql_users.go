// Package dao 封装数据访问：MySQL 账号（gorm）与 Redis 会话。
package dao

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"

	sqlmysql "github.com/go-sql-driver/mysql"
	gormmysql "gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"simple_tuan/internal/models"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate account")
)

// account 是账号表的 ORM 模型。
type account struct {
	ID           int64     `gorm:"primaryKey;autoIncrement"`
	Username     string    `gorm:"type:varchar(32);uniqueIndex"`
	PasswordHash []byte    `gorm:"type:varbinary(72)"`
	CreatedAt    time.Time `gorm:"type:timestamp;default:CURRENT_TIMESTAMP"`
}

// TableName 固定账号表名。
func (account) TableName() string { return "qiji_users" }

// MySQLUsers 是账号表的数据访问实现。
type MySQLUsers struct{ db *gorm.DB }

// NewMySQLUsers 连接 MySQL，可选创建专用数据库，并 AutoMigrate 账号表。
func NewMySQLUsers(ctx context.Context, addr, user, password, database string, createDB bool) (*MySQLUsers, error) {
	base := sqlmysql.Config{
		User: user, Passwd: password, Net: "tcp", Addr: addr,
		ParseTime:    true,
		Loc:          time.Local,
		Params:       map[string]string{"charset": "utf8mb4"},
		Timeout:      10 * time.Second,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
	}
	options := &gorm.Config{Logger: logger.Default.LogMode(logger.Silent), TranslateError: true}
	if createDB {
		root := base
		bootstrap, err := gorm.Open(gormmysql.Open(root.FormatDSN()), options)
		if err != nil {
			return nil, fmt.Errorf("MySQL 连接失败，请核对本地配置")
		}
		err = bootstrap.Exec("CREATE DATABASE IF NOT EXISTS `" + database + "` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci").Error
		if err != nil {
			return nil, fmt.Errorf("专用数据库初始化失败：%s", safeConnectionError(err))
		}
	}
	base.DBName = database
	db, err := gorm.Open(gormmysql.Open(base.FormatDSN()), options)
	if err != nil {
		return nil, fmt.Errorf("MySQL 连接失败，请核对本地配置")
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(3 * time.Minute)
	if err := sqlDB.PingContext(ctx); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("MySQL 连接失败，请核对本地配置")
	}
	if err := db.WithContext(ctx).AutoMigrate(&account{}); err != nil {
		sqlDB.Close()
		return nil, fmt.Errorf("账号表初始化失败")
	}
	return &MySQLUsers{db: db}, nil
}

// safeConnectionError 将底层连接错误转为可读提示。
func safeConnectionError(err error) string {
	var sqlErr *sqlmysql.MySQLError
	if errors.As(err, &sqlErr) {
		return fmt.Sprintf("MySQL 错误码 %d（请核对账号及建库权限）", sqlErr.Number)
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) && networkErr.Timeout() {
		return "连接或协议响应超时"
	}
	return "连接不可用"
}

func (u *MySQLUsers) Close() error {
	sqlDB, err := u.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// Create 插入新账号；用户名重复时返回 ErrDuplicate（依赖 gorm 错误翻译）。
func (u *MySQLUsers) Create(ctx context.Context, name string, hash []byte) (models.User, error) {
	row := account{Username: name, PasswordHash: hash}
	err := u.db.WithContext(ctx).Create(&row).Error
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return models.User{}, ErrDuplicate
	}
	if err != nil {
		return models.User{}, err
	}
	return models.User{ID: row.ID, Username: row.Username}, nil
}

// Find 按用户名查询账号；不存在时返回 ErrNotFound。
func (u *MySQLUsers) Find(ctx context.Context, name string) (models.Account, error) {
	var row account
	err := u.db.WithContext(ctx).Where("username = ?", name).First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return models.Account{}, ErrNotFound
	}
	if err != nil {
		return models.Account{}, err
	}
	return models.Account{User: models.User{ID: row.ID, Username: row.Username}, PasswordHash: row.PasswordHash}, nil
}
