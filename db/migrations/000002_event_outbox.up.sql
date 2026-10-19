-- 000002_event_outbox：事务性 Outbox 表（EDA 的最小实现）
--
-- 用法：业务事务内同时 INSERT 业务数据与该表，保证"业务成功 ⇒ 事件必达"；
-- 后台 Worker 轮询 dispatched_at IS NULL 且 available_at <= NOW() 的行。

CREATE TABLE IF NOT EXISTS event_outbox (
  id             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  topic          VARCHAR(64)     NOT NULL                COMMENT '事件主题，如 TripVerified',
  aggregate_type VARCHAR(64)     NOT NULL                COMMENT '聚合类型，如 trip',
  aggregate_id   VARCHAR(64)     NOT NULL                COMMENT '聚合根 ID',
  payload        JSON            NOT NULL                COMMENT '事件负载',
  attempts       INT             NOT NULL DEFAULT 0      COMMENT '已尝试投递次数',
  available_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3) COMMENT '最早可投递时间（支持延迟/退避重试）',
  dispatched_at  DATETIME(3)     NULL                    COMMENT '投递成功时间',
  dead_letter_at DATETIME(3)     NULL                    COMMENT '进入死信时间',
  last_error     VARCHAR(1024)   NULL                    COMMENT '最后一次失败原因',
  created_at     DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_outbox_pending (dispatched_at, dead_letter_at, available_at),
  KEY idx_outbox_aggregate (aggregate_type, aggregate_id)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '事务性发件箱（领域事件）';
