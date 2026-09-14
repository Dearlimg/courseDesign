// 遗传算法求解 TSP 系统（课程设计）：Go 后端 + Web 前端。
// 后端提供 REST API（实例管理 / GA 求解 / 参数扫描），并托管 web/ 静态页面。
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"gatsp/internal/api"
)

func main() {
	mux := api.NewMux()
	mux.Handle("/", http.FileServer(http.Dir("web")))

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	srv := &http.Server{
		Addr:         ":" + port,
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 120 * time.Second, // 参数扫描请求耗时较长
	}
	log.Printf("GA-TSP 系统已启动: http://localhost:%s", port)
	log.Fatal(srv.ListenAndServe())
}
