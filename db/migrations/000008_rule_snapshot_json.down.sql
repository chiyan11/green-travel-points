-- 000008 回滚：把 rule_snapshot 从 JSON 还原成迁移前的拼串格式。
UPDATE carbon_record
SET rule_snapshot = CONCAT(
        'carbon_per_point=', COALESCE(rule_snapshot ->> '$.carbonPerPoint', '0'),
        ';daily_cap=',       COALESCE(rule_snapshot ->> '$.dailyCap', '0'),
        ';raw_points=',      COALESCE(rule_snapshot ->> '$.rawPoints', '0'),
        ';granted_points=',  COALESCE(rule_snapshot ->> '$.grantedPoints', '0')
    )
WHERE rule_snapshot LIKE '{%';
