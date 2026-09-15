package controller

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"simple_tuan/internal/logic"
)

// securityHeaders 设置安全响应头并拦截跨站 POST。
func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "same-origin")
		w.Header().Set("X-Frame-Options", "DENY")
		if r.Method == "POST" {
			origin := r.Header.Get("Origin")
			if origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Host != r.Host || (u.Scheme != "http" && u.Scheme != "https") {
					writeError(w, 403, "不允许跨站提交")
					return
				}
			}
			if r.Header.Get("Sec-Fetch-Site") == "cross-site" {
				writeError(w, 403, "不允许跨站提交")
				return
			}
		}
		next.ServeHTTP(w, r)
	})
}

// requireAuth 对页面与业务 API 执行登录校验。
func requireAuth(svc *logic.AuthService, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		isAPI := strings.HasPrefix(r.URL.Path, "/api/")
		isPage := r.URL.Path == "/" || r.URL.Path == "/index.html" || r.URL.Path == "/tsp.html" || r.URL.Path == "/experiment.html"
		if !isAPI && !isPage {
			next.ServeHTTP(w, r)
			return
		}
		ctx, cancel := context.WithTimeout(r.Context(), 4*time.Second)
		_, err := svc.User(ctx, requestToken(r))
		cancel()
		if errors.Is(err, logic.ErrNotFound) {
			if isPage {
				http.Redirect(w, r, "/auth.html", http.StatusSeeOther)
				return
			}
			writeError(w, 401, "登录已失效，请重新登录")
			return
		}
		if err != nil {
			writeError(w, 503, "会话服务暂不可用")
			return
		}
		w.Header().Set("Cache-Control", "no-store")
		next.ServeHTTP(w, r)
	})
}
