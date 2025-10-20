-- 创建白名单用户表
-- 执行命令：docker-compose exec -T mysql mysql -u pocw_user -ppocw_password pocw_db < scripts/database/add_whitelist_users.sql

USE pocw_db;

-- 创建白名单用户表
CREATE TABLE IF NOT EXISTS whitelist_users (
    wallet_address VARCHAR(42) PRIMARY KEY COMMENT 'User wallet address',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'When user was added to whitelist',
    created_by VARCHAR(42) NOT NULL COMMENT 'Admin who added this user',
    reason VARCHAR(500) NULL COMMENT 'Reason for whitelisting',
    status ENUM('active', 'inactive') DEFAULT 'active' COMMENT 'Whitelist status',
    
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    
    FOREIGN KEY (wallet_address) REFERENCES user_profiles(wallet_address) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_0900_ai_ci 
COMMENT='Whitelist users who can create multiple subnets';

-- 验证创建结果
DESCRIBE whitelist_users;
