

CREATE TABLE IF NOT EXISTS `billing_events` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `user_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关联 users.id（可选）',
  `uid` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户对外 UID（可选）',

  `platform` VARCHAR(16) NOT NULL COMMENT '平台: ios/android',
  `event_type` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '事件类型: verify/restore/rtdn/apple_notification/refund/revoke/renew 等',
  `event_source` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '事件来源: client/google/apple/system',

  `order_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '订单号/交易号',
  `original_order_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '原始订单号/原始交易号',
  `purchase_token` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'Google purchase token / Apple receipt token',
  `product_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '商品ID（可选）',

  `event_status` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '事件处理状态: pending/processed/failed/ignored',
  `error_code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '处理失败时的错误码',
  `error_message` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '处理失败时的错误信息',

  `event_time` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '事件发生时间（秒级时间戳）',
  `processed_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '事件处理完成时间（秒级时间戳）',

  `raw_payload` JSON NULL COMMENT '平台原始通知/原始请求负载',

  `created_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间（秒级时间戳）',
  `updated_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间（秒级时间戳）',

  PRIMARY KEY (`id`),
  KEY `idx_billing_event_user` (`tenant_id`, `user_id`),
  KEY `idx_billing_event_uid` (`tenant_id`, `uid`),
  KEY `idx_billing_event_platform_type` (`platform`, `event_type`),
  KEY `idx_billing_event_order_id` (`platform`, `order_id`),
  KEY `idx_billing_event_original_order_id` (`platform`, `original_order_id`),
  KEY `idx_billing_event_purchase_token` (`platform`, `purchase_token`),
  KEY `idx_billing_event_status` (`event_status`),
  KEY `idx_billing_event_event_time` (`event_time`),
  KEY `idx_billing_event_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Billing 事件流水表';