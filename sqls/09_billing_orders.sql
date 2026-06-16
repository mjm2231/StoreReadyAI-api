

CREATE TABLE IF NOT EXISTS `billing_orders` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `user_id` BIGINT UNSIGNED NOT NULL COMMENT '关联 users.id',
  `uid` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户对外 UID',

  `platform` VARCHAR(16) NOT NULL COMMENT '平台: ios/android',
  `product_id` VARCHAR(128) NOT NULL COMMENT '商店商品ID',
  `subscription_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '订阅ID（可选）',
  `base_plan_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT 'Google Base Plan ID（可选）',

  `order_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '订单号/交易号',
  `original_order_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '原始订单号/原始交易号',
  `purchase_token` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'Google purchase token / Apple receipt token',
  `receipt_data` MEDIUMTEXT NULL COMMENT '原始 receipt / JWS / base64 数据（可选）',

  `purchase_state` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '订单状态: purchased/pending/canceled/refunded/expired 等',
  `acknowledged` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否已 acknowledge / complete',
  `auto_renewing` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '是否自动续费',

  `purchase_time` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '购买时间（秒级时间戳）',
  `expire_time` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '到期时间（秒级时间戳）',

  `currency` VARCHAR(8) NOT NULL DEFAULT '' COMMENT '币种（可选）',
  `amount_micros` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '金额（微单位，可选）',

  `verify_status` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '校验状态: pending/success/failed',
  `verify_error_code` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '最近一次校验错误码',
  `verify_error_message` VARCHAR(255) NOT NULL DEFAULT '' COMMENT '最近一次校验错误信息',
  `last_verified_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '最近一次校验时间（秒级时间戳）',

  `raw_payload` JSON NULL COMMENT '平台原始返回负载',

  `created_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间（秒级时间戳）',
  `updated_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间（秒级时间戳）',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_order_platform_token` (`platform`, `purchase_token`),
  KEY `idx_billing_order_user` (`tenant_id`, `user_id`),
  KEY `idx_billing_order_uid` (`tenant_id`, `uid`),
  KEY `idx_billing_order_order_id` (`platform`, `order_id`),
  KEY `idx_billing_order_original_order_id` (`platform`, `original_order_id`),
  KEY `idx_billing_order_verify_status` (`verify_status`),
  KEY `idx_billing_order_expire_time` (`expire_time`),
  KEY `idx_billing_order_updated_at` (`updated_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Billing 订单流水表';