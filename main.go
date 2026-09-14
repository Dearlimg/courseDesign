// 迷宫求解系统（课程设计题目十二）：Go 后端 + Web 前端。
// 后端提供 REST API（预置迷宫 / 随机生成 / BFS-DFS 求解），并托管 web/ 静态页面。
package main

import (
	"log"
	"net/http"
	"time"

	"mazeweb/internal/api"
)

func main() {
	mux := api.NewMux()
	mux.Handle("/", http.FileServer(http.Dir("web")))

	srv := &http.Server{
		Addr:         ":8080",
		Handler:      mux,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}
	log.Println("迷宫求解系统已启动: http://localhost:8080")
	log.Fatal(srv.ListenAndServe())
}
