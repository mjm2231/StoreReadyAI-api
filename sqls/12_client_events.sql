
-- 客户端事件表，用于记录客户端上报的各种事件，如查询商品、发起支付、登录登出等，便于后续分析用户行为和排查问题。
CREATE TABLE IF NOT EXISTS `client_events` (
  `id`              BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键',
  `tenant_id`       BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `uid`             BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '业务用户UID',
  `event_id` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '客户端事件唯一ID',
  `received_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '服务端接收时间(秒)',
  `event_group`     VARCHAR(32) NOT NULL DEFAULT '' COMMENT '事件分组: billing/sync/login/app 等',
  `event_name`      VARCHAR(64) NOT NULL DEFAULT '' COMMENT '事件名称: billing_query_products 等',
  `event_source`    VARCHAR(32) NOT NULL DEFAULT '' COMMENT '事件来源: android/ios/web/server',
  `platform`        VARCHAR(16) NOT NULL DEFAULT '' COMMENT '平台: android/ios/web',

  `app_version`     VARCHAR(32) NOT NULL DEFAULT '' COMMENT '应用版本号',
  `build_number`    VARCHAR(32) NOT NULL DEFAULT '' COMMENT '构建号',
  `package_name`    VARCHAR(128) NOT NULL DEFAULT '' COMMENT '应用包名',

  `device_id`       VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备ID',
  `device_model`    VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备型号',
  `os_version`      VARCHAR(64) NOT NULL DEFAULT '' COMMENT '系统版本',

  `network_type`    VARCHAR(32) NOT NULL DEFAULT '' COMMENT '网络类型: wifi/cellular/unknown',
  `store_available` TINYINT(1) NOT NULL DEFAULT 0 COMMENT '商店是否可用: 0否 1是',

  `event_code`      VARCHAR(64) NOT NULL DEFAULT '' COMMENT '业务码/错误码: store_unavailable/products_not_found',
  `event_message`   VARCHAR(255) NOT NULL DEFAULT '' COMMENT '事件说明/错误摘要',
  `payload`         JSON NULL COMMENT '扩展载荷(JSON): product_ids/not_found_ids/query结果等',

  `created_at`      BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间(秒)',

  PRIMARY KEY (`id`),
  UNIQUE KEY `uk_client_event_event_id` (`event_id`),
  KEY `idx_client_event_uid` (`tenant_id`, `uid`),
  KEY `idx_client_event_group_name` (`event_group`, `event_name`),
  KEY `idx_client_event_platform_created` (`platform`, `created_at`),
  KEY `idx_client_event_created` (`created_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='客户端埋点事件表';