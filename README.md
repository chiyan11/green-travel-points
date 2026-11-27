# 绿行积分 · 城市绿色出行碳积分系统

<p>
  <a href="https://github.com/chiyan11/green-travel-points/actions/workflows/ci.yml"><img alt="CI" src="https://github.com/chiyan11/green-travel-points/actions/workflows/ci.yml/badge.svg?branch=main"></a>
  <img alt="Go" src="https://img.shields.io/badge/Go-1.24%2B-00ADD8?logo=go&logoColor=white">
  <img alt="React" src="https://img.shields.io/badge/React-19-61DAFB?logo=react&logoColor=black">
  <img alt="TypeScript" src="https://img.shields.io/badge/TypeScript-7.0-3178C6?logo=typescript&logoColor=white">
  <img alt="MySQL" src="https://img.shields.io/badge/MySQL-8.4%2B-4479A1?logo=mysql&logoColor=white">
  <img alt="用途" src="https://img.shields.io/badge/用途-课程实验-6B7280">
</p>

> 把一次「步行 / 骑行 / 地铁 / 公交 / 共享单车」出行，变成可计量、可核查、可兑换的碳资产。
> 后端 Go + chi，前端 React + TypeScript 7 + Vite，MySQL 8 持久化，采用事务性发件箱（Outbox）落地教学级事件驱动架构。

**当前进度：Sprint 0–S2 文档已补齐（需求规格 + 设计规格 + 用户故事地图/Backlog）；后端主链路与商城主体已实现；Outbox Worker 已支持轮询、重试和死信；repo 集成测试与双链路 E2E 已落地；Docker 全栈实跑仍待补。**

已实现：认证 / 行程上报与碳核算 / 积分账本与查询 / 商品目录 / 兑换订单与状态机 / 运营后台（含积分对账）；
待补齐：Outbox 业务处理器（当前默认处理器仅记录投递）、Docker 编排实跑。前端 service 均已联调真实接口；repo 集成测试 14 例（`REPO_TEST_DSN` 门控）、消费者主链路 E2E（含 11 张验收截图）与运营后台 E2E 都已落地。

进度与偏差详见 [Sprint 0 骨架说明](docs/Sprint0-骨架说明.md)，分工详见 [项目分工与迭代计划](docs/项目分工与迭代计划.md)，接口细节见 [P3 接口说明](docs/api/README.md)。

---

## 目录

- [项目简介](#项目简介)
- [功能范围](#功能范围)
- [技术栈](#技术栈)
- [系统架构](#系统架构)
- [目录结构](#目录结构)
- [快速开始](#快速开始)
- [接口](#接口)
- [领域规则](#领域规则)
- [测试](#测试)
- [部署](#部署)
- [开发规范](#开发规范)
- [团队与分工](#团队与分工)
- [项目进度](#项目进度)
- [常见问题](#常见问题)
- [相关文档](#相关文档)
- [许可说明](#许可说明)

---

## 项目简介

《软件工程实验》（2026 版）第 14 题：**碳中和主题的城市绿色出行积分系统**。

### 背景

城市交通碳排放占比持续上升，而市民的步行、骑行、公共交通出行缺乏可量化的记录与正向激励。现有碳普惠平台普遍有三个痛点：**出行数据不可信**（无从校验）、**积分规则不透明**（用户算不出自己的分）、**多方结算难**（商户供了货，积分怎么核销、怎么结账说不清）。

### 目标

系统实现四件事：

1. **出行数据采集（模拟）**：接收并校验市民上报的行程（方式、距离、时间）；
2. **碳减排量计算**：相对"燃油私家车基准"计算少排放的 CO₂ 当量；
3. **积分累积与兑换**：按公开规则换算积分、记账、在积分商城兑换商品；
4. **商城运营管理**：商品 / 库存 / 订单 / 商户 / 对账的运营闭环。

### 核心业务闭环

```mermaid
flowchart LR
  A[市民上报行程] --> B[碳减排核算]
  B --> C[积分核发<br/>账本流水]
  C --> D[积分商城兑换]
  D --> E[兑换订单<br/>状态流转]
  E --> F[核销与对账]
  C -. Outbox 事件 .-> G[统计 / 徽章 / 排行榜]
  E -. Outbox 事件 .-> H[库存扣减 / 商户通知]
```

### 成功标准

| 编号 | 成功标准 | 验收方式 |
| --- | --- | --- |
| G1 | 5 种出行方式的碳减排与积分结果唯一且可复现 | `apps/api/internal/domain` 单测全绿，与需求文档数值表逐行对照 |
| G2 | 账本等价恒等式任何时刻成立 | 对账命令输出"偏差 0" |
| G3 | 50 并发兑换同一 SKU，超卖 = 0 | 并发集成测试报告 |
| G4 | "上报 → 得积分 → 兑换 → 核销"端到端可演示 | E2E 用例 + 演示脚本 |
| G5 | 核心图表与代码 / 数据库命名一致 | 设计说明书自查表（用例名 ↔ 接口 ↔ 表名） |

---

## 功能范围

**本期交付**（模块编号见分工计划 §3 WBS）

| 模块 | 内容 |
| --- | --- |
| M1 用户与认证 | 注册、登录、JWT、角色（市民 / 管理员 / 商户） |
| M2 行程与碳核算 | 行程上报、5 种出行方式、距离校验、碳减排计算 |
| M3 积分账本 | 积分核发、账户余额、流水、日上限、过期 |
| M4 事件总线 | Outbox 表 + Worker + 重试 + 死信 |
| M5 积分查询 | 余额、流水（分页 / 筛选）、碳减排看板 |
| M6 商品与库存 | 商品 / SKU 管理、上下架、库存扣减 |
| M7 兑换与订单 | 兑换下单、幂等、状态机、核销码 |
| M8 运营后台 | 商品 / 订单 / 参数与审计（后台三屏 + 对应 API）；用户 / 流水 / 报表 / 对账页面未做 |
| M9–M10 前端 | 用户端页面 + 运营端表格页 |

**明确不做**（避免范围蔓延）

| 编号 | 非目标 | 原因 |
| --- | --- | --- |
| NG-1 | 真实 GPS / 轨迹采集与地图 SDK | 教学场景无真实数据源，用"起终点 + 距离"模拟 |
| NG-2 | 真实支付与资金结算 | 兑换只用积分，不涉及货币 |
| NG-3 | 真实物流对接 | 商户履约用"发货 + 核销码"模拟 |
| NG-4 | 原生 App / 小程序 | 只做响应式 Web |
| NG-5 | 多城市碳因子区划 | 单一城市口径，因子可配置但不做区划 |
| NG-6 | 真上 Kafka / RabbitMQ | 用 Outbox 表实现教学级 EDA，答辩说明取舍 |

---

## 技术栈

| 层 | 选型 | 版本 | 选型要点 |
| --- | --- | --- | --- |
| 后端 | Go + [chi](https://github.com/go-chi/chi) | go 1.24 / chi v5.2.3 | 标准库 `net/http` 语义 + 轻量路由，无框架魔法，便于讲清分层 |
| 数据访问 | `database/sql`，手写 SQL | mysql 驱动 v1.9.3 | SQL 可见可审；`apps/api/sqlc.yaml` 已就绪但 sqlc 尚未启用 |
| 迁移 | golang-migrate 风格的 `db/migrations` | — | up / down 成对，可回滚 |
| 事件 | 事务性 Outbox 表 + 轮询 Worker | — | 不引入 Kafka / Redis，仍在关系库里保证"业务成功 ⇒ 事件必达"；Worker 按批轮询、退避重试并写死信，业务处理器待各模块接入 |
| 前端 | React + TypeScript 7 + Vite + TanStack Query | 19.3 / 7.0.2 / 8.3 / 5.104 | TS7 原生编译器（Go 实现），构建快；Query 统一服务端状态 |
| 设计系统 | [Astryx](https://astryx.atmeta.com/)（`@astryxdesign/core` + `theme-neutral`，品牌子主题在 `apps/web/src/themes/green-travel/`） | 0.7.17 | `main.tsx` 按层导入 4 份样式表并挂 `Theme` + `LinkProvider`；控件用 Astryx，业务样式仍是一个 `styles.css`（自带设计 token：品牌绿 / 积分琥珀 / 字号阶梯 / 12px 卡片圆角 / 单档阴影），无 StyleX/Tailwind 编译器 |
| 图形 | 手写线性图标 + CSS 柱状图 | — | `src/components/icons.tsx` 是 24×24 单色 SVG 集合，不引图标库；近 7 日碳减排用 CSS 柱状（`CarbonTrendChart`），不引图表库 |
| 接口契约 | OpenAPI 3.1 + swagger-typescript-api（devDependency） | 13.15 | `docs/api/openapi.yaml` 是唯一契约，`npm run gen:api` 生成 `apps/web/src/types/api.ts`，门禁校验生成物与 yaml 一致 |
| 测试 | Go `testing` / Vitest + Testing Library + MSW / Playwright / k6 | 1.65 | 单测、页面测试、repo 集成（`REPO_TEST_DSN` 门控）、双链路 E2E 已落地；性能压测待建 |
| 数据库 | MySQL | 8.4+（本机 9.7） | `utf8mb4_0900_ai_ci`、`DATETIME(3)`、JSON 存事件负载 |
| 部署 | Docker Compose + Nginx + GitHub Actions | — | 一条命令起全栈，CI 双 job 门禁 |

---

## 系统架构

前后端分离 + 分层结构。后端依赖方向：`cmd`（装配根）→ `httpserver`（HTTP 层）→ `app`（DTO 与端口接口）← `repo`（SQL 实现），`domain` 在最内层、不 import 任何内部包；业务规则（碳核算、积分、状态机）做成无 I/O 的纯函数，便于单测与复用。

业务事件采用**事务性发件箱（Transactional Outbox）**：

```
上报行程（单事务）
  INSERT trip_record + INSERT carbon_record + INSERT point_txn
  + 更新 point_account + INSERT event_outbox(type=TripVerified, payload=...)
后台 Worker（轮询 outbox）——按批锁定待投递事件，成功后写 `dispatched_at`，失败按指数退避重试，达到上限写 `dead_letter_at`
  TripVerified → 更新碳减排统计 / 触发成就徽章 / 写排行榜
  OrderPaid    → 扣减库存 / 通知商户 / 生成核销码
```

> **实现现状**：生产端已落地——行程上报、下单 / 取消、运营流转都在各自事务里 `INSERT event_outbox`；消费端已具备轮询、行锁、指数退避和死信处理，默认处理器只记录投递。统计、徽章、通知等业务副作用尚未定义处理器，须由对应模块接入。事件主题已统一使用 `eventbus` 常量。

选择 EDA 而非“经典分层同步调用”的理由、与其他架构方案的对比，写在 [设计规格说明书](docs/设计规格说明书.md) 的「2.1.1 架构决策与选型」（含候选方案对比、约束支持度与决策记录）；逻辑架构（包图）、物理架构（部署图）见 [docs/uml](docs/uml)。

---

## 目录结构

```
green-travel-points/
├── apps/
│   ├── api/                          # Go 后端（module greentravelpoints）
│   │   ├── cmd/server/main.go        # 入口：配置 → 连接池 → 路由装配 → 优雅关闭
│   │   ├── cmd/server/dbendpoint.go  # 启动日志播报数据库目标（本机 / 共享库，不打印账号密码）
│   │   └── internal/
│   │       ├── app/                  # DTO、哨兵错误、端口接口（Catalog / Points / Orders / Role）
│   │       │   └── admin_test.go     # 运营后台接口用例（admin 列表返回 SKU 等，注入假存储不连 MySQL）
│   │       ├── config/config.go      # 环境变量 → Config，含默认值与取值范围校验
│   │       ├── domain/               # 纯业务规则：碳核算、积分换算、订单状态机（45 个用例）
│   │       ├── eventbus/             # 领域事件主题、处理器契约与 Worker
│   │       ├── httpserver/           # 路由、中间件、统一响应壳、各业务 handler（49 个测试函数）
│   │       └── repo/                 # 手写 SQL：连接池、行程、商品、积分、订单、运营后台；orders_test.go / audit_test.go / batches_test.go / outbox_integration_test.go 为 REPO_TEST_DSN 门控的集成测试（13 例）
│   └── web/                          # React 前端：用户端 9 页 + 运营后台 3 页（/admin）
│       ├── AGENTS.md                 # 前端专项代理指南（Astryx 工作流与规则，由 astryx init 维护）
│       ├── src/{pages,components,services,lib,themes,types}/  # themes/ = Astryx 品牌主题；pages/admin = M8 后台三屏；types/api.ts = 生成契约
│       └── e2e/ + playwright.config.ts                        # 消费者主链路（*.spec.ts）与运营后台（*.e2e.ts）用例
├── db/
│   ├── migrations/                   # 8 组 up/down：参数与出行方式 / 发件箱 / 账本 / 用户行程碳记录 / 商城订单 / 批次流水 / 审计日志 / 规则快照改 JSON（数据迁移）
│   └── seed/dev_demo.sql             # 演示数据：用户 1001（1200 分）+ 3 个上架商品
├── deploy/                           # Dockerfile.api / Dockerfile.web / nginx.conf / docker-compose.yml
├── docs/
│   ├── api/README.md                 # M5–M8 接口契约：字段、事务步骤、错误码、联调步骤
│   ├── uml/                          # 24 个 .puml 源图 + out/ 24 张 PNG + .tools/(plantuml.jar, graphviz)
│   ├── 需求规格说明书.md              # 实验一：角色、用例、域模型、规则、非功能需求
│   ├── 设计规格说明书.md              # 实验二：架构对比/决策、包图/部署图、表2-1/2-2/2-3、偏差
│   ├── 用户故事地图与Backlog.md       # 故事地图、40 条 Backlog、7 个 Sprint 切片
│   ├── 实验报告-绿行积分.md / .docx    # 五章完整报告（可单独提交）
│   └── 其他 .md                       # 分工计划、Sprint 0 记录、联通记录、建模指南、接口需求清单
├── scripts/                          # dev-env.ps1（缓存重定向）/ verify.ps1（一键门禁）/ uml-export.ps1
├── .github/workflows/ci.yml          # 双 job：Go（gofmt→vet→test -race→build）+ Node（tsc→vitest→build）
├── AGENTS.md                         # 代理 / 新人作业手册：分层规范、领域规则、迁移与测试要求、门禁
└── README.md                         # 项目总览（本文件）
```

---

## 快速开始

### 环境要求

| 依赖 | 版本 | 说明 |
| --- | --- | --- |
| Go | 1.24+ | 本机实测 1.27.2 |
| Node.js | 24+ | 本机实测 24.16.0，前端统一用 **npm** |
| MySQL | 8.4+ | 本机实测 9.7.2，或使用 Docker Desktop |
| golang-migrate | 可选 | 已提供 mysql 客户端逐条执行的替代方案 |

Windows 下若默认缓存目录不可写（例如受限沙箱），先点源环境脚本，把 Go / npm 缓存重定向到仓库内 `.cache/`：

```powershell
. .\scripts\dev-env.ps1
```

### 1. 克隆仓库

```powershell
git clone https://github.com/chiyan11/green-travel-points.git
cd green-travel-points
```

### 2. 准备数据库

```sql
-- 用 root 登录一次，建库建账号
CREATE DATABASE IF NOT EXISTS green_travel_points
  DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
CREATE USER IF NOT EXISTS 'greentravel'@'%' IDENTIFIED BY 'greentravel';
GRANT ALL PRIVILEGES ON green_travel_points.* TO 'greentravel'@'%';
FLUSH PRIVILEGES;
```

执行迁移（任选一种）：

```powershell
# 方式 A：golang-migrate CLI
migrate -path db/migrations `
  -database "mysql://greentravel:greentravel@tcp(127.0.0.1:3306)/green_travel_points" up

# 方式 B：mysql 客户端逐个执行（注意文件名顺序）
Get-ChildItem db/migrations/*.up.sql | Sort-Object Name | ForEach-Object {
  Get-Content $_ -Raw | mysql -h 127.0.0.1 -u greentravel -pgreentravel green_travel_points
}

# 方式 C：Docker Compose（连带起 API；首次启动自动导入迁移）
docker compose -f deploy/docker-compose.yml up -d --build
```

> 本机已按上述步骤实跑过一次（建库 → 两条迁移 → `/readyz` 200），完整命令与输出见 [数据库联通记录](docs/数据库联通记录.md)。

### 3. 启动后端

```powershell
cd apps/api
go run ./cmd/server              # 默认监听 :8080
```

后端**只读环境变量**、不解析 `.env` 文件；默认值已指向本机 `127.0.0.1:3306` 的 `greentravel` 账号，所以零配置即可启动。要换库或改端口就设环境变量（`DB_DSN` / `HTTP_ADDR` / `AUTH_SECRET` …，完整清单见 `apps/api/.env.example`），例如在 PowerShell 里：

```powershell
$env:DB_DSN = "root:root@tcp(127.0.0.1:3306)/green_travel_points?parseTime=true&loc=Local&charset=utf8mb4"
```

无数据库时服务仍可启动（启动日志只 Warn）：`/healthz` 返回 200，`/readyz` 返回 503，便于前端先联调。

> 演示账号可直接登录：`db/seed/dev_demo.sql` 自带 `pbkdf2$600000$salt$digest` 口令哈希——市民 `13800000001` / `demo123456`（user 1001，1200 分），运营管理员 `13800000002` / `admin123456`（user 1002，`/admin` 用它登录）。也可注册新账号（`POST /api/v1/auth/register` 或登录页「注册」）；开发期身份头 `X-User-Id: 1001` 只在显式设置 `AUTH_ALLOW_DEV_HEADER=true` 时可用，`APP_ENV=prod` 开启会拒绝启动。共享库（`10.213.180.114`）仍是旧占位哈希且「演示用户」id=1，**禁止重跑 seed**，按 `db/seed/dev_demo.sql` 头部说明做定向 UPDATE。

### 4. 启动前端

```powershell
cd apps/web
npm install
npm run dev        # http://localhost:5173，/api 已代理到 127.0.0.1:8080
```

### 5. 一键校验

```powershell
pwsh -File .\scripts\verify.ps1
```

依次执行 `gofmt -l` → `go vet` → `go test` → `go build` → `tsc --noEmit` → `vitest run` → `vite build`，任一失败即非 0 退出；CI 跑的是同一套门禁。

---

## 接口

统一响应壳：`{"code":"OK","message":"ok","data":{...}}`，错误时省略 `data`；错误码与 HTTP 状态映射见 `apps/api/internal/httpserver/response.go`。
字段口径、分页约定、事务步骤与业务错误码见 **[P3 接口说明](docs/api/README.md)**。

**已实现**（均有 httpserver 单测覆盖，测试用假存储、不连 MySQL）：

| 分组 | 端点 | 说明 |
| --- | --- | --- |
| 探针 | `GET /healthz`、`GET /readyz` | 存活恒 200；就绪含 DB ping，失败返回 503 |
| 认证 | `POST /api/v1/auth/register`、`POST /api/v1/auth/login`、`GET /api/v1/me` | 自签 HMAC-SHA256 token，有效期 2 小时 |
| 行程 | `POST /api/v1/trips`、`GET /api/v1/trips` | 单事务：行程 + 碳记录 + 积分流水 + 余额 + Outbox |
| 商品 | `GET /api/v1/products`、`GET /api/v1/products/{id}` | 公开接口，支持分类 / 关键词 / 仅看有货 |
| 积分 | `GET /api/v1/points/summary`、`GET /api/v1/points/transactions`、`GET /api/v1/points/daily-carbon` | 需身份；流水支持类型多选与分页；近 7 日碳减排按天聚合 |
| 兑换 | `POST /api/v1/orders`、`GET /api/v1/orders`、`POST /api/v1/orders/{id}/cancel` | 支持 `Idempotency-Key`；按 FIFO 消耗积分批次 |
| 运营 | `/api/v1/admin/products`、`/api/v1/admin/skus/{id}/stock`、`/api/v1/admin/orders`（含 confirm / ship / complete）、`POST /api/v1/admin/points/adjust`、`/api/v1/admin/points/reconcile`、`/api/v1/admin/audit-logs`、`/api/v1/admin/points/expire`、`/api/v1/admin/config`、`/api/v1/admin/travel-modes/{code}` | 需 `ADMIN` 角色，非法流转返回 409 |

**身份鉴权**：只认 `Authorization: Bearer <token>`（登录颁发，2 小时过期），未带 token 即 401；开发期要在脚本里指定用户，必须显式设 `AUTH_ALLOW_DEV_HEADER=true` 才接受 `X-User-Id: <正整数>`，`APP_ENV=prod` 开启会拒绝启动。运营后台再叠加 `RequireAdmin` 角色校验。

**路由可裁剪**：`NewRouter` 的 `Catalog` / `Points` / `Orders` / `Admin` / `Roles` 端口为 nil 时不挂载对应分组，单测可以只测目标模块。

---

## 领域规则

已冻结，单测覆盖（`apps/api/internal/domain`）：

```
碳减排量 carbon_saved(g) = max(0, 距离(km) × (EF_base − EF_actual))
积分 points              = floor(carbon_saved / carbon_per_point)
当日发放                 = min(points, daily_points_cap − 当日已发放)
```

| 出行方式 | EF_actual (gCO₂e/km) | 说明 |
| --- | --- | --- |
| 步行 walk | 0 | |
| 骑行 bike | 0 | |
| 地铁 subway | 41 | |
| 公交 bus | 68 | |
| 共享单车 shared_bike | 0 | |
| **基准 EF_base** | **192** | 燃油私家车人均排放口径 |

可调参数存 `sys_config`、排放因子存 `travel_mode`，**业务代码不硬编码**；默认值与 `domain.DefaultPointsRule()` 一致（100 g = 1 分、单日上限 500 分、单次距离 0.1–100 km）。

**5 条不变量**（写进测试断言）

1. 账本等价：`point_account.balance == Σ(point_txn.amount)`；
2. 幂等：同一 `user_id + trip_hash` 重复上报只产生 1 条记录、1 笔流水；
3. 不超兑：`balance - frozen >= 0`，扣减走条件更新 / 行锁，禁止先查后写；
4. 状态机：订单 `PENDING → PAID → SHIPPED → COMPLETED`，可 `CANCELLED`，非法跃迁必须拒绝；
5. 可追溯：任何一笔积分变动都能反查到来源（行程 / 兑换 / 人工调整 / 过期）。

---

## 测试

| 层级 | 范围 | 命令 | 现状 |
| --- | --- | --- | --- |
| 单元测试（Go · domain） | 碳核算、积分换算、日上限、订单状态机 | `go test ./internal/domain -v` | 12 个测试函数 / 45 个用例（含表驱动子用例），全绿 |
| 单元测试（Go · http） | 路由、中间件、认证、各 handler（假存储，不连库） | `go test ./internal/httpserver -v` | 49 个测试函数 / 49 例，全绿 |
| 单元测试（Go · app） | `sys_config` / `travel_mode` 参数校验（UC-14） | `go test ./internal/app -v` | 2 个函数 / 34 例（表驱动），全绿 |
| 单元测试（Go · cmd/server） | 启动日志的 DSN 目标判定：本机 / 局域网 / ZeroTier / socket / 非法 DSN；写超时与请求超时的关系 | `go test ./cmd/server -v` | 2 个函数 / 15 个用例 |
| 单元测试（前端） | Envelope 解包、错误映射、表单校验 | `npx vitest run` | 2 个文件 19 例（http 5 + validators 14） |
| 页面测试（前端） | 登录失败态、行程上报与重复提交、商城空态/库存不足、订单取消 | `npx vitest run src/pages` | 5 个文件 16 例，jsdom + Testing Library + MSW |
| E2E（前端） | 注册 → 上报两笔行程 → 积分到账 → 商城兑换 → 取消返还；登录失败提示 | `npm run test:e2e` | Playwright + 真后端 + 共享库，2 例全绿，11 张截图存 `docs/screenshots/` |
| 类型检查 | 前端全量 | `npx tsc --noEmit` | 零错误 |
| 契约一致性 | `docs/api/openapi.yaml` ↔ `apps/web/src/types/api.ts` | `npm run gen:api` 后 `git diff --exit-code` | 已接入 `verify.ps1` 与 CI |
| E2E（消费者主链路） | 注册 → 上报 → 积分到账 → 兑换 → 取消返还 | `E2E_API_TARGET=… npm run test:e2e` | 1 条用例绿，11 张截图在 `docs/screenshots/` |
| E2E（运营后台） | 建商品 → 上架 → 改库存 → 下单 → 后台确认接单 | `npx playwright test`（apps/web） | 1 条用例绿；截图证据在 `docs/p5-evidence/` |
| 集成测试 | repo + 真实 MySQL：兑换成功 / 积分不足 / 批次不足回滚 / 库存不足 / 限购 / 取消回填批次 / 20 并发防超卖 / 对账 / 人工调账与审计写入 / 积分批次发放 / 行程幂等与账本对账 / Outbox 重试与死信 | `REPO_TEST_DSN=… go test ./internal/repo -v` | `apps/api/internal/repo/` 下 `orders_test.go`、`audit_test.go`、`batches_test.go`、`trips_test.go`、`outbox_integration_test.go` 共 14 例；未设 DSN 时数据库用例 SKIP；尚未进 CI |
| 性能 | 行程上报、兑换并发 | 待建（k6） | 未建：计划 S6 起 |

指导书要求的**基本路径法**（组件测试）与**等价类划分法**（功能测试）用例表，在 S6 补齐并写入测试报告。

---

## 部署

```powershell
docker compose -f deploy/docker-compose.yml up -d --build
```

当前 compose 只编排 `mysql` + `api` 两个服务：MySQL 首次启动按文件名顺序导入 8 组迁移，API 等 `mysql` 健康检查通过后启动。

`web` 与反向代理 nginx 的镜像配置已就绪（`deploy/Dockerfile.web` 多阶段构建 → nginx 托管 `dist/`，`deploy/nginx.conf` 含 SPA 回退与 `/api` 反代），但尚未编入 compose。物理部署拓扑见 [部署图](docs/uml/06-部署图.puml)。

> 本机 Docker daemon 未启动，上述编排**尚未实跑验证**，属"配置就绪"状态。

---

## 开发规范

- **分支**：`main` 受保护，只接受 PR；功能分支命名 `feat/M2-trip-report`、`fix/M7-oversell`。
- **提交**：Conventional Commits，一条提交只做一件事，例：`feat(api): 实现行程上报接口 (#12)`。
- **PR**：1 人 Review + CI 绿才可合并；描述里写"对应报告章节 / 图号"，便于后期统稿。
- **节奏**：周一 Planning → 周三站会 → 周五 Review + 打 Tag `sprint-N`；每个 Sprint 末必须有可运行增量。
- **目录归属**：每人只改自己模块的 `internal/<模块>` 与页面目录，跨模块改动先提 Issue 对齐。
- **文档**：每个 Sprint 末同步产出对应报告章节草稿，文档即交付物，不留到最后一周。

---

## 团队与分工

| 编号 | 角色 | 一句话职责 | Sprint 1 起手位置 |
| --- | --- | --- | --- |
| P1 | 组长 / 架构与建模 ______（待填写） | 业务规则定稿、全套 UML、文档统稿、进度与互评 | 出用例图、域模型、包图；维护契约与自查表 |
| P2 | 后端 / 核心域与事件引擎 ______（待填写） | 工程骨架、数据模型、认证、行程上报链路 | `internal/httpserver/router.go` 预留的 `/api/v1/trips`，复用 `domain.EvaluateTrip` |
| P3 | 后端 / 积分商城与运营 ______（待填写） | 积分账户与流水、商品、订单、兑换 | 续写账本与商品表迁移；实现 `PointsGranted` 消费者 |
| P4 | 前端 / 用户端 ______（待填写） | 前端工程化与用户端页面 | `apps/web/src/routes.tsx` 的占位页逐个替换；`src/lib/http.ts` 已封装 Envelope |
| P5 | 测试与部署 / 运营后台 ______（待填写） | 三层测试、CI / 容器化、测试报告 | `scripts/verify.ps1` + `deploy/`；补基本路径法 / 等价类用例表 |

完整职责、WBS 与逐周任务矩阵见 [项目分工与迭代计划](docs/项目分工与迭代计划.md)。

---

## 项目进度

| Sprint | 周期 | 目标 | 对齐实验 | 状态 |
| --- | --- | --- | --- | --- |
| S0 | 第 1 周 | 技术栈验证、业务规则冻结、骨架搭建 | 准备 | ✅ 已完成 |
| S1 | 第 2–4 周 | 需求建模：角色 / 用例 / 活动图 / 域模型 / 非功能需求 | 实验一 | ✅ 已完成（12 张 UML 源图 + 成图） |
| S2 | 第 5–7 周 | 设计建模 + 垂直切片（登录 → 上报 → 碳核算 → 积分） | 实验二 | ✅ 主链路已通（认证 + 行程 + 积分） |
| S3 | 第 8–10 周 | 商城闭环 + 订单状态机 + Outbox Worker | 实验三 | 🚧 商城闭环、状态机与 Worker 已实现；业务处理器待接入 |
| S4 | 第 11–13 周 | 三层测试 + 性能 + 容器化部署 + 报告四 / 五章 | 实验四 | ⏳ 未开始 |
| S1 | 第 2–3 周 | 需求建模：角色 / 用例 / 活动图 / 域模型 / 非功能需求 + 故事地图 | 实验一 | ✅ 已完成（19 个用例、12 张分析图、40 条 Backlog） |
| S2 | 第 4–5 周 | 设计建模：三方案对比、包图/部署图、表2-1/2-2/2-3 | 实验二 | ✅ 设计规格与 51 张 UML 已补；19/19 用例均有鲁棒图与顺序图（新增 27 张，图2-18～2-44） |
| S3 | 第 6–7 周 | 主链路：认证 → 行程 → 碳核算 → 积分发放 | 实验二/三 | ✅ 主链路已通（含幂等与日上限）；JWT 为临时实现 |
| S4 | 第 8–9 周 | 商城闭环：商品 → 兑换 → 订单状态机 → 取消退分 | 实验三 | 🚧 兑换/订单/取消主体已实现；批次发放已接入 `/trips`；并发/取消已有 repo 集成用例 |
| S5 | 第 10–11 周 | 运营后台 + Outbox Worker + 对账/审计 | 实验三 | 🚧 运营 API 与后台三屏已实现（含参数配置 / 过期清算 / 调账与审计查询）；Worker 业务处理器待接入 |
| S6 | 第 12–13 周 | 三层测试、部署、报告四/五章、答辩 | 实验四 | ⏳ 未开始 |

**Sprint 0 验收**

- [x] 前后端可独立启动，`/healthz` `/readyz` `/api/v1/ping` 行为符合预期
- [x] 领域规则单元测试全绿（12 个测试函数 / 45 个用例：碳核算、积分换算、日上限、订单状态机）
- [x] 前端 TS7 类型检查通过、单测 19 例通过、生产构建成功
- [x] 五组迁移可执行（`000001`–`000005`；Sprint 0 当时口径，现为 8 组 / 15 张表，见 `db/migrations/` 与 AGENTS §6），种子数据 `db/seed/dev_demo.sql` 可导入
- [x] CI 配置就绪（Go + Node 双 job）
- [x] 数据库连通性与 `migrate up` 实跑：2026-10-20 本机 MySQL 9.7.2，`/readyz` 返回 `{"database":"up"}`

---

## 常见问题

**1. `tsc` 报 `TS5102: Option 'baseUrl' has been removed`**
TypeScript 7 移除了 `baseUrl`。路径别名改为相对形式：`"@/*": ["./src/*"]`。

**2. `tsc` 报无法解析 `@typescript/typescript-win32-x64`**
TS7 把编译器编译成各平台二进制，通过 `optionalDependencies` 分发，包管理器必须正确安装平台包。npm 正常，pnpm 在 Windows 下会失败。

**3. 前端为什么用 npm 而不是 pnpm**
pnpm 11 的 isolated 链接器用 Junction 指向虚拟 store，Windows + Node 24 下解析 `@typescript/typescript-win32-x64` 与 `std-env` 均失败；改用 npm 并锁定 `package-lock.json`。

**4. `vite build` / `vitest` 报 `spawn EPERM`**
受限沙箱（某些编辑器内置终端、带沙箱的 Agent 工具）会拦截子进程，`vite` / `vitest` 需要 spawn 子进程来做配置打包与 worker 池，因此失败。在普通终端执行即正常；CI 使用标准 runner 不受影响。

**5. `docker compose` 报无法连接 daemon**
本机 Docker Desktop 未启动。启动后再执行部署命令，或改用方式 B 手动跑迁移、本地 `go run` + `npm run dev`。

**6. Windows 下读写仓库子目录报 `EPERM`（不是代码问题）**
仓库若是从别处复制 / 解压来的，既有目录可能带着一份不含当前用户的 ACL：表现为根目录能写、`apps` `docs` `db` 等子目录全被拒。用当前账户补一次完全控制即可（只增不删，可用 `/remove:g` 撤销）：

```powershell
icacls "D:\coding\green-travel-points" /grant "$env:USERNAME:(OI)(CI)F" /T /C
```

**7. 前端页面报 401 / 列表为空**
前端已全部走真实接口，先确认后端在跑并导入种子：`db/seed/dev_demo.sql`（用户 1001 / 管理员 1002）。前端只带 `Authorization: Bearer <token>`，不再注入 `X-User-Id`；未登录的匿名请求会 401，由 `http.ts` 统一清会话并跳 `/login`。

---

## 相关文档

| 文档 | 内容 |
| --- | --- |
| [参考-2026版软件工程实验指导书](docs/参考-2026版软件工程实验指导书.txt) | 课程要求与报告样本（原文） |
| [项目分工与迭代计划](docs/项目分工与迭代计划.md) | 角色、WBS、7 个 Sprint（13 周）、协作规范、风险预案 |
| [需求规格说明书](docs/需求规格说明书.md) | 角色、用例、域模型、非功能需求（实验一） |
| [设计规格说明书](docs/设计规格说明书.md) | 架构方案对比与决策、包图/部署图、表2-1/2-2/2-3、实现偏差（实验二） |
| [用户故事地图与Backlog](docs/用户故事地图与Backlog.md) | 故事地图、40 条用户故事、Sprint 切片与验收标准 |
| [实验报告（Markdown）](docs/实验报告-绿行积分.md) / [DOCX](docs/实验报告-绿行积分.docx) | 五章完整报告：需求、设计、原型、测试、总结与互评 |
| [Sprint 0 骨架说明](docs/Sprint0-骨架说明.md) | 实施记录、实测数据、踩坑与验收证据 |
| [数据库联通记录](docs/数据库联通记录.md) | 建库、迁移执行、连通性验证全过程 |
| [PlantUML 建模指南](docs/PlantUML建模指南.md) | 出图流程与 `scripts/uml-export.ps1` 用法 |
| [AGENTS.md](AGENTS.md) | 代理 / 新人作业手册：分层与依赖方向、领域规则、迁移与测试要求、门禁与排错 |
| [P3 接口说明](docs/api/README.md) | M5–M8 端点、字段口径、事务步骤、业务错误码、联调步骤 |
| [OpenAPI 契约](docs/api/openapi.yaml) | 全部端点的机器可读契约（3.1），前端类型由它生成 |
| [E2E 验收截图](docs/screenshots) | 主链路 11 张截图：登录 / 注册 / 首页 / 行程 / 积分 / 商城 / 商品详情 / 订单 / 取消 / 登录失败 |
| [前端接口需求清单](docs/前端接口需求清单.md) | 页面 ↔ 端点对照、字段与错误码需求（P4 → P2/P3） |
| [UML 源图与导出](docs/uml) | 24 个 `.puml`：4 张用例图（总览 + 出行/积分/运营三个分区）、4 张活动图、域模型、包图、部署图、2 张鲁棒图（市民 / 运营管理员各一张）、9 张序列图、2 张状态图；`out/` 下为 24 张 PNG |

---

## 许可说明

本项目为《软件工程实验》课程作业，仅供课程教学、答辩与团队内部学习使用，未授予公开分发或商业使用许可。文档与 UML 模型版权归小组成员共同所有；第三方依赖遵循其各自许可。
