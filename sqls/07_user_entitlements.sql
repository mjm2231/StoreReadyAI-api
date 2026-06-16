-- 07_user_entitlements.sql（VIP 状态来源：最小可用）
CREATE TABLE IF NOT EXISTS `user_entitlements` (
  `id`             BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `user_id`        BIGINT UNSIGNED NOT NULL COMMENT '用户ID',

  `entitlement`    VARCHAR(32) NOT NULL COMMENT '权益标识: vip',
  `product_code`   VARCHAR(64) NOT NULL DEFAULT '' COMMENT '内部商品编码，如 vip_monthly/vip_yearly',
  `product_id`     VARCHAR(128) NOT NULL DEFAULT '' COMMENT '商店商品ID，如 Google Play/App Store productId',
  `source`         TINYINT NOT NULL DEFAULT 0 COMMENT '来源:0=manual,1=ios_iap,2=google_play,3=promo',
  `status`         TINYINT NOT NULL DEFAULT 1 COMMENT '状态:1=active,2=expired,3=revoked',
  `started_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '开始时间戳秒',
  `expired_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '到期时间戳秒',

  `ref_id`         VARCHAR(128) NULL COMMENT '外部订单/交易ID(可选)',
  `auto_renew`     TINYINT NOT NULL DEFAULT 0 COMMENT '是否自动续期(0/1，可选)',
  `created_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间戳秒',
  `updated_at`     BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间戳秒',

  PRIMARY KEY (`id`),
  KEY `idx_ent_user` (`tenant_id`,`user_id`,`entitlement`,`status`),
  -- 幂等：ref_id 非空时可防止重复写入（MySQL 允许多个 NULL）
  UNIQUE KEY `uk_ent_ref` (`tenant_id`,`entitlement`,`ref_id`),
  KEY `idx_ent_expired` (`tenant_id`,`user_id`,`entitlement`,`expired_at`),

  CONSTRAINT `fk_ent_user`
    FOREIGN KEY (`user_id`) REFERENCES `users` (`id`)
    ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci COMMENT='用户权益(VIP)';