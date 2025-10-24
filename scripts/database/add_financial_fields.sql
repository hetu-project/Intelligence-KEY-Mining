-- 为 subnets 表添加 TVL 和 Valuation 字段
-- 执行命令：docker-compose exec -T mysql mysql -u pocw_user -ppocw_password pocw_db < scripts/database/add_financial_fields.sql

USE pocw_db;

-- 添加 TVL 和 Valuation 字段
-- 检查并添加 tvl 字段
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = 'pocw_db' AND TABLE_NAME = 'subnets' AND COLUMN_NAME = 'tvl') = 0,
    'ALTER TABLE subnets ADD COLUMN tvl DECIMAL(15,2) DEFAULT 0.00 COMMENT ''Total Value Locked in USD'' AFTER website',
    'SELECT ''tvl column already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 检查并添加 valuation 字段
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = 'pocw_db' AND TABLE_NAME = 'subnets' AND COLUMN_NAME = 'valuation') = 0,
    'ALTER TABLE subnets ADD COLUMN valuation DECIMAL(15,2) DEFAULT 0.00 COMMENT ''Project Valuation in USD'' AFTER tvl',
    'SELECT ''valuation column already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 验证修改
DESCRIBE subnets;
