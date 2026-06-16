

-- 用户反馈表：用于收集 App 内用户提交的问题反馈、建议、投诉等信息，便于后台跟进处理。
CREATE TABLE IF NOT EXISTS `user_feedbacks` (
  `id` BIGINT UNSIGNED NOT NULL AUTO_INCREMENT COMMENT '主键ID',
  `tenant_id` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '租户ID',
  `uid` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '业务用户UID，未登录可为0',

  `category` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '反馈分类：1普通反馈 2问题报错 3功能建议 4支付订阅 5账号登录 6其他',
  `title` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '反馈标题，客户端可选填',
  `content` TEXT NOT NULL COMMENT '反馈内容',
  `contact` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '用户联系方式：邮箱/手机号/其他',

  `status` TINYINT UNSIGNED NOT NULL DEFAULT 1 COMMENT '处理状态：1待处理 2处理中 3已处理 4已关闭',
  `priority` TINYINT UNSIGNED NOT NULL DEFAULT 2 COMMENT '优先级：1低 2普通 3高 4紧急',

  `reply_content` TEXT NULL COMMENT '后台回复内容',
  `handled_by` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '处理人管理员ID',
  `handled_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '处理时间，Unix秒',

  `app_version` VARCHAR(32) NOT NULL DEFAULT '' COMMENT 'App版本号',
  `build_number` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '构建号',
  `platform` VARCHAR(16) NOT NULL DEFAULT '' COMMENT '平台：ios/android/web',
  `device_model` VARCHAR(128) NOT NULL DEFAULT '' COMMENT '设备型号',
  `os_version` VARCHAR(64) NOT NULL DEFAULT '' COMMENT '系统版本',
  `locale` VARCHAR(32) NOT NULL DEFAULT '' COMMENT '客户端语言，如 zh-CN/en-US',

  `extra` JSON NULL COMMENT '扩展信息JSON，如页面路径、截图URL、错误日志ID等',

  `created_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '创建时间，Unix秒',
  `updated_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '更新时间，Unix秒',
  `deleted_at` BIGINT UNSIGNED NOT NULL DEFAULT 0 COMMENT '软删除时间，0表示未删除',

  PRIMARY KEY (`id`),
  KEY `idx_feedback_tenant_uid` (`tenant_id`, `uid`),
  KEY `idx_feedback_status_priority` (`tenant_id`, `status`, `priority`),
  KEY `idx_feedback_category_created` (`tenant_id`, `category`, `created_at`),
  KEY `idx_feedback_created` (`tenant_id`, `created_at`),
  KEY `idx_feedback_deleted` (`tenant_id`, `deleted_at`)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='用户反馈表';