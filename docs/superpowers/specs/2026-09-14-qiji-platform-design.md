# 「骑迹」外卖骑手智能调度平台 · 产品设计文档

> 状态：草案 v2（纳入 0/1 背包 = 骑手自动最优接单）· 待用户复审
> 日期：2026-09-14
> 关联项目：`maze-bfs-dfs`（BFS/DFS 路径搜索）、`ga-tsp`（GA·TSP 巡回 + GA·0/1 背包接单 + CEC 基准）
> 整合方案：**方案 B · 深整合单服务**（合并为一个 Go 二进制 + 统一 API + 单页应用）

---

## 1. 产品定位

**「骑迹」（QiJi）** 是一个面向**外卖骑手调度中心**的 Web 工作台，用四种智能搜索/优化算法解决骑手调度的四类核心问题：同城最短送达、片区可达性勘探、多单巡回优化、自动最优接单。

- **一句话定位**：给骑手调度员用的浏览器工作台，用 BFS/DFS/遗传算法分别算"单的最短送达路、片区可不可达、多单怎么排最省、接哪些单最赚"。
- **目标用户**：外卖平台城市站调度员 / 配送站长（答辩演示对象：指导老师与同学）。
- **核心价值**：把三个孤立的算法演示 + 一个背包扩展，串成一个有真实业务背景、有统一交互、可对比的调度软件，既覆盖课程设计算法考点（最短路 BFS、深度搜索 DFS、进化优化 GA——含 TSP 巡回与 0/1 背包接单两个 GA 应用），又给答辩一条完整故事线。

> 产品名为暂定「骑迹」（骑手 + 轨迹）。可在实现前替换，不影响架构。

---

## 2. 背景故事

四个算法的共同本质是 **"在图/网络上搜索、在有限预算下选择与优化"**，这恰好是骑手调度的底层问题。把现有"迷宫"重新定义为 **"城市栅格路网"**（墙=建筑/禁行，通道=可骑行街道），三个图算法天然落地成骑手的三件正事；再把 0/1 背包映射成"接单决策"，凑成调度闭环——**背包决定"接哪几单"，TSP 决定"接了怎么送"**：

| 算法 | 骑手业务角色 | 业务问题 | 用户故事 |
|---|---|---|---|
| **BFS** | 最短送达引擎 | 取餐商家 → 送达用户的最短骑行路线（最少步数） | "这一单从 A 商家送到 B 用户，走哪条最短？" |
| **DFS** | 配送可达性勘探 | 新签约片区/新楼宇是否都在骑手可达网络内（穷举可达区）；并给一条可行路线与最优对比 | "新承包这片街区，所有地址都送得到吗？" |
| **GA·TSP** | 多单合并调度优化 | 一个骑手同时配送多单，排送达顺序使总里程/超时最小（多单 TSP） | "手里 10 单合并送，怎么排顺序最省时？" |
| **GA·背包** | 自动最优接单引擎 | 单次出车时间预算内，从订单池挑哪些单接使总收益最大（0/1 背包） | "候选单一堆、时间有限，接哪几单这趟最赚？" |

> **背包 → 接单 映射**：背包容量 W = 单次出车时间预算；物品 = 待接订单；重量 w_i = 订单预计耗时；价值 v_i = 订单收益；约束 Σw_i≤W（不超时）；目标 Σv_i 最大；0/1 = 接/不接。
>
> 一条调度员一天的故事线讲完四个算法应用，答辩叙事完整。

---

## 3. 用户角色与典型场景

**角色**：骑手调度员（操作「骑迹」Web 工作台）。

**典型场景（一天四件事）**：

1. **上午 · 单单最短送达（BFS）**
   调度员在路网编辑器里画好片区栅格路网，设取餐点（起点）和送达点（终点），点"最短送达"。BFS 一层层扩散的搜索过程被可视化，给出最短路线与步数、耗时。
2. **上午 · 片区可达性勘探（DFS）**
   新签约片区接入，调度员要确认所有目标地址都从配送站可达。DFS 从配送站出发穷举可达区，标出不可达的"孤岛"，并给一条可行路线与 BFS 最短路对比，直观看到"可行 vs 最优"的差距。
3. **中午 · 多单巡回优化（GA·TSP）**
   高峰期一个骑手手里 10 单，调度员把多个配送点交给巡回调度引擎。遗传算法种群进化过程可视化（种群云、收敛曲线、多样性、最优差距），给出最优送达顺序与总里程，并能做参数对比实验找更好配置。
4. **下午 · 自动最优接单（GA·背包）**
   下一轮高峰前的接单窗口，调度员把候选订单池和骑手单次出车时间预算交给接单引擎。遗传算法在"接/不接"组合空间进化，可视化订单网格逐代筛选（接=高亮、不接=灰）、收益柱状、收敛曲线，给出最优接单子集、总收益、总耗时与是否超预算，并能与 DP 精确最优对比、切换贪心修复开关看收敛差异。

---

## 4. 功能规格

### 4.1 模块清单

| # | 模块 | 算法 | 复用来源 |
|---|---|---|---|
| M1 | 路网编辑器 | —（数据准备） | `maze-bfs-dfs/web` 编辑交互 |
| M2 | 最短送达引擎 | BFS | `maze-bfs-dfs/internal/maze` + `solver` |
| M3 | 可达性勘探器 | DFS | 同上（`solver.Solve(...,"dfs")`） |
| M4 | 巡回调度引擎 | GA·TSP | `ga-tsp/internal/tsp` + `ga` |
| M5 | 算法对比面板 | BFS/DFS/GA | 新增（聚合各模块统计） |
| M6 | 参数对比实验 | GA | `ga-tsp/internal/api` 的 `/api/scan` |
| M7 | 自动最优接单引擎 | GA·0/1背包 | `ga-tsp/internal/optimization`(knapsack) |

### 4.2 功能详述

**M1 路网编辑器**
- 画布栅格路网（墙=建筑/禁行、通路=街道、起点=取餐点、终点=送达点）；点击格子切换墙/通路；行列数、墙密度可调；随机生成 + 预置示例（题目十二 5×5）。
- 多配送点模式：可在巡回调度模块里把多个格点标为配送点，喂给 GA。

**M2 最短送达引擎（BFS）**
- 输入：路网 + 起点 + 终点。输出：最短路径坐标序列、探索过程快照序列（每步访问的格子）、路径长度、访问格子数、耗时。
- 可视化：BFS 层层扩散动画、最短路径高亮；播放/暂停/单步/重置/调速/进度条。

**M3 可达性勘探器（DFS）**
- 输入：路网 + 起点（配送站）。输出：一条可行路径、DFS 探索过程（含死路回溯）、全可达区集合、不可达格子集合、路径长度、访问格子数、耗时。
- 可视化：DFS 深入-回溯动画、可达区高亮、死路标记；与 BFS 同框对比"最短 vs 可行"统计表。

**M4 巡回调度引擎（GA·TSP）**
- 输入：TSP 实例（内置 att48 / 随机欧氏 / 手选多配送点）+ GA 参数（种群、代数、交叉/变异概率、精英、种子、选择/交叉/变异算子、2-opt 局部搜索开关）。
- 输出：逐代快照（当代最优/平均/多样性/已知最优差距/较初始改善）、最终最优回路、收敛代数、耗时、最优差距 Gap。
- 可视化：城市地图 + 种群云 + 逐代最优回路动画 + 收敛曲线 + 多样性曲线；播放/单步/首末代/跳收敛点/调速/进度条 + 键盘快捷键。

**M5 算法对比面板**
- 在同一（或各自）路网/实例上横向对比 BFS / DFS / GA(TSP) / GA(背包) 的：搜索步数、访问节点数、路径长度/收益、耗时、最优性；给出文字结论。

**M6 参数对比实验（GA）**
- 固定种子，每次只变一个参数（变异/交叉概率、种群、精英），跑最多 8 组，对比收敛曲线与解质量表。

**M7 自动最优接单引擎（GA·0/1 背包）**
- 输入：订单池（每个订单含预计耗时 w_i、收益 v_i）+ 骑手单次出车时间预算 W + GA 参数（含 `GreedyRepair` 贪心修复开关）。
- 输出：最优接单子集（接/不接位串）、总收益 Σv、总耗时 Σw、是否超预算、逐代快照、与 DP 精确最优的对比。
- 可视化：订单网格逐代筛选动画（接=高亮、不接=灰）、收益/耗时柱状、收敛曲线、贪心修复开关 A/B 对比；播放/单步/调速/进度条。

### 4.3 统一交互约定（贯穿所有模块）

播放 / 暂停 / 单步 / 重置(首) / 跳末 / 调速滑块 / 进度条 + 文字进度。键盘：空格播放暂停、←→单步、Home/End 首末、C 跳收敛点（M4）。所有 Canvas 可视化复用现有实现，不重写交互。

---

## 5. 系统架构（方案 B · 深整合单服务）

### 5.1 整体

- **单一 Go 二进制** `qiji`，单一 module `qiji`，单一端口（默认 `:8080`，`PORT` 环境变量覆盖）。
- 后端：`net/http` + `http.ServeMux`（Go 1.22+ 方法+路径模式），托管统一前端静态目录 `web/`。
- 前端：单页应用（SPA 风格，多视图切换，原生 HTML/CSS/JS + Canvas，**零第三方依赖**）。
- 算法核心包直接复用：`maze`/`solver`/`tsp`/`ga`/`optimization` 为纯逻辑，仅需改 import path。背包求解复用 `optimization.SolveKnapsack`（含 `GreedyRepair` 开关与 DP 精确最优对比）。

### 5.2 目录结构

```
qiji/
├── go.mod                 # module qiji
├── main.go                # 统一入口：NewMux() + FileServer(web/) + :8080
├── internal/
│   ├── maze/              # 复用自 maze-bfs-dfs/internal/maze（栅格路网）
│   ├── solver/            # 复用自 maze-bfs-dfs/internal/solver（BFS/DFS）
│   ├── tsp/               # 复用自 ga-tsp/internal/tsp（实例 + att48）
│   ├── ga/                # 复用自 ga-tsp/internal/ga（遗传算法）
│   ├── optimization/      # 复用自 ga-tsp（背包 knapsack + CEC 基准）
│   └── api/               # 新写：统一路由注册 + 各 handler
│       ├── mux.go         #   NewMux：注册 /api/nav/* /api/tsp/* /api/order/* /api/exp/cec/* + 公共工具
│       ├── nav.go         #   导航/勘探 handler（preset/generate/solve）
│       ├── tsp.go         #   巡回调度 handler（instances/solve/scan）
│       ├── order.go       #   自动接单 handler（instance/solve/scan，复用 knapsack）
│       ├── experiments.go #   复用 CEC 基准（knapsack 已提升为 order）
│       └── util.go        #   decodeBody/writeJSON/writeError（统一）
└── web/
    ├── index.html         # 单页应用骨架 + 骑迹品牌顶栏 + 侧栏视图切换
    ├── style.css          # 统一品牌主题（骑手配色）
    ├── app.js             # 路由分发 + 各模块视图 + 复用可视化
    ├── views/
    │   ├── nav.html       # M2 最短送达（BFS）视图
    │   ├── explore.html   # M3 可达性勘探（DFS）视图
    │   ├── dispatch.html  # M4 巡回调度（GA）视图
    │   ├── editor.html    # M1 路网编辑器视图
    │   └── compare.html   # M5 对比视图
    └── ...                # 各视图配套 js/css 按需拆分
```

> 视图文件可内联进 `index.html` 或按需 fetch，实现期定。原则：单页内多视图切换，不跳整页。

### 5.3 后端复用与改动清单

| 来源 | 处置 | 改动 |
|---|---|---|
| `maze-bfs-dfs/internal/maze` | 直接复用 | 仅改 package import path（`mazeweb` → `qiji`） |
| `maze-bfs-dfs/internal/solver` | 直接复用 | 同上 |
| `maze-bfs-dfs/internal/api/handler.go` | 重写为 `nav.go` | 路由前缀改 `/api/nav/*`；handler 逻辑同 |
| `ga-tsp/internal/tsp` | 直接复用 | 改 import path |
| `ga-tsp/internal/ga` | 直接复用 | 改 import path |
| `ga-tsp/internal/optimization` | 直接复用（背包接单 + CEC 基准） | 改 import path |
| `ga-tsp/internal/api/handler.go` | 重写为 `tsp.go` | 路由前缀改 `/api/tsp/*`；handler 逻辑同 |
| `ga-tsp/internal/api/experiments.go` | 拆分 | knapsack → `order.go`（`/api/order/*`）；cec 保留 `/api/exp/cec/*` |
| `maze-bfs-dfs/main.go` / `ga-tsp/main.go` | 合并为 `qiji/main.go` | 单一 `NewMux` + 单端口 |
| 两边 `util`（decodeBody/writeJSON/writeError） | 合并为 `api/util.go` | 去重 |

> 算法核心零改动，风险集中在 api 路由重写 + 前端单页化。

---

## 6. 统一 API 设计

### 6.1 端点总表

**导航/勘探模块（原 maze-bfs-dfs）**

| 方法 | 路径 | 说明 | 请求体 | 来源 |
|---|---|---|---|---|
| GET | `/api/nav/preset` | 预置路网示例（题目十二 5×5） | — | 原 `/api/preset` |
| POST | `/api/nav/generate` | 随机生成路网 | `{width,height,density,seed?}` | 原 `/api/maze/generate` |
| POST | `/api/nav/solve` | BFS/DFS 求解 | `{maze,algorithm:"bfs"\|"dfs"}` | 原 `/api/solve` |

**巡回调度模块（原 ga-tsp）**

| 方法 | 路径 | 说明 | 请求体 | 来源 |
|---|---|---|---|---|
| GET | `/api/tsp/instances` | 内置实例列表 | — | 原 `/api/instances` |
| GET | `/api/tsp/instances/{name}` | 按名取实例（att48） | — | 原 `/api/instances/{name}` |
| POST | `/api/tsp/instance/random` | 随机欧氏实例 | `{n,seed}` | 原 `/api/instance/random` |
| POST | `/api/tsp/solve` | GA 求解（逐代快照） | `{instance,params}` | 原 `/api/solve` |
| POST | `/api/tsp/scan` | 参数扫描对比 | `{instance,params,param,values}` | 原 `/api/scan` |

**自动接单模块（原 knapsack 实验，0/1 背包 = 骑手接单）**

| 方法 | 路径 | 说明 | 请求体 | 来源 |
|---|---|---|---|---|
| GET | `/api/order/instance` | 生成订单池（随机耗时+收益+时间预算） | query:`n,seed` | 原 `/api/knapsack/instance` |
| POST | `/api/order/solve` | GA 求最优接单子集 | `{orders,capacity,params}` | 原 `/api/experiments/solve`(problem=knapsack) |
| POST | `/api/order/scan` | 接单参数扫描对比 | `{orders,capacity,params,param,values}` | 原 `/api/experiments/scan`(knapsack) |

**CEC 算法基准（可选保留）**

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/api/exp/cec/functions` | 基准函数列表 |
| GET | `/api/exp/cec/landscape` | 函数景观网格 |
| POST | `/api/exp/cec/solve` | CEC 求解（problem=cec） |

**静态资源**：`/` → `http.FileServer(http.Dir("web"))`。

### 6.2 请求/响应约定

- 请求体与响应体均为 `application/json; charset=utf-8`。
- 统一错误响应：`{"error":"<msg>"}`，HTTP 4xx。
- `algorithm` 字段取值 `"bfs"` / `"dfs"`（与现有保持一致，前端 UI 文案换成骑手话术，字段名不变）。
- TSP 巡回用 `ga.Params`（Normalize 校验）；接单用 `optimization.Params`（`Validate(problem)` 校验，含 `GreedyRepair` 开关）。
- 端口：默认 `:8080`，`PORT` 覆盖；`ReadTimeout` 30s，`WriteTimeout` 120s（扫描耗时长）。

---

## 7. 前端单页应用结构

### 7.1 布局

- 顶栏：品牌「骑迹 · 外卖骑手智能调度平台」+ 全局状态。
- 侧栏导航：路网编辑器 / 最短送达(BFS) / 可达勘探(DFS) / 巡回调度(GA·TSP) / 自动接单(GA·背包) / 算法对比 / 参数实验。
- 主视图区：按侧栏切换视图，各视图自带控制面板 + Canvas + 结果统计。

### 7.2 视图与复用映射

| 视图 | 复用前端来源 | 改动 |
|---|---|---|
| 路网编辑器 | `maze-bfs-dfs/web` 画布+编辑 | 套骑迹品牌、话术换"路网/街道/取餐点/送达点" |
| 最短送达(BFS) | `maze-bfs-dfs/web` 求解+回放 | 同上话术；统计表标"BFS 最短送达" |
| 可达勘探(DFS) | 同上（DFS 分支） | 新增"全可达区高亮""不可达孤岛标记""BFS 对比列" |
| 巡回调度(GA·TSP) | `ga-tsp/web/index.html` 全套 | 话术换"配送点/巡回路线/送达顺序"；保留 HUD/收敛/多样性/实验 |
| 自动接单(GA·背包) | `ga-tsp/web/experiment.html`(knapsack) | 话术换"订单池/接单/时间预算/收益"；订单网格接/不接双色、收益柱状、贪心修复开关 |
| 算法对比 | 新增 | 聚合各模块统计的横向表 + 文字结论 |
| 参数实验 | `ga-tsp/web` 参数扫描区 | 套骑迹品牌 |

### 7.3 交互

所有视图统一：播放/暂停/单步/重置(首)/跳末/调速滑块/进度条；GA 视图额外：跳收敛点、键盘快捷键（空格/←→/Home/End/C）。复用现有 `app.js` 的回放控制逻辑。

---

## 8. 数据流

**M2/M3 BFS/DFS**：前端画布 → 构造 `maze` 二维数组 → `POST /api/nav/solve{maze,algorithm}` → 后端 `maze.New` + `solver.Solve` → 返回路径+探索快照+统计 → 前端 Canvas 逐帧回放。

**M4 GA**：前端选/随机实例 → `POST /api/tsp/solve{instance,params}` → 后端 `buildInstance` + `ga.Solve` → 返回逐代快照+最终回路+统计 → 前端地图+曲线逐代回放。

**M6 扫描**：前端选参数+取值 → `POST /api/tsp/scan` → 后端循环 `ga.Solve` → 返回各组精简收敛数据 → 前端对比曲线+表。

**M7 自动接单（GA·背包）**：前端选订单数/种子 → `GET /api/order/instance` → 设时间预算+GA参数 → `POST /api/order/solve` → 后端 `optimization.SolveKnapsack` → 返回最优接单子集+逐代快照+DP精确最优对比 → 前端订单网格逐代回放+收益曲线。

**M5 对比**：前端在求解后把三模块统计聚合同框展示（纯前端聚合，无新端点）。

---

## 9. 错误处理

- 请求体 JSON 解析失败、参数越界（迷宫尺寸、城市数<4、边类型非法、订单数、扫描值数量超 [1,8]、接单评估总次数上限等）→ 400 + `{"error"}`。
- GA `Params.Normalize()` 失败 → 400。
- 未知实例名 → 404。
- 前端：fetch 失败/非 2xx → 控制面板状态栏显示错误文案，不破坏当前画布。

---

## 10. 测试策略（TDD）

- **复用现有测试**：`maze-bfs-dfs/internal/api/api_test.go`、`ga-tsp/internal/api/{api_test,experiments_test}.go` 的用例平移到 `qiji/internal/api`，断言改为新路由前缀（`/api/nav/*`、`/api/tsp/*`、`/api/order/*`）。
- **算法包测试随复用迁移**：`maze`/`solver`/`tsp`/`ga`/`optimization` 的既有测试一并搬入，确保行为不变。
- **新增测试**：`api/mux.go` 注册完整性（所有端点 405/404 行为）、统一错误格式、对比聚合逻辑。
- 流程：先迁移测试 → 跑通（应全绿，因逻辑未变）→ 再做前端单页化（前端无单测，靠浏览器手测 + API 测试覆盖后端）。

---

## 11. 部署

- **本地**：`go build -o qiji.exe .` → `Start-Process -WorkingDirectory qiji` → `http://localhost:8080`。
- **阿里云 ECS（121.40.235.227）**：交叉编译 `GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -o qiji` → scp → Dockerfile（`FROM alpine:3.20`, `COPY qiji /app/qiji`, `WORKDIR /app`, `EXPOSE 8080`, `ENTRYPOINT ["/app/qiji"]`）→ `docker run -d -p 8002:8080 --name qiji --restart unless-stopped qiji:latest`。
- **安全组**：ECS 控制台入方向放行 `8002/TCP`（沿用既有 ga-tsp 8001 的放行流程）。
- 容器工作目录须含 `web/`（前端静态资源随二进制一同打包进镜像）。

---

## 12. 里程碑（建议）

1. 建 `qiji/` 骨架 + go.mod + 合并 main.go，迁移 6 个算法包（maze/solver/tsp/ga/optimization），改 import，跑通编译。
2. 写统一 `api/mux.go` + `nav.go` + `tsp.go` + `order.go` + `util.go`，迁移并改写测试至新前缀，后端全绿。
3. 本地起服务，curl 验证全部端点（沿用既有 curl 脚本）。
4. 前端单页化：品牌外壳 + 侧栏 + 各视图接入（含接单视图），复用现有可视化 JS。
5. M5 对比面板（含背包收益维度）+ 话术品牌化收尾。
6. 浏览器全流程手测 → 部署 ECS。

---

## 13. 非目标（YAGNI）

- 不做用户登录/权限/多租户（课程设计单机演示）。
- 不做真实地图/经纬度路网（栅格抽象足够讲清算法）。
- 不做实时订单接入/外部 API 集成。
- 不做 GA 之外的启发式（蚁群、模拟退火等）——聚焦四算法应用（BFS/DFS/GA-TSP/GA-背包）。
- 不引入任何前端框架/构建工具/第三方 Go 依赖（保持零依赖、可离线答辩）。
- 不重构既有算法核心实现，只迁移复用。

---

## 14. 风险

| 风险 | 缓解 |
|---|---|
| 合并两 module 改 import 易遗漏 | 用 IDE 全局替换 + `go build` 兜底；算法包零逻辑改动 |
| 前端单页化工作量被低估 | 后端先全绿再动前端；前端按视图逐个接入，每接一个即可手测 |
| 既有 ga-tsp CEC 基准扩展拖慢整合 | CEC 列为可选（`/api/exp/cec/*`），主流程（含背包接单）通了再挂 |
| ECS 新端口放行 | 提前在控制台放行 8002/TCP |

---

## 15. 开放问题（待确认）

1. **产品名**：暂定「骑迹」，是否采用？或换「速迹/智骑」？
2. **CEC 基准扩展**：背包已包装为"自动接单"保留；CEC 算法基准是否保留（`/api/exp/cec/*`），还是砍掉只留四业务算法？
3. **ECS 端口**：用 8002 还是复用 8001？
4. **对比面板 M5**：是否本期必做，还是作为加分项后续补？
