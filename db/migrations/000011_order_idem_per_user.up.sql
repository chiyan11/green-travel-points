-- 000011：兑换幂等键由「全局唯一」改为「用户内唯一」。
-- 原因：uk_order_idem(idempotency_key) 全局唯一，而查询按 (user_id, idempotency_key)，
-- 不同用户偶然使用同一 key 时，后者插入触发 1062 后回读不到自己的订单，返回 500。
-- 共享库不能重建，因此以新迁移改索引；000005 保持原样（与已执行历史一致）。
ALTER TABLE redeem_order
  DROP INDEX uk_order_idem,
  ADD UNIQUE KEY uk_order_idem (user_id, idempotency_key);
