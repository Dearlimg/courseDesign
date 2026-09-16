# 智能方法与系统设计 · 课程设计

> 智能 22 级《智能方法与系统设计》课程设计实现。以「校园外卖配送调度」为主线，把**遗传算法**（TSP 巡回优化、0/1 背包接单、CEC 基准）与**图搜索算法**（BFS / DFS）做成可交互、可验证、可复现实验的 Web 系统。

工作区包含两个可独立运行的 Go 项目：

| 项目 | 题目 | 核心算法 | 默认端口 |
| --- | --- | --- | --- |
| **某团 · 校园配送决策**（`ga-tsp/`） | 自选综合题（主项目） | GA 选单（0/1 背包 + 约束修复）、GA 送单（TSP + 2-opt）、精确对照验证 | 8081 |
| **迷宫求解系统**（`maze-bfs-dfs/`） | 指导书题目十二 | BFS 最短通路 / DFS 对照，探索过程动画 | 8080 |

两个项目均为 **Go 后端 + 原生 HTML/CSS/JS 前端**，前端零框架、零构建；后端提供 REST API 并托管静态页面。

---

## 一、主项目：某团 · 校园配送决策（`ga-tsp/`）

面向校园配送站调度员，解决两个问题：**这一趟接哪些单**、**这些单按什么顺序送**，并完整展示求解过程与实验数据。

### 功能

**01 校园订单池**
- 随机生成 20～50 笔仿真订单，六种场景可切换：`uniform` 均匀分布、`clustered` 宿舍集中、`near` 近距离低收益、`far` 远距离高收益、`outlier` 少数偏远低收益、`same-place` 多单同址；支持数据种子复现
- 订单卡片展示标准重量、配送收入、单笔配送距离，可直接修改参数
- 服务端保存订单批次与配送方案（仅当前账号可见，最近 50 条）

**02 配送决策（三种规划模式）**
- `business`（默认）：最大净收益 = 配送收入 − 里程成本 − 时间成本，同时受载重上限与时长上限约束
- `knapsack`：0/1 背包视角的最大收入，附动态规划精确最优值对照，明确标注是否达到容量模型最优
- `route`：固定全部送达点，比较「最近邻 + 2-opt」基准与 GA 精修后的路线

**03 校园地图与道路路线**
- 西安邮电大学长安校区西区仿真路网（23 个配送入口、81 个道路节点），支持缩放、拖动、地点定位、展开大图、只看有订单的地点
- 停靠顺序清单 + 逐段路线高亮；订单选取、每段里程与地图轨迹使用同一套道路数据

**03+ 路线进化回放**
- 计算完成后回放**真实逐代记录**的历史最佳路线（不是事后重算）：播放 / 单步 / 从头回放 / 跳至最终 / 调速 / 拖动进度条
- 同步展示「历史最佳总路程 ↓」与「路线适应度 ↑」双曲线；得分 = 1000 ÷ (1 + 总米数)
- 回放开启时关闭局部搜索并采用页面请求的 GA 参数，曲线与实际求解过程一致

**04 对照实验**
- 同一批订单对比纯背包、贪心、业务 GA 三种策略，固定种子 7、17、27、37、47 各跑一次
- 统计最高 / 平均净收益、样本标准差（n−1）、满足载重与时长约束的次数，支持导出 CSV

**原始算法工作台（保留）**
- `/legacy.html`：实例管理与 GA 求解、单参数扫描对照
- `/tsp.html`：att48 基准（已知最优 10628）实验页
- `/experiment.html?problem=knapsack|cec`：标准背包与 CEC 基准函数实验页

### 技术栈

| 层 | 选型 |
| --- | --- |
| Web 框架 | gin v1.12 |
| 分层 | `controller`（HTTP 适配）/ `logic`（业务）/ `dao`（存储）/ `models` |
| 存储 | MySQL（gorm v1.31，用户与快照）、Redis v9（会话） |
| 认证 | bcrypt 口令摘要 + 服务端会话 cookie（`qiji_session`，HttpOnly + SameSite=Strict），按 IP 限流 |
| 前端 | 原生 ES Module + DOM / SVG，无框架无构建 |
| 配置 | `.env` 文件或环境变量（`QIJI_*` 前缀，`.env` 不入库） |

### 目录结构

```
ga-tsp/
├── main.go                    # 入口：加载 .env → 连接 MySQL/Redis → 装配路由 → 监听端口
├── internal/
│   ├── config/                # .env 与环境变量读取
│   ├── controller/            # HTTP 适配层：认证、校园、实验、快照、静态托管与守卫
│   ├── logic/                 # 业务逻辑：plan（选单+路线+回放帧）、campus、analysis、auth
│   ├── dao/                   # gorm MySQL（用户、快照）、go-redis（会话、限流）
│   └── models/                # 请求 / 响应 / 领域结构
├── pkg/
│   ├── campus/                # 校园仿真路网（west.json 内嵌，含数据说明 README）
│   ├── tsp/                   # TSP 实例（att48 基准 / 随机欧氏）
│   ├── ga/                    # 遗传算法核心（排列 / 随机键编码、交叉变异、2-opt）
│   └── optimization/          # 0/1 背包 GA 与 CEC 基准函数
├── web/                       # 前端页面与脚本（index.html 为校园决策主页面）
├── work/report_measurements/  # 课程报告数据测量脚本
└── Dockerfile
```

### 快速开始

前置条件：Go 1.26.1 或兼容版本、可用的 MySQL 与 Redis。

```powershell
cd ga-tsp

# 1) 创建本地配置 .env（已被 .gitignore 忽略，不会入库），内容见下方示例

# 2) 启动（静态资源按当前工作目录的 web/ 加载，必须在 ga-tsp 目录内启动）
go run .
```

`.env` 示例：

```ini
QIJI_MYSQL_ADDR=127.0.0.1:3306
QIJI_MYSQL_USER=root
QIJI_MYSQL_PASSWORD=你的密码
QIJI_MYSQL_DATABASE=qiji_dispatch
QIJI_CREATE_DATABASE=true     # 首次运行自动建库
QIJI_REDIS_ADDR=127.0.0.1:6379
QIJI_COOKIE_SECURE=false      # HTTPS 部署时改 true
PORT=8081
```

打开 <http://localhost:8081>，未登录会跳转到 `auth.html`；注册账号即可使用（用户名 3～32 位字母、数字或下划线，密码 8～72 字节），账号与历史记录存放在 MySQL。

### API 一览

所有接口返回 JSON，错误为 HTTP 4xx/5xx + `{"error":"说明"}`；除认证接口外均需登录，请求体上限 1 MiB。

| 分组 | 接口 | 说明 |
| --- | --- | --- |
| 认证 | `POST /api/auth/register` · `login` · `logout` | 注册（每 IP 5 次/分）、登录（15 次/分）、退出 |
| 认证 | `GET /api/auth/me` | 当前登录用户 |
| 校园 | `GET /api/campus/map` | 校园路网（地点、节点、距离） |
| 校园 | `POST /api/campus/batches/generate` | 随机订单批次（数量 / 种子 / 场景） |
| 校园 | `POST /api/campus/plan` | 配送规划（三模式，含逐代 `routeFrames` 回放帧） |
| 校园 | `POST /api/campus/compare` | 三策略对照实验（固定种子集） |
| 校园 | `POST /api/campus/batches/save` · `plans/save` | 保存批次 / 方案快照 |
| 校园 | `GET /api/campus/history?kind=batch\|plan` · `GET /api/campus/history/:id` | 历史批次 / 方案（仅本人可见） |
| 基准 | `GET /api/instances` · `/api/instances/:name` · `POST /api/instance/random` | TSP 实例（att48 / 随机欧氏） |
| 求解 | `POST /api/solve` · `POST /api/scan` | GA 求解（逐代快照）、单参数扫描对照 |
| 实验 | `GET /api/knapsack/instance` · `/api/cec/functions` · `/api/cec/landscape` | 背包 / CEC 实验实例 |
| 实验 | `POST /api/experiments/solve` · `POST /api/experiments/scan` | 背包 / CEC 求解与多组对照 |
| 骑手 | `POST /api/dispatch/select` · `POST /api/dispatch/route` | 接单建议（背包）、多单巡回路线（TSP） |
| 骑手 | `POST /api/analysis/compare` | 多组配置 × 多种子的重复实验统计 |

### 算法与复现

- **选单 GA**：0/1 位串编码、锦标赛选择、单点交叉（0.9）、位翻转变异（0.03）；不可行解按「每单位约束缓解代价最小」的顺序修复；小规模订单用动态规划给出精确最优收入做对照
- **送单 GA**：排列 / 随机键编码、半数最近邻混合初始化、OX / PMX 交叉、逆转 / 交换 / 插入变异、精英保留，配合 2-opt 局部搜索；业务模式保留「基准与 GA 结果中较优者」，不虚构提升
- **可复现性**：固定种子可复现解与曲线，耗时随机器不同
- **计算预算**：单次规划 ≤ 30000 次个体评估、最长 45 秒；对照实验 ≤ 150 万次评估、最长 75 秒；前端取消请求后后端在迭代边界响应

### 测试与验证

```powershell
cd ga-tsp
go test ./...
go vet ./...
node --test web/export.test.mjs
```

覆盖：31 元接单实例、整数分金额、空 / 单订单、重复编号、同址合并、闭合路线、基准回退、两种编码合法性、固定种子复现、显式零参数、统计公式、预算与取消、CSV 转义、路线回放帧与最终方案一致性等。

### 报告测量脚本

```powershell
cd ga-tsp/work/report_measurements
go run . report.json    # 生成 att48 基准、参数敏感性、校园对照实验数据（JSON），供课程报告使用
```

---

## 二、题目十二：迷宫求解（`maze-bfs-dfs/`）

指导书题目十二的实验：在同一张迷宫上用 BFS 与 DFS 求解并对比。

- 预置指导书示例迷宫；支持自定义尺寸、密度、种子随机生成
- Canvas 绘制：墙、通路、已探索、回溯（死路）、当前格、最终路径六种状态着色
- 探索过程逐步动画回放，输出访问格子数、路径长度、耗时等统计
- 后端仅用标准库 `net/http`，无第三方依赖

### 快速开始

```powershell
cd maze-bfs-dfs
go run .
```

打开 <http://localhost:8080>（端口固定 8080）。

### API

| 接口 | 说明 |
| --- | --- |
| `GET /api/preset` | 返回指导书示例迷宫 |
| `POST /api/maze/generate` | 随机迷宫（`width`、`height`、`density`、`seed`） |
| `POST /api/solve` | 求解（`algorithm` = `bfs` / `dfs`），返回 `path`、`steps`、`visitedCount`、`pathLength`、`elapsedMs` |

### 测试

```powershell
cd maze-bfs-dfs
go test ./...
```

---

## 三、部署（Docker，主项目）

1. 本地交叉编译静态二进制（strip 后约 30 MB）：

```powershell
cd ga-tsp
cmd /c "set GOOS=linux&&set GOARCH=amd64&&set CGO_ENABLED=0&&go build -ldflags ""-s -w"" -o gatsp ."
```

2. 打包 `gatsp` + `web/` 上传服务器（`Dockerfile` 只 COPY 这两项，镜像基于 alpine:3.20，服务器无需 Go 工具链）
3. 构建并运行容器（需与 MySQL / Redis 网络互通）：

```bash
docker build -t ga-tsp:latest .
docker run -d --name ga-tsp --restart unless-stopped -p 8001:8001 \
  -e PORT=8001 \
  -e QIJI_MYSQL_ADDR=mysql:3306 -e QIJI_MYSQL_USER=root -e QIJI_MYSQL_PASSWORD=*** \
  -e QIJI_MYSQL_DATABASE=qiji_dispatch -e QIJI_CREATE_DATABASE=true \
  -e QIJI_REDIS_ADDR=redis:6379 \
  -e QIJI_COOKIE_SECURE=false \
  ga-tsp:latest
```

经 HTTPS 反向代理部署时，请设置 `QIJI_COOKIE_SECURE=true`。

---

## 四、工作区结构

```
courseDesign/
├── ga-tsp/            # 主项目：某团 · 校园配送决策（Go module: simple_tuan）
├── maze-bfs-dfs/      # 题目十二：迷宫求解（Go module: mazeweb）
├── docs/              # 设计与重构文档
├── .docx_text.txt     # 课程设计指导书文本提取
└── README.md          # 本文件
```

## 五、文档索引

- `ga-tsp/README.md` —— 原始算法工作台（选单 / 路线 / 参数扫描）详细说明
- `ga-tsp/pkg/campus/README.md` —— 校园路网数据来源、比例与编辑方式
- `ga-tsp/pkg/optimization/data/README.md` —— 背包与 CEC 数据说明
- `docs/superpowers/specs/2026-09-14-qiji-platform-design.md` —— 平台整合设计文档
- `docs/superpowers/specs/2026-09-15-simple-tuan-restructure-design.md` —— 三层重构设计
- `docs/superpowers/plans/2026-09-15-phase1-layered-structure.md` —— 重构实施计划

## 六、说明与边界

- 本系统为**仿真教学软件**：不执行真实派单，不含骑手端、配送跟踪、实时地图或支付结算
- 校园地图为参考真实布局绘制的**离线估算路网**，入口位置与路段长度未经测绘，页面已明确标注
- GA 为近似搜索：只有在精确对照（DP 最优值）支持时才标注「最优」
- 金额以整数分计算与存储，界面显示元；标准差使用样本标准差（n−1）
