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

**当前状态：Sprint 0（工程骨架）已完成并通过一键校验**，Sprint 1 起按模块交付业务接口。
进度与偏差详见 [Sprint 0 骨架说明](docs/Sprint0-骨架说明.md)，分工详见 [项目分工与迭代计划](docs/项目分工与迭代计划.md)。

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
| M8 运营后台 | 商品 / 订单 / 用户 / 流水 / 报表 / 对账 |
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
| 数据访问 | `database/sql` + sqlc（待启用） | mysql 驱动 v1.9.3 | SQL 可见可审，生成强类型代码，直接支撑"数据库字段级清单" |
| 迁移 | golang-migrate 风格的 `db/migrations` | — | up / down 成对，可回滚 |
| 事件 | 事务性 Outbox 表 + 轮询 Worker | — | 不引入 Kafka / Redis，仍在关系库里保证"业务成功 ⇒ 事件必达" |
| 前端 | React + TypeScript 7 + Vite + TanStack Query | 19.3 / 7.0.2 / 8.3 / 5.104 | TS7 原生编译器（Go 实现），构建快；Query 统一服务端状态 |
| 测试 | Go `testing` / Vitest / Playwright | — | 三层测试，对齐实验"基本路径法 + 等价类划分" |
| 数据库 | MySQL | 8.4+（本机 9.7） | `utf8mb4_0900_ai_ci`、`DATETIME(3)`、JSON 存事件负载 |
| 部署 | Docker Compose + Nginx + GitHub Actions | — | 一条命令起全栈，CI 双 job 门禁 |

---

## 系统架构

前后端分离 + 分层结构，后端内部按 `config → httpserver → usecase → repo → domain` 单向依赖，业务规则（碳核算、积分）做成无 I/O 的纯函数，便于单测与复用。

业务事件采用**事务性发件箱（Transactional Outbox）**：

```
上报行程（单事务）
  INSERT trip_record + INSERT carbon_record + INSERT point_txn
  + 更新 point_account + INSERT event_outbox(type=TripVerified, payload=...)
后台 Worker（轮询 outbox）
  TripVerified → 更新碳减排统计 / 触发成就徽章 / 写排行榜
  OrderPaid    → 扣减库存 / 通知商户 / 生成核销码
```

选择 EDA 而非"经典分层同步调用"的理由、与其他架构方案的对比，写在设计说明书的「2.1.1 架构决策与选型」；逻辑架构（包图）、物理架构（部署图）见 [docs/uml](docs/uml)。

---

## 目录结构

```
green-travel-points/
├── apps/
│   ├── api/                     # Go 后端
│   │   ├── cmd/server/          # 入口：配置 → 日志 → 路由 → 优雅关闭
│   │   └── internal/
│   │       ├── config/          # 环境变量装配
│   │       ├── domain/          # 纯业务规则（碳核算、积分）+ 单测
│   │       ├── eventbus/        # 领域事件主题与处理器契约
│   │       ├── httpserver/      # 路由、中间件、统一响应壳、健康检查
│   │       └── repo/            # 连接池；sqlc 生成代码将落 sqlcgen/
│   ├── web/                     # React 用户端（路由骨架 + 连通性自检）
│   └── admin/                   # 运营后台（Sprint 3 起）
├── packages/api-types/          # OpenAPI 生成的前端类型（Sprint 2 起）
├── db/migrations/               # 000001 参数与出行方式 / 000002 事务性发件箱
├── deploy/                      # Dockerfile.api / Dockerfile.web / nginx.conf / compose
├── docs/
│   ├── uml/                     # PlantUML 源图 + out/ 导出 PNG
│   └── *.md                     # 需求、分工、Sprint 记录、建模指南
├── scripts/                     # dev-env.ps1 / verify.ps1 / uml-export.ps1
└── .github/workflows/ci.yml     # Go + Node 双 job 门禁
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
Copy-Item .env.example .env      # 按需修改 DB_DSN
go run ./cmd/server              # 默认监听 :8080
```

无数据库时服务仍可启动（启动日志只 Warn）：`/healthz` 返回 200，`/readyz` 返回 503，便于前端先联调。

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

当前已实现（Sprint 0）：

| 接口 | 用途 |
| --- | --- |
| `GET /healthz` | 存活探针（恒 200） |
| `GET /readyz` | 就绪探针（DB ping 失败返回 503） |
| `GET /api/v1/ping` | 前后端连通性自检 |

所有响应统一壳：`{"code":"OK","message":"ok","data":{...}}`，错误码定义见 `apps/api/internal/httpserver/response.go`。

规划中（Sprint 1–3，契约先行）：

| 分组 | 端点 | 负责 |
| --- | --- | --- |
| 认证 | `POST /api/v1/auth/register`、`POST /api/v1/auth/login` | P2 |
| 行程 | `POST /api/v1/trips`、`GET /api/v1/trips` | P2 |
| 积分 | `GET /api/v1/points/summary`、`GET /api/v1/points/transactions` | P3 |
| 商城 | `GET /api/v1/products`、`GET /api/v1/products/{id}` | P3 |
| 兑换 | `POST /api/v1/orders`、`GET /api/v1/orders`、`POST /api/v1/orders/{id}/cancel` | P3 |
| 运营 | 商品 / 库存 / 订单发货 / 对账 | P5 + P3 |

> 接口契约文件 `docs/api/openapi.yaml` 随 Sprint 1 产出；前端类型由它生成，不手写 DTO。

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
| 单元测试（Go） | 碳核算、积分规则、HTTP 层 | `go test ./... -count=1` | 领域规则 35 例 + HTTP 4 例，全绿 |
| 单元测试（前端） | Envelope 解包、错误映射 | `npx vitest run` | 5 例通过 |
| 类型检查 | 前端全量 | `npx tsc --noEmit` | 零错误 |
| 集成测试 | API + MySQL | 待建（testcontainers） | Sprint 2 起 |
| E2E | 申报 → 得积分 → 兑换 | 待建（Playwright） | Sprint 3 起 |
| 性能 | 行程上报、兑换并发 | 待建（k6） | Sprint 4 起 |

指导书要求的**基本路径法**（组件测试）与**等价类划分法**（功能测试）用例表，在 Sprint 3–4 补齐并写入测试报告。

---

## 部署

```powershell
docker compose -f deploy/docker-compose.yml up -d --build
```

包含 `mysql` + `api` + `web` + `nginx` 四个服务，首次启动自动执行迁移；物理部署拓扑见 [部署图](docs/uml/06-部署图.puml)。

> 本机 Docker daemon 尚未启动，上述编排**尚未实跑验证**，属"配置就绪"状态，由 P5 在第一次集成分支上补执行证据。

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
| P1 | 组长 / 架构与建模 ______ | 业务规则定稿、全套 UML、文档统稿、进度与互评 | 出用例图、域模型、包图；维护契约与自查表 |
| P2 | 后端 / 核心域与事件引擎 ______ | 工程骨架、数据模型、认证、行程上报链路 | `internal/httpserver/router.go` 预留的 `/api/v1/trips`，复用 `domain.EvaluateTrip` |
| P3 | 后端 / 积分商城与运营 ______ | 积分账户与流水、商品、订单、兑换 | 续写账本与商品表迁移；实现 `PointsGranted` 消费者 |
| P4 | 前端 / 用户端 ______ | 前端工程化与用户端页面 | `apps/web/src/routes.tsx` 的占位页逐个替换；`src/lib/http.ts` 已封装 Envelope |
| P5 | 测试与部署 / 运营后台 ______ | 三层测试、CI / 容器化、测试报告 | `scripts/verify.ps1` + `deploy/`；补基本路径法 / 等价类用例表 |

完整职责、WBS 与逐周任务矩阵见 [项目分工与迭代计划](docs/项目分工与迭代计划.md)。

---

## 项目进度

| Sprint | 周期 | 目标 | 对齐实验 | 状态 |
| --- | --- | --- | --- | --- |
| S0 | 第 1 周 | 技术栈验证、业务规则冻结、骨架搭建 | 准备 | ✅ 已完成 |
| S1 | 第 2–4 周 | 需求建模：角色 / 用例 / 活动图 / 域模型 / 非功能需求 | 实验一 | 🚧 进行中 |
| S2 | 第 5–7 周 | 设计建模 + 垂直切片（登录 → 上报 → 碳核算 → 积分） | 实验二 | ⏳ 未开始 |
| S3 | 第 8–10 周 | 商城闭环 + 订单状态机 + Outbox Worker | 实验三 | ⏳ 未开始 |
| S4 | 第 11–13 周 | 三层测试 + 性能 + 容器化部署 + 报告四 / 五章 | 实验四 | ⏳ 未开始 |

**Sprint 0 验收**

- [x] 前后端可独立启动，`/healthz` `/readyz` `/api/v1/ping` 行为符合预期
- [x] 领域规则单元测试全绿（碳核算 12 例、积分 23 例）
- [x] 前端 TS7 类型检查通过、单测 5 例通过、生产构建成功
- [x] 两条迁移可执行（`000001` 参数与出行方式、`000002` 发件箱）
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
受限沙箱下子进程被拦截，在普通终端执行即正常；CI 使用标准 runner 不受影响。

**5. `docker compose` 报无法连接 daemon**
本机 Docker Desktop 未启动。启动后再执行部署命令，或改用方式 B 手动跑迁移、本地 `go run` + `npm run dev`。

---

## 相关文档

| 文档 | 内容 |
| --- | --- |
| [参考-2026版软件工程实验指导书](docs/参考-2026版软件工程实验指导书.txt) | 课程要求与报告样本（原文） |
| [项目分工与迭代计划](docs/项目分工与迭代计划.md) | 角色、WBS、13 周迭代、协作规范、风险预案 |
| [需求规格说明书](docs/需求规格说明书.md) | 角色、用例、域模型、非功能需求（实验一） |
| [Sprint 0 骨架说明](docs/Sprint0-骨架说明.md) | 实施记录、实测数据、踩坑与验收证据 |
| [数据库联通记录](docs/数据库联通记录.md) | 建库、迁移执行、连通性验证全过程 |
| [PlantUML 建模指南](docs/PlantUML建模指南.md) | 出图流程与 `scripts/uml-export.ps1` 用法 |
| [UML 源图与导出](docs/uml) | 用例图、活动图、域模型、包图、部署图、鲁棒图、序列图、状态图 |

---

## 许可说明

本项目为《软件工程实验》课程作业，仅供课程教学、答辩与团队内部学习使用，未授予公开分发或商业使用许可。文档与 UML 模型版权归小组成员共同所有；第三方依赖遵循其各自许可。
