-- 21_security_events.sql
-- 企业级安全事件（Firewall/RateLimit/AntiBrush/Recovery 等统一沉淀）

CREATE TABLE IF NOT EXISTS `security_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',

  `type` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '事件类型',
  `severity` VARCHAR(16) NOT NULL DEFAULT 'info' COMMENT '严重级别',
  `rid` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '请求ID',
  `created_at` BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间（unix秒）',

  `uid` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '用户ID',
  `tenant_id` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '租户ID',
  `role` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '角色',

  `ip` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '客户端IP',
  `ua` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'User-Agent',
  `device` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备标识（可选）',

  `route` VARCHAR(256) NOT NULL DEFAULT '' COMMENT '路由（FullPath/Path）',
  `method` VARCHAR(16) NOT NULL DEFAULT '' COMMENT 'HTTP方法',

  `details` TEXT COMMENT '事件详情（JSON字符串，截断）',

  PRIMARY KEY (`id`),
  KEY `idx_se_ip_time` (`ip`, `created_at`),
  KEY `idx_se_uid_time` (`uid`, `created_at`),
  KEY `idx_se_tenant_time` (`tenant_id`, `created_at`),
  KEY `idx_se_type_time` (`type`, `created_at`),
  KEY `idx_se_severity_time` (`severity`, `created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='安全事件';