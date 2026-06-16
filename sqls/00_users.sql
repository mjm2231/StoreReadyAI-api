CREATE TABLE IF NOT EXISTS `users` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '自增ID',
  `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID（MVP固定0，后期多租户扩展）',
  `uid`            BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户唯一ID（对外暴露；创建用户后可回填为id或雪花ID）',

  `status`         TINYINT NOT NULL DEFAULT 1 COMMENT '状态:0=unknown,1=active,2=banned,3=deleted',
  `role`           VARCHAR(32) NOT NULL DEFAULT 'user' COMMENT '用户角色:user,admin,owner',

  `email`          VARCHAR(255) NULL COMMENT '邮箱（Web账号密码登录使用；第三方登录可为空）',
  `email_verified` TINYINT NOT NULL DEFAULT 0 COMMENT '邮箱是否已验证:0=否,1=是',
  `password_hash`  VARCHAR(255) NULL COMMENT '密码哈希（账号密码登录使用；第三方登录可为空）',
  `login_type`     VARCHAR(32) NOT NULL DEFAULT 'password' COMMENT '主登录方式:password,google,github,apple,firebase',
  `name`           VARCHAR(128) NULL COMMENT '昵称',
  `avatar`         VARCHAR(512) NULL COMMENT '头像URL',

  `locale`         VARCHAR(32)  NULL COMMENT '语言/地区（可选）',
  `timezone`       VARCHAR(64)  NULL COMMENT '时区（可选）',

  `is_vip`         TINYINT NOT NULL DEFAULT 0 COMMENT 'VIP标记（后期付费）',
  `vip_started_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'VIP开始时间戳秒',
  `vip_expired_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT 'VIP到期时间戳秒',

  `last_login_at`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最近登录时间戳秒',
  `created_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间戳秒',
  `updated_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间戳秒',

  PRIMARY KEY (`id`),
  KEY `idx_users_tenant_id` (`tenant_id`),
  KEY `idx_users_email` (`email`),
  KEY `idx_users_uid` (`uid`),
  KEY `idx_users_status` (`status`),
  KEY `idx_users_role` (`role`),
  KEY `idx_users_login_type` (`login_type`),
  KEY `idx_users_last_login_at` (`last_login_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户主表';