-- 000003_points_ledger：积分账户（余额快照）与积分流水（只追加账本）
--
-- 字段与枚举严格对齐 P1 的域模型（docs/uml/04-域模型.puml）与
-- 需求规格说明书 BR-07 / BR-08 / BR-09 / BR-11，保证"报告里的图 = 数据库里的表"：
--   PointAccount: id, userId, balance, frozen, totalSavedG, version
--   PointTxn    : id, accountId, txnType, amount, bizRefType, bizRefId, balanceAfter, occurredAt
--
-- 不变量：
--   1. 账本等价：point_account.balance == SUM(point_txn.amount)（BR-07）
--   2. 只追加：point_txn 不 UPDATE / DELETE，冲正写 REFUND 反向流水（BR-08）
--   3. 不超兑：可兑额度 = balance - frozen，兑换前 SELECT ... FOR UPDATE 锁后复核（BR-09）
--   4. 幂等：同一业务单据的同一类流水由唯一索引兜底（如一个行程只能有一条 TRAVEL）
--
-- 说明：
--   - user 表由 M1（P2）创建，本迁移不建物理外键，仅用 user_id 逻辑关联，
--     与 000001 / 000002 风格一致。
--   - frozen 是**风控锁定**金额（不是兑换冻结）：兑换直接扣 balance，不走冻结；
--     可兑额度 = balance - frozen。
--   - 积分批次与 365 天 FIFO 过期（BR-11）计划由 000004 的 point_batch 表承载。

CREATE TABLE IF NOT EXISTS point_account (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  user_id       BIGINT UNSIGNED NOT NULL                COMMENT '用户ID（逻辑关联 user.id）',
  balance       BIGINT          NOT NULL DEFAULT 0      COMMENT '积分余额（正确性以 point_txn 汇总为准，此处为快照）',
  frozen        BIGINT          NOT NULL DEFAULT 0      COMMENT '风控锁定积分；可兑额度 = balance - frozen（BR-09）',
  total_saved_g DECIMAL(14, 3)  NOT NULL DEFAULT 0.000  COMMENT '累计碳减排量（gCO2e），用于成就/报表',
  version       INT UNSIGNED    NOT NULL DEFAULT 0      COMMENT '乐观锁版本号，每次余额变动 +1',
  created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_point_account_user (user_id),
  CONSTRAINT ck_point_account_nonneg CHECK (balance >= 0 AND frozen >= 0),
  CONSTRAINT ck_point_account_frozen CHECK (frozen <= balance)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '积分账户（余额快照，一人一户）';

CREATE TABLE IF NOT EXISTS point_txn (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  account_id    BIGINT UNSIGNED NOT NULL                COMMENT '积分账户ID（逻辑关联 point_account.id）',
  txn_type      ENUM('TRAVEL','REDEEM','REFUND','ADJUST','EXPIRE') NOT NULL COMMENT '流水类型：行程发放/兑换消耗/退款冲正/人工调整/过期',
  amount        BIGINT          NOT NULL                COMMENT '变动值：正数=入账，负数=出账（BR-07）',
  balance_after BIGINT          NOT NULL                COMMENT '本笔入账后的余额快照（对账与审计用）',
  biz_ref_type  VARCHAR(32)     NOT NULL                COMMENT '业务来源类型：trip_record / redeem_order / audit_log',
  biz_ref_id    BIGINT UNSIGNED NOT NULL                COMMENT '业务单据ID（行程ID / 兑换单ID / 审计记录ID）',
  occurred_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '流水发生时间',
  PRIMARY KEY (id),
  UNIQUE KEY uk_point_txn_biz (biz_ref_type, biz_ref_id, txn_type),
  KEY idx_point_txn_account_time (account_id, occurred_at),
  CONSTRAINT ck_point_txn_amount_nonzero CHECK (amount <> 0),
  CONSTRAINT ck_point_txn_balance_after CHECK (balance_after >= 0)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '积分流水（append-only 账本，余额的唯一真相来源）';
