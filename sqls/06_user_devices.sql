-- 06_user_devices.sql（多设备同步：设备登记 & last_seen/last_sync）
CREATE TABLE IF NOT EXISTS `user_devices` (
  `id`          BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id`   BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `user_id`     BIGINT UNSIGNED NOT NULL COMMENT '用户ID',

  `device_id`   VARCHAR(128) NOT NULL COMMENT '设备唯一ID(客户端生成/持久化)',
  `platform`    TINYINT NOT NULL COMMENT '平台:1=ios,2=android,3=web,9=unknown',
  `device_name` VARCHAR(128) NULL COMMENT '设备名(可选)',
  `app_version` VARCHAR(32)  NULL COMMENT 'App版本(可选)',

  `push_token` VARCHAR(256) NULL COMMENT '推送Token(APNs/FCM，可选)',
  `last_ip`    VARCHAR(64)  NULL COMMENT '最近IP(可选)',
  `user_agent` VARCHAR(256) NULL COMMENT 'User-Agent(可选)',

  `status`     TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1=active,2=revoked',

  `last_seen_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最近活跃时间戳秒',
  `last_sync_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最近同步时间戳秒(可选)',

  `created_at`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间戳秒',
  `updated_at`  BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间戳秒',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_user_device` (`tenant_id`,`user_id`,`device_id`),
  KEY `idx_device_seen` (`tenant_id`,`user_id`,`last_seen_at`),
  KEY `idx_device_status` (`tenant_id`,`user_id`,`status`),

  CONSTRAINT `fk_device_user`
    FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户设备表(同步/多端)';