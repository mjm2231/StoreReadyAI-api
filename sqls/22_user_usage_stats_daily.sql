CREATE TABLE  IF NOT EXISTS user_usage_stats_daily (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY COMMENT '主键ID',

  tenant_id BIGINT NOT NULL COMMENT '租户ID',
  user_id BIGINT NOT NULL COMMENT '用户ID',

  stat_date DATE NOT NULL COMMENT '统计日期',

  -- 核心指标
  subscription_count INT DEFAULT 0 COMMENT '订阅总数',
  reminder_count INT DEFAULT 0 COMMENT '提醒总数',

  created_subscriptions INT DEFAULT 0 COMMENT '当日新增订阅数',
  triggered_reminders INT DEFAULT 0 COMMENT '当日触发提醒数',

  -- 行为指标
  app_open_count INT DEFAULT 0 COMMENT '应用打开次数',
  active_minutes INT DEFAULT 0 COMMENT '活跃时长（分钟）',

  -- 转化相关
  is_vip TINYINT DEFAULT 0 COMMENT '是否为VIP：0否 1是',

  created_at BIGINT COMMENT '创建时间（Unix秒）',
  updated_at BIGINT COMMENT '更新时间（Unix秒）',

  UNIQUE KEY uk_user_date (tenant_id, user_id, stat_date)
) COMMENT='用户使用统计日报表';