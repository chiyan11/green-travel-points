-- 执行前协调共享库；异常快照会使迁移失败，不会静默写入零分。
SET @old_sql_mode = @@SESSION.sql_mode;
SET SESSION sql_mode = 'STRICT_ALL_TABLES';
SET @ddl = IF(
  EXISTS (SELECT 1 FROM information_schema.COLUMNS
          WHERE TABLE_SCHEMA = DATABASE() AND TABLE_NAME = 'carbon_record' AND COLUMN_NAME = 'raw_points'),
  'SELECT 1',
  'ALTER TABLE carbon_record ADD COLUMN raw_points BIGINT NULL'
);
PREPARE stmt FROM @ddl;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

UPDATE carbon_record
SET raw_points = CASE
  WHEN JSON_VALID(rule_snapshot) THEN
    CASE WHEN JSON_TYPE(JSON_EXTRACT(rule_snapshot, '$.rawPoints')) = 'INTEGER'
           AND CAST(JSON_UNQUOTE(JSON_EXTRACT(rule_snapshot, '$.rawPoints')) AS DECIMAL(65,0)) BETWEEN 0 AND 9223372036854775807
         THEN CAST(JSON_UNQUOTE(JSON_EXTRACT(rule_snapshot, '$.rawPoints')) AS SIGNED)
         ELSE NULL END
  ELSE NULL END
WHERE raw_points IS NULL;

ALTER TABLE carbon_record MODIFY COLUMN raw_points BIGINT NOT NULL;
SET SESSION sql_mode = @old_sql_mode;
