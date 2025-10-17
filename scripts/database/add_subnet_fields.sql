-- 为 subnets 表添加 x_url 和 website 字段
-- 执行命令：docker-compose exec -T mysql mysql -u pocw_user -ppocw_password pocw_db < scripts/database/add_subnet_fields.sql

USE pocw_db;

-- 添加项目 X/Twitter URL 和官网字段
-- 检查并添加 x_url 字段
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = 'pocw_db' AND TABLE_NAME = 'subnets' AND COLUMN_NAME = 'x_url') = 0,
    'ALTER TABLE subnets ADD COLUMN x_url VARCHAR(500) NULL COMMENT ''Project X/Twitter URL'' AFTER icon',
    'SELECT ''x_url column already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 检查并添加 website 字段
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = 'pocw_db' AND TABLE_NAME = 'subnets' AND COLUMN_NAME = 'website') = 0,
    'ALTER TABLE subnets ADD COLUMN website VARCHAR(500) NULL COMMENT ''Project official website'' AFTER x_url',
    'SELECT ''website column already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 验证修改
DESCRIBE subnets;
