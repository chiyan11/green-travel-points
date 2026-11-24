-- dev_demo.sql —— 本地联调 / 演示用种子数据（不属于迁移序列，不参与 CI）
--
-- ⚠️ 语义：**只补缺失行，绝不覆盖已有数据**。
--   已有行一律 no-op（`ON DUPLICATE KEY UPDATE id = id`），唯一例外是下方两个演示账号的
--   password_hash / role / nickname——那是"让演示账号能登录"的明确意图，且不碰账本。
--   原因：覆盖式写法会破坏账本与库存，实测踩过：
--     · point_account 曾被 `user_id = VALUES(user_id)` 改指向 → 账户与用户脱钩；
--     · point_batch 曾把 remaining 冲回 1200 → 已消费的余量被"复活"，账实不符；
--     · product_sku 曾把 stock 冲回原值、product 曾把 OFFLINE 强制改回 ONLINE → 冲掉运营操作。
--   要恢复演示基线请 **DROP DATABASE → 迁移 → 本文件**，不要靠重跑本文件"覆盖"。
--
-- ⚠️ 共享库（docs/dev/shared-db.md，10.213.180.114）**禁止**执行本文件：那边的用户 ID 与
--   本文件不一致（例如「演示用户」是 id=1 而不是 1001），重跑会插入一整套重复的 1001 系列行。
--   共享库要修只做定向 UPDATE（见 docs/api/README.md §5）。
--
-- 用法（PowerShell）：
--   $env:MYSQL_PWD="greentravel"
--   cmd /c "mysql --user=greentravel --host=127.0.0.1 --default-character-set=utf8mb4 green_travel_points < db/seed/dev_demo.sql"
--
-- 账号（可直接登录；口令与 apps/web/e2e 的默认值一致，前端 E2E 无需再配环境变量）：
--   1001 13800000001 演示用户 / CITIZEN，口令 demo123456，当前积分 1200
--   1002 13800000002 运营管理员 / ADMIN，口令 admin123456（运营后台 /admin 用它登录）
-- 口令口径：sha256$<hex salt>$<hex digest>，与 httpserver/passwordHash 一致。
-- 商品：3 个上架商品，SKU 价格 100 / 300 / 800。

-- ---------- 演示用户与积分账户 ----------
INSERT INTO user (id, phone, nickname, password_hash, role, status)
VALUES (1001, '13800000001', '演示用户',
        'sha256$726211e994a96a50be61dd7bd08aedee$e5b7d2e7e88ce334803ac33029c83cb451059ee6207f55309ecce7d0e8ca80d4',
        'CITIZEN', 'ACTIVE')
ON DUPLICATE KEY UPDATE nickname = VALUES(nickname), password_hash = VALUES(password_hash);

INSERT INTO point_account (id, user_id, balance, frozen, total_saved_g, version)
VALUES (1001, 1001, 1200, 0, 12000.000, 0)
ON DUPLICATE KEY UPDATE id = id; -- no-op：绝不改已有账户的归属与余额

-- 初始 1200 分对应一笔历史行程奖励（流水只追加，重复执行由主键兜底）
INSERT INTO point_txn (id, account_id, txn_type, amount, balance_after, biz_ref_type, biz_ref_id)
VALUES (1001, 1001, 'TRAVEL', 1200, 1200, 'trip_record', 9001)
ON DUPLICATE KEY UPDATE id = id; -- no-op

INSERT INTO point_batch (id, account_id, grant_txn_id, granted_amount, remaining, expires_at, status)
VALUES (1001, 1001, 1001, 1200, 1200, DATE_ADD(NOW(3), INTERVAL 365 DAY), 'ACTIVE')
ON DUPLICATE KEY UPDATE id = id; -- no-op：绝不把已消费/已过期的 remaining 冲回原值

-- 运营后台管理员（role = ADMIN），用于 M8 后台接口联调 / 运营后台 /admin
INSERT INTO user (id, phone, nickname, password_hash, role, status)
VALUES (1002, '13800000002', '运营管理员',
        'sha256$f6b61c75d1b8eb0d6939a5c363b03f12$ae2a6d8dd0f42821f2791d813f4fc5fa5c5ce9d4ee05658433d9669b3fe783d3',
        'ADMIN', 'ACTIVE')
ON DUPLICATE KEY UPDATE nickname = VALUES(nickname), role = VALUES(role), password_hash = VALUES(password_hash);

-- ---------- 商户与商品 ----------
INSERT INTO merchant (id, name, contact, settle_cycle, status)
VALUES (1001, '绿色生活商户', '张三 13800000000', 'MONTHLY', 'ACTIVE')
ON DUPLICATE KEY UPDATE id = id; -- no-op

INSERT INTO product (id, merchant_id, name, description, category, cover_url, status, sort_order, limit_per_user)
VALUES
  (1001, 1001, '环保帆布袋', '可重复使用的棉质帆布袋，减少塑料袋消耗', '文创', '', 'ONLINE', 10, 2),
  (1002, 1001, '咖啡馆兑换券', '合作咖啡馆中杯饮品一杯', '餐饮', '', 'ONLINE', 20, 3),
  (1003, 1001, '骑行安全头盔', '城市通勤骑行头盔，带反光条', '装备', '', 'ONLINE', 30, 1)
ON DUPLICATE KEY UPDATE id = id; -- no-op：绝不把运营手动下架的商品强制改回 ONLINE

INSERT INTO product_sku (id, product_id, sku_name, points_price, stock, sold_count)
VALUES
  (1001, 1001, '标准款', 100, 50, 0),
  (1002, 1002, '中杯', 300, 30, 0),
  (1003, 1003, '均码', 800, 5, 0)
ON DUPLICATE KEY UPDATE id = id; -- no-op：绝不把已售出的库存冲回原值
