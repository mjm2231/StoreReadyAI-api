CREATE TABLE IF NOT EXISTS `admin_user_roles` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `admin_user_id`  BIGINT UNSIGNED NOT NULL COMMENT '管理员ID，对应 admin_users.id',
  `role_id`        BIGINT UNSIGNED NOT NULL COMMENT '角色ID，对应 admin_roles.id',
  `created_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间（秒）',
  `updated_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间（秒）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_admin_ur_tenant_user_role` (`tenant_id`, `admin_user_id`, `role_id`),
  KEY `idx_admin_ur_tenant_admin_user_id` (`tenant_id`, `admin_user_id`),
  KEY `idx_admin_ur_tenant_role_id` (`tenant_id`, `role_id`),
  KEY `idx_admin_ur_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='后台管理员角色关联表';