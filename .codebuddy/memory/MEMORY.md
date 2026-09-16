# 长期记忆 · courseDesign（智能方法与系统设计课程设计）

## 项目布局（工作区 c:\Users\14925\Desktop\code\go\courseDesign）
- `maze-bfs-dfs/`：指导书**题目十二**迷宫求解（BFS 最短路 + DFS 对比），模块名 `mazeweb`，端口 **8080**，**保底交付项目**，功能已完整
- `ga-tsp/`：**主项目** GA 遗传算法求解 TSP（att48 基准）+ 进化过程可视化，模块名 `simple_tuan`（某团），端口 **8081**，已分层 controller/logic/dao/models + pkg + gin + gorm
- 根目录保留指导书 docx 与提取文本 `.docx_text.txt`

## 技术约定（用户偏好）
- 技术栈：**Go 后端 + 原生 HTML/CSS/JS + Canvas 前端**，前后端分离；maze 项目保持纯标准库（net/http），ga-tsp 已换 **gin + gorm**（分层 controller/logic/dao），前端零框架
- 必须提供**可视化界面**：既要展示算法执行/搜索/进化过程，也要展示最优结果；交互含 播放/暂停/单步/调速/进度条
- 开发流程：先给设计方案 → 用户确认 → **TDD**（先测试后实现）→ 本地跑起来并验证 API → 浏览器预览
- 每完成一步用中文简要汇报实测数据（耗时、Gap、路径长度等）

## ga-tsp 重构现状（simple_tuan，2026-09-15 三阶段完成）
- 6 commit：`c0f9591` 模块更名 / `26bfedf` pkg 迁移 / `f480684` models+logic / `272c63c` 认证拆分+api→controller / `5445a3b` gin / `bb93cf8` gorm
- 依赖：gin v1.12.0、gorm v1.31.2 + driver/mysql v1.6.0、go-redis v9.22、bcrypt；目录 `internal/{controller,logic,dao,models,config}` + `pkg/{tsp,ga,optimization}`
- 关键装配：`controller.NewApp(svc, secure, staticDir)`；`dao.NewMySQLUsers(...)` gorm 版（AutoMigrate + TranslateError 判 409/404）；错误契约 `{"error": msg}` 不变
- 遗留待办：logic/tsp.go 薄封装（controller 仍直连 pkg）、cmd/ 入口化（main.go 在 ga-tsp 根、二进制仍 gatsp.exe）、Task 6 命名统一（QIJI_→ST_、表 st_users、cookie st_session、品牌某团）

## 环境与命令注意事项
- Go 1.26.1，Windows + PowerShell
- `execute_command` 直传 PowerShell 命令时 `$` 变量会被吞掉 → 写成 `.ps1` 脚本文件执行；或 `curl.exe -d @file.json`
- 启动后端：`go build -o xxx.exe .` + `Start-Process -WorkingDirectory <项目目录>`（托管静态目录时工作目录必须正确）
- **git 操作（切分支/合并/stash）前必须先停掉 gatsp 进程**：Windows 下运行中的 exe 被锁，git 会报 `unable to unlink old 'ga-tsp/gatsp.exe': Invalid argument` 并连带 `Could not reset index file to revision 'HEAD'`，使 stash/checkout 半成功；`go build -o gatsp.exe` 却能在运行时覆盖成功，故容易忽略这个锁
- PowerShell 里引用 stash 用**单引号**：`'stash@{0}'`（裸写 `stash@{0}` 会被当成 scriptblock，git 收到 `stash@` `MAA=` `xml` `text` 报 "Too many revisions specified"）
- **改完后端代码必须重新 `go build` + 重启进程**（静态前端是实时读盘，后端不是）；否则新接口未注册会落到 `NoRoute` → `http.FileServer` 返回 404 纯文本，前端 `JSON.parse` 报 `Unexpected non-whitespace character after JSON at position 4`
- 排查「接口是否真的注册了」用 `findstr /M /C:"campus/plan" gatsp.exe` + 对比源码与 exe 的 mtime，比 curl 打接口更直接（NoRoute 会先回 401 掩盖 404）
- **PowerShell 5.1 执行无 BOM 的 `.ps1` 时按 GBK 解码**：脚本里的中文字面量会变乱码，导致中文 `Replace` 全部静默失效（输出出现乱码即征兆）→ 中文用 `[char]0xXXXX` 码点构造，或改成纯 ASCII 脚本；读写文本用 `[IO.File]::ReadAllText/WriteAllText` + `UTF8Encoding($false)`（`-Encoding UTF8` 会写 BOM）
- **前端改动看不到效果时先查两层缓存**：(1) 浏览器启发式缓存 —— ga-tsp 现在对 `.css`/`.js` 也返回 `Cache-Control: no-store`；(2) 后端没重新编译重启。改 `*.html` 白名单判断要**排除 auth.html**，否则未登录会被 303 重定向到自己形成死循环
- PS 5.1 查响应头用 `curl.exe -s -I`（`Invoke-WebRequest -Method Head` 会抛 NullReferenceException）；括号内不要写多语句 `(a; b | c)`，PS 会报语法错误
- 验证接口优先 curl.exe 或 PowerShell `Invoke-RestMethod`（脚本文件内）

## 服务器部署（阿里云 ECS 121.40.235.227）
- SSH：`ssh aliyun`（~/.ssh/config 别名，IdentityFile ~/.ssh/id_ed25519，密钥免密）；密钥不可用时可显式 `-i C:/Users/14925/.ssh/id_ed25519 -o IdentitiesOnly=yes`
- 服务端曾出现 sshd 预握手关闭（`kex_exchange_identification: Connection closed`）→ 重启 sshd 后恢复；无 fail2ban/无 hosts.deny
- 已部署服务：blog-front（/opt/blog，host 网络 80）、im-toy（8080+nginx）、ga-tsp（/opt/ga-tsp，-p 8001:8001，容器名 ga-tsp，镜像 ga-tsp:latest，--restart unless-stopped）；**ga-tsp 已可公网访问** <http://121.40.235.227:8001>
- ga-tsp 部署方式（2026-09-15 更新版）：本地交叉编译 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -ldflags "-s -w" -o gatsp .`（**必须 strip**，否则 ELF 43MB vs 30MB）→ `tar -czf` 打包 gatsp + web → scp 到 /opt/ga-tsp → `docker build -t ga-tsp:latest .`（Dockerfile 仅 COPY gatsp/web，FROM alpine:3.20；服务器无需 Go 工具链）
- **容器网络：ga-tsp 必须 `--network workspace_default`**（mysql/redis 所在网络，172.18.0.0/16）；DB 用容器名直连：`-e QIJI_MYSQL_ADDR=mysql:3306 -e QIJI_REDIS_ADDR=redis:6379`，凭据 root/sta_go、库 qiji_dispatch、`QIJI_CREATE_DATABASE=true`（自动建库）、`QIJI_COOKIE_SECURE=false`；重建命令：`docker rm -f ga-tsp && docker run -d --name ga-tsp --network workspace_default --restart unless-stopped -p 8001:8001 -e PORT=8001 -e QIJI_... ga-tsp:latest`
- 安全组：**8001/TCP 已放行**（本地 curl 公网返回 303）；其它新端口仍需在 ECS 控制台安全组入方向手动放行

## 应用包装项目：骑迹（外卖骑手调度平台·进行中）
- 目标：把 maze-bfs-dfs(BFS/DFS) + ga-tsp(GA·TSP + GA·0/1背包 + CEC) 包装成面向"外卖骑手调度中心"的单页 Web 应用，共 **四算法四模块**
- 业务映射：迷宫=城市栅格路网；BFS=最短送达、DFS=片区可达性勘探、GA·TSP=多单巡回优化、**GA·背包=自动最优接单**（背包容量=出车时间预算，物品=订单，重量=耗时，价值=收益）；背包与 TSP 形成闭环：接哪几单→怎么送
- 整合：方案 B 深整合——新建 module `qiji`，单二进制/单端口:8080/单页前端；6 算法包(maze/solver/tsp/ga/optimization)复用仅改 import；统一 API `/api/nav/*` `/api/tsp/*` `/api/order/*`(背包接单) `/api/exp/cec/*`(可选)
- 设计文档：`docs/superpowers/specs/2026-09-14-qiji-platform-design.md` **v2**（纳入背包=自动接单）；**产品名已定：某团（simple_tuan）**（2026-09-15 用户拍板，模块/二进制/env/表名未来统一 ST_ 前缀）；开放问题：CEC是否保留/ECS端口/M5必做
- **落地进展（2026-09-15，分支 feat/qiji-dispatch）**：三层重构（gin+gorm）6 commit 完成；**校园调度 campus 模块 5 commit 完成**（仿真路网 pkg/campus → 订单批次 → 三模式规划 logic/plan.go → gorm 快照持久化 → 三策略对照+精确验证）；前端首页 index.html 已改为「校园配送决策」四区页面，旧工作台保留在 legacy.html
- **已合并到 master（2026-09-15 13:39）**：`git merge --ff-only feat/qiji-dispatch` 快进成功，master 由 `b892a1b` → `d9d98ce`（含西校区路网建模 west.json + 交互式校区图 campus-map.js）；本地领先 origin/master **26 个提交，尚未 push**；feat 分支保留在 d9d98ce
- **2026-09-15 收尾 4 个提交（全部入库，工作区干净）**：`6ec97a7` 品牌统一为某团 + 归档 docs/ 与 memory（含 `.gitignore` 忽略 `ga-tsp/run*.log`）；`f4143ca` 订单改点餐式卡片 + 单笔配送距离 + 配送箱飞入动画（新增 campus.js 379 行、Map.DepotMeters）；`c0b1dd2` `.css`/`.js` 禁用缓存 + `.html` 后缀页面白名单；`74bd591` 刷新构建产物
- **已知缺口（2026-09-15）**：~~ECS 8001 容器仍是 09-14 旧版~~ → **已更新为 d9d98ce 新版（13:45 部署）**；ST_ 命名统一（Task 6）未做；`gatsp`(30MB)/`gatsp.exe`(43MB) 构建产物仍被 git 跟踪，本地工作区有未提交的重新构建版本
