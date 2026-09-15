package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"simple_tuan/internal/logic"
)

// NewMux 构建仅含业务路由的路由器（测试与预览用，无登录守卫）。
func NewMux() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	registerBusiness(r, nil)
	return r
}

// NewApp 组装完整应用：安全头、认证端点、登录守卫、业务路由与静态托管。
func NewApp(svc *logic.AuthService, secure bool, staticDir string, stores ...logic.CampusStore) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(securityHeaders(), gin.Recovery())
	h := &authHandlers{svc: svc, secure: secure}
	r.POST("/api/auth/register", h.register)
	r.POST("/api/auth/login", h.login)
	r.POST("/api/auth/logout", h.logout)
	r.GET("/api/auth/me", h.me)
	registerBusiness(r, svc, stores...)
	r.NoRoute(noRouteGuard(svc, staticDir))
	return r
}

// registerBusiness 注册业务 API；svc 非空时挂登录守卫。
func registerBusiness(r *gin.Engine, svc *logic.AuthService, stores ...logic.CampusStore) {
	api := r.Group("/api")
	if svc != nil {
		api.Use(requireLogin(svc))
	}
	api.GET("/instances", handleListInstances)
	api.GET("/instances/:name", handleGetInstance)
	api.POST("/instance/random", handleRandomInstance)
	api.POST("/solve", handleSolve)
	api.POST("/scan", handleScan)
	registerExperiments(api)
	registerCampus(api)
	if len(stores) > 0 {
		registerSnapshots(api, stores[0])
	}
	api.POST("/analysis/compare", handleCompare)
	api.POST("/dispatch/route", handleRoute)
	api.POST("/dispatch/select", handleSelection)
}
