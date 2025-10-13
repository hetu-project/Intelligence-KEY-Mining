-- 添加社交平台绑定字段到 user_profiles 表
-- 执行命令：docker-compose exec -T mysql mysql -u pocw_user -ppocw_password pocw_db < scripts/database/add_social_binding_fields.sql

USE pocw_db;

-- 添加 Telegram 和 Discord 绑定字段
ALTER TABLE user_profiles 
ADD COLUMN IF NOT EXISTS telegram_id VARCHAR(100) NULL COMMENT 'Telegram user ID' AFTER twitter_id,
ADD COLUMN IF NOT EXISTS discord_id VARCHAR(100) NULL COMMENT 'Discord user ID' AFTER telegram_id;

-- 添加索引以提高查询性能（如果不存在）
CREATE INDEX IF NOT EXISTS idx_user_profiles_telegram_id ON user_profiles(telegram_id);
CREATE INDEX IF NOT EXISTS idx_user_profiles_discord_id ON user_profiles(discord_id);

-- 验证修改
DESCRIBE user_profiles;
