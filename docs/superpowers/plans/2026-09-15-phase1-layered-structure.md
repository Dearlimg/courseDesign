# 某团重构 · 阶段①（目录分层）实施计划（已执行完毕）

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在保持 net/http + database/sql 旧栈的前提下，将 `ga-tsp` 重构为 `simple_tuan` 模块的 controller/logic/dao/models/pkg 分层结构，API 契约与全部测试不变。

**Architecture:** 编译器驱动的原子迁移：每任务移动一组包→修复 import→`go build ./...` + `go test ./...` 全绿→commit。阶段②（gin）、阶段③（gorm）另行出计划。

**Tech Stack:** Go 1.26 标准库（本阶段零新增依赖）、go-sql-driver/mysql、go-redis/v9、bcrypt。

**Spec:** `docs/superpowers/specs/2026-09-15-simple-tuan-restructure-design.md`

## 执行结果总览（2026-09-15）

三阶段全部完成，共 6 个 commit：

| 阶段 | Commit | 内容 |
|---|---|---|
| ① | `c0f9591` | Task 1 模块更名 simple_tuan |
| ① | `26bfedf` | Task 2 算法包迁移 pkg/ |
| ① | `f480684` | Task 3 业务域三层化 models+logic |
| ① | `272c63c` | Task 4+5 认证三层拆分 + api→controller（合并提交） |
| ② gin | `5445a3b` | HTTP 层迁移 gin（严格解码与错误契约不变） |
| ③ gorm | `bb93cf8` | 账号持久层迁移 gorm（AutoMigrate+TranslateError） |

**与计划的差异（未执行项）:** Task 5 Step 1（logic/tsp.go+experiment.go 薄封装未建，controller 仍直接 import pkg）、Step 3（cmd/ 入口化未做，main.go 仍在 ga-tsp/ 根目录）；Task 6 全部未做（QIJI_ 前缀、品牌骑迹、表/键/cookie 命名沿用旧名）。

## Global Constraints

- 工作目录：`c:\Users\14925\Desktop\code\go\courseDesign\ga-tsp`（仓库根为上一级）
- API 路径与 JSON 响应契约**完全不变**（前端零改动）
- att48 最优回路 == 10628 金标准测试必须原样保留并通过
- 每任务结束 `go test ./...` 全绿才可 commit；不 push
- PowerShell 中 `$` 变量可用，但复杂管道写 .ps1 临时脚本执行
- 依赖方向：controller → logic → dao → models；logic/models → pkg；dao 不 import config；pkg 不 import 项目内任何层

---

### Task 1: 模块更名 simple_tuan

**Files:**
- Modify: `go.mod`（module 行）
- Modify: 全部 `*.go`（import 前缀）

**Interfaces:** 无代码接口变化，仅 import 路径 `gatsp/...` → `simple_tuan/...`

- [x] **Step 1: 改 go.mod module 名**

`go.mod` 第一行 `module gatsp` → `module simple_tuan`

- [x] **Step 2: 批量替换 import 前缀**

PowerShell（在 ga-tsp 目录）：
```powershell
Get-ChildItem -Recurse -Include *.go | ForEach-Object {
  (Get-Content $_.FullName -Raw) -replace '"gatsp/', '"simple_tuan/' | Set-Content -NoNewline $_.FullName
}
```

- [x] **Step 3: 验证编译与测试**

Run: `go build ./... ; go vet ./... ; go test ./...`
Expected: 全部通过（无失败包）

- [x] **Step 4: Commit** —— `c0f9591`

```
git add -A ga-tsp
git commit -m "refactor: 模块更名 simple_tuan"
```

---

### Task 2: 算法包迁移至 pkg/

**Files:**
- Move: `internal/tsp/` → `pkg/tsp/`
- Move: `internal/ga/` → `pkg/ga/`
- Move: `internal/optimization/` → `pkg/optimization/`（含 .txt 数据与 README）
- Modify: 引用方 import（api、dispatch、analysis 包内）

**Interfaces:** 包名不变（tsp/ga/optimization），仅路径变化

- [x] **Step 1: git mv 三个目录**

```powershell
New-Item -ItemType Directory -Force pkg | Out-Null
git mv internal/tsp pkg/tsp
git mv internal/ga pkg/ga
git mv internal/optimization pkg/optimization
```

- [x] **Step 2: 批量替换引用**

```powershell
Get-ChildItem -Recurse -Include *.go | ForEach-Object {
  (Get-Content $_.FullName -Raw) `
    -replace '"simple_tuan/internal/tsp"', '"simple_tuan/pkg/tsp"' `
    -replace '"simple_tuan/internal/ga"', '"simple_tuan/pkg/ga"' `
    -replace '"simple_tuan/internal/optimization"', '"simple_tuan/pkg/optimization"' |
    Set-Content -NoNewline $_.FullName
}
```

- [x] **Step 3: 验证**

Run: `go build ./... ; go test ./...`
Expected: 全绿；`internal/` 下仅剩 api/auth/config/dispatch/analysis

- [x] **Step 4: Commit** `refactor: 算法包迁移至 pkg` —— `26bfedf`

---

### Task 3: 业务域三层化（models + logic，dispatch/analysis 消失）

**Files:**
- Create: `internal/models/user.go`（User、Account，自 `internal/auth`）
- Create: `internal/models/dispatch.go`（Order、ExcludedOrder、SelectionRequest、SelectionResult、Point、RouteRequest、RouteResult，自 `internal/dispatch`）
- Create: `internal/models/analysis.go`（Group、Request、Run、Summary、Result，自 `internal/analysis`）
- Create: `internal/logic/selection.go`（Select、ValidateOrders、DefaultSelectionParams、validCoordinate）
- Create: `internal/logic/route.go`（Route、RouteInstance、DefaultRouteParams、rotateDepot）
- Create: `internal/logic/analysis.go`（Compare、validate、summarize）
- Delete: `internal/dispatch/`、`internal/analysis/`
- Modify: `internal/api/`（引用改 `simple_tuan/internal/logic` + `simple_tuan/internal/models`）

**Interfaces（后续任务依赖）:**
```go
// logic 包导出（业务编排入口，签名与原 dispatch/analysis 包一致）
func Select(ctx context.Context, req models.SelectionRequest) (models.SelectionResult, error)
func Route(ctx context.Context, req models.RouteRequest) (models.RouteResult, error)
func RouteInstance(req models.RouteRequest) (*tsp.Instance, []models.Point, error)
func ValidateOrders(orders []models.Order) error
func Compare(ctx context.Context, req models.Request) (models.Result, error)
```

**迁移规则:**
- 模型结构体整体剪切到 models 包，字段与 json tag 逐字保留
- `dispatch/route.go` 的 `validCoordinate` 与 `selection.go` 重复定义 → logic 包内保留一份
- 测试文件跟随：`dispatch/selection_test.go`、`route_test.go` → `logic/`（改包名与 import）；`analysis/compare_test.go` → `logic/`

- [x] **Step 1: 建 models 三文件**（结构体剪切，包名 `models`）
- [x] **Step 2: 建 logic 三文件**（函数剪切，签名中 dispatch.X → models.X）
- [x] **Step 3: 迁测试**（git mv 到 `internal/logic/`，改 `package logic` 与 import）
- [x] **Step 4: 改 api 包引用**（`internal/dispatch` → `internal/logic` + `internal/models`；`internal/analysis` → 同）
- [x] **Step 5: 删空目录** `internal/dispatch`、`internal/analysis`
- [x] **Step 6: 验证** `go build ./... ; go test ./...` 全绿
- [x] **Step 7: Commit** `refactor: 业务逻辑归位 logic 层` —— `f480684`

---

### Task 4: 认证模块三层拆分（auth 包消失）

**Files:**
- Create: `internal/dao/mysql_users.go`（MySQLUsers + NewMySQLUsers，自 `auth/store.go` 前半）
- Create: `internal/dao/redis_sessions.go`（RedisSessions + NewRedisSessions + sessionKey/digest/rateScript，自 `auth/store.go` 后半）
- Create: `internal/logic/auth.go`（Users/Sessions 接口、ErrNotFound/ErrDuplicate、Service、业务方法）
- Create: `internal/controller/response.go`（writeJSON/writeError/decodeBody/timeNowNanos）
- Create: `internal/controller/auth.go`（四个 endpoint + cookie 读写 + readCredentials）
- Create: `internal/controller/middleware.go`（安全头 + Origin/Sec-Fetch-Site 防护 + 登录守卫 require）
- Create: `internal/controller/app.go`（NewApp：auth 路由 + 中间件包装 + 静态托管）
- Delete: `internal/auth/`
- Modify: `main.go`（装配改 dao.New* + logic.NewAuth）

**Interfaces（Task 5 依赖）:**
```go
// internal/dao
func NewMySQLUsers(ctx context.Context, addr, user, password, database string, createDB bool) (*MySQLUsers, error)
func NewRedisSessions(ctx context.Context, addr, password string) (*RedisSessions, error)

// internal/logic
type Users interface { Create(context.Context, string, []byte) (models.User, error); Find(context.Context, string) (models.Account, error) }
type Sessions interface { Put(context.Context, string, models.User, time.Duration) error; Get(context.Context, string) (models.User, error); Delete(context.Context, string) error; Allow(context.Context, string, int, time.Duration) (bool, error) }
func NewAuth(users Users, sessions Sessions, secure bool) *AuthService
// 方法（零 HTTP）：Register(ctx, name, pass) (models.User, error)
//                  Login(ctx, name, pass) (models.User, token string, error)  // token 生成与 Redis 写入在 logic
//                  Logout(ctx, token) error
//                  User(ctx, token) (models.User, error)
//                  Allow(ctx, key string, limit int) (bool, error)  // 限流决策，key 由 controller 组装

// internal/controller
func NewApp(svc *logic.AuthService, next http.Handler) http.Handler
```

**拆分规则:**
- `store.go`：MySQL 部分与 Redis 部分按上述切开；`Open(ctx, config.Config)` 拆成两个 New*（构造函数收具体参数，dao 不 import config）
- `http.go`：cookie 读写、`readCredentials`、限流 key（path+ip）留在 controller；`Service` 的 bcrypt 比对、dummyHash、token 生成（crypto/rand 32B）、会话增删查移入 logic.Login/User/Logout
- 错误语义保持：注册重复 409、登录失败 401、限流 429、会话服务故障 503 的判定逻辑逐条保留
- 测试迁移：`http_test.go` → `controller/`（httptest 打 NewApp，memory 实现随迁）；`infrastructure_test.go` → `dao/`（改 `ST_INTEGRATION_TEST` 在 Task 6，本任务保持 QIJI_）；`browser_preview_test.go` → `controller/`（暂保留，Task 5 调整装配引用）

- [x] **Step 1: 建 dao 两文件**（自 store.go 剪切，建库/建表 SQL 原样）
- [x] **Step 2: 建 logic/auth.go**（接口 + Service + 五个业务方法）
- [x] **Step 3: 建 controller 四文件**（auth.go/middleware.go/response.go/app.go）
- [x] **Step 4: 改 main.go 装配**（NewMySQLUsers + NewRedisSessions + logic.NewAuth + controller.NewApp）
- [x] **Step 5: 迁测试并删 internal/auth/**
- [x] **Step 6: 验证** `go build ./... ; go test ./...` 全绿
- [x] **Step 7: Commit** —— 实际与 Task 5 合并提交：`272c63c refactor: 目录分层 controller/logic/dao/models`

---

### Task 5: api → controller + 入口 cmd 化

**Files:**
- Move: `internal/api/{handler,dispatch,experiments,analysis,time}.go` → `internal/controller/`
- Create: `internal/logic/tsp.go`（ListInstances/GetInstance/RandomInstance/SolveTSP/ScanTSP + buildInstance + scanParamDefs）
- Create: `internal/logic/experiment.go`（RandomKnapsack/Benchmarks/CECLandscape/SolveExperiment/ScanExperiment 编排，自 api/experiments.go）
- Move: `main.go` → `cmd/simple_tuan/main.go`
- Delete: `internal/api/`
- Create: `internal/controller/router.go`（NewRouter() 注册全部业务 API，替代原 NewMux）

**Interfaces:**
```go
// internal/controller — main 装配入口
func NewRouter() *http.ServeMux                       // 业务 API（无认证，供测试直打）
// NewApp(svc, next) 已在 Task 4 提供                    // 完整应用（认证 + 守卫 + 静态）

// internal/logic（新增薄封装，controller 不再 import pkg）
type InstanceSummary struct { Name string; Size int; Optimal float64; EdgeType string }
func ListInstances() []InstanceSummary
func GetInstance(name string) (*tsp.Instance, error)
func RandomInstance(n int, seed int64) (*tsp.Instance, error)
type SolveRequest struct { Instance InstancePayload; Params ga.Params }
func SolveTSP(req SolveRequest) (*ga.Result, error)   // 含 buildInstance + Normalize
type ScanRequest struct { Instance InstancePayload; Params ga.Params; Param string; Values []float64 }
func ScanTSP(req ScanRequest) ([]map[string]any, error) // 含 scanParamDefs + 预算校验
```

**迁移规则:**
- handler.go 的 `instancePayload`/`buildInstance`/`scanParamDefs` → logic/tsp.go；controller 的 tsp.go 只留 bind + 调 logic + 响应
- experiments.go 的 `experimentRequest`/`solveExperiment`/scan 编排与预算校验 → logic/experiment.go；controller 留 bind + 响应
- 测试迁移：`api_test.go`/`experiments_test.go`/`dispatch_test.go`/`decision_integration_test.go` → `controller/`（改打 NewRouter；断言不变）
- `browser_preview_test.go` 装配改 `NewRouter()` + 静态
- main.go：`go build -o simple_tuan.exe ./cmd/simple_tuan`（注意构建目录变化）

- [ ] **Step 1: 建 logic/tsp.go + logic/experiment.go**（未做：薄封装未建，controller 仍直接 import pkg）
- [x] **Step 2: api 四文件迁 controller，删 internal/api**（实际以 mux.go 保留业务路由，未新建 router.go）
- [ ] **Step 3: main.go → cmd/simple_tuan/main.go**（未做：入口仍在 ga-tsp/main.go，构建仍 `go build -o gatsp.exe .`）
- [x] **Step 4: 迁测试**（4 个 api 测试 → controller）
- [x] **Step 5: 验证** `go build ./... ; go test ./...` 全绿（cmd 构建因 Step 3 未做而无）
- [x] **Step 6: Commit** —— 与 Task 4 合并：`272c63c`

---

### Task 6: 命名统一 ST_ 前缀 + 品牌某团 + 全链路验收

**Files:**
- Modify: `internal/config/config.go`（QIJI_* → ST_* 共 8 键）
- Modify: `.env.example`（ST_ 键）+ 本地 `.env`（不提交）
- Modify: `internal/dao/mysql_users.go`（表 `qiji_users` → `st_users`）
- Modify: `internal/dao/redis_sessions.go`（key 前缀 `qiji:session:v1:`/`qiji:rate:v1:` → `st:session:v1:`/`st:rate:v1:`）
- Modify: `internal/controller/auth.go`（cookie 名 `qiji_session` → `st_session`）
- Modify: `internal/dao/infrastructure_test.go`（`QIJI_INTEGRATION_TEST` → `ST_INTEGRATION_TEST`）、`browser_preview_test.go`（`QIJI_BROWSER_PREVIEW` → `ST_BROWSER_PREVIEW`）
- Modify: `cmd/simple_tuan/main.go` 启动日志「骑迹」→「某团」
- Modify: `web/index.html`（title、侧栏品牌「骑迹 QIJI·DISPATCH」→「某团 SIMPLE_TUAN·DISPATCH」、footer）、`web/auth.html`（品牌、title、副标、footer）
- Modify: `README.md`（品牌与目录结构说明）
- Modify: `.gitignore`（`ga-tsp/qiji.exe`/`ga-tsp/qiji` → `ga-tsp/simple_tuan.exe`；新增 `ga-tsp/gatsp.exe`、`ga-tsp/gatsp`；`git rm --cached ga-tsp/gatsp ga-tsp/gatsp.exe` 退出跟踪）
- Modify: `Dockerfile`（COPY 路径适配新二进制名）

**验证（全部执行）:**
- [ ] **Step 1: 代码与配置改名**（上表逐项）
- [ ] **Step 2: 本地 .env 键名同步**（QIJI_ → ST_，值不变；库名 qiji_dispatch → st_dispatch）
- [ ] **Step 3: `go build ./... ; go test ./...` 全绿**
- [ ] **Step 4: 真实库联调** `$env:ST_INTEGRATION_TEST="1"; go test ./internal/dao/ -run TestInfrastructure -v`（新表 st_users 自动建立）
- [ ] **Step 5: 启动服务**（`go build -o simple_tuan.exe ./cmd/simple_tuan` + Start-Process，工作目录 ga-tsp）+ 浏览器验收：登录页（某团品牌）→ 注册新账号 → 登录 → 工作台四模块 → 退出（表名已换，旧 demodispatch 账号不在新表，属预期）
- [ ] **Step 6: Commit** `refactor: 命名统一为某团 ST_ 前缀`

---

## 阶段②③预告（已完成，2026-09-15）

- **阶段② gin** ✅ `5445a3b`：controller 全量换 gin.HandlerFunc/gin.Context，`c.Request.Context()` 下沉；NoRoute 静态兜底；自定义 decode 保留 DisallowUnknownFields + 二次 Decode 拒多 JSON；`{"error": msg}` 契约不变，契约测试原样通过
- **阶段③ gorm** ✅ `bb93cf8`：dao/mysql_users.go 换 gorm v1.31.2 + driver/mysql v1.6.0（AutoMigrate 建表、WithContext 透传、TranslateError 判 409 重复注册/404 不存在）；真实库集成联调 11.78s 通过
