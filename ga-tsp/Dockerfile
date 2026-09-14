# 遗传算法 TSP 系统镜像：仅 COPY 本地交叉编译好的静态二进制 + web 静态资源，服务器无需 Go 工具链
FROM alpine:3.20

# 换阿里云源（与同服务器其它服务一致，加速 apk；本镜像无需额外包，仅备用）
RUN sed -i 's#dl-cdn.alpinelinux.org#mirrors.aliyun.com#g' /etc/apk/repositories

WORKDIR /app

# 二进制与前端静态资源
COPY gatsp /app/gatsp
COPY web /app/web

# 修复可执行位（scp 上传可能不带 unix 执行位）
RUN chmod +x /app/gatsp

# 容器内监听 8001（由 main.go 读取 PORT 环境变量）
ENV PORT=8001
EXPOSE 8001

CMD ["/app/gatsp"]
