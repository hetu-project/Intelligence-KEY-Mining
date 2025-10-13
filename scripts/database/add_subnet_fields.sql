-- 为 subnets 表添加 x_url 和 website 字段
-- 执行命令：docker-compose exec -T mysql mysql -u pocw_user -ppocw_password pocw_db < scripts/database/add_subnet_fields.sql

USE pocw_db;

-- 添加项目 X/Twitter URL 和官网字段
ALTER TABLE subnets 
ADD COLUMN IF NOT EXISTS x_url VARCHAR(500) NULL COMMENT 'Project X/Twitter URL' AFTER icon,
ADD COLUMN IF NOT EXISTS website VARCHAR(500) NULL COMMENT 'Project official website' AFTER x_url;

-- 验证修改
DESCRIBE subnets;
