# P3 接口说明（M5 / M6 / M7）

> 实现：`apps/api/internal/httpserver`（HTTP 层）+ `apps/api/internal/repo`（SQL 层）+ `apps/api/internal/app`（DTO 与端口）。
> 对应报告：实验三"系统原型构建与实现"、实验二"设计类清单 / 序列图"。
> 状态：✅ 已实现并通过单元测试 + 本地 MySQL 联调。身份已支持 `Authorization: Bearer <token>`（`POST /api/v1/auth/login` 自签签发，有效期 2 小时）；未带 token 时回退开发期请求头 `X-User-Id`，便于不登录联调。

## 机器可读契约：[openapi.yaml](openapi.yaml)

本目录下的 `openapi.yaml` 是前后端唯一的传输契约（OpenAPI 3.1），覆盖 auth / products / points / trips / orders / admin 全部端点。
上文各小节的表格是给人读的说明，字段以 `openapi.yaml` 为准；两者不一致时按 yaml 修文档。

流程：

```bash
cd apps/web
npm run gen:api     # 读 ../../docs/api/openapi.yaml，生成 src/types/api.ts（勿手改）
```

- 生成物 `apps/web/src/types/api.ts` 需要提交；`apps/web/src/services/*.ts` 全部按它标注传输类型，
  再把 wire 字段映射成页面视图模型（`apps/web/src/types/domain.ts`）。
- 门禁会校验「生成物与 yaml 一致」：`scripts/verify.ps1` 的 `web: api contract` 步骤、
  CI 的「契约类型与 openapi.yaml 一致」步骤（`npm run gen:api` 后 `git diff --exit-code`）。
- 改接口的顺序固定为：改 `apps/api` → 改 `openapi.yaml` → `npm run gen:api` → 修 `services/` 的映射 → 提交。

生成器选型说明：原计划用 `openapi-typescript`，但它把 `typescript@^5.x` 声明为 peer 依赖，
而本项目前端是 TypeScript 7（Go 实现的编译器），`npm install` 会 ERESOLVE、`npm ci` 直接失败；
`--legacy-peer-deps` 能装上但 CI 的 `npm ci` 仍然报错，且它运行时依赖 TS 编译器 API（`ts.SyntaxKind`），
在 TS7 下会 `TypeError`。`@hey-api/openapi-ts` 同样卡在这个运行时 API 上。
因此改用 `swagger-typescript-api`（devDependency，自带 `typescript@^6` 副本，不改动项目 TS 版本），
`--no-client --enum-style union` 只产出 data contract，正好是 wire 类型所需的东西。
等前端升到官方 TS 5.x/6.x 后可以再评估换回 `openapi-typescript`，契约与生成物路径都不受影响。

## 0. 约定

- 统一响应壳：`{"code":"OK","message":"ok","data":...}`；错误时 `data` 省略。
- 分页参数：`page`（从 1 起，默认 1）、`pageSize`（默认 20，上限 100）；响应 `{items, page, pageSize, total}`。
- 时间：RFC3339（带时区）。
- 金额/积分：整数 `int64`。
- 身份：优先 `Authorization: Bearer <token>`（登录 / 注册响应里的 `token`）；缺失时回退 `X-User-Id: <user_id>`（开发期便利，生产须关闭）。
  运营接口额外要求角色为 `ADMIN`（`RequireAdmin`），越权返回 403 `FORBIDDEN`。
  注意：当前 token 是自签 HMAC-SHA256 的紧凑格式，并非标准 JWT（无 header 段），仅课程范围内可用。

## 1. 商品目录（M6，公开）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/products` | 商品列表；`category`、`keyword`、`onlyInStock=true` 可选 |
| GET | `/api/v1/products/{id}` | 商品详情（含 SKU 列表） |

返回字段：`id, name, description, category, imageUrl, pointsPrice（最低 SKU 价）, stock（SKU 库存和）, active, limitPerUser, skus[]`。

## 2. 积分查询（M5，需身份）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/points/summary` | 余额 / 冻结 / 可用 / 累计减碳 / 本月获得 / 今日获得 / 日上限（`earnedToday` 已排除退款） |
| GET | `/api/v1/points/transactions` | 流水；`type` 可逗号多选，支持 `TRAVEL/REDEEM/REFUND/ADJUST/EXPIRE` 及前端别名 `EARN/SPEND/REVERSE` |
| GET | `/api/v1/points/daily-carbon?days=7` | 最近 N 天逐日碳减排量（缺失日期补 0），供前端趋势图 |

每日碳减排返回：`[{"date":"2026-11-24","grams":483.2}, ...]`（`days` 默认 7，上限 90）。

流水字段：`id, type, amount（正得负耗）, balanceAfter, reason, refType, refId, createdAt`。

## 3. 兑换订单（M7，需身份）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/v1/orders` | 兑换下单；请求头可选 `Idempotency-Key`（BR-15） |
| GET | `/api/v1/orders` | 我的订单；`status` 可选 |
| POST | `/api/v1/orders/{id}/cancel` | 取消订单（仅 `PENDING`），body `{"reason":"..."}` |

下单请求：`{"productId": 1001, "receiver": "本人"}`（`skuId` 可选，缺省由后端挑第一个有库存的 SKU）。

事务步骤（与 `docs/uml/10-序列图-兑换下单.puml` 一致）：
1. 幂等键命中 → 返回原订单；
2. `point_account` 行锁 → `product_sku` 行锁（固定顺序防死锁）；
3. 锁后复核：商品在线 / 库存 > 0 / `balance - frozen ≥ 价格` / 未超 `limit_per_user`；
4. 写 `redeem_order` + `point_txn(REDEEM)` + 扣余额 + 扣库存 + 按 FIFO 消耗 `point_batch`；
5. 写 `event_outbox(OrderPaid)`，提交。

批次一致性（BR-11）：
- 扣减严格按 `expires_at, id` 升序（真 FIFO），每笔消耗写入 `point_batch_txn`（`amount` 为负）；
- 取消订单时按 `point_batch_txn` 原路回填到**原批次**（`expires_at` 不变，`amount` 为正）；
  找不到消费明细（历史数据）才按 `points_expire_days` 新建批次；
- 恒等式：`Σ(point_account.balance) − Σ(point_batch.remaining WHERE status='ACTIVE') = 0`
  （行程发分侧由 `repo.GrantPointsWithBatch` 写批次，P2 的 `/trips` 需接这一行）。

取消订单（BR-12）：补 `point_txn(REFUND)`、退余额、回补库存、生成退款批次（重新计有效期）、订单置 `CANCELLED`、写 `event_outbox(OrderCancelled)`。

## 4. 业务错误码

| HTTP | code | 场景 |
| --- | --- | --- |
| 400 | `INVALID_ARGUMENT` | 参数非法 / 请求体不是合法 JSON |
| 401 | `UNAUTHORIZED` | 缺少 `X-User-Id`（或未来 JWT 失效） |
| 404 | `NOT_FOUND` | 商品 / 订单不存在或已下架 |
| 409 | `INSUFFICIENT_POINTS` | 积分不足 |
| 409 | `OUT_OF_STOCK` | 库存不足 |
| 409 | `LIMIT_EXCEEDED` | 超出单用户限兑数量 |
| 409 | `ILLEGAL_TRANSITION` | 订单当前状态不允许该操作 |
| 500 | `INTERNAL` | 未预期错误（细节只进日志，不外泄） |

## 5. 本地联调

```powershell
# 1) 起库并导入表结构（已建库前提下，按文件名顺序执行迁移）
Get-ChildItem db/migrations/*.up.sql | Sort-Object Name | ForEach-Object {
  Get-Content $_ -Raw | mysql --user=greentravel --host=127.0.0.1 --default-character-set=utf8mb4 green_travel_points
}

# 2) 导入演示数据（user_id = 1001，1200 积分，3 个商品）
$env:MYSQL_PWD="greentravel"
cmd /c "mysql --user=greentravel --host=127.0.0.1 --default-character-set=utf8mb4 green_travel_points < db/seed/dev_demo.sql"

# 3) 起后端
cd apps/api; go run ./cmd/server

# 4) 冒烟
curl http://127.0.0.1:8080/api/v1/products
curl -H "X-User-Id: 1001" http://127.0.0.1:8080/api/v1/points/summary
```

> ⚠️ Windows 下用 curl 传中文 JSON 时，命令行参数可能被转成 GBK 导致入库乱码；
> 中文请求体建议写入 UTF-8 文件后 `--data-binary @file.json`。

## 6. 运营后台（M8，需 ADMIN 身份）

开发期要求 `X-User-Id` 对应用户的 `role = 'ADMIN'`（种子数据里 1002 是管理员），否则返回 403。

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/admin/products` | 全部商品（含 DRAFT/OFFLINE）；`status`、`keyword` 可选 |
| POST | `/api/v1/admin/products` | 新建商品 + SKU（默认 DRAFT） |
| PATCH | `/api/v1/admin/products/{id}` | 更新商品（含上下架：`status`） |
| PUT | `/api/v1/admin/skus/{id}/stock` | 设置 SKU 库存（绝对值） |
| GET | `/api/v1/admin/orders` | 全部订单；`status` 可选 |
| POST | `/api/v1/admin/orders/{id}/confirm` | `PENDING → PAID`（商户确认） |
| POST | `/api/v1/admin/orders/{id}/ship` | `PAID → SHIPPED`（发货） |
| POST | `/api/v1/admin/orders/{id}/complete` | `SHIPPED → COMPLETED`（核销完成） |
| GET | `/api/v1/admin/points/reconcile` | 全量积分对账：`balance == Σ(txn)`，返回不一致账户（BR-07 证据） |

新建商品示例：

```json
{
  "merchantId": 1001,
  "name": "环保水杯",
  "description": "可重复使用",
  "category": "文创",
  "status": "DRAFT",
  "sortOrder": 40,
  "limitPerUser": 1,
  "skus": [{"name": "标准款", "pointsPrice": 120, "stock": 20}]
}
```

订单流转非法时返回 `409 ILLEGAL_TRANSITION`；每次流转都会写 `event_outbox`
（`OrderConfirmed` / `OrderShipped` / `OrderCompleted`），交给 Worker 消费。

## 7. 测试

```powershell
cd apps/api
go test ./... -count=1        # 领域规则 + 状态机 + HTTP 处理层（假存储，不依赖 MySQL）
go vet ./...
go build ./...
```
