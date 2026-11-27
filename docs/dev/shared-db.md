# 团队共享数据库（ZeroTier 内网）

> 用途：全组共用一个 MySQL 实例，避免各自本地数据不一致。
> **共享库主机是 `DESKTOP-IGAKB82`**，不是「每台机器自己」：ZeroTier 网络 `743993800f43676f`，
> 主机 ZeroTier 内网 IP `10.213.180.114`，MySQL 9.7.2（Windows 服务 `MySQL97`，数据目录 `D:\coding\mysql\Data\`）。
> 宿主机上 `127.0.0.1` 与共享 DSN 是同一个实例；**其余成员机一律连 `10.213.180.114`**。
> 判断自己是不是主机：`hostname` 为 `DESKTOP-IGAKB82` 才是；只看 ZeroTier 网络号不够，
> 成员机（如 `MSI` / `10.213.180.75`）也在同一个网络号里。
> 成员机本地 3306 上可能另有自己的 MySQL（本仓库开发机上实测是 `MySQL80` 8.0.46），
> 里面没有 `greentravel` 账号、也不是共享库，直连会 `ERROR 1045 (28000)`。
> 早期本节把组员 B 的 `19c56b20f1 / 10.213.180.43` 误记成「本机」，那是组员设备，别混；
> 主机地址以 `ipconfig` 里 `ZeroTier One [743993800f43676f]` 的实际值为准，成员对照见 `docs/数据库联通记录.md`。

## 1. 连接信息

| 项 | 值 |
| --- | --- |
| 主机 | `10.213.180.114` |
| 端口 | `3306` |
| 数据库 | `green_travel_points` |
| 用户 | `greentravel` |
| 密码 | `greentravel` |
| MySQL 版本 | 9.7.2 |

**怎么确认自己连的是共享实例**：别人（非主机）执行下面一条，应拿到与主机一致的结果；
`hostname` 或 `datadir` 对不上，就是连到了自己机器上的另一个库。

```sql
SELECT @@hostname, @@port, @@version, @@datadir;
-- DESKTOP-IGAKB82 | 3306 | 9.7.2 | D:\coding\mysql\Data\
```

后端启动日志也会播报目标（判据见 `apps/api/cmd/server/dbendpoint.go`）：地址不是 loopback 时打印
`数据库目标：远程共享实例，你的读写对其他成员实时可见  db_addr=<地址>`。

## 2. 后端连接串（DB_DSN）

```powershell
$env:DB_DSN='greentravel:greentravel@tcp(10.213.180.114:3306)/green_travel_points?parseTime=true&loc=Local&charset=utf8mb4&timeout=30s&readTimeout=30s&writeTimeout=30s'
$env:HTTP_REQUEST_TIMEOUT='60s'
cd apps\api
go run ./cmd/server
```

验证：

```powershell
curl http://127.0.0.1:8080/readyz
```

> **延迟说明（实测）**：共享库走 ZeroTier 中转，链路延迟较高——单次探针约 0.7s，
> 登录/积分查询 1.5~6s，**上报一次行程（多步事务）约 20~25s**。
> 因此必须把 `HTTP_REQUEST_TIMEOUT` 调到 `60s`，否则上报行程会 504；
> `DB_DSN` 里也建议带上 `timeout/readTimeout/writeTimeout`。
> 组员：日常可用自己机器的本地库，联调/演示再切共享库。
> 主机：`127.0.0.1` 与共享 DSN 是**同一个实例**，切换只影响延迟，不改变数据。
>
> **冷启动 503 是可预期的**：`/readyz` 的探针超时写死在 `apps/api/internal/httpserver/health.go:29`（2s），
> 而共享库冷连接要 2–3s。后端刚起时连续几次 `/readyz` 都会 503，**光重试探针把连接池捂不热**：
> 先打一次真实接口（`curl http://127.0.0.1:8080/api/v1/products`，实测约 3.0s），再探即 200（实测 0.59s）。

## 3. 演示数据

共享库的演示账号以库中现有数据为准，**不要重跑 `db/seed/dev_demo.sql`**：

| 数据 | 说明 |
| --- | --- |
| 用户 `1` | 演示市民，手机号 13800000001；余额以库中为准，旧占位哈希按 seed 头部说明做定向 UPDATE |
| 用户 `1002` | 运营管理员（`role=ADMIN`），用于后台接口 |
| 商品 `1001–1003` | 环保帆布袋 100 分 / 咖啡券 300 分 / 骑行头盔 800 分 |
| 积分批次 / 流水 | 以库中现有数据为准 |

后台接口测试头：`X-User-Id: 1002`；普通用户头：`X-User-Id: 1`。仅 `APP_ENV=dev` 且 `AUTH_ALLOW_DEV_HEADER=true` 时生效，否则 401。

## 4. 相关命令

```powershell
# 查表
$env:MYSQL_PWD="greentravel"
mysql --user=greentravel --host=10.213.180.114 --port=3306 --default-character-set=utf8mb4 green_travel_points -e "SHOW TABLES;"

# 共享库禁止重跑 seed；演示账号口令按 db/seed/dev_demo.sql 头部说明做定向 UPDATE
```

## 5. 注意事项

- 共享库**不要执行 `down` 迁移或清表**，会影响全组；建表/改表先确认。
- 每台电脑都要装 ZeroTier 并加入 `743993800f43676f`，且需要网络管理员在 ZeroTier Central 授权后才拿到 `10.213.180.x` 地址。
- **`127.0.0.1:3306` 和 `10.213.180.114:3306` 是同一个实例**（同一台机器的同一个 MySQL），
  不存在「本地备用库」那第二份。对着 `127.0.0.1` 跑 `down` 迁移、清表、批量造数据，等同于直接改全组的库；
  本机可以用 `127.0.0.1` 提速，但写的仍是这一份共享库。
- 共享库主机（本机）关机、睡眠或退出 ZeroTier，全组立刻连不上库；这是单人单机承担的服务，没有冗余。

## 6. 迁移同步约定（改了表结构怎么让全组跟上）

> 关键前提：**共享库里的表结构只有一份**。谁执行了 DDL，所有人立刻生效，
> 不存在「发给组员同步」这一步；`git pull` 拿到迁移文件是为了**留档和让本地库能复现**。

### 6.1 三条铁律

1. **只新增，不改历史**。新结构建下一个未执行序号（当前仓库已有 `000001`–`000011`），
   已执行的历史迁移一个字都不再动——别人库里那几组已经跑过，改了不会重跑，
   只会让两边悄悄不一致。
2. **共享库的 DDL 同一时间只由一个人执行**。各自同时 `ALTER` 会撞表锁或互相覆盖；
   执行前在群里说一句「我现在改共享库」，改完再说「好了」。
3. **执行者负责贴证据**。执行完把 `SHOW TABLES` / `DESCRIBE <新表>` 的输出发群里，
   并在 §6.4 登记一行。没有输出就等于没改。

### 6.2 改库的人：五步

```powershell
cd D:\coding\green-travel-points
$env:MYSQL_PWD="greentravel"
$mysql     = "mysql"       # 已加入 PATH 就用它；否则写全路径：
$mysqldump = "mysqldump"   # "C:\Program Files\MySQL\MySQL Server 9.7\bin\mysqldump.exe"

# 0) 业务代码要用的新表 / 新列，写成成对迁移（up 必须可重复执行）

# 1) 改结构前先备份共享库（库里已有真实账号和订单，误删回不来）
New-Item -ItemType Directory -Force .cache | Out-Null
& $mysqldump -h 10.213.180.114 -u greentravel --single-transaction --default-character-set=utf8mb4 green_travel_points > ".cache\backup-$(Get-Date -Format yyyyMMdd-HHmm).sql"

# 2) 执行迁移（可以全跑：现有 up 都是 IF NOT EXISTS / 幂等写法，
#    新增迁移也必须遵守这条，否则只能单跑缺的那个）
Get-ChildItem db\migrations\*.up.sql | Sort-Object Name | ForEach-Object {
  Get-Content $_ -Raw | & $mysql -h 10.213.180.114 -u greentravel --default-character-set=utf8mb4 green_travel_points
  if ($LASTEXITCODE -eq 0) { "OK   $($_.Name)" } else { "FAIL $($_.Name)" }
}

# 3) 验证并把输出贴到群里
& $mysql -h 10.213.180.114 -u greentravel --default-character-set=utf8mb4 -e "SHOW TABLES; DESCRIBE <新表名>;" green_travel_points
```

第 4、5 步：`git commit -m "feat(db): 新增 xxx 表"` 并 push，然后群里发一句话——
「共享库已执行 `000006`，连共享库的**重启后端**即可，不用迁移」。

### 6.3 其他人：看情况，最多两步

```powershell
git pull --rebase          # 拿迁移文件（留档 / 本地库复现用）
# 只有「用本地库」的人需要继续：
#   从 db\migrations\000006_*.up.sql 起按文件名顺序跑到最新（当前含 000011）
# 最后重启后端（连共享库的人只做这一步）
```

- 连**共享库**：表已经在共享库里了，**不用执行任何迁移**，重启后端即可。
- 用**本地库**：按序号补执行比自己库里新的迁移（`000006` 起），再重启后端。
- 拉完代码报 `Error 1146: Table ... doesn't exist`：说明后端连的是本地库而本地没跑新迁移，
  要么补跑迁移，要么把 `DB_DSN` 切到共享库。

### 6.4 执行登记表

每次在共享库改完结构，在下面加一行，别人排查时能直接对时间线。

| 日期 | 迁移 | 执行人 | 目标库 | 结果证据 |
| --- | --- | --- | --- | --- |
| 2026-11-17 | `000003`–`000005` | 李响（本机） | 本机 127.0.0.1 | `SHOW TABLES` 13 张，`/readyz` 200 |
| TODO | `000006`–`000011` | 待执行者确认 | 共享库 | 本表尚无登记；需执行者补 `SHOW TABLES` / `DESCRIBE` 证据 |

### 6.5 三个常见坑

| 坑 | 说明 |
| --- | --- |
| Docker 的 initdb 只对空数据卷生效 | `deploy/docker-compose.yml` 把迁移挂到 `/docker-entrypoint-initdb.d`，但 `./mysql-data` 已有数据时**不会重跑**；新增迁移得手动执行。 |
| 没有 `schema_migrations` 表 | 没人知道别人跑到第几版，只能靠「up 幂等 + 全跑」兜底。这是技术债，不是设计。 |
| `down` 只回滚结构，不回滚数据 | 删表删列后数据回不来，所以 §6.2 第 1 步的备份不能省；共享库上不要随手 `down`（另见 §5）。 |

### 6.6 延迟实测（2026-11-24，跨 ZeroTier）

共享库在 `DESKTOP-IGAKB82`（`10.213.180.114:3306`，MySQL 9.7.2），本机 `MSI` 经 ZeroTier 访问，ping 426–541 ms。用后端 `:18080` 实测：

| 接口 | 单次耗时 | 说明 |
| --- | --- | --- |
| `GET /api/v1/products` | 1.7 s | 冷启动 3.0 s，首条请求会把连接池捂热 |
| `GET /api/v1/trips` | 1.1 s | |
| `GET /api/v1/points/summary` | 3.5 s | |
| `GET /api/v1/points/transactions` | 4.6 s | |
| `POST /api/v1/trips` | 12–20 s | 一个事务里至少 9 条 SQL（travel_mode 查询 / 账户行锁 / 当日流水汇总 / 行程 / 碳记录 / 积分流水 / 批次 / 账户更新 / Outbox），每条都要一次跨网往返 |
| `POST /api/v1/orders` | 12 s | 同理，行锁 + 账本写入 |

两个直接后果：

1. **前端 E2E 的超时必须放宽**：`apps/web/playwright.config.ts` 里 `timeout: 300_000`、`expect.timeout` 与 `actionTimeout` 都是 `120_000`，压到 5–10 s 会稳定假失败（第一次跑就是被 30 s 的 URL 断言卡住的）。
2. **`/readyz` 冷启动会 503**：`apps/api/internal/httpserver/health.go` 里探活超时写死 2 s，而冷连接要 2–3 s。想让就绪检查变绿，先打一次真实接口把连接池热起来，或者等代码把超时调大。
3. **写请求会比服务器的写超时先撞线**：`apps/api/cmd/server/main.go` 原本把 `WriteTimeout` 写死 30 s，比 `HTTP_REQUEST_TIMEOUT` 还早到期，于是 30 s 左右的兑换/取消会以 502 出现在浏览器侧（Vite dev 代理把被掐断的连接报成 502），而后端日志里是一条 `status=200 duration_ms≈30000`。现在 `WriteTimeout = serverWriteTimeout(cfg.HTTPRequestTimeout)`（请求超时 + 15 s）；对着共享库跑时 `HTTP_REQUEST_TIMEOUT=60s` 是底线，写入仍接近上限再抬到 `90s`。
4. **运营后台 E2E 的种子账号只在本机开发库有效**：共享库里 `13800000002`（ADMIN）的口令哈希与本机不同，`apps/web/e2e/admin-console.e2e.ts` 已改成开跑前探一次登录、失败就跳过并给出原因；要真跑，用 `E2E_ADMIN_PHONE` / `E2E_ADMIN_PASSWORD` / `E2E_CITIZEN_PHONE` / `E2E_CITIZEN_PASSWORD` 覆盖。

