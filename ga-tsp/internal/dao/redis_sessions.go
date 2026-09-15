package dao

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"simple_tuan/internal/models"
)

// RedisSessions 是基于 Redis 的不透明会话与频率计数实现。
type RedisSessions struct{ client *redis.Client }

// NewRedisSessions 连接 Redis 并校验可用性。
func NewRedisSessions(ctx context.Context, addr, password string) (*RedisSessions, error) {
	client := redis.NewClient(&redis.Options{Addr: addr, Password: password,
		DialTimeout: 5 * time.Second, ReadTimeout: 3 * time.Second, WriteTimeout: 3 * time.Second, MaxRetries: 1})
	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, fmt.Errorf("Redis 连接失败，请核对本地配置")
	}
	return &RedisSessions{client: client}, nil
}

func (s *RedisSessions) Close() error { return s.client.Close() }

func digest(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}
func sessionKey(token string) string { return "qiji:session:v1:" + digest(token) }

func (s *RedisSessions) Put(ctx context.Context, token string, u models.User, ttl time.Duration) error {
	body, err := json.Marshal(u)
	if err != nil {
		return err
	}
	return s.client.Set(ctx, sessionKey(token), body, ttl).Err()
}
func (s *RedisSessions) Get(ctx context.Context, token string) (models.User, error) {
	var u models.User
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
