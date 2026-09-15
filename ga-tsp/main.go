// 遗传算法求解 TSP 系统（课程设计）：Go 后端 + Web 前端。
// 后端提供 REST API（实例管理 / GA 求解 / 参数扫描），并托管 web/ 静态页面。
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"simple_tuan/internal/config"
	"simple_tuan/internal/controller"
	"simple_tuan/internal/dao"
	"simple_tuan/internal/logic"
)

func main() {
	if err := config.LoadEnv(".env"); err != nil {
		log.Fatal(err)
	}
	settings, err := config.Read()
	if err != nil {
		log.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	users, err := dao.NewMySQLUsers(ctx, settings.MySQLAddr, settings.MySQLUser, settings.MySQLPassword, settings.Database, settings.CreateDatabase)
	if err != nil {
		cancel()
		log.Fatal(err)
	}
	sessions, err := dao.NewRedisSessions(ctx, settings.RedisAddr, settings.RedisPassword)
	cancel()
	if err != nil {
		users.Close()
		log.Fatal(err)
	}
	defer users.Close()
	defer sessions.Close()
	dbctx, dbcancel := context.WithTimeout(context.Background(), 20*time.Second)
	store, err := users.CampusStore(dbctx)
	dbcancel()
	if err != nil {
		log.Fatal(err)
	}
	svc := logic.NewAuth(users, sessions)
	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      controller.NewApp(svc, settings.CookieSecure, "web", store),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // 参数扫描请求耗时较长
	}
	log.Printf("某团配送决策系统已启动: http://localhost:%s", port)
	log.Fatal(srv.ListenAndServe())
}
