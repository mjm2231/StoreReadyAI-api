

CREATE TABLE IF NOT EXISTS `billing_products` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',

  `product_code` VARCHAR(64) NOT NULL COMMENT '业务商品编码，如 vip_monthly / vip_yearly',
  `platform` VARCHAR(16) NOT NULL COMMENT '平台: ios/android',
  `store_product_id` VARCHAR(128) NOT NULL COMMENT '商店商品ID（App Store / Google Play）',

  `product_type` VARCHAR(32) NOT NULL DEFAULT 'subscription' COMMENT '商品类型: subscription / consumable / non_consumable',
  `subscription_group` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '订阅组（用于 iOS 分组 / 业务分组）',

  `status` TINYINT NOT NULL DEFAULT 1 COMMENT '状态: 1=启用, 2=禁用',
  `is_recommended` TINYINT NOT NULL DEFAULT 0 COMMENT '是否推荐（用于 UI 默认高亮）',
  `sort` INT NOT NULL DEFAULT 0 COMMENT '排序权重（越小越靠前）',

  `created_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间（秒级时间戳）',
  `updated_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间（秒级时间戳）',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_billing_product_code_platform` (`tenant_id`, `product_code`, `platform`),
  UNIQUE KEY `uk_billing_product_store_id` (`platform`, `store_product_id`),
  KEY `idx_billing_product_status` (`status`),
  KEY `idx_billing_product_sort` (`sort`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Billing 商品配置表';