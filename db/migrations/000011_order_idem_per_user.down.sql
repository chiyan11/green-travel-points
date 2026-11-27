-- 回滚：恢复全局唯一。若已有两个用户使用同一 key，回滚会因 1062 失败，
-- 需要先清理重复 key 的数据再执行。
ALTER TABLE redeem_order
  DROP INDEX uk_order_idem,
  ADD UNIQUE KEY uk_order_idem (idempotency_key);
