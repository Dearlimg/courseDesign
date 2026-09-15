# 某团（simple_tuan）三层架构重构设计

日期：2026-09-15
状态：待用户审阅
范围：`courseDesign/ga-tsp`（Go 后端 + web 前端）

## 1. 背景与目标

骑迹配送决策系统已完成六大功能 + 认证。现按标准 Go 单体分层规范重构目录与技术栈：

- **分层**：`cmd` 放入口、`internal` 装全部私有实现（controller → logic → dao 单向依赖）、`models` 独立、`pkg` 放可外借的库
- **技术栈**：net/http → **gin**；database/sql → **gorm**（Redis 保持 go-redis）
- **改名**：模块 `gatsp` → **`simple_tuan`**，产品品牌「骑迹」→「某团」

## 2. 已确认决策（用户拍板）

| 决策点 | 结论 |
|---|---|
| 模块名 | `simple_tuan`（某团），二进制 `simple_tuan.exe`，原 `gatsp` 弃用 |
| 算法包 | `pkg/tsp`、`pkg/ga`、`pkg/optimization`（可外借纯算法库） |
| 节奏 | 三阶段：①目录分层（保持旧栈）→ ②换 gin → ③换 gorm，各自测试提交 |

## 3. 目标目录树（阶段①完成后）

```
ga-tsp/                              # 仓库子目录名保持（见 §8 开放问题）
├── cmd/
│   └── simple_tuan/
│       └── main.go                  # 仅装配：config → dao → logic → controller → router → 静态托管
├── internal/
│   ├── controller/                  # 协议适配：bind 参数、校验格式、统一响应
│   │   ├── router.go                # 路由注册总入口（原 api.NewMux）
│   │   ├── tsp.go                   # 实例/求解/扫描（原 handler.go 主体）
│   │   ├── dispatch.go              # 接单/路线/对比（原 api/dispatch.go）
│   │   ├── experiments.go           # 重复实验（原 api/experiments.go）
│   │   ├── auth.go                  # 认证四端点（原 auth/http.go 路由部分）
│   │   ├── middleware.go            # 登录守卫 + 安全头 + 跨站防护
│   │   └── response.go              # writeJSON / writeError / decodeBody
│   ├── logic/                       # 业务编排、事务边界，零 HTTP 依赖
│   │   ├── auth.go                  # 原认证 Service：bcrypt、会话、限流决策
│   │   ├── selection.go             # 原 dispatch/selection.go（智能接单）
│   │   ├── route.go                 # 原 dispatch/route.go（路线规划）
│   │   ├── analysis.go              # 原 analysis/compare.go（重复实验统计）
│   │   └── tsp.go                   # 新增薄封装：实例获取/求解/参数扫描编排（pkg/ga + pkg/tsp）
│   ├── dao/                         # 数据读写
│   │   ├── mysql_users.go           # 账号存取（① database/sql → ③ gorm）
│   │   └── redis_sessions.go        # 会话 + 限流计数（go-redis，始终不变）
│   ├── models/                      # PO/DTO，不 import 任何层
│   │   ├── user.go                  # User（响应 DTO）/ Account（含 hash 的 PO）
│   │   ├── dispatch.go              # Order / SelectionRequest / SelectionResult / RouteRequest / RouteResult
│   │   └── analysis.go              # 重复实验请求/结果 DTO
│   └── config/
│       └── config.go                # .env 读取（键名改 ST_ 前缀，见 §6）
├── pkg/                             # 可外借库：纯算法、零业务依赖
│   ├── tsp/                         # ← internal/tsp（att48 金标准 10628 测试原样迁移）
│   ├── ga/                          # ← internal/ga
│   └── optimization/                # ← internal/optimization（背包/CEC）
├── web/                             # 前端（品牌文字 → 某团，逻辑零改动）
├── go.mod                           # module simple_tuan
├── .env.example                     # ST_ 前缀键名
└── README.md                        # 品牌与结构说明更新
```

## 4. 依赖方向（编译器强制）

```
main(cmd) → controller → logic → dao → models
    │            │          │
    └── config   └────┬─────┘
                     ↓
                pkg/{tsp,ga,optimization}   （被 logic 与 models 引用；controller 不直接 import pkg）
```

> 说明：`models` 的业务 DTO（如 `SelectionResult.Evolution`）内嵌 `pkg/ga`、`pkg/optimization` 的结果类型，故允许 models → pkg。pkg 是零依赖可外借库而非分层成员，依赖图仍无环单向。

硬规则：
1. controller 只 import logic + models，**不得出现 SQL、业务判断**
2. logic 只 import dao + models + pkg，**零 HTTP 痕迹**（无 gin.Context，标准 context.Context 透传——现有代码已是此形态，保持）
3. dao 只 import models，构造函数收具体参数（地址/账号等字符串），**不 import config**
4. models 不 import 任何层
5. interface 定义在消费者侧：`logic` 定义 `UserStore`/`SessionStore` 接口（沿用现有 auth.Users/auth.Sessions 模式），dao 实现，仅单测 mock 时使用

## 5. 三阶段计划

### 阶段①：目录分层（保持 net/http + database/sql）
1. `go.mod` 改 module `simple_tuan`；全量 import 路径 `gatsp/...` → `simple_tuan/...`
2. `internal/api` 拆入 `internal/controller`（response.go 抽公共 writeJSON/writeError）；TSP 实验端点的 `tsp.*`/`ga.*` 直调改经 `logic/tsp.go` 薄封装，落实 controller 不碰 pkg 的硬规则
3. `internal/auth` 三拆：http.go 路由 → controller/{auth.go,middleware.go}；Service → logic/auth.go；store.go → dao/{mysql_users.go,redis_sessions.go}；User/Account → models/user.go
4. `internal/dispatch`、`internal/analysis` → `internal/logic`；其跨层模型 → `internal/models`
5. `internal/{tsp,ga,optimization}` → `pkg/`（算法代码零改动，仅 import 前缀）
6. `main.go` → `cmd/simple_tuan/main.go`（纯装配）
7. 前端品牌：index.html/auth.html/README「骑迹 QIJI」→「某团 SIMPLE_TUAN」
8. 验收：`go test ./...` 全过；浏览器全功能回归；API 契约不变

### 阶段②：gin 替换 net/http
1. 引入 `github.com/gin-gonic/gin`
2. controller 全部 handler 改 `func(c *gin.Context)`：`ShouldBindJSON`（保留 DisallowUnknownFields 语义需自定义校验）、`c.JSON`、`c.Param`
3. middleware.go 改 gin 中间件链：登录守卫 / 安全头 / Origin+Sec-Fetch-Site 跨站防护
4. 静态托管：`router.NoRoute` + `http.FileServer`（保持现有缓存头行为）
5. context 规则：`c.Request.Context()` 下沉到 logic（硬规则）
6. 统一响应保持 `{"error": msg}` 原契约，前端零改动
7. 测试迁移：httptest 直接打 gin engine；断言不变
8. 验收：同阶段①

### 阶段③：gorm 替换 database/sql
1. 引入 `gorm.io/gorm` + `gorm.io/driver/mysql`
2. `dao/mysql_users.go` 改 gorm：
   - 建库：bootstrap 连接 + `Exec("CREATE DATABASE IF NOT EXISTS ...")`（gorm 不建库）
   - 建表：`AutoMigrate(&models.Account{})` 替代手写 SQL
   - 查询：`db.WithContext(ctx)` 透传超时/取消（硬规则）
   - 唯一约束冲突：判断 `errors.Is(err, gorm.ErrDuplicatedKey)`（需 `TranslateError: true`）
3. models PO 加 gorm 标签：`TableName() = "st_users"`、列名映射、`PasswordHash` 不打 json tag（沿用 Account 嵌入 User 的响应隔离）
4. 验收：单元测试 + `ST_INTEGRATION_TEST=1` 真实库联调 + 浏览器回归

## 6. 命名变更清单

| 旧 | 新 |
|---|---|
| module `gatsp` | `simple_tuan` |
| 二进制 `gatsp.exe` / `qiji` | `simple_tuan`（.gitignore 同步） |
| 品牌展示「骑迹 QIJI·DISPATCH」 | 「某团 SIMPLE_TUAN·DISPATCH」 |
| 环境变量 `QIJI_*`（ADDR/USER/PASSWORD/DATABASE/CREATE_DATABASE/REDIS_*/COOKIE_SECURE） | `ST_*` 同名映射 |
| 联调开关 `QIJI_INTEGRATION_TEST` | `ST_INTEGRATION_TEST` |
| 浏览器夹具开关 `QIJI_BROWSER_PREVIEW` | `ST_BROWSER_PREVIEW` |
| MySQL 库 `qiji_dispatch` | `st_dispatch` |
| MySQL 表 `qiji_users` | `st_users` |
| Cookie 名 `qiji_session` | `st_session` |
| Redis key 前缀 `qiji:session:v1:` / `qiji:rate:v1:` | `st:session:v1:` / `st:rate:v1:` |

> 服务器上旧 `qiji_*` 库表与 Redis key 自然遗留、无冲突，无需清理。

## 7. 不变量（每阶段验收标准）

1. **API 路径与 JSON 契约完全不变**（前端零改动）：`/api/auth/*`、`/api/dispatch/*`、`/api/analysis/compare`、`/api/instances*`、`/api/solve`、`/api/scan`、`/api/experiments/*`
2. `go test ./...` 全过；att48 最优回路 **10628** 金标准不破坏
3. 认证行为不变：未登录页面 303 → `/auth.html`、API 401、限流阈值（注册 5/分、登录 15/分）、会话 24h
4. Docker 部署流程不变（交叉编译 + COPY alpine）

## 8. 风险与开放问题

| 项 | 对策 / 默认决策 |
|---|---|
| import 全量重写易漏 | 阶段① 用编译器驱动：改 go.mod 后 `go build ./...` 逐包修复，全绿才提交 |
| `DisallowUnknownFields` gin 不原生支持 | 阶段② 保留自定义 decode（decoder.DisallowUnknownFields）作为 bind 实现，行为不回退 |
| gorm 唯一键错误翻译 | 开 `TranslateError`，单测覆盖 409 重复注册路径 |
| 大请求体限制（1MB/4KB） | 保留 `http.MaxBytesReader` 包装 `c.Request.Body` |
| 仓库子目录名 `ga-tsp/` | **默认保持**（避免大规模 git rename 与部署路径变化；模块名/品牌已彻底改，目录仅是物理位置）。如需改名另起任务 |
| 三阶段各自提交 | commit 信息：`refactor: 分层目录重构`、`refactor: 引入 gin`、`refactor: 引入 gorm` |
