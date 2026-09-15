package controller

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"simple_tuan/internal/logic"
)

// securityHeaders 设置安全响应头并拦截跨站 POST。
func securityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "same-origin")
		c.Header("X-Frame-Options", "DENY")
		if c.Request.Method == "POST" {
			origin := c.Request.Header.Get("Origin")
			if origin != "" {
				u, err := url.Parse(origin)
				if err != nil || u.Host != c.Request.Host || (u.Scheme != "http" && u.Scheme != "https") {
					writeError(c, 403, "不允许跨站提交")
					c.Abort()
					return
				}
			}
			if c.Request.Header.Get("Sec-Fetch-Site") == "cross-site" {
				writeError(c, 403, "不允许跨站提交")
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

// requireLogin 对业务 API 执行登录校验。
func requireLogin(svc *logic.AuthService) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 4*time.Second)
		defer cancel()
		user, err := svc.User(ctx, requestToken(c.Request))
		if errors.Is(err, logic.ErrNotFound) {
			writeError(c, 401, "登录已失效，请重新登录")
			c.Abort()
			return
		}
		if err != nil {
			writeError(c, 503, "会话服务暂不可用")
			c.Abort()
			return
		}
		c.Header("Cache-Control", "no-store")
		c.Set("userID", user.ID)
		c.Next()
	}
}

// noRouteGuard 兜底处理未注册路径：页面与业务 API 做登录校验，其余交静态托管。
func noRouteGuard(svc *logic.AuthService, staticDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := c.Request.URL.Path
		isAPI := strings.HasPrefix(p, "/api/")
		// auth.html 自身不做登录校验，否则会 303 重定向到自己形成死循环。
		isPage := p == "/" || (strings.HasSuffix(p, ".html") && p != "/auth.html")
		if isAPI || isPage {
			ctx, cancel := context.WithTimeout(c.Request.Context(), 4*time.Second)
			_, err := svc.User(ctx, requestToken(c.Request))
			cancel()
			if errors.Is(err, logic.ErrNotFound) {
				if isPage {
					c.Redirect(http.StatusSeeOther, "/auth.html")
					return
				}
				writeError(c, 401, "登录已失效，请重新登录")
				return
			}
			if err != nil {
				writeError(c, 503, "会话服务暂不可用")
				return
			}
			c.Header("Cache-Control", "no-store")
		}
		// 样式与脚本也禁止缓存：开发期改前端后刷新即可生效，避免浏览器用旧副本。
		if strings.HasSuffix(p, ".css") || strings.HasSuffix(p, ".js") {
			c.Header("Cache-Control", "no-store")
		}
		http.FileServer(http.Dir(staticDir)).ServeHTTP(c.Writer, c.Request)
	}
}
