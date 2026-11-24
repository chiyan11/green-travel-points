# AGENTS.md

> 给在此仓库里干活的 AI 代理（以及新加入的同学）的作业手册。
> 先读这一页，再动手；与本文冲突时，以代码和 `README.md` 为准，并回头修正本文。

项目：**绿行积分（Green Travel Points）** ——《软件工程实验》2026 版第 14 题「碳中和城市绿色出行积分系统」。
一句话：把一次步行 / 骑行 / 地铁 / 公交 / 共享单车的出行，换算成碳减排量与可兑换的积分。

- 后端：Go 1.27 + chi v5 + `database/sql` 手写 SQL，MySQL 8
- 前端：React 19 + TypeScript 7 + Vite 8 + TanStack Query，纯 CSS
- 架构：分层单体 + 事务性发件箱（Outbox）教学级事件驱动

---

## 1. 六条硬约束

以下六条破坏一次，后面所有工作都要返工。

1. **结论要有命令证据。** 说「改好了」之前必须跑过对应验证（`scripts/verify.ps1` 或针对性的 `go test` / `vitest`），并把输出贴出来。不要用「应该没问题」交付。
2. **不擅自扩大改动面。** 不升级依赖、不新增第三方库、不改技术选型、不动领域常量、不重排目录、不顺手格式化全仓。确需变更，先说明理由并取得确认。
3. **代码与文档同步。** 改了接口 → 改 `docs/api/README.md`；改了业务规则 → 改 `README.md` 规则表 + 对应单测；增删文件 → 改 `README.md` 目录结构。文档过期比没文档更糟。
4. **开发阶段，不做旧数据兼容。** 项目仍在开发期，**禁止**为旧的历史数据写兼容代码分支：不保留旧字段、旧格式、旧状态值的解析与回退路径，不写 `legacy` 判断，不做双写。表结构或数据模型变了就直接改迁移与种子，**丢掉旧数据**，用新数据跑通并测试。唯一例外是团队共享库：动它之前先按 §2「团队共享库」对齐，别把全组的数据洗掉。
5. **每次修改，三方体验一起变好。** 用户（更快完成目标、界面更清楚）、开发者（更好改、更好测）、AI Agent（更好定位、更好验证）三者都必须变好或至少不变差，并且**不能破坏现有功能**。任何一方明显变差就停下来重做，不要用「后面再优化」交付。
6. **只走分支 + PR，不直推 `main`。** 任何改动上传 GitHub 都必须先开独立分支（`<type>/<简短主题>`，如 `docs/pr-only-merge-rule`），在分支上提交并推送，然后开 PR；PR 说明写清改了什么、怎么验证，CI 通过后再合并进 `main`。**禁止** `git push origin main`、禁止在本地 `main` 上直接提交后推送、禁止跳过 PR 强推；紧急修复也一样（可以快速合并，但不能省略分支与 PR）。

---

## 2. 环境与启动

### 前置

| 依赖 | 版本要求 | 说明 |
| --- | --- | --- |
| Go | ≥ 1.24 | `go.mod` 声明 `go 1.24`；本机工具链为 go1.27.2，能编译 |
| Node.js | ≥ 22（本机 24.x） | 前端构建与测试 |
| npm | ≥ 10（本机 11.x） | **必须用 npm，不要用 pnpm**（见 FAQ 2/3） |
| MySQL | 8.0+（本机 9.x 亦可） | 也可用 `deploy/docker-compose.yml` 起 |

### 三条启动路径

```powershell
# ① 只跑后端（需要 MySQL；没有库也能起，/readyz 会 503）
cd apps/api
go run ./cmd/server                 # :8080

# ② 只跑前端（页面可开，列表为空或 401）
cd apps/web
npm install
npm run dev                         # http://localhost:5173，/api 代理到 127.0.0.1:8080

# ③ 全栈 + 数据库（Docker）
docker compose -f deploy/docker-compose.yml up -d --build
```

### 必须知道的四个事实

1. **配置只来自环境变量。** `internal/config/config.go` 只调用 `os.LookupEnv`，**不解析 `.env` 文件**；复制 `.env.example` 成 `.env` 不会生效。默认值可直接连本机库：`greentravel:greentravel@tcp(127.0.0.1:3306)/green_travel_points`。
2. **端口写死。** 后端 `:8080`，前端 `5173`（`apps/web/vite.config.ts`），代理规则 `/api → 127.0.0.1:8080`。改端口要同时改这两处。
3. **缓存重定向。** 若全局缓存目录不可写（受限沙箱、只读用户目录），先执行 `. .\scripts\dev-env.ps1`，它把 `GOCACHE` / `GOMODCACHE` / `npm` 缓存指到仓库内 `.cache/`。
4. **种子账号不能登录。** `db/seed/dev_demo.sql` 的 `password_hash` 是占位串，后端校验的是 `sha256$<salt>$<digest>` 格式，二者对不上。要么注册新账号，要么用开发期身份头 `X-User-Id: 1001`。

> **团队共享库（ZeroTier 内网）**：连接方式见 [`docs/dev/shared-db.md`](dev/shared-db.md)。
> 该链路延迟较高，后端启动需带 `HTTP_REQUEST_TIMEOUT=60s`，`DB_DSN` 要加 `timeout/readTimeout/writeTimeout`；
> 日常开发建议本地库，联调/演示切共享库。
> **若本机就是共享库主机**（`docs/dev/shared-db.md` 顶部写明主机名的那台），`127.0.0.1` 与共享库是**同一实例**：
> 切换只影响延迟不改数据，任何 `down` 迁移 / 清表 / 批量造数据都等于直接改全组。

### 常用命令

| 目的 | 命令 | 工作目录 |
| --- | --- | --- |
| 跑全部测试（后端） | `go test ./... -count=1` | `apps/api` |
| 跑单个包 | `go test ./internal/domain -v` | `apps/api` |
| 竞态检测（同 CI） | `go test ./... -race -count=1` | `apps/api` |
| 格式化 | `gofmt -w .` | `apps/api` |
| 静态检查 | `go vet ./...` | `apps/api` |
| 编译产物 | `go build -o ../../.cache/bin/server.exe ./cmd/server` | `apps/api` |
| 前端类型检查 | `npx tsc --noEmit` | `apps/web` |
| 前端单测 | `npx vitest run` | `apps/web` |
| 前端单测（监听） | `npx vitest` | `apps/web` |
| 生产构建 | `npx vite build` | `apps/web` |
| **一键门禁** | `pwsh -File .\scripts\verify.ps1` | 仓库根 |
| 导出 UML 成图 | `pwsh -File .\scripts\uml-export.ps1` | 仓库根 |
| 导入种子数据 | `mysql ... green_travel_points < db/seed/dev_demo.sql` | 仓库根 |

---

## 3. 仓库地图

```
green-travel-points/
├── apps/
│   ├── api/                      Go 后端（module: greentravelpoints）
│   │   ├── cmd/server/main.go    进程入口：装配 config → repo → 端口实现 → httpserver
│   │   ├── cmd/server/dbendpoint.go  从 DB_DSN 解析目标地址（启动日志提示本机库 / 共享库，不打印凭据）
│   │   ├── internal/
│   │   │   ├── config/           环境变量 → Config（无 .env 解析）
│   │   │   ├── domain/           纯业务规则：碳核算、积分、订单状态机（无 I/O、可裸测）
│   │   │   ├── app/              DTO、端口接口（Ports）、哨兵错误
│   │   │   ├── repo/             SQL 实现：连接池、事务、行锁、Outbox 写入
│   │   │   ├── httpserver/       路由、中间件、统一响应壳、各 handler
│   │   │   └── eventbus/         事件契约（Worker 尚未实现）
│   │   ├── sqlc.yaml             已配置，**未启用**（当前是手写 SQL）
│   │   └── .env.example          可用环境变量清单（不是被自动加载的文件）
│   └── web/                      React 前端
│       ├── src/
│       │   ├── main.tsx          应用引导（QueryClientProvider + RouterProvider）
│       │   ├── routes.tsx        路由表：/login 公开，其余 RequireAuth + AppLayout
│       │   ├── pages/            9 个页面：Home / Login / Trips / Points / Mall / ProductDetail / Orders / Profile / Placeholder
│       │   ├── components/       AppLayout（外壳+导航）、RequireAuth（守卫）、CarbonTrendChart（折线图）
│       │   ├── services/         按域拆的 API 调用：auth / trips / products / points / orders（按生成契约标注传输类型）
│       │   ├── lib/              http（Envelope 解包、错误、401 处理）、session、validators、format、orderStatus、productCover、queryClient
│       │   ├── types/api.ts      由 docs/api/openapi.yaml 生成的 wire 类型（npm run gen:api，勿手改）
│       │   ├── types/domain.ts   页面视图模型（id 转 string、coverUrl、startedAt 等展示口径）
│       │   └── styles.css        全站样式（单文件，无 CSS 框架）
│       └── vite.config.ts        端口 5173 + /api 代理
├── db/
│   ├── migrations/               5 组 up/down 迁移（见 §6）
│   └── seed/dev_demo.sql         演示数据：用户 1001、管理员 1002、3 个商品
├── deploy/                       docker-compose.yml、Dockerfile.api、Dockerfile.web、nginx.conf
├── docs/                         需求规格、分工计划、Sprint 0 记录、数据库记录、接口说明、前端接口清单、UML
├── scripts/                      dev-env.ps1（缓存重定向）、verify.ps1（一键门禁）、uml-export.ps1、toc_pages.py、build_team_docx.py
├── .github/workflows/ci.yml      Go job + Web job
└── README.md                     面向人的总览（本文是面向干活的细节补充）
```

**先看哪里**

| 你要做的事 | 先读 |
| --- | --- |
| 改业务规则（碳、积分、状态机） | `apps/api/internal/domain/*.go` 与其 `_test.go`，再读 `README.md` 规则表 |
| 加 / 改接口 | `apps/api/internal/httpserver/router.go` → 同目录对应 handler → `docs/api/README.md` |
| 改数据库 | `db/migrations/` 最后一组；表关系见 `docs/uml/04-域模型.puml` |
| 接前端页面 | `apps/web/src/services/` 与 `types/domain.ts`，再看对应 `pages/` |
| 查「为什么这么设计」 | `docs/需求规格说明书.md`、`docs/项目分工与迭代计划.md`、`docs/Sprint0-骨架说明.md` |
| 查环境 / 连库问题 | `docs/数据库联通记录.md`、`README.md` 常见问题 |

---

## 4. 后端规范

### 依赖方向（单向，禁止反向）

```
cmd/server  ──►  httpserver  ──►  app（DTO + 端口接口 + 哨兵错误）
                    │                    ▲
                    └──►  repo  ─────────┘（repo 实现 app 定义的接口）
                              │
domain ◄──────────────────────┘（最内层，谁都可以依赖它，它不依赖任何人）
```

- `domain` **只能** import 标准库；一旦出现 `database/sql`、`net/http` 就说明放错层了。
- `httpserver` 不写 SQL；`repo` 不出现 `http` 类型；跨层错误用 `app` 里的哨兵错误（`errors.Is`）翻译。
- 端口接口在 `app` 定义，实现在 `repo`；handler 只依赖接口，因此单测可以注入假存储、**不连 MySQL**。

### 代码该放哪（决策表）

| 你在写什么 | 放哪 | 备注 |
| --- | --- | --- |
| 计算公式、状态流转、参数校验的业务部分 | `internal/domain` | 纯函数，输入输出之外无副作用 |
| 请求 / 响应结构体、分页结构 | `internal/app/types.go` | JSON tag 与 `docs/api/README.md` 对齐 |
| 存储能力抽象 | `internal/app/ports.go` | 一个能力一个接口，方法少而稳 |
| SQL、事务、行锁、Outbox | `internal/repo` | 文件名按域拆：`orders.go` / `points.go` / `products.go` / `admin.go` |
| 路由、中间件、响应壳 | `internal/httpserver` | 路由集中在 `router.go` |
| HTTP handler | `internal/httpserver/<域>.go` | 只做：解析参数 → 调端口 → 写响应 |
| 环境变量 | `internal/config` | 新增配置要给默认值，否则本地跑不起来 |

### 约定

- **响应壳统一**：`{"code":"OK","message":"ok","data":...}`，用 `httpserver.OK` / `Fail` / `FailInternal` 写，别自己 `json.NewEncoder`。
- **错误码稳定**：`OK` / `INVALID_ARGUMENT` / `UNAUTHORIZED` / `FORBIDDEN` / `NOT_FOUND` / `CONFLICT` / `INTERNAL` / `UNAVAILABLE`，以及业务细分 `INSUFFICIENT_POINTS` / `OUT_OF_STOCK` / `LIMIT_EXCEEDED` / `ILLEGAL_TRANSITION`（见 `internal/httpserver/response.go`）。码是给前端和测试断言的契约，不要随改随变。
- **内部错误不外泄**：用 `FailInternal`（日志记详情、客户端只看到「服务器内部错误」）。
- **请求体严格解码**：用 `DisallowUnknownFields`，多余字段直接 400，避免前后端字段漂移悄悄通过。
- **日志**：`log/slog` 结构化，key 用小写短词（`"err"`、`"user_id"`）；handler 不打印业务数据明细。
- **超时与关闭**：中间件 `Timeout` 取 `HTTP_REQUEST_TIMEOUT`（默认 15s）；`main.go` 用 `HTTP_SHUTDOWN_TIMEOUT` 优雅关闭，改动启动逻辑时不要绕过。
- **路由可裁剪**：`NewRouter` 的 `Catalog` / `Points` / `Orders` / `Admin` / `Roles` 为 nil 时不挂载该组，单测可只装被测模块——新加端口时保持这个惯例。

### 接口清单（改接口前先看这张表）

| 分组 | 端点 | 鉴权 |
| --- | --- | --- |
| 探针 | `GET /healthz`、`GET /readyz` | 无 |
| 认证 | `POST /api/v1/auth/register`、`POST /api/v1/auth/login`、`GET /api/v1/me` | 公开 / 需身份 |
| 行程 | `POST /api/v1/trips`、`GET /api/v1/trips` | 需身份 |
| 商品 | `GET /api/v1/products`、`GET /api/v1/products/{id}` | 公开 |
| 积分 | `GET /api/v1/points/summary`、`GET /api/v1/points/transactions` | 需身份 |
| 兑换 | `POST /api/v1/orders`、`GET /api/v1/orders`、`POST /api/v1/orders/{id}/cancel` | 需身份 |
| 运营 | `GET|POST /api/v1/admin/products`、`PATCH /api/v1/admin/products/{id}`、`PUT /api/v1/admin/skus/{id}/stock`、`GET /api/v1/admin/orders`、`POST /api/v1/admin/orders/{id}/(confirm|ship|complete)`、`GET /api/v1/admin/points/reconcile` | 需 ADMIN |

身份：优先 `Authorization: Bearer <token>`（登录签发，2 小时过期）；无 token 时回退 `X-User-Id: <id>`（开发便利，生产须关）。运营组再叠加 `RequireAdmin`，越权 403。

---

## 5. 领域规则（改之前先读，改之后必须改测试）

| 规则 | 值 / 公式 | 位置 |
| --- | --- | --- |
| 基准排放因子 | `EmissionFactorBase = 192.0` g CO₂/km（燃油小汽车） | `apps/api/internal/domain/carbon.go` |
| 各方式因子 | 步行 0、骑行 0、地铁 41、公交 68、共享单车 0（g/km） | 同上 `SeedEmissionFactors` |
| 碳减排量 | `max(0, 距离 × (192 − 该方式因子))` 克 | 同上 |
| 积分换算 | `floor(碳减排克数 / 100 + 1e-9)`，即 100 g = 1 分 | `apps/api/internal/domain/points.go:30` 的 `DefaultPointsRule` |
| 每日积分上限 | 500 分 | 同上 `DailyCap` |
| 有效距离 | 0.1 km ≤ d ≤ 100 km | 同上 `MinDistanceKm` / `MaxDistanceKm` |
| 订单状态机 | `PENDING → PAID | CANCELLED`；`PAID → SHIPPED | CANCELLED`；`SHIPPED → COMPLETED | CANCELLED` | `apps/api/internal/domain/order.go` 的 `orderTransitions` |

改规则的固定动作：

1. 改 `domain` 里的常量 / 公式（**不要**在 handler 或 repo 里就地写魔法数）。
2. 补 / 改 `domain` 表驱动用例，覆盖边界（0 km、0.099 km、100 km、100.001 km、日上限耗尽、非法状态流转）。
3. 同步 `README.md` 的规则表与 `docs/需求规格说明书.md` 的相关段落。

---

## 6. 数据层

### 表清单（5 组迁移，13 张表）

| 迁移 | 表 | 作用 |
| --- | --- | --- |
| `000001` | `sys_config`、`travel_mode` | 系统参数、出行方式与排放因子 |
| `000002` | `event_outbox` | 事务性发件箱（`topic/aggregate/payload/attempts/available_at/dispatched_at/dead_letter_at`） |
| `000003` | `point_account`、`point_txn` | 积分账户（余额/冻结/累计减碳/version）与**只追加**流水 |
| `000004` | `user`、`trip_record`、`carbon_record` | 用户、行程、碳记录 |
| `000005` | `merchant`、`product`、`product_sku`、`redeem_order`、`point_batch` | 商城与兑换（批次用于 FIFO 扣减与过期） |

### 不变量（写 SQL 时不能破）

- **流水只追加**：`point_txn` 不更新、不删除；余额变更必须同时写一条流水，`balance_after` 与该次变更后余额一致。
- **一个业务事务一个事务**：行程上报 = `trip_record` + `carbon_record` + `point_txn` + `point_account` 余额 + `event_outbox`，全在一个 `tx`，任一步失败整体回滚。
- **扣分按批次 FIFO**：先扣最早的 `point_batch`（`remaining`、`expires_at`），下单与取消都走批次，不直接改余额了事。
- **并发安全**：涉及余额/库存的读改写必须 `SELECT ... FOR UPDATE`（现有实现见 `apps/api/internal/repo/orders.go` 的 `CreateOrder`）。
- **幂等**：`POST /orders` 支持 `Idempotency-Key`，命中即返回原订单，不重复扣分。

### 加一个迁移

1. 新建成对文件，序号连续：`db/migrations/000006_xxx.up.sql` + `000006_xxx.down.sql`。
2. `up` 用 `CREATE TABLE IF NOT EXISTS`、幂等 `ALTER`；`down` 必须能真正回滚（删表 / 删列 / 删索引）。
3. 同步更新本节表格与 `README.md` 目录结构里的迁移列表。
4. 开发阶段**优先直接改已有迁移**并重建库（`DROP DATABASE` 后从空库重跑），不要为旧表结构追加兼容迁移；提交前确认从空库能一路跑到最新。只有在联调 / 共享库上才按「新变更新文件」处理，先看 §2「团队共享库」。
5. 演示数据放 `db/seed/`，**不要**混进迁移序列（种子不参与 CI）。

---

## 7. 前端规范

### 分层

```
pages/*.tsx  ──►  services/*.ts  ──►  lib/http.ts  ──►  fetch
    │                  │
    └── types/domain.ts（视图模型）      lib/session.ts（token 存取）
                       └── types/api.ts（wire 契约，由 openapi.yaml 生成）
```

- **组件不直接 `fetch`**：请求只写在 `services/`，页面用 TanStack Query（`useQuery` / `useMutation`）调用。
- **两层类型，不许混用**：`types/api.ts` 是后端 wire 契约（`npm run gen:api` 从 `docs/api/openapi.yaml` 生成，禁止手改）；`types/domain.ts` 是页面视图模型（`id` 统一 `string`、`imageUrl→coverUrl`、`traveledAt→startedAt`、`bizRefType/occurredAt` 等）。映射只写在 `services/` 的 `toXxx()` 里。
- **接口变更顺序**：改 `apps/api` → 改 `docs/api/openapi.yaml` → `cd apps/web && npm run gen:api` → 修 `services/` 映射 → 提交生成物。门禁会校验生成物与 yaml 一致。
- **后端字段差异在 service 层吸收**，页面只看到干净的前端类型。现有例子：`status`/`active` 归一化、`traveledAt`/`startedAt`、`refType→bizRefType`、`createdAt→occurredAt`、`rawPoints`/`grantedPoints` 推算 `cappedByDaily`。新增接口不许把原始 JSON 直接漏给页面。
- **401 统一处理**：`lib/http.ts` 收到 401 会清会话并跳 `/login`，页面不要重复写这套逻辑。
- **会话存 `localStorage` 的 `gtp_session`**（`lib/session.ts`）；不要另起 key。
- **路由守卫**：`routes.tsx` 里 `/login` 公开，其余包 `RequireAuth` + `AppLayout`；新增页面挂到对应子路由下。
- **样式**：全站单文件 `styles.css`，用现有类名（`btn` / `card` / `field-error` / `badge--ok` 等）；不引入 CSS 框架，不写内联大段样式。
- **类型**：`tsc --noEmit` 必须零错误；不用 `any` 绕过（确需时先说明）。

### 界面文案与信息密度

- **默认先删**：删掉多余的标语（slogan）、说明文字、次要标签（badge / 副标题 / 单位提示 / 重复标题）。每段文字都要承担信息或操作，否则删掉。
- **删完看不懂就重设计**：如果去掉说明文字后页面无法独立看懂，说明布局与信息架构本身有问题——重排布局、调整层级、改控件形态，而不是把文字加回来。
- **文案不给布局打补丁**：不用「帮助说明」「温馨提示」「* 为必填」之类的长段落掩盖入口难找、主次不清。

---

## 8. 测试

### 现状（写新测试时对齐这个风格）

| 层级 | 位置 | 规模 | 特点 |
| --- | --- | --- | --- |
| 领域单测 | `apps/api/internal/domain/*_test.go` | 12 个函数 / 45 例 | 表驱动，覆盖边界与非法输入 |
| HTTP 单测 | `apps/api/internal/httpserver/*_test.go` | 24 例 | 注入假存储，**不连 MySQL**；断言响应壳与错误码 |
| 启动日志单测 | `apps/api/cmd/server/dbendpoint_test.go` | 1 个函数 / 9 例 | 表驱动：默认 DSN、局域网 / ZeroTier、localhost、IPv6 回环、unix socket、省略地址、非法 DSN |
| 前端单测 | `apps/web/src/lib/*.test.ts` | 2 文件 19 例 | `http.test.ts` 用 `vi.stubGlobal('fetch')` 测解包与错误映射；`validators.test.ts` 测校验规则 |

### 新增功能的底线

- 新的领域规则 → 至少 1 组表驱动用例，含边界值与非法输入。
- 新的 HTTP 端点 → `api_test.go` 加用例：正常路径 + 至少一个错误路径（参数错 / 未授权 / 冲突），断言 `code` 与状态码。
- 新的前端校验或解包逻辑 → `vitest` 用例；纯展示改动不强制。
- 不加「只为了覆盖率」的空测试；不为通过而放宽断言。

### 命令

```powershell
cd apps/api; go test ./... -count=1              # 全量
cd apps/api; go test ./internal/domain -run TestCarbon -v
cd apps/web; npx vitest run                      # 全量
cd apps/web; npx vitest run src/lib/http.test.ts # 单文件
```

---

## 9. 门禁（提交前必须过）

```powershell
pwsh -File .\scripts\verify.ps1
```

八步，任一失败即非 0 退出并打印 `VERIFY FAILED: <步骤>`：

| # | 步骤 | 失败常见原因 |
| --- | --- | --- |
| 1 | `gofmt -l .` | 忘记格式化；直接 `gofmt -w .` |
| 2 | `go vet ./...` | 未使用的变量、错的格式化占位符 |
| 3 | `go test ./...` | 改了规则没改测试，或断言写死旧数值 |
| 4 | `go build` | 导入未使用的包、接口实现不全 |
| 5 | `web: api contract`（`npm run gen:api` + `git status`） | 改了 `openapi.yaml` 没重新生成，或手改了 `types/api.ts`；跑 `npm run gen:api` 并提交 |
| 6 | `tsc --noEmit` | 类型漂移；TS7 不认 `baseUrl`，路径别名用相对形式 |
| 7 | `vitest run` | 断言过时、mock 返回值形状变了 |
| 8 | `vite build` | 仅构建期才暴露的引用错误 |

CI（`.github/workflows/ci.yml`）两个 job 与本地一致，差异：CI 用 `go test -race -coverprofile`、`npm ci`。

---

## 10. 文档与图

| 文档 | 什么时候必须更新 |
| --- | --- |
| `README.md` | 增删文件 / 目录、改技术栈、改接口分组、改进度、加 FAQ |
| `docs/api/README.md` | 端点、请求/响应字段、错误码、事务步骤、身份约定变化 |
| `docs/需求规格说明书.md` | 业务规则、角色、用例、非功能指标变化 |
| `docs/uml/*.puml` | 类图 / 序列图 / 状态机随实现变化；改完跑 `scripts/uml-export.ps1` 重新出图（`out/` 下 PNG 一并提交） |
| `docs/项目分工与迭代计划.md` | 里程碑或分工调整 |
| 本文件 | 规范变化、踩到新坑 |

约定：文档用中文，代码注释用中文，标识符用英文；Markdown 用 LF、UTF-8 无 BOM。

---

## 11. 已知缺口与技术债

| 项 | 现状 | 若要接手 |
| --- | --- | --- |
| Outbox Worker | 事件只写入 `event_outbox`，**没有消费者**；`internal/eventbus` 只有契约。生产端事件名还没统一：订单侧写死字面量 `OrderPaid` / `OrderCancelled`，常量却是 `TopicRedeemOrderPaid = "RedeemOrderPaid"`，且 `OrderCancelled` 无常量 | 先统一 topic 常量（含补 `OrderCancelled`），再实现轮询 + 退避 + 死信，读 `dispatched_at IS NULL AND available_at <= NOW()` |
| 密码哈希 | 加盐 SHA-256（`sha256$salt$digest`），注释已标"临时方案" | 换 bcrypt / argon2，并给出存量数据迁移方案 |
| Token | 自签 HMAC-SHA256 紧凑串，**不是标准 JWT**（无 header 段） | 需要标准 JWT 时统一改签发与校验两端 |
| sqlc | `apps/api/sqlc.yaml` 已配置但未启用，repo 全是手写 SQL | 启用即要重写 repo 层，别半途混用 |
| 集成 / E2E | 均未建（计划 testcontainers + Playwright） | 先补「上报 → 得分 → 兑换 → 核销」主链路 |
| 部署 | compose 只有 `mysql` + `api`；`web` 与 nginx 配置就绪但未编入；本机未实跑 docker | 补 web 服务与反向代理，留下实跑证据 |
| 前端包体 | `vite build` 单包约 1.49 MB（gzip 489 kB），已触发 500 kB 警告 | 用路由级 `import()` 分包，图表库按需加载 |

---

## 12. 排错速查

| 症状 | 原因 | 处理 |
| --- | --- | --- |
| `spawn EPERM`（vitest / vite） | 受限沙箱拦截子进程 | 换普通终端执行；CI 不受影响 |
| 仓库子目录读写 `EPERM` | 目录 ACL 不含当前用户（复制/解压继承） | `icacls "<仓库根>" /grant "$env:USERNAME:(OI)(CI)F" /T /C` |
| `/readyz` 503 | 连不上 MySQL | 确认库在跑、账号 `greentravel/greentravel` 存在、`DB_DSN` 正确 |
| 页面 401 | token 过期（2h）或未登录且 `X-User-Id` 被改 | 重新登录；或保持回退头 |
| 商城 / 积分列表为空 | 没导种子；库里确实没数据 | 导入 `db/seed/dev_demo.sql` |
| `docker compose` 连不上 daemon | Docker Desktop 未启动 | 启动后重试，或改本地 `go run` + `npm run dev` |
| `tsc` 报 `TS5102 baseUrl` | TS7 移除该选项 | 路径别名写成相对形式 |
| pnpm 安装依赖失败 | TS7 平台二进制 + Windows Junction 问题 | 改用 npm |

---

## 13. 干活的固定动作

**开工前**

1. `git status` 确认工作区干净（有未提交改动先问清楚是谁的、要不要保留）。
2. 按 §3「先看哪里」定位文件，读相关代码与它的测试，别凭印象改。
3. 明确验收方式：这次改动由哪条命令证明它是对的。

**改动中**

- 小步走：先让测试红，再让它绿；一次只解决一件事。
- 遵守 §4 依赖方向与 §7 分层，别为了省事把 SQL 写进 handler、把 fetch 写进组件。
- 涉及界面时先按 §7「界面文案与信息密度」删一轮标语 / 说明文字 / 次要标签，再判断布局要不要重做。
- 动数据结构时按 §1 第 4 条：直接改迁移、丢旧数据，不写兼容分支。
- 发现的顺手问题记下来，不要在同一提交里捎带修（除非你是被要求做这次清理）。

**收尾**

1. 跑 `scripts/verify.ps1`，绿了再说话。
2. `git diff` 自查：有没有调试残留、打印、临时注释、被顺手改掉的无关行。
3. 同步文档（§10），更新 §11 技术债（修掉一项就划掉一项）。
4. 提交信息：`<type>(<scope>): <中文摘要>`，type 用 `feat|fix|docs|refactor|test|chore`，scope 用 `api|web|db|docs|deploy`。例：`feat(api): 行程上报支持每日积分上限`。
5. 上传 GitHub：`git switch -c <type>/<主题>` → 提交 → `git push -u origin <分支>` → 开 PR（写清改动与验证方式）→ CI 绿 → 合并到 `main`；合并后切回 `main` 并 `git pull`，删掉已合并的本地分支。**不直接推 `main`。**

**不要做**

- 不要 `git commit --no-verify` 跳过门禁。
- 不要升级 `go.mod` / `package.json` 里的版本，除非任务明确要求。
- 不要在 `domain` 里引入 I/O，不要在 handler 里写 SQL。
- 不要用 `any` / `//nolint` / 空 `catch` 掩盖错误。
- 不要把密钥、真实手机号、个人数据写进代码或种子。
- 不要为旧数据 / 旧字段 / 旧格式写兼容分支，也不要留「以后再删」的过渡代码。
- 不要用标语、说明文字、次要标签补救布局（见 §7）。
- 不要直接 `git push` 到 `main`；一律分支 + PR（见 §1 第 6 条）。
