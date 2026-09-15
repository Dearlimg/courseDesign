package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const cookieName = "qiji_session"
const sessionTTL = 24 * time.Hour

var usernamePattern = regexp.MustCompile(`^[a-z0-9_]{3,32}$`)

type Service struct {
	users     Users
	sessions  Sessions
	secure    bool
	dummyHash []byte
}

func New(users Users, sessions Sessions, secure bool) *Service {
	dummy, _ := bcrypt.GenerateFromPassword([]byte("timing-only-placeholder"), bcrypt.DefaultCost)
	return &Service{users: users, sessions: sessions, secure: secure, dummyHash: dummy}
}
func respond(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
func failure(w http.ResponseWriter, status int, message string) {
	respond(w, status, map[string]string{"error": message})
}

func (s *Service) Handler(next http.Handler) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", s.register)
	mux.HandleFunc("POST /api/auth/login", s.login)
	mux.HandleFunc("POST /api/auth/logout", s.logout)
	mux.HandleFunc("GET /api/auth/me", s.me)
	mux.Handle("/", s.require(next))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		if r.Method == "POST" {
			origin := r.Header.Get("Origin")
			if origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Host != r.Host || (u.Scheme != "http" && u.Scheme != "https") {
					failure(w, 403, "不允许跨站提交")
					return
				}
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				failure(w, 403, "不允许跨站提交")
				return
			}
		}
		mux.ServeHTTP(w, r)
	})
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

func readCredentials(w http.ResponseWriter, r *http.Request) (credentials, error) {
	var c credentials
	r.Body = http.MaxBytesReader(w, r.Body, 4096)
	d := json.NewDecoder(r.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(&c); err != nil {
		return c, errors.New("请求格式不正确")
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return c, errors.New("请求格式不正确")
	}
	c.Username = strings.ToLower(strings.TrimSpace(c.Username))
	if !usernamePattern.MatchString(c.Username) {
		return c, errors.New("用户名须为 3～32 位英文字母、数字或下划线")
	}
	if len(c.Password) < 8 || len(c.Password) > 72 {
		return c, errors.New("密码须为 8～72 字节")
	}
	return c, nil
}
func (s *Service) limited(w http.ResponseWriter, r *http.Request) bool {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	limit := 15
	if r.URL.Path == "/api/auth/register" {
		limit = 5
	}
	ok, err := s.sessions.Allow(r.Context(), r.URL.Path+":"+ip, limit, time.Minute)
	if err != nil {
		failure(w, 503, "会话服务暂不可用")
		return true
	}
	if !ok {
		w.Header().Set("Retry-After", "60")
		failure(w, 429, "操作过于频繁，请稍后重试")
		return true
	}
	return false
}
func (s *Service) register(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	if s.limited(w, r) {
		return
	}
	c, err := readCredentials(w, r)
	if err != nil {
		failure(w, 400, err.Error())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(c.Password), bcrypt.DefaultCost)
	if err != nil {
		failure(w, 500, "账号创建失败")
		return
	}
	user, err := s.users.Create(ctx, c.Username, hash)
	if errors.Is(err, ErrDuplicate) {
		failure(w, 409, "该用户名已被使用")
		return
	}
	if err != nil {
		failure(w, 503, "账号服务暂不可用")
		return
	}
	respond(w, 201, map[string]any{"user": user, "message": "注册成功，请登录"})
}
func (s *Service) cookie(w http.ResponseWriter, token string, maxAge int) {
	expires := time.Now().Add(sessionTTL)
	if maxAge < 0 {
		expires = time.Unix(1, 0)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: true, Secure: s.secure,
		SameSite: http.SameSiteStrictMode, MaxAge: maxAge, Expires: expires})
}
func (s *Service) login(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	if s.limited(w, r) {
		return
	}
	c, err := readCredentials(w, r)
	if err != nil {
		failure(w, 400, err.Error())
		return
	}
	a, err := s.users.Find(ctx, c.Username)
	if err != nil && !errors.Is(err, ErrNotFound) {
		failure(w, 503, "账号服务暂不可用")
		return
	}
	hash := a.PasswordHash
	if errors.Is(err, ErrNotFound) {
		hash = s.dummyHash
	}
	match := bcrypt.CompareHashAndPassword(hash, []byte(c.Password)) == nil
	if err != nil || !match {
		failure(w, 401, "用户名或密码不正确")
		return
	}
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		failure(w, 500, "登录失败")
		return
	}
	token := hex.EncodeToString(bytes)
	if old, err := r.Cookie(cookieName); err == nil {
		if err := s.sessions.Delete(ctx, old.Value); err != nil {
			failure(w, 503, "会话服务暂不可用")
			return
		}
	}
	if err := s.sessions.Put(ctx, token, a.User, sessionTTL); err != nil {
		failure(w, 503, "会话服务暂不可用")
		return
	}
	s.cookie(w, token, int(sessionTTL.Seconds()))
	respond(w, 200, map[string]any{"user": a.User, "expiresIn": int(sessionTTL.Seconds())})
}
func (s *Service) user(r *http.Request) (User, error) {
	cookie, err := r.Cookie(cookieName)
	if err != nil || len(cookie.Value) != 64 {
		return User{}, ErrNotFound
	}
	if _, err := hex.DecodeString(cookie.Value); err != nil {
		return User{}, ErrNotFound
	}
	return s.sessions.Get(r.Context(), cookie.Value)
}
func (s *Service) me(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	user, err := s.user(r.WithContext(ctx))
	if errors.Is(err, ErrNotFound) {
		failure(w, 401, "请先登录")
		return
	}
	if err != nil {
		failure(w, 503, "会话服务暂不可用")
		return
	}
	respond(w, 200, map[string]any{"user": user})
}
func (s *Service) logout(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	if cookie, err := r.Cookie(cookieName); err == nil {
		if err := s.sessions.Delete(ctx, cookie.Value); err != nil {
			failure(w, 503, "退出失败，请稍后重试")
			return
		}
	}
	s.cookie(w, "", -1)
	respond(w, 200, map[string]bool{"ok": true})
}
func (s *Service) require(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAPI := strings.HasPrefix(r.URL.Path, "/api/")
		isPage := r.URL.Path == "/" || r.URL.Path == "/index.html" || r.URL.Path == "/tsp.html" || r.URL.Path == "/experiment.html"
		if !isAPI && !isPage {
			next.ServeHTTP(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		_, err := s.user(r.WithContext(ctx))
		cancel()
		if errors.Is(err, ErrNotFound) {
			if isPage {
				http.Redirect(w, r, "/auth.html", http.StatusSeeOther)
				return
			}
			failure(w, 401, "登录已失效，请重新登录")
			return
		}
		if err != nil {
			failure(w, 503, "会话服务暂不可用")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
