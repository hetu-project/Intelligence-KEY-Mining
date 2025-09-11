-- Migration: Add Twitter task support
-- Version: 001
-- Description: Add database support for Twitter ID binding and new task types

-- ============================================
-- 1. Update tasks table to support new task types
-- ============================================

-- Add indexes for new task types
ALTER TABLE tasks ADD INDEX idx_task_type_status (task_type, status);
ALTER TABLE tasks ADD INDEX idx_user_wallet_task_type (user_wallet, task_type);

-- ============================================
-- 2. Create Twitter-task-related tables
-- ============================================

-- Twitter task details table
CREATE TABLE IF NOT EXISTS twitter_tasks (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_id VARCHAR(36) NOT NULL,
    twitter_id VARCHAR(100) NOT NULL,
    tweet_id VARCHAR(100) NOT NULL,
    project_name VARCHAR(200) NULL,
    project_icon VARCHAR(500) NULL,
    twitter_username VARCHAR(100) NULL,
    twitter_link VARCHAR(500) NULL,
    retweet_url VARCHAR(500) NULL,
    verification_status VARCHAR(30) DEFAULT 'pending',
    verified_at TIMESTAMP NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    INDEX idx_task_id (task_id),
    INDEX idx_twitter_id (twitter_id),
    INDEX idx_tweet_id (tweet_id),
    INDEX idx_verification_status (verification_status),
    INDEX idx_created_at (created_at)
);

-- ============================================
-- 3. Create batch-verification tables
-- ============================================

-- Batch verification record table
CREATE TABLE IF NOT EXISTS batch_verifications (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    batch_id VARCHAR(36) NOT NULL,
    task_id VARCHAR(36) NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    total_tasks INT DEFAULT 0,
    verified_tasks INT DEFAULT 0,
    unverified_tasks INT DEFAULT 0,
    vlc_increment INT DEFAULT 0,
    status VARCHAR(30) DEFAULT 'pending',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    INDEX idx_batch_id (batch_id),
    INDEX idx_task_id (task_id),
    INDEX idx_status (status),
    INDEX idx_start_time (start_time),
    INDEX idx_end_time (end_time)
);

-- Batch verification result details table
CREATE TABLE IF NOT EXISTS batch_verification_results (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    batch_id VARCHAR(36) NOT NULL,
    twitter_id VARCHAR(100) NOT NULL,
    tweet_id VARCHAR(100) NOT NULL,
    verified BOOLEAN NOT NULL,
    verification_details JSON NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_batch_id (batch_id),
    INDEX idx_twitter_id (twitter_id),
    INDEX idx_tweet_id (tweet_id),
    INDEX idx_verified (verified)
);

-- ============================================
-- 4. Create VLC event history table
-- ============================================

-- VLC event history table
CREATE TABLE IF NOT EXISTS vlc_events (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_id VARCHAR(36) NOT NULL,
    task_type VARCHAR(50) NOT NULL,
    stage VARCHAR(30) NOT NULL, -- 'submission' or 'verification'
    description TEXT NOT NULL,
    increment_count INT NOT NULL,
    vlc_before JSON NOT NULL,
    vlc_after JSON NOT NULL,
    payload JSON NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    INDEX idx_task_id (task_id),
    INDEX idx_task_type (task_type),
    INDEX idx_stage (stage),
    INDEX idx_created_at (created_at)
);

-- ============================================
-- 5. Update user profile table (if needed)
-- ============================================

-- Check if twitter_id column exists; if not, add it
-- Note: According to the existing schema, twitter_id already exists; this part is only for compatibility
SET @col_exists = (
    SELECT COUNT(*)
    FROM INFORMATION_SCHEMA.COLUMNS
    WHERE TABLE_SCHEMA = DATABASE()
    AND TABLE_NAME = 'user_profiles'
    AND COLUMN_NAME = 'twitter_id'
);

-- Add column if it does not exist
SET @sql = IF(@col_exists = 0,
    'ALTER TABLE user_profiles ADD COLUMN twitter_id VARCHAR(100) NULL AFTER display_name, ADD INDEX idx_twitter_id (twitter_id)',
    'SELECT "twitter_id column already exists" as message'
);

PREPARE stmt FROM @sql;
EXECUTE stmt;
DEALLOCATE PREPARE stmt;

-- ============================================
-- 6. Create views for easy querying
-- ============================================

-- Task details view
CREATE OR REPLACE VIEW task_details_view AS
SELECT 
    t.id,
    t.user_wallet,
    t.task_type,
    t.status,
    t.payload,
    t.proof,
    t.attempts,
    t.created_at,
    t.updated_at,
    t.completed_at,
    t.event_id,
    t.vlc_clock,
    up.display_name,
    up.twitter_id as user_twitter_id,
    tt.twitter_id as task_twitter_id,
    tt.tweet_id,
    tt.project_name,
    tt.project_icon,
    tt.twitter_username,
    tt.verification_status as twitter_verification_status
FROM tasks t
LEFT JOIN user_profiles up ON t.user_wallet = up.wallet_address
LEFT JOIN twitter_tasks tt ON t.id = tt.task_id;

-- Batch verification statistics view
CREATE OR REPLACE VIEW batch_verification_stats AS
SELECT 
    bv.batch_id,
    bv.task_id,
    bv.start_time,
    bv.end_time,
    bv.total_tasks,
    bv.verified_tasks,
    bv.unverified_tasks,
    bv.vlc_increment,
    bv.status,
    bv.created_at,
    bv.completed_at,
    COUNT(bvr.id) as result_count,
    SUM(CASE WHEN bvr.verified = TRUE THEN 1 ELSE 0 END) as verified_count
FROM batch_verifications bv
LEFT JOIN batch_verification_results bvr ON bv.batch_id = bvr.batch_id
GROUP BY bv.batch_id;

-- VLC event statistics view
CREATE OR REPLACE VIEW vlc_event_stats AS
SELECT 
    task_type,
    stage,
    COUNT(*) as event_count,
    SUM(increment_count) as total_increment,
    AVG(increment_count) as avg_increment,
    MIN(created_at) as first_event,
    MAX(created_at) as last_event
FROM vlc_events
GROUP BY task_type, stage;

-- ============================================
-- 7. Update task-type enum (if CHECK constraint is used)
-- ============================================

-- Note: MySQL 8.0 supports CHECK constraints, but the existing schema uses VARCHAR
-- Add a stored function to validate task type

DELIMITER //

CREATE OR REPLACE FUNCTION validate_task_type(task_type VARCHAR(50))
RETURNS BOOLEAN
READS SQL DATA
DETERMINISTIC
BEGIN
    DECLARE valid_count INT DEFAULT 0;
    
    SELECT COUNT(*) INTO valid_count
    FROM (
        SELECT 'twitter_retweet' as type
        UNION SELECT 'discord_message'
        UNION SELECT 'email_confirm'
        UNION SELECT 'task_creation'
        UNION SELECT 'batch_verification'
    ) valid_types
    WHERE valid_types.type = task_type;
    
    RETURN valid_count > 0;
END //

DELIMITER ;

-- ============================================
-- 8. Insert sample data (optional)
-- ============================================

-- Insert sample validator config (if needed)
INSERT IGNORE INTO validator_configs (id, role, weight, public_key) VALUES
('twitter-validator-1', 'twitter_validator', 0.30, 'twitter_validator_public_key_placeholder');

-- ============================================
-- 9. Permission settings
-- ============================================

-- Grant necessary permissions to application user
-- Note: Assume pocw_user is the application database user
GRANT SELECT, INSERT, UPDATE, DELETE ON twitter_tasks TO 'pocw_user'@'%';
GRANT SELECT, INSERT, UPDATE, DELETE ON batch_verifications TO 'pocw_user'@'%';
GRANT SELECT, INSERT, UPDATE, DELETE ON batch_verification_results TO 'pocw_user'@'%';
GRANT SELECT, INSERT, UPDATE, DELETE ON vlc_events TO 'pocw_user'@'%';
GRANT SELECT ON task_details_view TO 'pocw_user'@'%';
GRANT SELECT ON batch_verification_stats TO 'pocw_user'@'%';
GRANT SELECT ON vlc_event_stats TO 'pocw_user'@'%';

-- ============================================
-- 10. Migration completion mark
-- ============================================

-- Create migration record table (if not exists)
CREATE TABLE IF NOT EXISTS migrations (
    id INT AUTO_INCREMENT PRIMARY KEY,
    version VARCHAR(20) NOT NULL UNIQUE,
    description TEXT NOT NULL,
    executed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
);

-- Record this migration
INSERT IGNORE INTO migrations (version, description) VALUES 
('001', 'Add Twitter task support with new task types, VLC events tracking, and batch verification');