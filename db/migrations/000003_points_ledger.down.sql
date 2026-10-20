-- 000003_points_ledger 回滚：先删子表（流水），再删父表（账户）
DROP TABLE IF EXISTS point_txn;
DROP TABLE IF EXISTS point_account;
