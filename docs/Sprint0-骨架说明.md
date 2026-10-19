# Sprint 0 骨架说明（实施记录与验收证据）

> 本文是**实施记录**，不是计划。计划见 [`项目分工与迭代计划.md`](项目分工与迭代计划.md)。
> 用途：① 交接给组内其余四人；② 实验报告"系统原型构建与实现""技术选型"两节的素材来源。

## 1. 本轮交付了什么

| 交付物 | 位置 | 状态 |
| --- | --- | --- |
| Go 后端骨架（配置 / 日志 / 路由 / 中间件 / 统一响应壳 / 健康检查） | `apps/api/` | 可编译、可启动、可测 |
| 领域规则实现 + 单元测试（碳核算、积分） | `apps/api/internal/domain/` | 35 个用例全绿 |
| 数据库迁移 000001（参数与出行方式）、000002（事务性发件箱） | `db/migrations/` | 文件就绪，待真库执行 |
| React + TS7 前端骨架（路由、布局、连通性自检） | `apps/web/` | 类型检查 + 单测 + 构建通过 |
| 部署与 CI | `deploy/`、`.github/workflows/ci.yml` | 配置就绪，Docker 未实跑 |
| 一键校验脚本 | `scripts/verify.ps1` | 已验证全绿 |

## 2. 环境实测结果

| 项 | 实测 | 影响 |
| --- | --- | --- |
| go | 1.27.2（GOROOT `D:\go`） | `go.mod` 声明 `go 1.24`，向后兼容 |
| node / npm | v24.16.0 / 11.18.0 | 满足 Node 24 要求 |
| MySQL | 服务 MySQL97 Running，客户端 9.7 | root 密码未知，**迁移未在真库执行** |
| Docker | CLI 29.8.2，daemon 未运行 | `docker compose` 未验证 |
| TypeScript | 7.0.2（`latest`，原生 Go 编译器） | 可用，但有两处破坏性变化（见下） |
| Vite | 8.3.4 | 构建 152 ms，产物 352 kB（gzip 112 kB） |

### 2.1 踩到的坑与处理（写报告时原样可用）

1. **TS7 移除 `baseUrl`**：`tsc` 报 `TS5102: Option 'baseUrl' has been removed`。改为相对路径别名 `"@/*": ["./src/*"]`，并删除 `baseUrl`。
2. **TS7 平台包未安装**：`tsc` 报 `Unable to resolve @typescript/typescript-win32-x64`。TS7 把编译器编译成各平台二进制，通过 `optionalDependencies` 分发，包管理器必须正确安装（npm 正常）。
3. **pnpm isolated 链接器在 Windows 失效**：Junction 指向 `.pnpm/` 虚拟 store 时，Node ESM 解析既找不到 TS 平台包，也找不到 `std-env`（`ERR_MODULE_NOT_FOUND`）。改为 hoisted 仍不稳定，**最终前端统一使用 npm**，锁 `package-lock.json`。
4. **受限沙箱下 npm/Vite 子进程 EPERM**：`vite build` / `vitest` 内部 `exec` 被沙箱拦截（`spawn EPERM`）。在普通终端执行即正常——这也是 CI 用标准 runner 的原因。

## 3. 验收证据（真实命令输出）

### 3.1 后端

```
$ go vet ./...                       → 0（无输出）
$ go test ./... -count=1
ok  greentravelpoints/internal/domain      0.357s
ok  greentravelpoints/internal/httpserver  0.518s
$ gofmt -l .                         → 空
$ go build ./cmd/server              → 0
```

启动冒烟（无数据库场景）：

```
GET /healthz        → 200 {"code":"OK","message":"ok","data":{"status":"ok"}}
GET /readyz         → 503 {"code":"UNAVAILABLE","message":"数据库不可用"}
GET /api/v1/ping    → 200 {"code":"OK","message":"ok","data":{"env":"dev","pong":"green-travel-points"}}
GET /api/v1/nope    → 404 {"code":"NOT_FOUND","message":"接口不存在: /api/v1/nope"}
```

### 3.2 前端

```
$ npx tsc --version          → Version 7.0.2
$ npx tsc --noEmit           → 0
$ npx vitest run
 ✓ src/lib/http.test.ts (5 tests) 22ms
 Test Files 1 passed | Tests 5 passed
$ npx vite build
 ✓ 74 modules transformed
 dist/assets/index-BAQ9x6UN.js  352.47 kB │ gzip: 111.62 kB
 ✓ built in 152ms
```

## 4. 领域规则与测试覆盖（报告"组件测试"素材）

实现位置 `apps/api/internal/domain/points.go`：

```go
func (r PointsRule) PointsFor(carbonSavedGrams float64) (int64, error)  // floor(x/100 + 1e-9)
func (r PointsRule) CapDaily(raw, alreadyToday int64) int64             // 0 表示不限制
func EvaluateTrip(distanceKm float64, e Emission, rule PointsRule, alreadyToday int64) (Grant, error)
```

| 测试 | 覆盖要点 | 用例数 |
| --- | --- | --- |
| `carbon_test.go` | 5 种出行方式、0.1/100 km 边界、0/负/NaN/Inf 非法输入、EF 倒挂下界 0 | 12 |
| `points_test.go` | 取整（99.999→0、100→1、300→3 浮点误差、453→4）、距离区间边界、单日上限削平、`EvaluateTrip` 端到端 | 23 |
| `router_test.go` | 健康检查、无库就绪探针 503、404 统一壳 | 4 |
| `http.test.ts` | Envelope 解包、业务错误转 `ApiError`、非 JSON 响应、POST 自动补头 | 5 |

**待补（Sprint 3）**：实验指导书要求"基本路径法"，需对 `EvaluateTrip` 与后续 `RedeemOrder` 状态机画出控制流图并计算 V(G)，当前仅覆盖了等价类划分。

## 5. 与计划的两处偏离

1. **前端包管理器 pnpm → npm**，原因见 2.1 第 3 条；`package-lock.json` 已入库。
2. **Docker/数据库未实跑**：本机 daemon 未启动、MySQL root 密码未知。`deploy/` 与迁移脚本是"纸面就绪"，需 P2/P5 在第一次集成分支上补真实执行截图。

## 6. 下一步（Sprint 1 起手）

1. **P1**：基于 `db/migrations` 与 `internal/domain` 出域模型、用例图、包图；把 `README` 的领域规则补进需求规格说明书。
2. **P2**：`/api/v1/auth` 注册登录 → `/api/v1/trips` 行程上报（复用 `EvaluateTrip`）→ 写入 `trip_record` 与 `event_outbox`。
3. **P3**：补 `point_account` / `point_txn` / `product` / `redeem_order` 迁移；实现 `PointsGranted` 消费者与兑换状态机。
4. **P4**：把 `apps/web/src/routes.tsx` 的 5 个占位页替换为真实页面，先做"行程上报 → 积分查询"闭环。
5. **P5**：真库跑迁移、补基本路径法用例、把 `scripts/verify.ps1` 接到 PR 门禁。
