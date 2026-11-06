-- 000005_mall_orders：商城目录（商户 / 商品 / SKU）、兑换订单与积分批次
--
-- 对齐：
--   docs/uml/04-域模型.puml              Merchant / Product / ProductSku / RedeemOrder
--   docs/uml/10-状态图-订单.puml          PENDING → PAID → SHIPPED → COMPLETED，可 CANCELLED（BR-10）
--   docs/uml/11-状态图-积分.puml / BR-11  积分批次 + 365 天先进先出过期
--   docs/前端接口需求清单.md              imageUrl / description / limitPerUser / receiver 等展示字段
--
-- 依赖：000003（point_account / point_txn）、000004（user / trip_record / carbon_record）。
-- 说明：
--   - 不建物理外键，全部用 id 逻辑关联，与 000001~000004 风格一致。
--   - redeem_order 保存商品名 / SKU 名 / 扣分快照：商品改名或下架后，历史订单仍可正确展示。
--   - point_batch 是一次发放（TRAVEL 流水）对应的积分批次，支撑 BR-11 的 FIFO 过期；
--     批次状态与 11-状态图-积分 的 可用 / 部分消耗 / 已消耗 / 已过期 / 已冲正 对应。

-- ============ 1. 商户 ============
CREATE TABLE IF NOT EXISTS merchant (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  name         VARCHAR(128)    NOT NULL                COMMENT '商户名称',
  contact      VARCHAR(64)     NOT NULL DEFAULT ''     COMMENT '联系人 / 联系方式',
  settle_cycle ENUM('MONTHLY','WEEKLY') NOT NULL DEFAULT 'MONTHLY' COMMENT '结算周期',
  status       ENUM('ACTIVE','DISABLED') NOT NULL DEFAULT 'ACTIVE' COMMENT '商户状态',
  created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_merchant_name (name),
  KEY idx_merchant_status (status)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '商户（商品供货方）';

-- ============ 2. 商品 ============
CREATE TABLE IF NOT EXISTS product (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  merchant_id    BIGINT UNSIGNED NOT NULL                COMMENT '商户ID（逻辑关联 merchant.id）',
  name           VARCHAR(128)    NOT NULL                COMMENT '商品名称',
  description    VARCHAR(500)    NOT NULL DEFAULT ''     COMMENT '商品描述（前端列表展示）',
  category       VARCHAR(32)     NOT NULL                COMMENT '商品分类',
  cover_url      VARCHAR(255)    NOT NULL DEFAULT ''     COMMENT '封面图 URL（对应前端 imageUrl）',
  status         ENUM('DRAFT','ONLINE','OFFLINE') NOT NULL DEFAULT 'DRAFT' COMMENT '商品状态：草稿/上架/下架',
  sort_order     INT             NOT NULL DEFAULT 0      COMMENT '排序权重，越小越靠前',
  limit_per_user INT             NOT NULL DEFAULT 0      COMMENT '单用户限兑数量，0 表示不限',
  created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_product_merchant_name (merchant_id, name),
  KEY idx_product_status_sort (status, sort_order),
  KEY idx_product_category (category),
  CONSTRAINT ck_product_limit_per_user CHECK (limit_per_user >= 0)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '积分商品';

-- ============ 3. 商品规格（SKU，库存与积分价格的最小兑换单位） ============
CREATE TABLE IF NOT EXISTS product_sku (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  product_id   BIGINT UNSIGNED NOT NULL                COMMENT '商品ID（逻辑关联 product.id）',
  sku_name     VARCHAR(64)     NOT NULL                COMMENT '规格名称',
  points_price BIGINT          NOT NULL                COMMENT '兑换所需积分（整数，禁止浮点）',
  stock        INT             NOT NULL DEFAULT 0      COMMENT '库存数量（BR-09 不超兑）',
  sold_count   INT             NOT NULL DEFAULT 0      COMMENT '累计兑换数量',
  created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_sku_product_name (product_id, sku_name),
  KEY idx_sku_product (product_id),
  CONSTRAINT ck_sku_points_price_positive CHECK (points_price > 0),
  CONSTRAINT ck_sku_stock_nonnegative CHECK (stock >= 0),
  CONSTRAINT ck_sku_sold_count_nonnegative CHECK (sold_count >= 0)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '商品规格（库存 / 积分价格）';

-- ============ 4. 兑换订单（状态机见 10-状态图-订单） ============
CREATE TABLE IF NOT EXISTS redeem_order (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  order_no        VARCHAR(32)     NOT NULL                COMMENT '订单号（对外展示）',
  user_id         BIGINT UNSIGNED NOT NULL                COMMENT '下单用户ID（逻辑关联 user.id）',
  product_id      BIGINT UNSIGNED NOT NULL                COMMENT '商品ID（快照来源）',
  sku_id          BIGINT UNSIGNED NOT NULL                COMMENT 'SKU ID（逻辑关联 product_sku.id）',
  product_name    VARCHAR(128)    NOT NULL                COMMENT '下单时的商品名快照',
  sku_name        VARCHAR(64)     NOT NULL                COMMENT '下单时的规格名快照',
  points_spent    BIGINT          NOT NULL                COMMENT '本单消耗积分（快照，对应一笔 REDEEM 流水）',
  quantity        INT             NOT NULL DEFAULT 1      COMMENT '兑换数量',
  receiver        VARCHAR(64)     NOT NULL DEFAULT ''     COMMENT '领取人 / 收货人',
  status          ENUM('PENDING','PAID','SHIPPED','COMPLETED','CANCELLED') NOT NULL DEFAULT 'PENDING' COMMENT '订单状态（BR-10 状态机）',
  redeem_code     VARCHAR(64)     NOT NULL                COMMENT '核销码（发货后生效）',
  idempotency_key VARCHAR(64)     NULL                    COMMENT '兑换幂等键（BR-15，24h 内重复请求返回原订单）',
  shipped_at      DATETIME(3)     NULL                    COMMENT '发货时间',
  completed_at    DATETIME(3)     NULL                    COMMENT '完成（核销）时间',
  cancelled_at    DATETIME(3)     NULL                    COMMENT '取消时间',
  cancel_reason   VARCHAR(255)    NOT NULL DEFAULT ''     COMMENT '取消原因',
  created_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_order_no (order_no),
  UNIQUE KEY uk_order_redeem_code (redeem_code),
  UNIQUE KEY uk_order_idem (idempotency_key),
  KEY idx_order_user_time (user_id, created_at),
  KEY idx_order_status (status),
  KEY idx_order_sku (sku_id),
  CONSTRAINT ck_order_points_spent_positive CHECK (points_spent > 0),
  CONSTRAINT ck_order_quantity_positive CHECK (quantity > 0)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '兑换订单（PENDING→PAID→SHIPPED→COMPLETED，可 CANCELLED）';

-- ============ 5. 积分批次（BR-11：365 天有效期 + 先进先出扣减） ============
CREATE TABLE IF NOT EXISTS point_batch (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  account_id     BIGINT UNSIGNED NOT NULL                COMMENT '积分账户ID（逻辑关联 point_account.id）',
  grant_txn_id   BIGINT UNSIGNED NOT NULL                COMMENT '发放来源流水ID（point_txn.id，TRAVEL 类型）',
  granted_amount BIGINT          NOT NULL                COMMENT '本批次发放积分（入账正数）',
  remaining      BIGINT          NOT NULL                COMMENT '本批次剩余可用积分（FIFO 扣减）',
  expires_at     DATETIME(3)     NOT NULL                COMMENT '过期时间 = 入账时间 + points_expire_days',
  status         ENUM('ACTIVE','CONSUMED','EXPIRED','REVERSED') NOT NULL DEFAULT 'ACTIVE' COMMENT '批次状态：可用/已耗尽/已过期/已冲正',
  created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_batch_grant_txn (grant_txn_id),
  KEY idx_batch_fifo (account_id, status, expires_at),
  CONSTRAINT ck_batch_granted_positive CHECK (granted_amount > 0),
  CONSTRAINT ck_batch_remaining_range CHECK (remaining >= 0 AND remaining <= granted_amount)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '积分批次（一次发放一行，支撑 FIFO 过期与部分消耗）';
