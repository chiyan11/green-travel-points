-- 000001_init：系统参数与出行方式（业务规则冻结，Sprint 0）
-- 这两张表承载"碳减排 → 积分"的全部可调参数，业务代码不得硬编码。

CREATE TABLE IF NOT EXISTS sys_config (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  config_key   VARCHAR(64)     NOT NULL                COMMENT '参数键',
  config_value VARCHAR(255)    NOT NULL                COMMENT '参数值（按 value_type 解析）',
  value_type   ENUM('string','int','decimal','bool') NOT NULL DEFAULT 'string' COMMENT '值类型',
  description  VARCHAR(255)    NOT NULL DEFAULT ''     COMMENT '说明',
  created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_sys_config_key (config_key)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '系统参数表';

CREATE TABLE IF NOT EXISTS travel_mode (
  id         BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  code       VARCHAR(32)     NOT NULL                COMMENT '出行方式代码',
  name       VARCHAR(64)     NOT NULL                COMMENT '显示名称',
  ef_base    DECIMAL(8, 3)   NOT NULL                COMMENT '基准排放因子 gCO2e/km（燃油私家车口径）',
  ef_actual  DECIMAL(8, 3)   NOT NULL                COMMENT '该方式实际排放因子 gCO2e/km',
  sort_order INT             NOT NULL DEFAULT 0      COMMENT '排序权重',
  enabled    BOOLEAN         NOT NULL DEFAULT TRUE   COMMENT '是否启用',
  created_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_travel_mode_code (code)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '出行方式与排放因子';

-- 种子数据：与 apps/api/internal/domain/carbon.go 的 SeedEmissionFactors 保持一致
INSERT INTO travel_mode (code, name, ef_base, ef_actual, sort_order) VALUES
  ('walk',        '步行',    192.000,   0.000, 10),
  ('bike',        '骑行',    192.000,   0.000, 20),
  ('subway',      '地铁',    192.000,  41.000, 30),
  ('bus',         '公交',    192.000,  68.000, 40),
  ('shared_bike', '共享单车', 192.000,   0.000, 50)
ON DUPLICATE KEY UPDATE name = VALUES(name);

-- 业务规则参数：与 domain.DefaultPointsRule() 保持一致
INSERT INTO sys_config (config_key, config_value, value_type, description) VALUES
  ('carbon_per_point',    '100',  'int',     '多少克碳减排量兑换 1 积分'),
  ('daily_points_cap',    '500',  'int',     '单日可获得积分上限，0 表示不限制'),
  ('trip_min_distance_km','0.1',  'decimal', '单次行程最短距离（km）'),
  ('trip_max_distance_km','100',  'decimal', '单次行程最长距离（km）'),
  ('points_expire_days',  '365',  'int',     '积分有效期（天）')
ON DUPLICATE KEY UPDATE description = VALUES(description);
