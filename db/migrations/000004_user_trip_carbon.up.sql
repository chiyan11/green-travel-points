-- 000004_user_trip_carbon：P2 的用户、行程与碳核算记录
--
-- user_id + trip_hash 是行程上报幂等键；
-- carbon_record 保存计算时的排放因子和规则快照，便于追溯与重算。

CREATE TABLE IF NOT EXISTS user (
  id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  phone         VARCHAR(20)     NOT NULL                COMMENT '手机号，登录标识',
  nickname      VARCHAR(64)     NOT NULL DEFAULT ''    COMMENT '昵称',
  password_hash VARCHAR(255)    NOT NULL                COMMENT '密码哈希，不存明文',
  role          ENUM('CITIZEN','ADMIN','MERCHANT') NOT NULL DEFAULT 'CITIZEN' COMMENT '用户角色',
  status        ENUM('ACTIVE','FROZEN') NOT NULL DEFAULT 'ACTIVE' COMMENT '账户状态',
  created_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at    DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_user_phone (phone),
  KEY idx_user_status (status)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '用户账户';

CREATE TABLE IF NOT EXISTS trip_record (
  id          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  user_id     BIGINT UNSIGNED NOT NULL                COMMENT '用户ID（逻辑关联 user.id）',
  mode_code   VARCHAR(32)     NOT NULL                COMMENT '出行方式代码（逻辑关联 travel_mode.code）',
  distance_km DECIMAL(8, 3)    NOT NULL                COMMENT '行程距离，单位 km',
  started_at  DATETIME(3)     NOT NULL                COMMENT '出发时间',
  ended_at    DATETIME(3)     NOT NULL                COMMENT '结束时间',
  trip_hash   CHAR(64)        NOT NULL                COMMENT '行程幂等哈希',
  source      ENUM('APP','SIMULATOR','ADMIN') NOT NULL DEFAULT 'APP' COMMENT '数据来源',
  status      ENUM('VALID','REVOKED') NOT NULL DEFAULT 'VALID' COMMENT '行程状态',
  created_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at  DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_trip_user_hash (user_id, trip_hash),
  KEY idx_trip_user_time (user_id, started_at),
  KEY idx_trip_mode (mode_code),
  CONSTRAINT ck_trip_distance_positive CHECK (distance_km > 0),
  CONSTRAINT ck_trip_time_order CHECK (ended_at >= started_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '用户上报的出行记录';

CREATE TABLE IF NOT EXISTS carbon_record (
  id              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  trip_id         BIGINT UNSIGNED NOT NULL                COMMENT '行程ID（逻辑关联 trip_record.id）',
  ef_base         DECIMAL(8, 3)    NOT NULL                COMMENT '计算时的基准排放因子',
  ef_actual       DECIMAL(8, 3)    NOT NULL                COMMENT '计算时的实际排放因子',
  carbon_saved_g  DECIMAL(12, 3)   NOT NULL                COMMENT '碳减排量，单位 gCO2e',
  rule_snapshot  VARCHAR(255)     NOT NULL                COMMENT '计算时的积分规则快照',
  computed_at     DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '计算时间',
  created_at      DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  updated_at      DATETIME(3)      NOT NULL DEFAULT CURRENT_TIMESTAMP(3) ON UPDATE CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  UNIQUE KEY uk_carbon_trip (trip_id),
  CONSTRAINT ck_carbon_saved_nonnegative CHECK (carbon_saved_g >= 0)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '行程碳减排核算记录';
