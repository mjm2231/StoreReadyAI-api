CREATE TABLE project_store_infos (
  id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '上架资料ID',
  tenant_id BIGINT NOT NULL DEFAULT 0 COMMENT '租户ID',
  user_id BIGINT NOT NULL DEFAULT 0 COMMENT '用户ID',
  project_id BIGINT NOT NULL DEFAULT 0 COMMENT '项目ID',

  app_name VARCHAR(100) NOT NULL DEFAULT '' COMMENT 'App名称',
  subtitle VARCHAR(255) NOT NULL DEFAULT '' COMMENT '副标题',
  keywords VARCHAR(500) NOT NULL DEFAULT '' COMMENT '关键词，逗号分隔',
  short_description VARCHAR(500) NOT NULL DEFAULT '' COMMENT '短描述',
  full_description TEXT COMMENT '完整描述',
  category VARCHAR(100) NOT NULL DEFAULT '' COMMENT '应用分类',
  content_rating VARCHAR(100) NOT NULL DEFAULT '' COMMENT '内容分级',
  privacy_policy_url VARCHAR(512) NOT NULL DEFAULT '' COMMENT '隐私政策URL',
  support_url VARCHAR(512) NOT NULL DEFAULT '' COMMENT '支持URL',
  marketing_url VARCHAR(512) NOT NULL DEFAULT '' COMMENT '营销URL',
  copyright VARCHAR(255) NOT NULL DEFAULT '' COMMENT '版权信息',
  contact_email VARCHAR(255) NOT NULL DEFAULT '' COMMENT '联系邮箱',

  status VARCHAR(32) NOT NULL DEFAULT 'draft' COMMENT '状态：draft/ready',
  created_at BIGINT NOT NULL DEFAULT 0 COMMENT '创建时间戳秒',
  updated_at BIGINT NOT NULL DEFAULT 0 COMMENT '更新时间戳秒',
  deleted_at BIGINT NOT NULL DEFAULT 0 COMMENT '软删除时间戳秒，0表示未删除',

  UNIQUE KEY uk_project_store_info (tenant_id, user_id, project_id, deleted_at),
  KEY idx_project_store_infos_user (tenant_id, user_id, deleted_at, updated_at),
  KEY idx_project_store_infos_project (tenant_id, project_id, deleted_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COMMENT='项目上架资料表';