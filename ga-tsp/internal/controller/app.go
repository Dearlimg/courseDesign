package controller

import (
	"net/http"

	"simple_tuan/internal/logic"
)

// NewApp 组装完整应用：认证端点、登录守卫与业务路由。
func NewApp(svc *logic.AuthService, secure bool, next http.Handler) http.Handler {
	h := &authHandlers{svc: svc, secure: secure}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/auth/register", h.register)
	mux.HandleFunc("POST /api/auth/login", h.login)
	mux.HandleFunc("POST /api/auth/logout", h.logout)
	mux.HandleFunc("GET /api/auth/me", h.me)
	mux.Handle("/", requireAuth(svc, next))
	return securityHeaders(mux)
}
