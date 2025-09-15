-- Migration: Add PENDING_REVIEW task status support
-- Version: 002
-- Description: Add support for new PENDING_REVIEW task status and related optimizations

-- ============================================
-- 1. Update task status documentation
-- ============================================

-- Add a comment to document the new status
ALTER TABLE tasks MODIFY COLUMN status VARCHAR(30) NOT NULL 
COMMENT 'Task status: SUBMITTED, PENDING_VERIFICATION, VERIFIED, FAILED, PENDING_REVIEW, MINER_OUTPUT_CREATED, VOTED, CONFIRMED, REJECTED';

-- ============================================
-- 2. Add Twitter verification metrics table
-- ============================================

-- Twitter verification metrics for monitoring
CREATE TABLE IF NOT EXISTS twitter_verification_metrics (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    date DATE NOT NULL,
    total_attempts INT DEFAULT 0,
    successful_calls INT DEFAULT 0,
    failed_calls INT DEFAULT 0,
    network_errors INT DEFAULT 0,
    api_errors INT DEFAULT 0,
    config_errors INT DEFAULT 0,
    avg_response_time_ms INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_date (date),
    INDEX idx_date (date)
);

-- ============================================
-- 3. Create Twitter verification log table
-- ============================================

-- Detailed log for Twitter verification attempts
CREATE TABLE IF NOT EXISTS twitter_verification_logs (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_id VARCHAR(36) NOT NULL,
    user_wallet VARCHAR(42) NOT NULL,
    tweet_id VARCHAR(100) NOT NULL,
    twitter_username VARCHAR(100) NOT NULL,
    user_twitter_id VARCHAR(100) NULL,
    api_request JSON NULL,
    api_response JSON NULL,
    verification_result BOOLEAN NULL,
    error_message TEXT NULL,
    attempt_number INT DEFAULT 1,
    response_time_ms INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    INDEX idx_task_id (task_id),
    INDEX idx_user_wallet (user_wallet),
    INDEX idx_tweet_id (tweet_id),
    INDEX idx_created_at (created_at),
    INDEX idx_verification_result (verification_result)
);

-- ============================================
-- 4. Update views to include new status
-- ============================================

-- Update task statistics view to include PENDING_REVIEW
CREATE OR REPLACE VIEW task_stats AS
SELECT 
    task_type,
    status,
    COUNT(*) as count,
    AVG(TIMESTAMPDIFF(SECOND, created_at, updated_at)) as avg_processing_time_seconds,
    COUNT(CASE WHEN status = 'PENDING_REVIEW' THEN 1 END) as pending_review_count
FROM tasks 
GROUP BY task_type, status;

-- Create Twitter verification monitoring view
CREATE OR REPLACE VIEW twitter_verification_monitoring AS
SELECT 
    t.id as task_id,
    t.user_wallet,
    t.status,
    t.attempts,
    t.created_at as task_created_at,
    t.updated_at as task_updated_at,
    up.twitter_id as user_twitter_id,
    JSON_EXTRACT(t.payload, '$.tweet_id') as tweet_id,
    JSON_EXTRACT(t.payload, '$.twitter_username') as twitter_username,
    tvl.verification_result,
    tvl.error_message,
    tvl.attempt_number,
    tvl.created_at as last_verification_at
FROM tasks t
LEFT JOIN user_profiles up ON t.user_wallet = up.wallet_address
LEFT JOIN twitter_verification_logs tvl ON t.id = tvl.task_id 
    AND tvl.id = (
        SELECT MAX(id) FROM twitter_verification_logs 
        WHERE task_id = t.id
    )
WHERE t.task_type = 'twitter_retweet';

-- ============================================
-- 5. Add stored procedures for maintenance
-- ============================================

DELIMITER //

-- Procedure to clean up old verification logs (keep 30 days)
CREATE PROCEDURE IF NOT EXISTS CleanupTwitterVerificationLogs()
BEGIN
    DELETE FROM twitter_verification_logs 
    WHERE created_at < DATE_SUB(NOW(), INTERVAL 30 DAY);
    
    SELECT ROW_COUNT() as deleted_rows;
END //

-- Procedure to update daily Twitter verification metrics
CREATE PROCEDURE IF NOT EXISTS UpdateTwitterVerificationMetrics()
BEGIN
    INSERT INTO twitter_verification_metrics (
        date, total_attempts, successful_calls, failed_calls, 
        network_errors, api_errors, config_errors, avg_response_time_ms
    )
    SELECT 
        CURDATE(),
        COUNT(*) as total_attempts,
        SUM(CASE WHEN verification_result = TRUE THEN 1 ELSE 0 END) as successful_calls,
        SUM(CASE WHEN verification_result = FALSE THEN 1 ELSE 0 END) as failed_calls,
        SUM(CASE WHEN error_message LIKE '%network%' OR error_message LIKE '%timeout%' THEN 1 ELSE 0 END) as network_errors,
        SUM(CASE WHEN error_message LIKE '%API%' OR error_message LIKE '%status%' THEN 1 ELSE 0 END) as api_errors,
        SUM(CASE WHEN error_message LIKE '%config%' OR error_message LIKE '%URL%' THEN 1 ELSE 0 END) as config_errors,
        AVG(COALESCE(response_time_ms, 0)) as avg_response_time_ms
    FROM twitter_verification_logs 
    WHERE DATE(created_at) = CURDATE()
    ON DUPLICATE KEY UPDATE
        total_attempts = VALUES(total_attempts),
        successful_calls = VALUES(successful_calls),
        failed_calls = VALUES(failed_calls),
        network_errors = VALUES(network_errors),
        api_errors = VALUES(api_errors),
        config_errors = VALUES(config_errors),
        avg_response_time_ms = VALUES(avg_response_time_ms),
        updated_at = CURRENT_TIMESTAMP;
END //

DELIMITER ;

-- ============================================
-- 6. Update task type validation function
-- ============================================

DELIMITER //

-- Update the validation function to include new task types
DROP FUNCTION IF EXISTS validate_task_type //

CREATE FUNCTION validate_task_type(task_type VARCHAR(50))
RETURNS BOOL
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
    ) valid_types
    WHERE valid_types.type = task_type;
    
    RETURN valid_count > 0;
END //

DELIMITER ;

-- ============================================
-- 7. Permission updates (Skipped in Docker environment)
-- ============================================

-- Grant permissions for new tables
-- Note: Skipping GRANT statements in Docker environment
-- GRANT SELECT, INSERT, UPDATE, DELETE ON twitter_verification_metrics TO 'miner_service'@'%';
-- GRANT SELECT, INSERT, UPDATE, DELETE ON twitter_verification_logs TO 'miner_service'@'%';
-- GRANT SELECT ON twitter_verification_monitoring TO 'miner_service'@'%';

-- ============================================
-- 8. Record migration
-- ============================================

INSERT IGNORE INTO migrations (version, description) VALUES 
('002', 'Add PENDING_REVIEW status support and Twitter verification monitoring');
