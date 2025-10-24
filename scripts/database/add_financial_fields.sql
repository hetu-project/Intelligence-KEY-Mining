-- 为 subnets 表添加 TVL 和 Valuation 字段
-- 执行命令：docker-compose exec -T mysql mysql -u pocw_user -ppocw_password pocw_db < scripts/database/add_financial_fields.sql

USE pocw_db;

-- 添加 TVL 和 Valuation 字段
ALTER TABLE subnets 
ADD COLUMN IF NOT EXISTS tvl DECIMAL(15,2) DEFAULT 0.00 COMMENT 'Total Value Locked in USD' AFTER website,
ADD COLUMN IF NOT EXISTS valuation DECIMAL(15,2) DEFAULT 0.00 COMMENT 'Project Valuation in USD' AFTER tvl;

-- 验证修改
DESCRIBE subnets;
