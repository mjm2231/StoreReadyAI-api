

CREATE TABLE IF NOT EXISTS `billing_transactions` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `user_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '关联 users.id',
  `uid` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '用户对外 UID',

  `platform` VARCHAR(16) NOT NULL COMMENT '平台: ios/android',
  `product_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '商品ID',

  `order_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '订单号/交易号',
  `original_order_id` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '原始订单号/原始交易号（订阅续费链路用）',
  `purchase_token` VARCHAR(512) NOT NULL DEFAULT '' COMMENT 'Google purchase token / Apple receipt token',

  `transaction_type` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '交易类型: purchase/renew/refund/revoke/restore',
  `transaction_state` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '交易状态: success/failed/pending',

  `amount_micros` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '金额（微单位）',
  `currency` VARCHAR(8) NOT NULL DEFAULT '' COMMENT '币种',

  `transaction_time` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '交易时间（秒级时间戳）',

  `raw_payload` JSON NULL COMMENT '平台原始交易数据',

  `created_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间（秒级时间戳）',
  `updated_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间（秒级时间戳）',

  PRIMARY KEY (`id`),
  KEY `idx_billing_tx_user` (`tenant_id`, `user_id`),
  KEY `idx_billing_tx_uid` (`tenant_id`, `uid`),
  KEY `idx_billing_tx_platform_order` (`platform`, `order_id`),
  KEY `idx_billing_tx_original_order` (`platform`, `original_order_id`),
  KEY `idx_billing_tx_purchase_token` (`platform`, `purchase_token`),
  KEY `idx_billing_tx_type` (`transaction_type`),
  KEY `idx_billing_tx_time` (`transaction_time`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Billing 交易流水表';