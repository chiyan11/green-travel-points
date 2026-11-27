-- 000008: carbon_record.rule_snapshot 从「a=1;b=2;」拼串统一改写成 JSON。
-- 运行期只解析 JSON（apps/api/internal/httpserver/trips.go 的 decodeTripSnapshot），
-- 旧格式在数据层一次性改写，Go 里不留兼容分支。
UPDATE carbon_record
SET rule_snapshot = JSON_OBJECT(
        'carbonPerPoint', CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(rule_snapshot, 'carbon_per_point=', -1), ';', 1) AS SIGNED),
        'dailyCap',       CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(rule_snapshot, 'daily_cap=', -1), ';', 1) AS SIGNED),
        'rawPoints',      CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(rule_snapshot, 'raw_points=', -1), ';', 1) AS SIGNED),
        'grantedPoints',  CAST(SUBSTRING_INDEX(SUBSTRING_INDEX(rule_snapshot, 'granted_points=', -1), ';', 1) AS SIGNED)
    )
WHERE rule_snapshot LIKE 'carbon_per_point=%';
