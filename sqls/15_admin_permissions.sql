

CREATE TABLE IF NOT EXISTS `admin_permissions` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `name`           VARCHAR(64) NOT NULL COMMENT '权限名称',
  `code`           VARCHAR(128) NOT NULL COMMENT '权限编码',
  `module`         VARCHAR(64) NOT NULL DEFAULT '' COMMENT '权限模块，如 user/subscription/vip/audit',
  `type`           TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '权限类型: 1=menu,2=page,3=action',
  `parent_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '父级权限ID，顶级为0',
  `path`           VARCHAR(255) NOT NULL DEFAULT '' COMMENT '前端路由路径或唯一标识',
  `icon`           VARCHAR(64) NOT NULL DEFAULT '' COMMENT '菜单图标',
  `sort`           INT NOT NULL DEFAULT 0 COMMENT '排序值，越小越靠前',
  `status`         TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '状态: 1=active,2=disabled',
  `is_system`      TINYINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '是否系统内置权限: 0=否,1=是',
  `remark`         VARCHAR(255) NOT NULL DEFAULT '' COMMENT '备注',
  `created_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间（秒）',
  `updated_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间（秒）',
  `deleted_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '软删除时间（秒）',
  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_admin_permission_tenant_code` (`tenant_id`,`code`),
  KEY `idx_admin_permission_tenant_id` (`tenant_id`),
  KEY `idx_admin_permission_module` (`module`),
  KEY `idx_admin_permission_type` (`type`),
  KEY `idx_admin_permission_parent_id` (`parent_id`),
  KEY `idx_admin_permission_status` (`status`),
  KEY `idx_admin_permission_sort` (`sort`),
  KEY `idx_admin_permission_deleted_at` (`deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='后台权限表';