package logic

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"golang.org/x/crypto/bcrypt"
	"simple_tuan/internal/dao"
	"simple_tuan/internal/models"
)

var (
	ErrNotFound     = errors.New("not found")
	ErrDuplicate    = errors.New("duplicate account")
	ErrUnauthorized = errors.New("invalid credentials")
)

// Users 是账号仓库的消费者侧接口，由 dao 层实现。
type Users interface {
	Create(context.Context, string, []byte) (models.User, error)
	Find(context.Context, string) (models.Account, error)
}

// Sessions 是会话存储的消费者侧接口，由 dao 层实现。
type Sessions interface {
	Put(context.Context, string, models.User, time.Duration) error
	Get(context.Context, string) (models.User, error)
	Delete(context.Context, string) error
	Allow(context.Context, string, int, time.Duration) (bool, error)
}

// SessionTTL 是登录会话的固定有效期。
const SessionTTL = 24 * time.Hour

// AuthService 编排账号与会话业务，全部方法零 HTTP 依赖。
type AuthService struct {
	users     Users
	sessions  Sessions
	dummyHash []byte
}

// NewAuth 装配认证业务服务。
func NewAuth(users Users, sessions Sessions) *AuthService {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("timing-only-placeholder"), bcrypt.DefaultCost)
	return &AuthService{users: users, sessions: sessions, dummyHash: dummy}
}

// Register 创建账号，用户名重复时返回 ErrDuplicate。
func (s *AuthService) Register(ctx context.Context, name, password string) (models.User, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return models.User{}, fmt.Errorf("账号创建失败")
	}
	user, err := s.users.Create(ctx, name, hash)
	if errors.Is(err, dao.ErrDuplicate) {
		return models.User{}, ErrDuplicate
	}
	return user, err
}

// Login 校验口令并签发新会话；oldToken 非空时先作废旧会话（防重放）。
// 口令或账号不匹配时返回 ErrUnauthorized。
func (s *AuthService) Login(ctx context.Context, name, password, oldToken string) (models.User, string, error) {
	a, err := s.users.Find(ctx, name)
	if err != nil && !errors.Is(err, dao.ErrNotFound) {
		return models.User{}, "", err
	}
	hash := a.PasswordHash
	if errors.Is(err, dao.ErrNotFound) {
		hash = s.dummyHash
	}
	match := bcrypt.CompareHashAndPassword(hash, []byte(password)) == nil
	if err != nil || !match {
		return models.User{}, "", ErrUnauthorized
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return models.User{}, "", err
	}
	token := hex.EncodeToString(buf)
	if oldToken != "" {
		if err := s.sessions.Delete(ctx, oldToken); err != nil {
			return models.User{}, "", err
		}
	}
	if err := s.sessions.Put(ctx, token, a.User, SessionTTL); err != nil {
		return models.User{}, "", err
	}
	return a.User, token, nil
}

// Logout 作废会话；无会话时静默成功。
func (s *AuthService) Logout(ctx context.Context, token string) error {
	if token == "" {
		return nil
	}
	return s.sessions.Delete(ctx, token)
}

// User 按会话令牌返回用户；令牌缺失、非法或过期时返回 ErrNotFound。
func (s *AuthService) User(ctx context.Context, token string) (models.User, error) {
	if len(token) != 64 {
		return models.User{}, ErrNotFound
	}
	if _, err := hex.DecodeString(token); err != nil {
		return models.User{}, ErrNotFound
	}
	u, err := s.sessions.Get(ctx, token)
	if errors.Is(err, dao.ErrNotFound) {
		return models.User{}, ErrNotFound
	}
	return u, err
}

// Allow 按 key 做固定窗口限流，窗口为一分钟。
func (s *AuthService) Allow(ctx context.Context, key string, limit int) (bool, error) {
	return s.sessions.Allow(ctx, key, limit, time.Minute)
}
