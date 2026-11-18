# 团队共享数据库（ZeroTier 内网）

> 用途：全组共用一个 MySQL 实例，避免各自本地数据不一致。
> **本机（`DESKTOP-IGAKB82`）就是共享库主机**：ZeroTier 网络 `743993800f43676f`，Node ID `0bab0cb69b`，
> 内网 IP `10.213.180.114`；本机 MySQL 9.7.2（Windows 服务 `MySQL97`，数据目录 `D:\coding\mysql\Data\`）
> 即为全组共用的那一个实例。
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
> 首次请求可能返回 503「数据库不可用」——冷连接超过探针 2s 超时，再请求一次即为 200。

## 3. 演示数据

已导入 `db/seed/dev_demo.sql`：

| 数据 | 说明 |
| --- | --- |
| 用户 `1` | 演示市民，手机号 13800000001，积分 1200 |
| 用户 `1002` | 运营管理员（`role=ADMIN`），用于后台接口 |
| 商品 `1001–1003` | 环保帆布袋 100 分 / 咖啡券 300 分 / 骑行头盔 800 分 |
| 积分批次 / 流水 | 与账户 1200 分对齐 |

后台接口测试头：`X-User-Id: 1002`；普通用户头：`X-User-Id: 1`。

## 4. 相关命令

```powershell
# 查表
$env:MYSQL_PWD="greentravel"
mysql --user=greentravel --host=10.213.180.114 --port=3306 --default-character-set=utf8mb4 green_travel_points -e "SHOW TABLES;"

# 重新导入演示数据（幂等，不会重复插入）
mysql --user=greentravel --host=10.213.180.114 --port=3306 --default-character-set=utf8mb4 green_travel_points < db/seed/dev_demo.sql
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

1. **只新增，不改历史**。新结构一律建 `000006_xxx.up.sql` + `000006_xxx.down.sql`，
   `000001`–`000005` 一个字都不再动——别人库里那几组已经跑过，改了不会重跑，
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
#   Get-ChildItem db\migrations\00000[6-9]*.up.sql | Sort-Object Name | ForEach-Object { ...同上... }
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

### 6.5 三个常见坑

| 坑 | 说明 |
| --- | --- |
| Docker 的 initdb 只对空数据卷生效 | `deploy/docker-compose.yml` 把迁移挂到 `/docker-entrypoint-initdb.d`，但 `./mysql-data` 已有数据时**不会重跑**；新增迁移得手动执行。 |
| 没有 `schema_migrations` 表 | 没人知道别人跑到第几版，只能靠「up 幂等 + 全跑」兜底。这是技术债，不是设计。 |
| `down` 只回滚结构，不回滚数据 | 删表删列后数据回不来，所以 §6.2 第 1 步的备份不能省；共享库上不要随手 `down`（另见 §5）。 |

