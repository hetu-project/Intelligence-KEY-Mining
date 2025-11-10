-- ============================================
-- Migration 004: Subnet Points Override System
-- Purpose: Allow subnet admins to manually override user's total points
-- ============================================

-- Create subnet_user_points_override table
CREATE TABLE IF NOT EXISTS subnet_user_points_override (
    id INT AUTO_INCREMENT PRIMARY KEY,
    subnet_id VARCHAR(255) NOT NULL COMMENT 'Subnet ID',
    wallet_address VARCHAR(42) NOT NULL COMMENT 'User wallet address',
    override_points INT NOT NULL COMMENT 'Override total points as of created_at timestamp',
    admin_wallet VARCHAR(42) NOT NULL COMMENT 'Admin who performed the override',
    reason TEXT COMMENT 'Reason for the override',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Override timestamp (points after this continue to accumulate)',
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    -- Constraints
    UNIQUE KEY uk_subnet_wallet (subnet_id, wallet_address),
    INDEX idx_subnet (subnet_id),
    INDEX idx_wallet (wallet_address),
    INDEX idx_admin (admin_wallet),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Subnet user points override table';

-- ============================================
-- Usage Notes:
-- 
-- 1. Override Logic:
--    - override_points = total points up to created_at
--    - Points earned after created_at continue to accumulate
--    - Total = override_points + SUM(points after created_at)
--
-- 2. Example:
--    User has 5 points from 5 tasks (Nov 1-5)
--    Admin sets override_points=10 on Nov 4 15:00
--    User completes task on Nov 5 (+1 point)
--    Result: Total = 10 + 1 = 11 points
--
-- 3. Update Override:
--    INSERT ... ON DUPLICATE KEY UPDATE override_points=?, created_at=NOW()
--
-- 4. Remove Override:
--    DELETE FROM subnet_user_points_override WHERE ...
-- ============================================

