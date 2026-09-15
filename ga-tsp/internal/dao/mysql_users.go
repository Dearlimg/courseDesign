// Package dao 封装数据访问：MySQL 账号与 Redis 会话。
package dao

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"time"

	"github.com/go-sql-driver/mysql"
	"simple_tuan/internal/models"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate account")
)

// MySQLUsers 是账号表的数据库访问实现。
type MySQLUsers struct{ db *sql.DB }

// NewMySQLUsers 连接 MySQL，可选创建专用数据库与账号表。
func NewMySQLUsers(ctx context.Context, addr, user, password, database string, createDB bool) (*MySQLUsers, error) {
	cfg := mysql.NewConfig()
	cfg.Logger = &mysql.NopLogger{}
	cfg.User = user
	cfg.Passwd = password
	cfg.Net = "tcp"
	cfg.Addr = addr
	cfg.ParseTime = true
	cfg.Timeout = 10 * time.Second
	cfg.ReadTimeout = 10 * time.Second
	cfg.WriteTimeout = 10 * time.Second
	if createDB {
		connector, err := mysql.NewConnector(cfg)
		if err != nil {
			return nil, fmt.Errorf("MySQL 配置无效")
		}
		bootstrap := sql.OpenDB(connector)
		// Database has already been restricted to an identifier by config.Read.
		_, err = bootstrap.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+database+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
		bootstrap.Close()
		if err != nil {
			return nil, fmt.Errorf("专用数据库初始化失败：%s", safeConnectionError(err))
		}
	}
	cfg.DBName = database
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, fmt.Errorf("MySQL 配置无效")
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)
	fail := func(message string) (*MySQLUsers, error) {
		db.Close()
		return nil, errors.New(message)
	}
	if err := db.PingContext(ctx); err != nil {
		return fail("MySQL 连接失败，请核对本地配置")
	}
	_, err = db.ExecContext(ctx, `CREATE TABLE IF NOT EXISTS qiji_users (
 id BIGINT NOT NULL AUTO_INCREMENT PRIMARY KEY,
 username VARCHAR(32) CHARACTER SET ascii COLLATE ascii_bin NOT NULL UNIQUE,
 password_hash VARBINARY(72) NOT NULL,
 created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
 ) ENGINE=InnoDB`)
	if err != nil {
		return fail("账号表初始化失败")
	}
	return &MySQLUsers{db: db}, nil
}

func safeConnectionError(err error) string {
	var sqlErr *mysql.MySQLError
	if errors.As(err, &sqlErr) {
		return fmt.Sprintf("MySQL 错误码 %d（请核对账号及建库权限）", sqlErr.Number)
	}
	var networkErr net.Error
	if errors.As(err, &networkErr) && networkErr.Timeout() {
		return "连接或协议响应超时"
	}
	return "连接不可用"
}

func (u *MySQLUsers) Close() error { return u.db.Close() }

func (u *MySQLUsers) Create(ctx context.Context, name string, hash []byte) (models.User, error) {
	result, err := u.db.ExecContext(ctx, "INSERT INTO qiji_users (username,password_hash) VALUES (?,?)", name, hash)
	if err != nil {
		var driverErr *mysql.MySQLError
		if errors.As(err, &driverErr) && driverErr.Number == 1062 {
			return models.User{}, ErrDuplicate
		}
		return models.User{}, err
	}
	id, err := result.LastInsertId()
	return models.User{ID: id, Username: name}, err
}

func (u *MySQLUsers) Find(ctx context.Context, name string) (models.Account, error) {
	var a models.Account
	err := u.db.QueryRowContext(ctx, "SELECT id,username,password_hash FROM qiji_users WHERE username=?", name).
		Scan(&a.ID, &a.Username, &a.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}
