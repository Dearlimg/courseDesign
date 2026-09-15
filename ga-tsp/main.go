// 遗传算法求解 TSP 系统（课程设计）：Go 后端 + Web 前端。
// 后端提供 REST API（实例管理 / GA 求解 / 参数扫描），并托管 web/ 静态页面。
package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"time"

	"simple_tuan/internal/api"
	"simple_tuan/internal/auth"
	"simple_tuan/internal/config"
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
	users, sessions, err := auth.Open(ctx, settings)
	cancel()
	if err != nil {
		log.Fatal(err)
	}
	defer users.Close()
	defer sessions.Close()
	login := auth.New(users, sessions, settings.CookieSecure)
	mux := api.NewMux()
	mux.Handle("/", http.FileServer(http.Dir("web")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      login.Handler(mux),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // 参数扫描请求耗时较长
	}
	log.Printf("骑迹配送决策系统已启动: http://localhost:%s", port)
	log.Fatal(srv.ListenAndServe())
}
