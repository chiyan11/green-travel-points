-- 共享库先协调再执行；重复执行不会重复添加列。
SET @ddl = IF(
  EXISTS (SELECT 1 FROM information_schema.COLUMNS
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'event_outbox' AND COLUMN_NAME = 'lease_token'),
  'SELECT 1',
  'ALTER TABLE event_outbox ADD COLUMN lease_token CHAR(32) NULL, ADD COLUMN lease_until DATETIME(3) NULL'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;
