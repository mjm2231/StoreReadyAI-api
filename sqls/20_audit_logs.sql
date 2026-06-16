-- 20_audit_logs.sql
-- 企业级审计日志（关键操作可追溯）

CREATE TABLE IF NOT EXISTS `audit_logs` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',

  `rid` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '请求ID',
  `trace_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT 'TraceID（预留）',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间（unix秒）',

  `uid` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '用户ID',
  `tenant_id` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '租户ID',
  `role` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '角色',
  `scopes` VARCHAR(512) NOT NULL DEFAULT '' COMMENT '权限范围（逗号分隔）',

  `action` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '动作（枚举/规范化字符串）',
  `resource_type` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '资源类型',
  `resource_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '资源ID',

  `ip` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '客户端IP',
  `ua` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'User-Agent',
  `device` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备标识（可选）',
  `refer` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'Referer',

  `success` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否成功',
  `http_status` INT NOT NULL DEFAULT 0 COMMENT 'HTTP状态码',
  `err_code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '业务错误码',
  `latency_ms` BIGINT NOT NULL DEFAULT 0 COMMENT '耗时（毫秒）',

  `method` VARCHAR(16) NOT NULL DEFAULT '' COMMENT 'HTTP方法',
  `path` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '请求路径',

  `query_summary` TEXT COMMENT 'Query 摘要（脱敏/截断）',
  `body_summary`  TEXT COMMENT 'Body 摘要（脱敏/截断）',
  `resp_summary`  TEXT COMMENT '响应摘要（可选，脱敏/截断）',

  `request_size_b` BIGINT NOT NULL DEFAULT 0 COMMENT '请求体大小（字节，参考）',
  `response_size_b` BIGINT NOT NULL DEFAULT 0 COMMENT '响应体大小（字节，参考）',

  `risk_score` BIGINT NOT NULL DEFAULT 0 COMMENT '风控分（可选）',
  `risk_action` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '风控动作（可选）',
  `risk_reasons` TEXT COMMENT '风控原因（JSON字符串，可选）',

  PRIMARY KEY (`id`),
  KEY `idx_audit_tenant_time` (`tenant_id`, `created_at`),
  KEY `idx_audit_uid_time` (`uid`, `created_at`),
  KEY `idx_audit_action_time` (`action`, `created_at`),
  KEY `idx_audit_status_time` (`http_status`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='审计日志';