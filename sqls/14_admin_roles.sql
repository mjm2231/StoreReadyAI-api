CREATE TABLE IF NOT EXISTS `admin_roles` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `name`           VARCHAR(64) NOT NULL COMMENT '角色名称',
  `code`           VARCHAR(64) NOT NULL COMMENT '角色编码',
  `status`         TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态: 1=active,2=disabled',
  `sort`           INT NOT NULL DEFAULT 0 COMMENT '排序值，越小越靠前',
  `is_system`      TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否系统内置角色: 0=否,1=是',
  `remark`         VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `created_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间（秒）',
  `updated_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间（秒）',
  `deleted_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '软删除时间（秒）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_admin_role_tenant_code` (`tenant_id`,`code`),
  KEY `idx_admin_role_tenant_id` (`tenant_id`),
  KEY `idx_admin_role_status` (`status`),
  KEY `idx_admin_role_sort` (`sort`),
  KEY `idx_admin_role_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='后台角色表';