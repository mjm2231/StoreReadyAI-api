-- 04_user_settings.sql（全局提醒设置/时区/默认币种）
CREATE TABLE IF NOT EXISTS `user_settings` (
  `id`                       BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id`                BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `user_id`                  BIGINT UNSIGNED NOT NULL COMMENT '用户ID',

  `default_currency`         CHAR(3) NOT NULL DEFAULT 'USD' COMMENT '默认币种',
  `default_remind_before_days` SMALLINT UNSIGNED NOT NULL DEFAULT 3 COMMENT '默认提前提醒天数(0~30)',
  `default_remind_on_day`      TINYINT NOT NULL DEFAULT 1 COMMENT '默认到期当天提醒(0/1)',
  `notification_enabled`     TINYINT NOT NULL DEFAULT 1 COMMENT '通知总开关(0/1)',
  `notification_time`        TIME NOT NULL DEFAULT '09:00:00' COMMENT '默认通知时间(本地时间，HH:MM:SS)',
  `timezone`                 VARCHAR(64) NOT NULL DEFAULT 'UTC' COMMENT '时区(IANA，如Asia/Shanghai)',

  `created_at`               BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间戳秒',
  `updated_at`               BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间戳秒(用于LWW冲突策略)',
  `deleted_at`               BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '软删除时间戳秒(0=未删除)',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_settings_user` (`tenant_id`,`user_id`),
  KEY `idx_settings_sync` (`tenant_id`,`updated_at`),

  CONSTRAINT `fk_settings_user`
    FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户全局设置';