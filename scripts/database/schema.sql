-- PoCW Database Schema
-- Designed for multi-service architecture

-- ============================================
-- 1. Task Management (used by MinerGateway)
-- ============================================

-- Tasks table
CREATE TABLE IF NOT EXISTS tasks (
    id VARCHAR(36) PRIMARY KEY,
    user_wallet VARCHAR(42) NOT NULL,
    task_type VARCHAR(50) NOT NULL,
    status VARCHAR(30) NOT NULL,
    payload JSON NOT NULL,
    proof JSON NULL,
    attempts INT DEFAULT 0,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    completed_at TIMESTAMP NULL,
    event_id VARCHAR(100) NULL,
    vlc_clock JSON NULL,
    
    INDEX idx_user_wallet (user_wallet),
    INDEX idx_task_type (task_type),
    INDEX idx_status (status),
    INDEX idx_created_at (created_at),
    INDEX idx_event_id (event_id)
);

-- Task status history
CREATE TABLE IF NOT EXISTS task_status_history (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    task_id VARCHAR(36) NOT NULL,
    old_status VARCHAR(30),
    new_status VARCHAR(30) NOT NULL,
    reason TEXT,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE,
    INDEX idx_task_id (task_id),
    INDEX idx_created_at (created_at)
);

-- ============================================
-- 2. SBT & User Profile (used by SBT Service)
-- ============================================

-- User profiles
CREATE TABLE IF NOT EXISTS user_profiles (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    wallet_address VARCHAR(42) UNIQUE NOT NULL,
    display_name VARCHAR(100) NOT NULL,
    twitter_id VARCHAR(100) NULL,
    registration_date TIMESTAMP NOT NULL,
    inviter VARCHAR(42) NULL,
    total_points INT DEFAULT 0,
    today_contribution INT DEFAULT 0,
    token_uri TEXT NOT NULL,
    token_id BIGINT NULL,
    image_uri TEXT NULL,
    ipfs_hash VARCHAR(100) NOT NULL,
    subnets JSON NULL,
    subnet_nfts JSON NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_wallet_address (wallet_address),
    INDEX idx_display_name (display_name),
    INDEX idx_registration_date (registration_date),
    INDEX idx_inviter (inviter),
    INDEX idx_token_id (token_id),
    INDEX idx_twitter_id (twitter_id)
);

-- Points history
CREATE TABLE IF NOT EXISTS points_history (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    wallet_address VARCHAR(42) NOT NULL,
    date DATE NOT NULL,
    source VARCHAR(100) NOT NULL,
    points INT NOT NULL,
    tx_ref VARCHAR(100) NULL,
    task_id VARCHAR(36) NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (wallet_address) REFERENCES user_profiles(wallet_address) ON DELETE CASCADE,
    INDEX idx_wallet_address (wallet_address),
    INDEX idx_date (date),
    INDEX idx_source (source),
    INDEX idx_task_id (task_id)
);

-- Invite relationships
CREATE TABLE IF NOT EXISTS invite_relations (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    inviter VARCHAR(42) NOT NULL,
    invitee VARCHAR(42) NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_invite (inviter, invitee),
    INDEX idx_inviter (inviter),
    INDEX idx_invitee (invitee)
);

-- ============================================
-- 3. Validator State (used by Validator Service)
-- ============================================

-- Validator configs
CREATE TABLE IF NOT EXISTS validator_configs (
    id VARCHAR(50) PRIMARY KEY,
    role VARCHAR(30) NOT NULL,
    weight DECIMAL(3,2) NOT NULL,
    public_key TEXT NOT NULL,
    endpoints JSON NULL,
    status VARCHAR(20) DEFAULT 'active',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_role (role),
    INDEX idx_status (status)
);

-- Validation records
CREATE TABLE IF NOT EXISTS validation_records (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    event_id VARCHAR(100) NOT NULL,
    validator_id VARCHAR(50) NOT NULL,
    task_id VARCHAR(36) NOT NULL,
    vote VARCHAR(10) NOT NULL,
    score DECIMAL(3,2) NOT NULL,
    weight DECIMAL(3,2) NOT NULL,
    reason TEXT NULL,
    vlc_state JSON NULL,
    signature TEXT NOT NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    FOREIGN KEY (validator_id) REFERENCES validator_configs(id),
    INDEX idx_event_id (event_id),
    INDEX idx_validator_id (validator_id),
    INDEX idx_task_id (task_id),
    INDEX idx_vote (vote),
    INDEX idx_created_at (created_at)
);

-- Consensus results
CREATE TABLE IF NOT EXISTS consensus_results (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    event_id VARCHAR(100) UNIQUE NOT NULL,
    task_id VARCHAR(36) NOT NULL,
    total_weight DECIMAL(3,2) NOT NULL,
    accept_weight DECIMAL(3,2) NOT NULL,
    reject_weight DECIMAL(3,2) NOT NULL,
    final_decision VARCHAR(10) NOT NULL,
    consensus_reached BOOLEAN NOT NULL,
    aggregator_id VARCHAR(50) NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_event_id (event_id),
    INDEX idx_task_id (task_id),
    INDEX idx_final_decision (final_decision),
    INDEX idx_created_at (created_at)
);

-- ============================================
-- 4. Twitter Task Support (from 001 migration)
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
-- 5. Twitter Verification Monitoring (from 002 migration)
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
-- 6. System Monitoring
-- ============================================

-- Service health status
CREATE TABLE IF NOT EXISTS service_health (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    service_id VARCHAR(50) NOT NULL,
    service_type VARCHAR(30) NOT NULL, -- 'miner', 'validator', 'aggregator', 'sbt'
    status VARCHAR(20) NOT NULL, -- 'healthy', 'degraded', 'error'
    endpoint VARCHAR(200) NOT NULL,
    last_heartbeat TIMESTAMP NOT NULL,
    error_message TEXT NULL,
    metadata JSON NULL,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY unique_service (service_id, service_type),
    INDEX idx_service_type (service_type),
    INDEX idx_status (status),
    INDEX idx_last_heartbeat (last_heartbeat)
);

-- Performance metrics
CREATE TABLE IF NOT EXISTS performance_metrics (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    service_id VARCHAR(50) NOT NULL,
    metric_name VARCHAR(50) NOT NULL,
    metric_value DECIMAL(10,4) NOT NULL,
    unit VARCHAR(20) NOT NULL,
    timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_service_id (service_id),
    INDEX idx_metric_name (metric_name),
    INDEX idx_timestamp (timestamp)
);

-- ============================================
-- 5. Seed Data
-- ============================================

-- Insert default validator configs
INSERT IGNORE INTO validator_configs (id, role, weight, public_key) VALUES
('validator-1', 'ui_validator', 0.40, 'ui_validator_public_key_placeholder'),
('validator-2', 'format_validator', 0.20, 'format_validator_2_public_key_placeholder'),
('validator-3', 'format_validator', 0.20, 'format_validator_3_public_key_placeholder'),
('validator-4', 'semantic_validator', 0.20, 'semantic_validator_public_key_placeholder');

-- ============================================
-- 6. Triggers & Stored Procedures
-- ============================================

DELIMITER //

-- Trigger to update user points
CREATE TRIGGER IF NOT EXISTS update_user_points 
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

-- Stored procedure to reset daily contributions
CREATE PROCEDURE IF NOT EXISTS ResetDailyContributions()
BEGIN
    UPDATE user_profiles 
    SET today_contribution = 0, 
        updated_at = CURRENT_TIMESTAMP;
END//

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

-- Task type validation function (updated)
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
-- 7. Index Optimization
-- ============================================

-- Composite indexes (MySQL 8.0 compatible)
CREATE INDEX idx_tasks_user_status ON tasks(user_wallet, status);
CREATE INDEX idx_tasks_type_status ON tasks(task_type, status);
CREATE INDEX idx_validation_records_event_validator ON validation_records(event_id, validator_id);
CREATE INDEX idx_points_history_wallet_date ON points_history(wallet_address, date);

-- ============================================
-- 8. Views
-- ============================================

-- User statistics
CREATE OR REPLACE VIEW user_stats AS
SELECT 
    up.wallet_address,
    up.display_name,
    up.total_points,
    up.today_contribution,
    COUNT(DISTINCT t.id) as total_tasks,
    COUNT(DISTINCT CASE WHEN t.status = 'CONFIRMED' THEN t.id END) as completed_tasks,
    up.registration_date,
    DATEDIFF(NOW(), up.registration_date) as days_active
FROM user_profiles up
LEFT JOIN tasks t ON up.wallet_address = t.user_wallet
GROUP BY up.wallet_address;

-- Validator performance
CREATE OR REPLACE VIEW validator_performance AS
SELECT 
    vc.id as validator_id,
    vc.role,
    vc.weight,
    COUNT(vr.id) as total_validations,
    COUNT(CASE WHEN vr.vote = 'accept' THEN 1 END) as accept_votes,
    COUNT(CASE WHEN vr.vote = 'reject' THEN 1 END) as reject_votes,
    AVG(vr.score) as avg_score,
    sh.status as current_status,
    sh.last_heartbeat
FROM validator_configs vc
LEFT JOIN validation_records vr ON vc.id = vr.validator_id
LEFT JOIN service_health sh ON vc.id = sh.service_id AND sh.service_type = 'validator'
GROUP BY vc.id;

-- Task statistics (updated with new status)
CREATE OR REPLACE VIEW task_stats AS
SELECT 
    task_type,
    status,
    COUNT(*) as count,
    AVG(TIMESTAMPDIFF(SECOND, created_at, updated_at)) as avg_processing_time_seconds,
    COUNT(CASE WHEN status = 'PENDING_REVIEW' THEN 1 END) as pending_review_count
FROM tasks 
GROUP BY task_type, status;

-- Task details view (enhanced)
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

-- Twitter verification monitoring view
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
-- 9. Data Retention Rules
-- ============================================

-- Enable event scheduler
-- SET GLOBAL event_scheduler = ON;

-- Clean up old performance metrics (keep 30 days)
-- CREATE EVENT IF NOT EXISTS cleanup_old_metrics
-- ON SCHEDULE EVERY 1 DAY
-- DO
--   DELETE FROM performance_metrics WHERE timestamp < DATE_SUB(NOW(), INTERVAL 30 DAY);

-- Clean up old health records (keep 7 days)
-- CREATE EVENT IF NOT EXISTS cleanup_old_health_records
-- ON SCHEDULE EVERY 1 DAY  
-- DO
--   DELETE FROM service_health WHERE updated_at < DATE_SUB(NOW(), INTERVAL 7 DAY);

-- ============================================
-- 10. Sample Permissions
-- ============================================

/*
-- MinerGateway user
CREATE USER IF NOT EXISTS 'miner_service'@'%' IDENTIFIED BY 'secure_password_1';
GRANT SELECT, INSERT, UPDATE ON tasks TO 'miner_service'@'%';
GRANT SELECT, INSERT ON task_status_history TO 'miner_service'@'%';
GRANT SELECT ON user_profiles TO 'miner_service'@'%';

-- Validator user
CREATE USER IF NOT EXISTS 'validator_service'@'%' IDENTIFIED BY 'secure_password_2';
GRANT SELECT, INSERT, UPDATE ON validation_records TO 'validator_service'@'%';
GRANT SELECT ON validator_configs TO 'validator_service'@'%';
GRANT SELECT, INSERT, UPDATE ON service_health TO 'validator_service'@'%';

-- SBT Service user
CREATE USER IF NOT EXISTS 'sbt_service'@'%' IDENTIFIED BY 'secure_password_3';
GRANT ALL PRIVILEGES ON user_profiles TO 'sbt_service'@'%';
GRANT ALL PRIVILEGES ON points_history TO 'sbt_service'@'%';
GRANT ALL PRIVILEGES ON invite_relations TO 'sbt_service'@'%';

FLUSH PRIVILEGES;
*/