package controller

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"regexp"
	"strings"
	"time"

	"simple_tuan/internal/logic"
)

const cookieName = "qiji_session"

var usernamePattern = regexp.MustCompile(`^[a-z0-9_]{3,32}$`)

type authHandlers struct {
	svc    *logic.AuthService
	secure bool
}

type credentials struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// requestToken 提取会话 cookie。
func requestToken(r *http.Request) string {
	cookie, err := r.Cookie(cookieName)
	if err != nil {
		return ""
	}
	return cookie.Value
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

// limited 按路径+IP 限流；被限时写出响应并返回 true。
func (h *authHandlers) limited(w http.ResponseWriter, r *http.Request) bool {
	ip, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		ip = r.RemoteAddr
	}
	limit := 15
	if r.URL.Path == "/api/auth/register" {
		limit = 5
	}
	ok, err := h.svc.Allow(r.Context(), r.URL.Path+":"+ip, limit)
	if err != nil {
		writeError(w, 503, "会话服务暂不可用")
		return true
	}
	if !ok {
		w.Header().Set("Retry-After", "60")
		writeError(w, 429, "操作过于频繁，请稍后重试")
		return true
	}
	return false
}

func (h *authHandlers) register(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	if h.limited(w, r) {
		return
	}
	c, err := readCredentials(w, r)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	user, err := h.svc.Register(ctx, c.Username, c.Password)
	if errors.Is(err, logic.ErrDuplicate) {
		writeError(w, 409, "该用户名已被使用")
		return
	}
	if err != nil {
		writeError(w, 503, "账号服务暂不可用")
		return
	}
	writeJSON(w, 201, map[string]any{"user": user, "message": "注册成功，请登录"})
}

func (h *authHandlers) cookie(w http.ResponseWriter, token string, maxAge int) {
	expires := time.Now().Add(logic.SessionTTL)
	if maxAge < 0 {
		expires = time.Unix(1, 0)
	}
	http.SetCookie(w, &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: true, Secure: h.secure,
		SameSite: http.SameSiteStrictMode, MaxAge: maxAge, Expires: expires})
}

func (h *authHandlers) login(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 8*time.Second)
	defer cancel()
	r = r.WithContext(ctx)
	if h.limited(w, r) {
		return
	}
	c, err := readCredentials(w, r)
	if err != nil {
		writeError(w, 400, err.Error())
		return
	}
	user, token, err := h.svc.Login(ctx, c.Username, c.Password, requestToken(r))
	if errors.Is(err, logic.ErrUnauthorized) {
		writeError(w, 401, "用户名或密码不正确")
		return
	}
	if err != nil {
		writeError(w, 503, "会话服务暂不可用")
		return
	}
	h.cookie(w, token, int(logic.SessionTTL.Seconds()))
	writeJSON(w, 200, map[string]any{"user": user, "expiresIn": int(logic.SessionTTL.Seconds())})
}

func (h *authHandlers) me(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	user, err := h.svc.User(ctx, requestToken(r))
	if errors.Is(err, logic.ErrNotFound) {
		writeError(w, 401, "请先登录")
		return
	}
	if err != nil {
		writeError(w, 503, "会话服务暂不可用")
		return
	}
	writeJSON(w, 200, map[string]any{"user": user})
}

func (h *authHandlers) logout(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
	defer cancel()
	if err := h.svc.Logout(ctx, requestToken(r)); err != nil {
		writeError(w, 503, "退出失败，请稍后重试")
		return
	}
	h.cookie(w, "", -1)
	writeJSON(w, 200, map[string]bool{"ok": true})
}
