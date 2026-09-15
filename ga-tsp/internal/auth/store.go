// Package auth implements MySQL accounts and Redis-backed opaque sessions.
package auth

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"time"

	"simple_tuan/internal/config"

	"github.com/go-sql-driver/mysql"
	"github.com/redis/go-redis/v9"
)

var (
	ErrNotFound  = errors.New("not found")
	ErrDuplicate = errors.New("duplicate account")
)

type User struct {
	ID       int64  `json:"id"`
	Username string `json:"username"`
}
type Account struct {
	User
	PasswordHash []byte
}
type Users interface {
	Create(context.Context, string, []byte) (User, error)
	Find(context.Context, string) (Account, error)
}
type Sessions interface {
	Put(context.Context, string, User, time.Duration) error
	Get(context.Context, string) (User, error)
	Delete(context.Context, string) error
	Allow(context.Context, string, int, time.Duration) (bool, error)
}

type MySQLUsers struct{ db *sql.DB }
type RedisSessions struct{ client *redis.Client }

func Open(ctx context.Context, c config.Config) (*MySQLUsers, *RedisSessions, error) {
	cfg := mysql.NewConfig()
	cfg.Logger = &mysql.NopLogger{}
	cfg.User = c.MySQLUser
	cfg.Passwd = c.MySQLPassword
	cfg.Net = "tcp"
	cfg.Addr = c.MySQLAddr
	cfg.ParseTime = true
	cfg.Timeout = 10 * time.Second
	cfg.ReadTimeout = 10 * time.Second
	cfg.WriteTimeout = 10 * time.Second
	if c.CreateDatabase {
		connector, err := mysql.NewConnector(cfg)
		if err != nil {
			return nil, nil, fmt.Errorf("MySQL 配置无效")
		}
		bootstrap := sql.OpenDB(connector)
		// Database has already been restricted to an identifier by config.Read.
		_, err = bootstrap.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS `"+c.Database+"` CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci")
		bootstrap.Close()
		if err != nil {
			return nil, nil, fmt.Errorf("专用数据库初始化失败：%s", safeConnectionError(err))
		}
	}
	cfg.DBName = c.Database
	connector, err := mysql.NewConnector(cfg)
	if err != nil {
		return nil, nil, fmt.Errorf("MySQL 配置无效")
	}
	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(3 * time.Minute)
	fail := func(message string) (*MySQLUsers, *RedisSessions, error) {
		db.Close()
		return nil, nil, errors.New(message)
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
	client := redis.NewClient(&redis.Options{Addr: c.RedisAddr, Password: c.RedisPassword,
		DialTimeout: 5 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, MaxRetries: 1})
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return fail("Redis 连接失败，请核对本地配置")
	}
	return &MySQLUsers{db: db}, &RedisSessions{client: client}, nil
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
func (u *MySQLUsers) Close() error    { return u.db.Close() }
func (s *RedisSessions) Close() error { return s.client.Close() }

func (u *MySQLUsers) Create(ctx context.Context, name string, hash []byte) (User, error) {
	result, err := u.db.ExecContext(ctx, "INSERT INTO qiji_users (username,password_hash) VALUES (?,?)", name, hash)
	if err != nil {
		var driverErr *mysql.MySQLError
		if errors.As(err, &driverErr) && driverErr.Number == 1062 {
			return User{}, ErrDuplicate
		}
		return User{}, err
	}
	id, err := result.LastInsertId()
	return User{ID: id, Username: name}, err
}
func (u *MySQLUsers) Find(ctx context.Context, name string) (Account, error) {
	var a Account
	err := u.db.QueryRowContext(ctx, "SELECT id,username,password_hash FROM qiji_users WHERE username=?", name).
		Scan(&a.ID, &a.Username, &a.PasswordHash)
	if errors.Is(err, sql.ErrNoRows) {
		return a, ErrNotFound
	}
	return a, err
}
func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func sessionKey(token string) string { return "qiji:session:v1:" + digest(token) }
func (s *RedisSessions) Put(ctx context.Context, token string, u User, ttl time.Duration) error {
	body, err := json.Marshal(u)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, sessionKey(token), body, ttl).Err()
}
func (s *RedisSessions) Get(ctx context.Context, token string) (User, error) {
	var u User
	body, err := s.client.Get(ctx, sessionKey(token)).Bytes()
	if errors.Is(err, redis.Nil) {
		return u, ErrNotFound
	}
	if err != nil {
		return u, err
	}
	err = json.Unmarshal(body, &u)
	return u, err
}
func (s *RedisSessions) Delete(ctx context.Context, token string) error {
	return s.client.Del(ctx, sessionKey(token)).Err()
}

var rateScript = redis.NewScript(`local n=redis.call('INCR',KEYS[1]); if n==1 then redis.call('PEXPIRE',KEYS[1],ARGV[1]) end; return n`)

func (s *RedisSessions) Allow(ctx context.Context, key string, limit int, ttl time.Duration) (bool, error) {
	n, err := rateScript.Run(ctx, s.client, []string{"qiji:rate:v1:" + digest(key)}, ttl.Milliseconds()).Int()
	return n <= limit, err
}
