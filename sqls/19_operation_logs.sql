

CREATE TABLE IF NOT EXISTS `operation_logs` (
  `id`               BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id`        BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `admin_user_id`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '操作管理员ID，对应 admin_users.id',
  `admin_username`   VARCHAR(64) NOT NULL DEFAULT '' COMMENT '操作管理员用户名快照',
  `module`           VARCHAR(64) NOT NULL DEFAULT '' COMMENT '操作模块，如 user/subscription/vip/audit/system',
  `action`           VARCHAR(64) NOT NULL DEFAULT '' COMMENT '操作动作，如 create/update/delete/ban/unban/grant/revoke',
  `target_type`      VARCHAR(64) NOT NULL DEFAULT '' COMMENT '目标对象类型，如 user/subscription/admin_user/role',
  `target_id`        VARCHAR(64) NOT NULL DEFAULT '' COMMENT '目标对象ID，统一用字符串存储便于兼容',
  `target_snapshot`  VARCHAR(255) NOT NULL DEFAULT '' COMMENT '目标对象摘要快照，如用户名/服务名/角色名',
  `request_id`       VARCHAR(64) NOT NULL DEFAULT '' COMMENT '请求ID，便于日志追踪',
  `ip`               VARCHAR(64) NOT NULL DEFAULT '' COMMENT '操作IP',
  `user_agent`       VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'User-Agent',
  `before_json`      JSON NULL COMMENT '操作前数据快照',
  `after_json`       JSON NULL COMMENT '操作后数据快照',
  `extra_json`       JSON NULL COMMENT '扩展字段，如批量操作参数、上下文信息',
  `reason`           VARCHAR(255) NOT NULL DEFAULT '' COMMENT '操作原因，敏感操作建议必填',
  `created_at`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间（秒）',
  PRIMARY KEY (`id`),
  KEY `idx_operation_tenant_id` (`tenant_id`),
  KEY `idx_operation_admin_user_id` (`admin_user_id`),
  KEY `idx_operation_module` (`module`),
  KEY `idx_operation_action` (`action`),
  KEY `idx_operation_target_type_target_id` (`target_type`, `target_id`),
  KEY `idx_operation_request_id` (`request_id`),
  KEY `idx_operation_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='后台操作审计日志表';