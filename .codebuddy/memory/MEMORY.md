# 长期记忆 · courseDesign（智能方法与系统设计课程设计）

## 项目布局（工作区 c:\Users\14925\Desktop\code\go\courseDesign）
- `maze-bfs-dfs/`：指导书**题目十二**迷宫求解（BFS 最短路 + DFS 对比），模块名 `mazeweb`，端口 **8080**，**保底交付项目**，功能已完整
- `ga-tsp/`：**主项目** GA 遗传算法求解 TSP（att48 基准）+ 进化过程可视化，模块名 `gatsp`，端口 **8081**
- 根目录保留指导书 docx 与提取文本 `.docx_text.txt`

## 技术约定（用户偏好）
- 技术栈固定：**Go 标准库后端（net/http）+ 原生 HTML/CSS/JS + Canvas 前端**，前后端分离，**零第三方依赖**（便于离线演示与答辩）
- 必须提供**可视化界面**：既要展示算法执行/搜索/进化过程，也要展示最优结果；交互含 播放/暂停/单步/调速/进度条
- 开发流程：先给设计方案 → 用户确认 → **TDD**（先测试后实现）→ 本地跑起来并验证 API → 浏览器预览
- 每完成一步用中文简要汇报实测数据（耗时、Gap、路径长度等）

## 环境与命令注意事项
- Go 1.26.1，Windows + PowerShell
- `execute_command` 直传 PowerShell 命令时 `$` 变量会被吞掉 → 写成 `.ps1` 脚本文件执行；或 `curl.exe -d @file.json`
- 启动后端：`go build -o xxx.exe .` + `Start-Process -WorkingDirectory <项目目录>`（托管静态目录时工作目录必须正确）
- 验证接口优先 curl.exe 或 PowerShell `Invoke-RestMethod`（脚本文件内）
