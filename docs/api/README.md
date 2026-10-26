# P3 接口说明（M5 / M6 / M7）

> 实现：`apps/api/internal/httpserver`（HTTP 层）+ `apps/api/internal/repo`（SQL 层）+ `apps/api/internal/app`（DTO 与端口）。
> 对应报告：实验三"系统原型构建与实现"、实验二"设计类清单 / 序列图"。
> 状态：✅ 已实现并通过单元测试 + 本地 MySQL 联调；⏳ 待 P2 的 JWT 认证接入后替换开发期身份头。

## 0. 约定

- 统一响应壳：`{"code":"OK","message":"ok","data":...}`；错误时 `data` 省略。
- 分页参数：`page`（从 1 起，默认 1）、`pageSize`（默认 20，上限 100）；响应 `{items, page, pageSize, total}`。
- 时间：RFC3339（带时区）。
- 金额/积分：整数 `int64`。
- 身份（**开发期临时方案**）：请求头 `X-User-Id: <user_id>`。
  上线前由 P2 的 `POST /api/v1/auth/login` 签发 JWT，把 `RequireUser` 中间件换成解析 JWT 即可，Handler 不用改。

## 1. 商品目录（M6，公开）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/products` | 商品列表；`category`、`keyword`、`onlyInStock=true` 可选 |
| GET | `/api/v1/products/{id}` | 商品详情（含 SKU 列表） |

返回字段：`id, name, description, category, imageUrl, pointsPrice（最低 SKU 价）, stock（SKU 库存和）, active, limitPerUser, skus[]`。

## 2. 积分查询（M5，需身份）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| GET | `/api/v1/points/summary` | 余额 / 冻结 / 可用 / 累计减碳 / 本月获得 / 今日获得 / 日上限 |
| GET | `/api/v1/points/transactions` | 流水；`type` 可逗号多选，支持 `TRAVEL/REDEEM/REFUND/ADJUST/EXPIRE` 及前端别名 `EARN/SPEND/REVERSE` |

流水字段：`id, type, amount（正得负耗）, balanceAfter, reason, refType, refId, createdAt`。

## 3. 兑换订单（M7，需身份）

| 方法 | 路径 | 说明 |
| --- | --- | --- |
| POST | `/api/v1/orders` | 兑换下单；请求头可选 `Idempotency-Key`（BR-15） |
| GET | `/api/v1/orders` | 我的订单；`status` 可选 |
| POST | `/api/v1/orders/{id}/cancel` | 取消订单（仅 `PENDING`），body `{"reason":"..."}` |

下单请求：`{"productId": 1001, "receiver": "本人"}`（`skuId` 可选，缺省由后端挑第一个有库存的 SKU）。

事务步骤（与 `docs/uml/09-序列图-兑换下单.puml` 一致）：
1. 幂等键命中 → 返回原订单；
2. `point_account` 行锁 → `product_sku` 行锁（固定顺序防死锁）；
3. 锁后复核：商品在线 / 库存 > 0 / `balance - frozen ≥ 价格` / 未超 `limit_per_user`；
4. 写 `redeem_order` + `point_txn(REDEEM)` + 扣余额 + 扣库存 + 按 FIFO 消耗 `point_batch`；
5. 写 `event_outbox(OrderPaid)`，提交。

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

## 6. 测试

```powershell
cd apps/api
go test ./... -count=1        # 领域规则 + 状态机 + HTTP 处理层（假存储，不依赖 MySQL）
go vet ./...
go build ./...
```
