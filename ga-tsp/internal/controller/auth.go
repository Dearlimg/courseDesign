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

	"github.com/gin-gonic/gin"

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

func readCredentials(c *gin.Context) (credentials, error) {
	var out credentials
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 4096)
	d := json.NewDecoder(c.Request.Body)
	d.DisallowUnknownFields()
	if err := d.Decode(&out); err != nil {
		return out, errors.New("请求格式不正确")
	}
	if err := d.Decode(&struct{}{}); err != io.EOF {
		return out, errors.New("请求格式不正确")
	}
	out.Username = strings.ToLower(strings.TrimSpace(out.Username))
	if !usernamePattern.MatchString(out.Username) {
		return out, errors.New("用户名须为 3～32 位英文字母、数字或下划线")
	}
	if len(out.Password) < 8 || len(out.Password) > 72 {
		return out, errors.New("密码须为 8～72 字节")
	}
	return out, nil
}

// limited 按路径+IP 限流；被限时写出响应并返回 true。
func (h *authHandlers) limited(c *gin.Context) bool {
	ip, _, err := net.SplitHostPort(c.Request.RemoteAddr)
	if err != nil {
		ip = c.Request.RemoteAddr
	}
	limit := 15
	if c.Request.URL.Path == "/api/auth/register" {
		limit = 5
	}
	ok, err := h.svc.Allow(c.Request.Context(), c.Request.URL.Path+":"+ip, limit)
	if err != nil {
		writeError(c, 503, "会话服务暂不可用")
		return true
	}
	if !ok {
		c.Header("Retry-After", "60")
		writeError(c, 429, "操作过于频繁，请稍后重试")
		return true
	}
	return false
}

func (h *authHandlers) register(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	if h.limited(c) {
		return
	}
	body, err := readCredentials(c)
	if err != nil {
		writeError(c, 400, err.Error())
		return
	}
	user, err := h.svc.Register(ctx, body.Username, body.Password)
	if errors.Is(err, logic.ErrDuplicate) {
		writeError(c, 409, "该用户名已被使用")
		return
	}
	if err != nil {
		writeError(c, 503, "账号服务暂不可用")
		return
	}
	writeJSON(c, 201, gin.H{"user": user, "message": "注册成功，请登录"})
}

func (h *authHandlers) cookie(c *gin.Context, token string, maxAge int) {
	expires := time.Now().Add(logic.SessionTTL)
	if maxAge < 0 {
		expires = time.Unix(1, 0)
	}
	http.SetCookie(c.Writer, &http.Cookie{Name: cookieName, Value: token, Path: "/", HttpOnly: true, Secure: h.secure,
		SameSite: http.SameSiteStrictMode, MaxAge: maxAge, Expires: expires})
}

func (h *authHandlers) login(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 8*time.Second)
	defer cancel()
	if h.limited(c) {
		return
	}
	body, err := readCredentials(c)
	if err != nil {
		writeError(c, 400, err.Error())
		return
	}
	user, token, err := h.svc.Login(ctx, body.Username, body.Password, requestToken(c.Request))
	if errors.Is(err, logic.ErrUnauthorized) {
		writeError(c, 401, "用户名或密码不正确")
		return
	}
	if err != nil {
		writeError(c, 503, "会话服务暂不可用")
		return
	}
	h.cookie(c, token, int(logic.SessionTTL.Seconds()))
	writeJSON(c, 200, gin.H{"user": user, "expiresIn": int(logic.SessionTTL.Seconds())})
}

func (h *authHandlers) me(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 4*time.Second)
	defer cancel()
	user, err := h.svc.User(ctx, requestToken(c.Request))
	if errors.Is(err, logic.ErrNotFound) {
		writeError(c, 401, "请先登录")
		return
	}
	if err != nil {
		writeError(c, 503, "会话服务暂不可用")
		return
	}
	writeJSON(c, 200, gin.H{"user": user})
}

func (h *authHandlers) logout(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 4*time.Second)
	defer cancel()
	if err := h.svc.Logout(ctx, requestToken(c.Request)); err != nil {
		writeError(c, 503, "退出失败，请稍后重试")
		return
	}
	h.cookie(c, "", -1)
	writeJSON(c, 200, gin.H{"ok": true})
}
