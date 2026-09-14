// 遗传算法求解 TSP 系统（课程设计）：Go 后端 + Web 前端。
// 后端提供 REST API（实例管理 / GA 求解 / 参数扫描），并托管 web/ 静态页面。
package main

import (
	"log"
	"net/http"
	"time"

	"gatsp/internal/api"
)

func main() {
	mux := api.NewMux()
	mux.Handle("/", http.FileServer(http.Dir("web")))

	srv := &http.Server{
		Addr:         ":8081",
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // 参数扫描请求耗时较长
	}
	log.Println("GA-TSP 系统已启动: http://localhost:8081")
	log.Fatal(srv.ListenAndServe())
}
