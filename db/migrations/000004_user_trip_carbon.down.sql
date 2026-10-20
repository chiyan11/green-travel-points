-- 000004_user_trip_carbon 回滚：先删行程关联记录，再删用户表。
DROP TABLE IF EXISTS carbon_record;
DROP TABLE IF EXISTS trip_record;
DROP TABLE IF EXISTS user;
