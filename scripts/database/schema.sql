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
-- 4. System Monitoring
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

DELIMITER ;

-- ============================================
-- 7. Index Optimization
-- ============================================

-- Composite indexes
CREATE INDEX IF NOT EXISTS idx_tasks_user_status ON tasks(user_wallet, status);
CREATE INDEX IF NOT EXISTS idx_tasks_type_status ON tasks(task_type, status);
CREATE INDEX IF NOT EXISTS idx_validation_records_event_validator ON validation_records(event_id, validator_id);
CREATE INDEX IF NOT EXISTS idx_points_history_wallet_date ON points_history(wallet_address, date);

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

-- Task statistics
CREATE OR REPLACE VIEW task_stats AS
SELECT 
    task_type,
    status,
    COUNT(*) as count,
    AVG(TIMESTAMPDIFF(SECOND, created_at, updated_at)) as avg_processing_time_seconds
FROM tasks 
GROUP BY task_type, status;

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