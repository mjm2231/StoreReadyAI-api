ALTER TABLE projects
ADD COLUMN description TEXT NULL COMMENT '项目描述，用于补充说明应用功能、目标用户、核心卖点等'
AFTER name;