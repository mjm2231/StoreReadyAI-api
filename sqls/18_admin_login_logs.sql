

CREATE TABLE IF NOT EXISTS `admin_login_logs` (
  `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `admin_user_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '管理员ID，登录成功时对应 admin_users.id，失败时可为0',
  `username`        VARCHAR(64) NOT NULL DEFAULT '' COMMENT '登录用户名快照',
  `login_type`      TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '登录类型: 1=password',
  `status`          TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '登录结果: 1=success,2=failed,3=logout',
  `failure_reason`  VARCHAR(255) NOT NULL DEFAULT '' COMMENT '失败原因，成功为空',
  `ip`              VARCHAR(64) NOT NULL DEFAULT '' COMMENT '登录IP',
  `user_agent`      VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'User-Agent',
  `request_id`      VARCHAR(64) NOT NULL DEFAULT '' COMMENT '请求ID，便于日志追踪',
  `created_at`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间（秒）',
  PRIMARY KEY (`id`),
  KEY `idx_admin_login_tenant_id` (`tenant_id`),
  KEY `idx_admin_login_admin_user_id` (`admin_user_id`),
  KEY `idx_admin_login_username` (`username`),
  KEY `idx_admin_login_status` (`status`),
  KEY `idx_admin_login_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='后台管理员登录日志表';