CREATE TABLE IF NOT EXISTS `admin_role_permissions` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `role_id`        BIGINT UNSIGNED NOT NULL COMMENT '角色ID，对应 admin_roles.id',
  `permission_id`  BIGINT UNSIGNED NOT NULL COMMENT '权限ID，对应 admin_permissions.id',
  `created_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间（秒）',
  `updated_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间（秒）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_admin_rp_tenant_role_permission` (`tenant_id`, `role_id`, `permission_id`),
  KEY `idx_admin_rp_tenant_role_id` (`tenant_id`, `role_id`),
  KEY `idx_admin_rp_tenant_permission_id` (`tenant_id`, `permission_id`),
  KEY `idx_admin_rp_created_at` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='后台角色权限关联表';