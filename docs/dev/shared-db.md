# 团队共享数据库（ZeroTier 内网）

> 用途：全组共用一个 MySQL 实例，避免各自本地数据不一致。
> 本机已加入 ZeroTier 网络 `743993800f43676f`，设备地址 `19c56b20f1`，内网 IP `10.213.180.43`。

## 1. 连接信息

| 项 | 值 |
| --- | --- |
| 主机 | `10.213.180.114` |
| 端口 | `3306` |
| 数据库 | `green_travel_points` |
| 用户 | `greentravel` |
| 密码 | `greentravel` |
| MySQL 版本 | 9.7.2 |

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
> 日常开发建议用本地库，联调/演示再切共享库。
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
- 本机 `127.0.0.1:3306` 还有一份本地 MySQL，仅作离线备用；跑联调请用上面的共享 DSN。
