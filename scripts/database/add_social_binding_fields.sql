-- 添加社交平台绑定字段到 user_profiles 表
-- 执行命令：docker-compose exec -T mysql mysql -u pocw_user -ppocw_password pocw_db < scripts/database/add_social_binding_fields.sql

USE pocw_db;

-- 添加 Telegram 和 Discord 绑定字段
-- 检查并添加 telegram_id 字段
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = 'pocw_db' AND TABLE_NAME = 'user_profiles' AND COLUMN_NAME = 'telegram_id') = 0,
    'ALTER TABLE user_profiles ADD COLUMN telegram_id VARCHAR(100) NULL COMMENT ''Telegram user ID'' AFTER twitter_id',
    'SELECT ''telegram_id column already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 检查并添加 discord_id 字段
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.COLUMNS 
     WHERE TABLE_SCHEMA = 'pocw_db' AND TABLE_NAME = 'user_profiles' AND COLUMN_NAME = 'discord_id') = 0,
    'ALTER TABLE user_profiles ADD COLUMN discord_id VARCHAR(100) NULL COMMENT ''Discord user ID'' AFTER telegram_id',
    'SELECT ''discord_id column already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 添加索引以提高查询性能（如果不存在）
-- 检查并创建 telegram_id 索引
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = 'pocw_db' AND TABLE_NAME = 'user_profiles' AND INDEX_NAME = 'idx_user_profiles_telegram_id') = 0,
    'CREATE INDEX idx_user_profiles_telegram_id ON user_profiles(telegram_id)',
    'SELECT ''idx_user_profiles_telegram_id index already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 检查并创建 discord_id 索引
SET @sql = (SELECT IF(
    (SELECT COUNT(*) FROM INFORMATION_SCHEMA.STATISTICS 
     WHERE TABLE_SCHEMA = 'pocw_db' AND TABLE_NAME = 'user_profiles' AND INDEX_NAME = 'idx_user_profiles_discord_id') = 0,
    'CREATE INDEX idx_user_profiles_discord_id ON user_profiles(discord_id)',
    'SELECT ''idx_user_profiles_discord_id index already exists'' as message'
));
PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- 验证修改
DESCRIBE user_profiles;
