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
0. 事务外先 `INSERT IGNORE INTO point_account` 建空账户——事务内先取共享锁再升排他锁会与并发的同名 INSERT 互等（MySQL 1213），见 `apps/api/internal/repo/orders.go` 注释；
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

取消订单（BR-12）：补 `point_txn(REFUND)`、退余额、回补库存、把 `point_batch_txn` 里消耗过的量原路回填原批次、订单置 `CANCELLED`、写 `event_outbox(OrderCancelled)`。

### 3.1 订单状态语义（前端文案口径，别写"待支付"）

状态机 `PENDING → PAID → SHIPPED → COMPLETED`，任意非终态可 `CANCELLED`（BR-10，见 `internal/domain/order.go`）：

| status | 含义 | 积分 | 谁能推进 |
| --- | --- | --- | --- |
| `PENDING` | **待商户确认**（下单即已扣分、已占库存） | 已扣 | 运营 `POST /admin/orders/{id}/confirm` |
| `PAID` | 已确认、待发货 | 已扣 | 运营 `POST /admin/orders/{id}/ship` |
| `SHIPPED` | 已发货、待核销 | 已扣 | 运营 `POST /admin/orders/{id}/complete` |
| `COMPLETED` | 已核销完成 | 已扣 | 终态 |
| `CANCELLED` | 已取消（用户或运营） | 已退还 | 终态 |

要点：积分兑换**没有支付环节**——下单事务里已经扣分 + 占库存，所以 `PENDING` 的正确文案是
「待商户确认 / 待发货」，不是「待支付」；用户端只有「取消」（且仅 `PENDING` 可取消），
状态推进全部在运营侧。演示时用运营身份（种子 `1002` 是 ADMIN）推进：

```powershell
curl -X POST -H "X-User-Id: 1002" http://127.0.0.1:8080/api/v1/admin/orders/1/confirm
curl -X POST -H "X-User-Id: 1002" http://127.0.0.1:8080/api/v1/admin/orders/1/ship
curl -X POST -H "X-User-Id: 1002" http://127.0.0.1:8080/api/v1/admin/orders/1/complete
```

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

# 2) 导入演示数据（1001 演示用户 1200 积分 / 1002 运营管理员 / 3 个商品）
$env:MYSQL_PWD="greentravel"
cmd /c "mysql --user=greentravel --host=127.0.0.1 --default-character-set=utf8mb4 green_travel_points < db/seed/dev_demo.sql"

# 3) 起后端
cd apps/api; go run ./cmd/server

# 4) 冒烟
curl http://127.0.0.1:8080/api/v1/products
curl -H "X-User-Id: 1001" http://127.0.0.1:8080/api/v1/points/summary
```

种子账号可直接登录，不需要再靠 `X-User-Id` 回退（口令与 `apps/web/e2e` 的默认值一致）：

| 账号 | 手机号 | 口令 | 角色 | 用途 |
| --- | --- | --- | --- | --- |
| 演示用户 | `13800000001` | `demo123456` | CITIZEN | 用户端：上报 / 兑换 / 取消 |
| 运营管理员 | `13800000002` | `admin123456` | ADMIN | 运营后台 `/admin`：确认 → 发货 → 核销 |

```powershell
# 登录并拿 token（前端就是这么做的）
curl -X POST -H "Content-Type: application/json" `
  --data-binary '{"account":"13800000002","password":"admin123456"}' `
  http://127.0.0.1:8080/api/v1/auth/login
```

### 5.1 种子的重跑语义（重要）

`db/seed/dev_demo.sql` 现在是**只补缺失行、绝不覆盖已有数据**：

- 账本/库存相关表（`point_account`、`point_txn`、`point_batch`、`product`、`product_sku`、`merchant`）
  一律 `ON DUPLICATE KEY UPDATE id = id`（no-op）——重跑不会改账户归属、不会把已消费的批次余量或库存冲回原值、
  不会把运营手动下架的商品强制改回 `ONLINE`；
- 唯一例外是两个演示账号的 `password_hash / role / nickname`：这是「让演示账号能登录」的明确意图，
  重跑会把口令修回 `demo123456` / `admin123456`，不碰账本。

要**恢复演示基线**（余额 1200、批次 1200、库存 50/30/5），正确做法是 `DROP DATABASE` → 按序执行
`db/migrations/*.up.sql` → 再导入本文件；**不要**靠重跑 seed 去"覆盖"。

> ⚠️ **共享库（`docs/dev/shared-db.md`，10.213.180.114）禁止执行 seed**：那边的用户 ID 与种子不一致
> （「演示用户」是 `id=1` 而不是 `1001`），重跑会插入一整套重复的 `1001` 系列行。
> 共享库要修口令等单点问题，只做定向 UPDATE：
> ```sql
> -- 让共享库的 13800000002 能登 /admin（口令改为 admin123456，与种子/前端 E2E 一致）
> UPDATE user SET password_hash =
>   'sha256$f6b61c75d1b8eb0d6939a5c363b03f12$ae2a6d8dd0f42821f2791d813f4fc5fa5c5ce9d4ee05658433d9669b3fe783d3'
>  WHERE id = 1002;
> -- 若共享库的「演示用户」也要能登录（id=1，口令 demo123456），同样只改这一列
> UPDATE user SET password_hash =
>   'sha256$726211e994a96a50be61dd7bd08aedee$e5b7d2e7e88ce334803ac33029c83cb451059ee6207f55309ecce7d0e8ca80d4'
>  WHERE id = 1;
> ```

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
| POST | `/api/v1/admin/points/expire` | 手动触发积分过期清算：到期批次置 `EXPIRED`、写 `EXPIRE` 流水、扣余额 |
| POST | `/api/v1/admin/points/adjust` | 人工调增 / 调减积分（UC-15），必填理由，变动写 `audit_log` |
| GET | `/api/v1/admin/audit-logs` | 审计日志查询（UC-19）：按操作人 / 动作 / 对象过滤 |
| GET | `/api/v1/admin/config` | 系统参数列表（UC-14）：键 / 值 / 值类型 / 说明 |
| PUT | `/api/v1/admin/config` | 批量修改系统参数，整批原子生效；任一项非法则全部回滚 |
| PUT | `/api/v1/admin/travel-modes/{code}` | 更新出行方式名称 / 排放因子 / 启用状态 |

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

> `GET /api/v1/admin/products` 会随每个商品返回 `skus` 明细（`id / name / pointsPrice / stock / soldCount`），
> 后台「改库存」直接用 `skus[].id` 调 `PUT /api/v1/admin/skus/{id}/stock`；
> `pointsPrice` / `stock` 仍是聚合值（最低价 / 总库存），不要拿它当 SKU 库存。

### 6.1 参数配置（UC-14）

业务规则参数在运行期全部来自数据库，运营改完**立即生效、无需发版**：
行程上报每次读 `sys_config` 与 `travel_mode`（`httpserver/trips.go`），积分概览的
`dailyCap` 与下单时的批次有效期也分别读 `daily_points_cap` / `points_expire_days`。

`GET /api/v1/admin/config` 返回 `sys_config` 全量条目（值一律是字符串，按 `valueType` 解析）：

```json
[
  {"key": "carbon_per_point", "value": "100", "valueType": "int", "description": "多少克碳减排量兑换 1 积分"},
  {"key": "daily_points_cap", "value": "500", "valueType": "int", "description": "单日可获得积分上限，0 表示不限制"},
  {"key": "trip_min_distance_km", "value": "0.1", "valueType": "decimal", "description": "单次行程最短距离（km）"},
  {"key": "trip_max_distance_km", "value": "100", "valueType": "decimal", "description": "单次行程最长距离（km）"},
  {"key": "points_expire_days", "value": "365", "valueType": "int", "description": "积分有效期（天）"}
]
```

`PUT /api/v1/admin/config` 批量提交，成功后返回**更新后的全量参数**：

```json
{"items": [{"key": "daily_points_cap", "value": "10"}]}
```

校验规则（任一条不满足 → `400 INVALID_ARGUMENT`，整批不落库）：

| 规则 | 说明 |
| --- | --- |
| 键必须已存在 | 只允许改 `sys_config` 里已有的参数；未知键拒绝，避免"改了但代码没消费" |
| 类型匹配 | `int` 需为整数；`decimal` 需为有限小数；`bool` 取 `true/false/1/0` |
| 语义约束 | `carbon_per_point > 0`（除零）、`points_expire_days > 0`（批次立即过期）、`trip_min_distance_km > 0`、`trip_max_distance_km > 0`、`daily_points_cap ≥ 0`（0 = 不限制） |
| 距离区间 | 提交后复核 `trip_min_distance_km < trip_max_distance_km`，反了会让所有行程被拒 |
| 原子性 | 整批在一个事务内完成，后续项非法时前面的改动一并回滚 |

`PUT /api/v1/admin/travel-modes/{code}` 改一种出行方式（`nil` 字段不动），成功返回该方式最新值：

```json
{"efActual": 50, "enabled": true}
```

返回：

```json
{"code": "bus", "name": "公交", "efBase": 192, "efActual": 50, "enabled": true}
```

约束：至少提交一个字段；`name` 非空；`efBase > 0`；`efActual ≥ 0`；提交后复核
`efActual < efBase`（否则该方式永远算不出碳减排量）。`enabled = false` 后该方式上报行程会返回
`400 INVALID_ARGUMENT`（未知或已停用的出行方式）。方式不存在返回 `404 NOT_FOUND`。

验收示例（P3-2）：把 `daily_points_cap` 改成 `10` 后上报 10 km 公交
（碳减排 `10 × (192 − 68) = 1240 g`，按 100 g/分应得 12 分），实际 `grantedPoints = 10`。

### 6.2 积分过期清算（P3-3）

`POST /api/v1/admin/points/expire` 手动触发（没有常驻定时任务，演示与运维都用它），返回：

```json
{"batches": 1, "points": 1200, "accounts": 1}
```

- `batches`：被置为 `EXPIRED` 的批次数；`points`：实际过期扣掉的积分；`accounts`：余额发生变化的账户数。

处理规则：

| 项 | 规则 |
| --- | --- |
| 选取范围 | `point_batch.status='ACTIVE' AND remaining>0 AND expires_at <= NOW()`，按 `expires_at, id` 升序（与 FIFO 口径一致） |
| 写流水 | `point_txn(txn_type='EXPIRE', amount<0, balance_after=扣除后余额)`，`biz_ref_type='point_batch'`、`biz_ref_id=batch_id` |
| 幂等 | `uk_point_txn_biz(biz_ref_type, biz_ref_id, txn_type)` 保证每批次最多一条 `EXPIRE` 流水；重复调用第二次返回全 0 |
| 批次 | 置 `status='EXPIRED'`、`remaining=0`；同时在 `point_batch_txn` 记一笔负明细，避免取消订单时把已过期的量回填成可用积分 |
| 余额 | `point_account.balance` 扣减并 `version+1`，全程单事务；锁顺序 `account → batch`（与下单一致，避免死锁） |
| 账实不符 | 若历史数据里余额小于批次余量，只扣得动的部分（`min(remaining, balance)`），批次仍归零置 `EXPIRED`，差额由 `/admin/points/reconcile` 暴露 |
| 不变量 | 正常数据下结束后仍满足 `Σ(point_account.balance) − Σ(point_batch.remaining WHERE status='ACTIVE') = 0` |

演示步骤（本地库）：

```powershell
# 1) 把某条批次改成已过期（造数据）
mysql --user=greentravel --password=greentravel --host=127.0.0.1 green_travel_points `
  -e "UPDATE point_batch SET expires_at = NOW(3) - INTERVAL 1 DAY WHERE id = 1001;"

# 2) 触发清算（1002 是 ADMIN）
curl -X POST -H "X-User-Id: 1002" http://127.0.0.1:8080/api/v1/admin/points/expire
# → {"batches":1,"points":1200,"accounts":1}

# 3) 复核：余额归零、批次 EXPIRED、账本仍平
#    GET /api/v1/admin/points/reconcile → {"ok":true,"mismatches":[]}
#    GET /api/v1/points/transactions?type=EXPIRE → 能看到「积分过期」流水
```

> 已知前提：行程发分侧尚未写 `point_batch`（P2 待接 `GrantPointsWithBatch`），所以本地库里
> 「行程发的分」没有批次，过期清算只会动到批次里真实存在的量；批次对账差值不是本接口引入的。

### 6.3 人工调账与审计（UC-15 / UC-19，BR-14 / NFR-12）

**调账** `POST /api/v1/admin/points/adjust`，理由必填（BR-14），返回新余额与审计记录 ID：

```json
{"userId": 1001, "amount": -50, "reason": "刷分扣回"}
```
```json
{"accountId": 1001, "userId": 1001, "balance": 950, "auditId": 3}
```

| 规则 | 说明 |
| --- | --- |
| 入参 | `userId > 0`、`amount ≠ 0`（正增负减）、`reason` 非空且 ≤255 字；缺一项 → `400` |
| 扣减下限 | `balance + amount < 0` → `409 INSUFFICIENT_POINTS`，整笔不落库 |
| 流水 | `point_txn(txn_type='ADJUST', amount=±, biz_ref_type='audit_log', biz_ref_id=auditId)`，积分变动可反查来源（NFR-12） |
| 批次 | 调增同时写 `point_batch`（有效期取 `points_expire_days`）；调减按 FIFO 消耗批次，批次不足的部分只扣余额、差额由对账接口暴露 → 正常数据下仍满足 `Σbalance == Σremaining(ACTIVE)` |
| 审计 | `audit_log(action='ADJUST', operator_id=当前管理员, target_key=userId, amount, before_value, after_value, reason)`，与流水、余额同事务 |
| 幂等 | 不支持幂等键：调账是运营人工动作，重复提交就是两笔——这正是要留审计的原因 |

**配置变更也留审计**（BR-14）：`PUT /admin/config` 每个真正变了的键写一条 `CONFIG_UPDATE`
（前后值 + 键名）；`PUT /admin/travel-modes/{code}` 写一条 `TRAVEL_MODE_UPDATE`
（`name=…;efBase=…;efActual=…;enabled=…`）。值没变不记，避免刷屏。

**审计查询** `GET /api/v1/admin/audit-logs?operatorId=&action=&targetType=&targetKey=&page=&pageSize=`：

```json
{
  "items": [{
    "id": 3, "operatorId": 1002, "action": "ADJUST", "targetType": "point_account",
    "targetId": 1001, "targetKey": "1001", "amount": -50,
    "beforeValue": "1000", "afterValue": "950", "reason": "刷分扣回",
    "createdAt": "2026-11-24T21:52:10.123+08:00"
  }],
  "page": 1, "pageSize": 20, "total": 1
}
```

| 过滤参数 | 说明 |
| --- | --- |
| `operatorId` | 操作人 `user.id` |
| `action` | `ADJUST` / `CONFIG_UPDATE` / `TRAVEL_MODE_UPDATE`（大小写不敏感，内部转大写） |
| `targetType` | `point_account` / `sys_config` / `travel_mode` |
| `targetKey` | 账户按 `userId`、参数按 `config_key`、出行方式按 `code` |

`audit_log` 只追加（无更新/删除接口），保留 ≥180 天（NFR-12）；表结构见
`db/migrations/000007_audit_log.up.sql`。

## 7. 测试

```powershell
cd apps/api
go test ./... -count=1        # 领域规则 + 状态机 + HTTP 处理层（假存储，不依赖 MySQL）
go vet ./...
go build ./...
```

### 7.1 repo 集成测试（真连 MySQL）

`internal/repo/orders_test.go` 覆盖兑换成功 / 积分不足 / 库存不足 / 限购 / 取消退款回填 /
20 并发防超卖 / 对账。**未设置 `REPO_TEST_DSN` 时全部跳过**，所以 CI 与没装库的机器不受影响。

用例会**整表清空**，因此 DSN 指向的库名必须以 `_test` 结尾，防止误洗演示库。

```powershell
# ① 准备专用库（需要 MySQL 管理员；本机 root 口令未知时可临时起一个独立实例，见下）
#    CREATE DATABASE green_travel_points_test DEFAULT CHARACTER SET utf8mb4 COLLATE utf8mb4_0900_ai_ci;
#    GRANT ALL PRIVILEGES ON green_travel_points_test.* TO 'greentravel'@'localhost';
# ② 按文件名顺序执行 db/migrations/*.up.sql 到该库
# ③ 跑测试
$env:REPO_TEST_DSN = 'greentravel:greentravel@tcp(127.0.0.1:3306)/green_travel_points_test?parseTime=true&loc=Local&charset=utf8mb4'
cd apps/api; go test ./internal/repo -count=1 -v
```

拿不到管理员权限时的等价做法：用已装好的 MySQL 二进制起一个**独立数据目录 + 独立端口**的临时实例，
不碰系统实例、不需要 root 口令：

```powershell
$bin = 'C:\Program Files\MySQL\MySQL Server 8.0\bin'
& "$bin\mysqld.exe" --initialize-insecure --datadir=.cache\mysql-test-data --basedir='C:\Program Files\MySQL\MySQL Server 8.0' --console
& "$bin\mysqld.exe" --datadir=.cache\mysql-test-data --basedir='C:\Program Files\MySQL\MySQL Server 8.0' --port=3307 --mysqlx=OFF --console
# 然后 root（空口令）建库建号、跑迁移，DSN 用 127.0.0.1:3307
```
