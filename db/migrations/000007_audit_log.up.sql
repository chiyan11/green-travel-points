-- 000007_audit_log：运营审计日志（BR-14 / NFR-12 / UC-19）
-- 人工调账与参数变更必须留痕：操作人、对象、前后值、理由；只追加，不提供更新/删除路径。

CREATE TABLE IF NOT EXISTS audit_log (
  id           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  operator_id  BIGINT UNSIGNED NOT NULL                COMMENT '操作人 user.id（后台管理员）',
  action       ENUM('ADJUST','CONFIG_UPDATE','TRAVEL_MODE_UPDATE') NOT NULL COMMENT '动作：人工调账/参数变更/出行方式变更',
  target_type  VARCHAR(32)     NOT NULL                COMMENT '对象类型：point_account / sys_config / travel_mode',
  target_id    BIGINT UNSIGNED NOT NULL DEFAULT 0      COMMENT '对象ID（账户ID / sys_config.id；出行方式无ID时为0）',
  target_key   VARCHAR(64)     NOT NULL DEFAULT ''     COMMENT '对象键（config_key / travel_mode.code），便于按对象检索',
  amount       BIGINT          NOT NULL DEFAULT 0      COMMENT '积分变动量（调账用，正增负减；其他动作为0）',
  before_value VARCHAR(255)    NOT NULL DEFAULT ''     COMMENT '变更前值（余额或配置值）',
  after_value  VARCHAR(255)    NOT NULL DEFAULT ''     COMMENT '变更后值',
  reason       VARCHAR(255)    NOT NULL                COMMENT '操作理由（BR-14 必填）',
  created_at   DATETIME(3)     NOT NULL DEFAULT CURRENT_TIMESTAMP(3),
  PRIMARY KEY (id),
  KEY idx_audit_operator_time (operator_id, created_at),
  KEY idx_audit_target_time (target_type, target_key, created_at),
  KEY idx_audit_action_time (action, created_at)
) ENGINE = InnoDB
  DEFAULT CHARSET = utf8mb4
  COLLATE = utf8mb4_0900_ai_ci COMMENT = '运营审计日志（只追加）';
