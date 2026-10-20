-- 000003_points_ledger：积分账户（余额快照）与积分流水（只追加账本）
--
-- 与 docs/项目分工与迭代计划.md §2.3 的"不变量"一一对应：
--   1. 账本等价：point_account.balance == SUM(point_txn.amount)，由对账接口验证
--   2. 只追加：point_txn 只有 INSERT，没有 UPDATE / DELETE；冲正一律写反向流水
--   3. 幂等：idem_key 唯一索引，重复上报 / 重复兑换只落一条流水
--   4. 可追溯：每笔流水带 biz_type + biz_id，可反查行程 / 兑换单 / 调整单
--
-- 说明：user 表由 M1（P2 负责）创建，本迁移不建外键，仅用 user_id 做逻辑关联，
--       与 000001 / 000002 的风格保持一致（现有迁移均未使用物理外键）。

CREATE TABLE IF NOT EXISTS point_account (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  user_id      BIGINT UNSIGNED NOT NULL                COMMENT '用户ID（逻辑关联 user.id）',
  balance      BIGINT          NOT NULL DEFAULT 0      COMMENT '可用积分余额（快照，正确性以 point_txn 汇总为准）',
  frozen       BIGINT          NOT NULL DEFAULT 0      COMMENT '兑换冻结积分（不参与可用余额）',
  total_earned BIGINT          NOT NULL DEFAULT 0      COMMENT '累计获得积分（只增不减）',
  total_spent  BIGINT          NOT NULL DEFAULT 0      COMMENT '累计消耗积分（只增不减）',
  version      BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '乐观锁版本号，每次余额变动 +1',
  created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_point_account_user (user_id),
  CONSTRAINT ck_point_account_nonneg CHECK (balance >= 0 AND frozen >= 0)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '积分账户（余额快照，一人一户）';

CREATE TABLE IF NOT EXISTS point_txn (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  user_id       BIGINT UNSIGNED NOT NULL                COMMENT '用户ID（逻辑关联 user.id）',
  txn_type      ENUM('earn','spend','expire','reverse','adjust') NOT NULL COMMENT '流水类型：获得/消耗/过期/冲正/人工调整',
  amount        BIGINT          NOT NULL                COMMENT '变动值：正数=入账，负数=出账',
  balance_after BIGINT          NOT NULL                COMMENT '本笔入账后的可用余额快照（对账与审计用）',
  biz_type      VARCHAR(32)     NOT NULL                COMMENT '业务来源：trip / redeem_order / admin / expire_job',
  biz_id        VARCHAR(64)     NOT NULL                COMMENT '业务单据ID（行程ID / 兑换单号 / 调整单号）',
  idem_key      VARCHAR(128)    NULL                    COMMENT '幂等键，如 trip:123、redeem:RO20261020001；唯一索引防重复入账',
  remark        VARCHAR(255)    NOT NULL DEFAULT ''     COMMENT '备注 / 冲正原因',
  created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_point_txn_idem (idem_key),
  KEY idx_point_txn_user (user_id, id),
  KEY idx_point_txn_biz (biz_type, biz_id),
  CONSTRAINT ck_point_txn_amount_nonzero CHECK (amount <> 0),
  CONSTRAINT ck_point_txn_balance_after CHECK (balance_after >= 0)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '积分流水（append-only 账本，余额的唯一真相来源）';
