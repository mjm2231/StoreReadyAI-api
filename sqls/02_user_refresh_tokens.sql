-- 02_user_refresh_tokens.sql（用户 Refresh Token 表；支持 Web 多端登录、续期、退出登录）
CREATE TABLE IF NOT EXISTS `user_refresh_tokens` (
  `id`           BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id`    BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `user_id`      BIGINT UNSIGNED NOT NULL COMMENT '用户ID',

  `token_hash`   CHAR(64) NOT NULL COMMENT 'refresh token SHA256哈希（不存明文）',
  `token_family` VARCHAR(64) NULL COMMENT 'Token族ID，用于刷新轮换时批量吊销同一登录会话',

  `device_id`    VARCHAR(128) NULL COMMENT '设备ID（Web可使用浏览器指纹/本地生成UUID）',
  `device_name`  VARCHAR(128) NULL COMMENT '设备名（如 Chrome on macOS）',
  `platform`     VARCHAR(32) NOT NULL DEFAULT 'web' COMMENT '平台:web,ios,android,admin',
  `ip`           VARCHAR(64)  NULL COMMENT '最近登录/刷新IP',
  `user_agent`   VARCHAR(512) NULL COMMENT 'UA',

  `status`       TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1=active,2=revoked,3=expired',
  `revoked_at`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '吊销时间戳秒，0=未吊销',
  `revoked_reason` VARCHAR(128) NULL COMMENT '吊销原因，如 logout,password_changed,token_reuse_detected',
  `expired_at`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '过期时间戳秒',
  `last_used_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最近使用时间戳秒',

  `created_at`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间戳秒',
  `updated_at`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间戳秒',

  PRIMARY KEY (`id`),

  UNIQUE KEY `uk_refresh_token_hash` (`token_hash`),
  KEY `idx_refresh_user` (`tenant_id`,`user_id`),
  KEY `idx_refresh_user_status` (`tenant_id`,`user_id`,`status`),
  KEY `idx_refresh_family` (`tenant_id`,`user_id`,`token_family`),
  KEY `idx_refresh_device` (`tenant_id`,`user_id`,`device_id`),
  KEY `idx_refresh_platform` (`tenant_id`,`platform`),
  KEY `idx_refresh_expired_at` (`expired_at`),
  KEY `idx_refresh_last_used_at` (`last_used_at`),

  CONSTRAINT `fk_refresh_user`
    FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户RefreshToken表';