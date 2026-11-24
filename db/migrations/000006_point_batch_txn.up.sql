-- 000006_point_batch_txn：批次消耗明细（FIFO 扣减 / 退款回填的凭据）
--
-- 背景：point_batch 只记录批次余额，不记录“哪一笔流水消耗了哪些批次”。
-- 取消订单需要把积分退回原批次（expires_at 不变），因此必须留下消费映射。
--
-- 约定：
--   - amount 为有符号：兑换扣减为负数，取消退款回填为正数；
--   - 同一 (batch_id, txn_id) 只允许一行，重复回填会被唯一索引拦下；
--   - 批次余额恒等式：point_batch.remaining = granted_amount + Σ(本表 amount) + 过期扣减。

CREATE TABLE IF NOT EXISTS point_batch_txn (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  batch_id   BIGINT UNSIGNED NOT NULL                COMMENT '批次ID（逻辑关联 point_batch.id）',
  txn_id     BIGINT UNSIGNED NOT NULL                COMMENT '积分流水ID（逻辑关联 point_txn.id）',
  amount     BIGINT          NOT NULL                COMMENT '批次变动量：消耗为负，回填为正',
  created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_pbt_batch_txn (batch_id, txn_id),
  KEY idx_pbt_txn (txn_id)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '积分批次消耗明细（FIFO 扣减与退款回填）';
