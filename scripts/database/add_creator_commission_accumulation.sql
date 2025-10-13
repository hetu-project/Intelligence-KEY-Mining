-- Add creator commission accumulation table
-- This table stores accumulated fractional commissions for task creators
-- 执行命令：docker-compose exec -T mysql mysql -u pocw_user -ppocw_password pocw_db < scripts/database/add_creator_commission_accumulation.sql

USE pocw_db;

CREATE TABLE IF NOT EXISTS creator_commission_accumulation (
    creator_wallet VARCHAR(42) PRIMARY KEY COMMENT 'Creator wallet address',
    accumulated_commission DECIMAL(10,3) NOT NULL DEFAULT 0.000 COMMENT 'Accumulated commission (supports 3 decimal places)',
    total_distributed INT NOT NULL DEFAULT 0 COMMENT 'Total integer points distributed so far',
    last_updated TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP COMMENT 'Last update time',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Record creation time',
    
    INDEX idx_creator_wallet (creator_wallet),
    INDEX idx_last_updated (last_updated),
    
    FOREIGN KEY (creator_wallet) REFERENCES user_profiles(wallet_address) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci 
COMMENT='Accumulates fractional creator commissions across PoCW rounds';

-- 验证创建结果
DESCRIBE creator_commission_accumulation;
