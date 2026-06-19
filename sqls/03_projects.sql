CREATE TABLE projects (
  id BIGINT AUTO_INCREMENT PRIMARY KEY COMMENT '项目ID',
  tenant_id BIGINT NOT NULL COMMENT '租户ID（MVP固定0，后期多租户扩展）',
  user_id BIGINT NOT NULL COMMENT '用户ID',

  name VARCHAR(100) NOT NULL COMMENT '项目名称',
  description TEXT NULL COMMENT '项目描述，用于补充说明应用功能、目标用户、核心卖点等',
  platform VARCHAR(32) NOT NULL DEFAULT '' COMMENT '平台',
  status VARCHAR(32) NOT NULL DEFAULT 'draft' COMMENT '状态',

  created_at BIGINT NOT NULL DEFAULT 0,
  updated_at BIGINT NOT NULL DEFAULT 0,
  deleted_at BIGINT NOT NULL DEFAULT 0,

  INDEX idx_user_projects (tenant_id, user_id, deleted_at, created_at),
  INDEX idx_tenant_status (tenant_id, status, deleted_at)
);