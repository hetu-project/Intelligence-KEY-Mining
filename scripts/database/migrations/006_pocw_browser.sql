-- ============================================
-- PoCW Browser Tables
-- ============================================
-- Purpose: Store PoCW consensus round data for browser visualization
-- Created: 2025-11-21
-- ============================================

-- PoCW Rounds Table
-- Stores complete round lifecycle data
CREATE TABLE IF NOT EXISTS pocw_rounds (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    round_id VARCHAR(100) UNIQUE NOT NULL COMMENT 'Unique round identifier (e.g., round_1731840000)',
    start_time TIMESTAMP NOT NULL COMMENT 'Round start timestamp',
    end_time TIMESTAMP NULL COMMENT 'Round completion timestamp',
    phase VARCHAR(30) NOT NULL DEFAULT 'idle' COMMENT 'Current phase: idle, task_process, vlc_verify, quality_vote, consensus, complete',
    
    -- Task statistics
    task_count INT DEFAULT 0 COMMENT 'Total number of tasks in this round',
    verified_count INT DEFAULT 0 COMMENT 'Number of verified tasks',
    failed_count INT DEFAULT 0 COMMENT 'Number of failed tasks',
    
    -- VLC tracking
    miner_vlc_before JSON COMMENT 'Miner VLC state before round processing',
    miner_vlc_after JSON COMMENT 'Miner VLC state after round processing',
    validator_vlc JSON COMMENT 'Validator VLC state during round',
    vlc_increment INT DEFAULT 0 COMMENT 'Total VLC increment in this round',
    
    -- Consensus results
    consensus_decision VARCHAR(20) COMMENT 'Final consensus decision: approved, rejected, no_consensus',
    consensus_approved_count INT DEFAULT 0 COMMENT 'Number of approved tasks',
    consensus_rejected_count INT DEFAULT 0 COMMENT 'Number of rejected tasks',
    consensus_metadata JSON COMMENT 'Detailed consensus calculation data',
    
    -- Round completion
    completion_result VARCHAR(50) COMMENT 'Completion status: success, task_processing_error, vlc_verification_error, etc.',
    duration_seconds DECIMAL(10,3) COMMENT 'Round processing duration in seconds',
    
    -- Metadata
    metadata JSON COMMENT 'Additional round metadata',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_round_id (round_id),
    INDEX idx_start_time (start_time),
    INDEX idx_phase (phase),
    INDEX idx_consensus_decision (consensus_decision),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='PoCW consensus rounds';

-- PoCW Votes Table
-- Stores validator votes for quality assessment
CREATE TABLE IF NOT EXISTS pocw_votes (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    round_id VARCHAR(100) NOT NULL COMMENT 'Associated round ID',
    task_id VARCHAR(36) NOT NULL COMMENT 'Task being voted on',
    validator_id VARCHAR(50) NOT NULL COMMENT 'Validator identifier',
    validator_role VARCHAR(30) COMMENT 'Validator role: ui_validator, format_validator, semantic_validator',
    
    -- Vote details
    vote VARCHAR(20) NOT NULL COMMENT 'Vote decision: approve, reject, abstain',
    quality_score DECIMAL(3,2) COMMENT 'Quality score (0.00-1.00)',
    weight DECIMAL(3,2) NOT NULL COMMENT 'Validator voting weight',
    reasoning TEXT COMMENT 'Vote reasoning or explanation',
    
    -- Timing
    vote_timestamp TIMESTAMP NOT NULL COMMENT 'When the vote was cast',
    
    -- Metadata
    metadata JSON COMMENT 'Additional vote metadata',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_round_id (round_id),
    INDEX idx_task_id (task_id),
    INDEX idx_validator_id (validator_id),
    INDEX idx_vote_timestamp (vote_timestamp),
    INDEX idx_vote (vote),
    
    FOREIGN KEY (round_id) REFERENCES pocw_rounds(round_id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='PoCW validator votes';

-- PoCW Round Tasks Table
-- Links tasks to rounds for tracking
CREATE TABLE IF NOT EXISTS pocw_round_tasks (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    round_id VARCHAR(100) NOT NULL COMMENT 'Associated round ID',
    task_id VARCHAR(36) NOT NULL COMMENT 'Task identifier',
    user_wallet VARCHAR(42) NOT NULL COMMENT 'User wallet address',
    task_type VARCHAR(50) NOT NULL COMMENT 'Task type (e.g., twitter_retweet)',
    subnet_id VARCHAR(36) COMMENT 'Associated subnet ID',
    
    -- VLC tracking
    vlc_increment INT DEFAULT 1 COMMENT 'VLC increment for this task',
    vlc_snapshot JSON COMMENT 'VLC state snapshot when task was processed',
    
    -- Consensus result
    consensus_result VARCHAR(20) COMMENT 'Consensus result for this task: approved, rejected',
    points_awarded INT DEFAULT 0 COMMENT 'Points awarded after consensus',
    
    -- Timing
    processed_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'When task was processed in round',
    
    INDEX idx_round_id (round_id),
    INDEX idx_task_id (task_id),
    INDEX idx_user_wallet (user_wallet),
    INDEX idx_task_type (task_type),
    INDEX idx_subnet_id (subnet_id),
    INDEX idx_consensus_result (consensus_result),
    
    UNIQUE KEY uk_round_task (round_id, task_id),
    FOREIGN KEY (round_id) REFERENCES pocw_rounds(round_id) ON DELETE CASCADE,
    FOREIGN KEY (task_id) REFERENCES tasks(id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='PoCW round task associations';

-- ============================================
-- Views for Easy Querying
-- ============================================

-- Round summary view with aggregated statistics
CREATE OR REPLACE VIEW pocw_round_summary AS
SELECT 
    r.round_id,
    r.start_time,
    r.end_time,
    r.phase,
    r.task_count,
    r.verified_count,
    r.failed_count,
    r.vlc_increment,
    r.consensus_decision,
    r.consensus_approved_count,
    r.consensus_rejected_count,
    r.duration_seconds,
    r.completion_result,
    COUNT(DISTINCT v.validator_id) as validator_count,
    COUNT(v.id) as total_votes,
    SUM(CASE WHEN v.vote = 'approve' THEN 1 ELSE 0 END) as approve_votes,
    SUM(CASE WHEN v.vote = 'reject' THEN 1 ELSE 0 END) as reject_votes,
    AVG(v.quality_score) as avg_quality_score,
    r.created_at
FROM pocw_rounds r
LEFT JOIN pocw_votes v ON r.round_id = v.round_id
GROUP BY r.id
ORDER BY r.start_time DESC;

-- Validator performance view
CREATE OR REPLACE VIEW pocw_validator_performance AS
SELECT 
    v.validator_id,
    v.validator_role,
    COUNT(*) as total_votes,
    SUM(CASE WHEN v.vote = 'approve' THEN 1 ELSE 0 END) as approve_count,
    SUM(CASE WHEN v.vote = 'reject' THEN 1 ELSE 0 END) as reject_count,
    SUM(CASE WHEN v.vote = 'abstain' THEN 1 ELSE 0 END) as abstain_count,
    AVG(v.quality_score) as avg_quality_score,
    AVG(v.weight) as avg_weight,
    MIN(v.vote_timestamp) as first_vote_time,
    MAX(v.vote_timestamp) as last_vote_time
FROM pocw_votes v
GROUP BY v.validator_id, v.validator_role;

-- ============================================
-- Example Queries
-- ============================================

-- Get recent rounds with statistics
-- SELECT * FROM pocw_round_summary LIMIT 20;

-- Get round details with all votes
-- SELECT r.*, v.*
-- FROM pocw_rounds r
-- LEFT JOIN pocw_votes v ON r.round_id = v.round_id
-- WHERE r.round_id = 'round_1731840000';

-- Get validator performance
-- SELECT * FROM pocw_validator_performance;

-- Get daily round statistics
-- SELECT 
--     DATE(start_time) as date,
--     COUNT(*) as round_count,
--     SUM(task_count) as total_tasks,
--     SUM(consensus_approved_count) as total_approved,
--     AVG(duration_seconds) as avg_duration
-- FROM pocw_rounds
-- WHERE start_time >= DATE_SUB(CURDATE(), INTERVAL 30 DAY)
-- GROUP BY DATE(start_time)
-- ORDER BY date DESC;
