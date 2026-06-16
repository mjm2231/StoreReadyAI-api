-- 01_user_identities.sql（第三方身份绑定表；用于 Google/GitHub/Apple/Firebase 等账号绑定）
CREATE TABLE IF NOT EXISTS `user_identities` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `user_id`      BIGINT UNSIGNED NOT NULL COMMENT '关联users.id',

  `provider`     VARCHAR(32) NOT NULL COMMENT '提供方: google,github,apple,firebase,password',
  `provider_uid` VARCHAR(191) NOT NULL COMMENT '提供方用户唯一ID，如 Google sub/GitHub id/Apple sub/Firebase UID',

  `email`        VARCHAR(255) NULL COMMENT '提供方邮箱（可空）',
  `display_name` VARCHAR(128) NULL COMMENT '提供方昵称（可空）',
  `avatar_url`   VARCHAR(512) NULL COMMENT '提供方头像URL（可空）',
  `raw_profile`  JSON NULL COMMENT '原始profile（可选，排查/扩展用）',

  `created_at`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间戳秒',
  `updated_at`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间戳秒',

  PRIMARY KEY (`id`),

  -- 关键唯一约束：同一租户下 provider+provider_uid 唯一，避免同一个第三方账号绑定到多个用户
  UNIQUE KEY `uk_identity_provider_uid` (`tenant_id`,`provider`,`provider_uid`),

  KEY `idx_identity_user_id` (`tenant_id`,`user_id`),
  KEY `idx_identity_provider` (`tenant_id`,`provider`),
  KEY `idx_identity_email` (`tenant_id`,`email`),

  CONSTRAINT `fk_identity_user`
    FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户第三方身份绑定表';