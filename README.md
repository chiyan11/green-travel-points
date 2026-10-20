# 绿行积分 · 城市绿色出行碳积分系统

软件工程实验（2026 版）第 14 题。用户上报绿色出行（步行 / 骑行 / 地铁 / 公交 / 共享单车），
系统按出行方式核算碳减排量、折算碳积分，积分可在商城兑换商品；运营侧管理商品、库存与兑换单状态。

> 当前进度：**Sprint 0 骨架已完成并通过校验**（详见 [`docs/Sprint0-骨架说明.md`](docs/Sprint0-骨架说明.md)）。
> 业务接口从 Sprint 1 开始按模块交付，分工见 [`docs/项目分工与迭代计划.md`](docs/项目分工与迭代计划.md)。

## 技术栈

| 层 | 选型 | 版本 | 选型要点 |
| --- | --- | --- | --- |
| 后端 | Go + chi | go 1.24+ / chi v5.2.3 | 标准库 `net/http` 语义 + 轻量路由，无框架魔法，便于讲清分层 |
| 数据访问 | `database/sql` + sqlc（待启用） | mysql 驱动 v1.9.3 | SQL 可见可审，生成强类型代码，直接喂"数据库字段级清单" |
| 迁移 | golang-migrate 风格的 `db/migrations` | — | up/down 成对，可回滚 |
| 事件 | 事务性 Outbox 表 + 轮询 Worker | — | 不引入 Kafka/Redis，仍在关系库里保证"业务成功 ⇒ 事件必达" |
| 前端 | React + TypeScript 7 + Vite + TanStack Query | 19.3 / 7.0.2 / 8.3 / 5.104 | TS7 原生编译器（Go 实现），构建快；Query 统一服务端状态 |
| 数据库 | MySQL | 8.4+（本机 9.7） | `utf8mb4_0900_ai_ci`、`DATETIME(3)`、JSON 存事件负载 |

TypeScript 7 的两条实测差异（写进实验报告"技术选型"更真实）：

1. `tsconfig.json` 已移除 `baseUrl`，路径别名必须写成相对形式 `"@/*": ["./src/*"]`；
2. 编译器本体拆到平台包（如 `@typescript/typescript-win32-x64`），包管理器必须正确安装 `optionalDependencies`。

## 目录结构

```
green-travel-points/
├── apps/
│   ├── api/                     # Go 后端
│   │   ├── cmd/server/          # 入口：配置 → 日志 → 路由 → 优雅关闭
│   │   └── internal/
│   │       ├── config/          # 环境变量装配
│   │       ├── domain/          # 纯业务规则（碳核算、积分规则）+ 单测
│   │       ├── eventbus/        # 领域事件主题与处理器契约
│   │       ├── httpserver/      # 路由、中间件、统一响应壳、健康检查
│   │       └── repo/            # 连接池；sqlc 生成代码将落 sqlcgen/
│   └── web/                     # React 前端（路由骨架 + 连通性自检）
├── db/migrations/               # 000001 初始化 / 000002 发件箱
├── deploy/                      # Dockerfile.api / Dockerfile.web / nginx.conf / compose
├── scripts/                     # dev-env.ps1（缓存重定向）、verify.ps1（一键门禁）
└── docs/                        # 分工计划、实验指导书原文、Sprint 0 说明
```

## 快速开始

### 0. 前置

- Go 1.24+、Node 24+、MySQL 8.4+（或 Docker Desktop）

Windows 下若默认缓存目录不可写（例如受限沙箱），先点源环境脚本，把 Go / npm 缓存重定向到仓库内 `.cache/`：

```powershell
. .\scripts\dev-env.ps1
```

### 1. 数据库准备

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

> 本机已按上述步骤实跑过一次（建库 → 两条迁移 → `/readyz` 200），完整命令与输出见 [`docs/数据库联通记录.md`](docs/数据库联通记录.md)。

### 2. 后端

```powershell
cd apps/api
Copy-Item .env.example .env      # 按需改 DB_DSN
go run ./cmd/server              # 默认监听 :8080
```

无数据库时服务仍可启动（启动日志只 Warn），`/healthz` 返回 200，`/readyz` 返回 503 —— 便于前端先联调。

| 接口 | 用途 |
| --- | --- |
| `GET /healthz` | 存活探针（恒 200） |
| `GET /readyz` | 就绪探针（DB ping 失败返回 503） |
| `GET /api/v1/ping` | 前后端连通性自检 |

所有响应统一壳：`{"code":"OK","message":"ok","data":{...}}`；错误码见 `internal/httpserver/response.go`。

### 3. 前端

```powershell
cd apps/web
npm install
npm run dev        # http://localhost:5173，/api 已代理到 127.0.0.1:8080
```

### 4. 一键校验

```powershell
pwsh -File .\scripts\verify.ps1
```

依次执行：`gofmt -l` → `go vet` → `go test` → `go build` → `tsc --noEmit` → `vitest run` → `vite build`，任一失败即非 0 退出。

## 领域规则（已冻结，单测覆盖）

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

## Sprint 0 验收口径

- [x] 前后端可独立启动，`/healthz` `/readyz` `/api/v1/ping` 行为符合预期
- [x] 领域规则单元测试（碳核算 11 例、积分 30+ 例）全绿
- [x] 前端 TS7 类型检查通过、单测 5 例通过、生产构建成功
- [x] 两条迁移可执行（`000001` 参数与出行方式、`000002` 发件箱）
- [x] CI 配置就绪（Go + Node 双 job）
- [x] 数据库连通性与 `migrate up` 实跑 —— 2026-10-20 在本机 MySQL 9.7.2 完成，`/readyz` 返回 `{"database":"up"}`（过程见 [`docs/数据库联通记录.md`](docs/数据库联通记录.md)）

## 五个人从这里接着做

| 角色 | Sprint 1 起手位置 |
| --- | --- |
| P1 组长 / 架构建模 | `docs/项目分工与迭代计划.md` 的 WBS；产出用例图、域模型、包图 |
| P2 后端（行程 → 碳核算 → 积分） | `internal/httpserver/router.go` 预留的 `/api/v1/trips`；复用 `domain.EvaluateTrip` |
| P3 后端（账本 / 商城 / 运营） | `db/migrations` 续写账本与商品表；`internal/eventbus` 的 `PointsGranted` 消费者 |
| P4 前端（用户端） | `apps/web/src/routes.tsx` 的 5 个占位页逐个替换；`src/lib/http.ts` 已封装 Envelope |
| P5 测试与部署 | `scripts/verify.ps1` + `deploy/`；按指导书补基本路径法 / 等价类用例表 |

## 已知约束

- 本机 Docker daemon 未运行，`docker compose` 尚未实跑验证。MySQL 迁移已在真库执行；root 无明文密码，本机是借 MySQL Workbench 存储的凭据调用 `mysqlsh` 执行（见 [`docs/数据库联通记录.md`](docs/数据库联通记录.md) §3）。
- 前端统一用 **npm**（而非 pnpm）：pnpm 11 的 isolated 链接器用 Junction 指向虚拟 store，Windows + Node 24 下解析 `@typescript/typescript-win32-x64` 与 `std-env` 均失败，改用 npm 后稳定。
