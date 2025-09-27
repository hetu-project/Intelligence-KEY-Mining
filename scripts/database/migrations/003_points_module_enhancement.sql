-- ============================================
-- Points Module Enhancement Migration
-- Adds subnet management, task expiry, duplicate prevention, and NFT integration
-- ============================================

-- 1. Subnets table - 子网管理
CREATE TABLE IF NOT EXISTS subnets (
    id VARCHAR(36) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    icon VARCHAR(500) NULL,
    creator_wallet VARCHAR(42) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    status VARCHAR(20) DEFAULT 'active',
    
    UNIQUE KEY unique_subnet_name (name),
    INDEX idx_creator_wallet (creator_wallet),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    
    FOREIGN KEY (creator_wallet) REFERENCES user_profiles(wallet_address) ON DELETE CASCADE
);

-- 2. Add subnet_id and expires_at to tasks table
ALTER TABLE tasks 
ADD COLUMN subnet_id VARCHAR(36) NULL,
ADD COLUMN expires_at TIMESTAMP NULL,
ADD INDEX idx_subnet_id (subnet_id),
ADD INDEX idx_expires_at (expires_at),
ADD FOREIGN KEY (subnet_id) REFERENCES subnets(id) ON DELETE SET NULL;

-- 3. User task completions - 防重复完成任务
CREATE TABLE IF NOT EXISTS user_task_completions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_wallet VARCHAR(42) NOT NULL,
    task_id VARCHAR(36) NOT NULL,
    subnet_id VARCHAR(36) NULL,
    completed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    vlc_increment INT DEFAULT 0,
    points_earned INT DEFAULT 0,
    
    UNIQUE KEY unique_user_task (user_wallet, task_id),
    INDEX idx_user_wallet (user_wallet),
    INDEX idx_task_id (task_id),
    INDEX idx_subnet_id (subnet_id),
    INDEX idx_completed_at (completed_at),
    
    FOREIGN KEY (user_wallet) REFERENCES user_profiles(wallet_address) ON DELETE CASCADE,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    FOREIGN KEY (subnet_id) REFERENCES subnets(id) ON DELETE SET NULL
);

-- 4. Daily distribution log - 每日分发日志防重复
CREATE TABLE IF NOT EXISTS daily_distribution_log (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    distribution_date DATE NOT NULL,
    status VARCHAR(20) NOT NULL, -- 'completed', 'failed', 'in_progress'
    total_points_distributed INT DEFAULT 0,
    total_users_affected INT DEFAULT 0,
    total_tasks_processed INT DEFAULT 0,
    started_at TIMESTAMP NOT NULL,
    completed_at TIMESTAMP NULL,
    error_message TEXT NULL,
    metadata JSON NULL, -- 存储分发详情
    
    UNIQUE KEY unique_distribution_date (distribution_date),
    INDEX idx_status (status),
    INDEX idx_started_at (started_at)
);

-- 5. NFT ownership cache - NFT持有状态缓存
CREATE TABLE IF NOT EXISTS nft_ownership_cache (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_wallet VARCHAR(42) NOT NULL,
    has_nft BOOLEAN NOT NULL,
    checked_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    expires_at TIMESTAMP NOT NULL, -- 缓存过期时间
    api_response JSON NULL, -- 存储API响应详情
    
    UNIQUE KEY unique_user_wallet (user_wallet),
    INDEX idx_user_wallet (user_wallet),
    INDEX idx_has_nft (has_nft),
    INDEX idx_expires_at (expires_at)
);

-- 6. Points history enhancement - 增强积分历史记录
-- 添加新的source类型和更多字段
ALTER TABLE points_history 
ADD COLUMN subnet_id VARCHAR(36) NULL,
ADD COLUMN nft_multiplier DECIMAL(3,2) DEFAULT 1.00,
ADD COLUMN base_points INT DEFAULT 0,
ADD COLUMN bonus_points INT DEFAULT 0,
ADD COLUMN description TEXT NULL,
ADD INDEX idx_subnet_id (subnet_id),
ADD INDEX idx_nft_multiplier (nft_multiplier),
ADD FOREIGN KEY (subnet_id) REFERENCES subnets(id) ON DELETE SET NULL;

-- 7. Invitation rewards tracking - 邀请奖励追踪
CREATE TABLE IF NOT EXISTS invitation_rewards (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    inviter_wallet VARCHAR(42) NOT NULL,
    invitee_wallet VARCHAR(42) NOT NULL,
    reward_points INT NOT NULL,
    inviter_has_nft BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    points_history_id BIGINT NULL, -- 关联积分历史记录
    
    UNIQUE KEY unique_invitation (inviter_wallet, invitee_wallet),
    INDEX idx_inviter_wallet (inviter_wallet),
    INDEX idx_invitee_wallet (invitee_wallet),
    INDEX idx_created_at (created_at),
    
    FOREIGN KEY (inviter_wallet) REFERENCES user_profiles(wallet_address) ON DELETE CASCADE,
    FOREIGN KEY (invitee_wallet) REFERENCES user_profiles(wallet_address) ON DELETE CASCADE,
    FOREIGN KEY (points_history_id) REFERENCES points_history(id) ON DELETE SET NULL
);

-- 8. Subnet statistics view - 子网统计视图
CREATE OR REPLACE VIEW subnet_stats AS
SELECT 
    s.id as subnet_id,
    s.name as subnet_name,
    s.icon as subnet_icon,
    s.creator_wallet,
    s.created_at as subnet_created_at,
    COUNT(DISTINCT t.id) as total_tasks,
    COUNT(DISTINCT CASE WHEN t.status = 'CONFIRMED' THEN t.id END) as completed_tasks,
    COUNT(DISTINCT utc.user_wallet) as unique_users,
    COALESCE(SUM(utc.points_earned), 0) as total_points_distributed,
    COALESCE(SUM(CASE WHEN DATE(utc.completed_at) = CURDATE() THEN utc.points_earned ELSE 0 END), 0) as today_points_distributed,
    COUNT(DISTINCT CASE WHEN DATE(utc.completed_at) = CURDATE() THEN utc.user_wallet END) as today_active_users
FROM subnets s
LEFT JOIN tasks t ON s.id = t.subnet_id
LEFT JOIN user_task_completions utc ON t.id = utc.task_id
WHERE s.status = 'active'
GROUP BY s.id;

-- 9. User points ranking view - 用户积分排名视图
CREATE OR REPLACE VIEW user_points_ranking AS
SELECT 
    up.wallet_address,
    up.display_name,
    up.total_points,
    up.today_contribution,
    up.registration_date,
    ROW_NUMBER() OVER (ORDER BY up.total_points DESC) as ranking,
    COUNT(DISTINCT utc.task_id) as completed_tasks_count,
    COUNT(DISTINCT utc.subnet_id) as active_subnets_count,
    COALESCE(noc.has_nft, FALSE) as has_nft
FROM user_profiles up
LEFT JOIN user_task_completions utc ON up.wallet_address = utc.user_wallet
LEFT JOIN nft_ownership_cache noc ON up.wallet_address = noc.user_wallet 
    AND noc.expires_at > NOW()
GROUP BY up.wallet_address
ORDER BY up.total_points DESC;

-- 10. Enhanced points history view - 增强积分历史视图
CREATE OR REPLACE VIEW points_history_detailed AS
SELECT 
    ph.id,
    ph.wallet_address,
    up.display_name,
    ph.date,
    ph.source,
    ph.points,
    ph.base_points,
    ph.bonus_points,
    ph.nft_multiplier,
    ph.description,
    ph.tx_ref,
    ph.task_id,
    ph.subnet_id,
    s.name as subnet_name,
    ph.created_at,
    CASE 
        WHEN ph.source LIKE '%Task%' THEN 'task_completion'
        WHEN ph.source LIKE '%NFT%' THEN 'nft_bonus'
        WHEN ph.source LIKE '%Invitation%' THEN 'invitation_reward'
        ELSE 'other'
    END as points_category
FROM points_history ph
LEFT JOIN user_profiles up ON ph.wallet_address = up.wallet_address
LEFT JOIN subnets s ON ph.subnet_id = s.id
ORDER BY ph.created_at DESC;

-- 11. Update existing triggers for new fields
DROP TRIGGER IF EXISTS update_user_points;

DELIMITER //
CREATE TRIGGER update_user_points 
AFTER INSERT ON points_history
FOR EACH ROW
BEGIN
    UPDATE user_profiles 
    SET total_points = total_points + NEW.points,
        today_contribution = CASE 
            WHEN NEW.date = CURDATE() THEN today_contribution + NEW.points 
            ELSE today_contribution 
        END,
        updated_at = CURRENT_TIMESTAMP
    WHERE wallet_address = NEW.wallet_address;
END//
DELIMITER ;

-- 12. Add stored procedures for common operations

DELIMITER //

-- Procedure to check if daily distribution is completed
CREATE PROCEDURE IF NOT EXISTS CheckDailyDistribution(IN check_date DATE)
BEGIN
    SELECT 
        CASE 
            WHEN COUNT(*) > 0 AND status = 'completed' THEN TRUE
            ELSE FALSE
        END as is_completed,
        status,
        total_points_distributed,
        total_users_affected,
        started_at,
        completed_at
    FROM daily_distribution_log 
    WHERE distribution_date = check_date;
END//

-- Procedure to get subnet statistics
CREATE PROCEDURE IF NOT EXISTS GetSubnetStatistics()
BEGIN
    SELECT * FROM subnet_stats ORDER BY total_points_distributed DESC;
END//

-- Procedure to get user ranking with pagination
CREATE PROCEDURE IF NOT EXISTS GetUserRanking(IN page_offset INT, IN page_limit INT)
BEGIN
    SELECT * FROM user_points_ranking 
    LIMIT page_offset, page_limit;
END//

-- Procedure to clean expired NFT cache
CREATE PROCEDURE IF NOT EXISTS CleanExpiredNFTCache()
BEGIN
    DELETE FROM nft_ownership_cache 
    WHERE expires_at < NOW();
    
    SELECT ROW_COUNT() as deleted_count;
END//

DELIMITER ;

-- 13. Insert sample data for testing (optional)
-- INSERT INTO subnets (id, name, icon, creator_wallet) VALUES
-- ('subnet-001', 'DeFi Protocol', 'https://example.com/defi-icon.png', '0x1234567890123456789012345678901234567890'),
-- ('subnet-002', 'NFT Marketplace', 'https://example.com/nft-icon.png', '0x2345678901234567890123456789012345678901');

-- 14. Add indexes for performance optimization
CREATE INDEX idx_tasks_subnet_status ON tasks(subnet_id, status);
CREATE INDEX idx_tasks_expires_status ON tasks(expires_at, status);
CREATE INDEX idx_points_history_date_source ON points_history(date, source);
CREATE INDEX idx_user_task_completions_date ON user_task_completions(completed_at);

-- 15. Add constraints for data integrity
ALTER TABLE tasks ADD CONSTRAINT chk_expires_at_future 
CHECK (expires_at IS NULL OR expires_at > created_at);

ALTER TABLE daily_distribution_log ADD CONSTRAINT chk_completed_after_started
CHECK (completed_at IS NULL OR completed_at >= started_at);

ALTER TABLE points_history ADD CONSTRAINT chk_nft_multiplier_positive
CHECK (nft_multiplier > 0 AND nft_multiplier <= 10.0);

-- Migration completed successfully
SELECT 'Points Module Enhancement Migration Completed Successfully' as status;
