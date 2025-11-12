-- ============================================
-- Migration 005: Subnet Points Adjustment (替代 Override 系统)
-- Purpose: 简化积分调整逻辑，支持增量调整和用户备注
-- ============================================

-- 创建积分调整表（增量方式）
CREATE TABLE IF NOT EXISTS subnet_points_adjustment (
    id INT AUTO_INCREMENT PRIMARY KEY,
    subnet_id VARCHAR(255) NOT NULL COMMENT 'Subnet ID',
    wallet_address VARCHAR(42) NOT NULL COMMENT 'User wallet address',
    adjustment_points INT NOT NULL COMMENT 'Adjustment points (can be positive or negative)',
    admin_wallet VARCHAR(42) NOT NULL COMMENT 'Admin who performed the adjustment',
    reason VARCHAR(50) COMMENT 'Reason for adjustment (max 20 Chinese characters = 60 bytes)',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP COMMENT 'Adjustment timestamp',
    
    -- Indexes
    INDEX idx_subnet_wallet (subnet_id, wallet_address),
    INDEX idx_subnet (subnet_id),
    INDEX idx_wallet (wallet_address),
    INDEX idx_admin (admin_wallet),
    INDEX idx_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci COMMENT='Subnet user points adjustment table';

-- ============================================
-- Usage Notes:
-- 
-- 1. 增量积分逻辑:
--    - 每次调整创建一条新记录
--    - 计算总积分 = 任务积分 + SUM(所有adjustment_points)
--    - 支持正数（增加）和负数（减少）
--
-- 2. 示例:
--    用户有 5 个任务积分
--    管理员调整 +10 分 (第一次)
--    总积分 = 5 + 10 = 15
--    管理员再调整 +5 分 (第二次)
--    总积分 = 5 + 10 + 5 = 20
--
-- 3. 查询用户某个子网的总积分:
--    SELECT 
--        COALESCE(SUM(ph.points), 0) as task_points,
--        COALESCE(SUM(spa.adjustment_points), 0) as adjustment_points,
--        COALESCE(SUM(ph.points), 0) + COALESCE(SUM(spa.adjustment_points), 0) as total_points
--    FROM user_profiles up
--    LEFT JOIN points_history ph ON up.wallet_address = ph.wallet_address AND ph.subnet_id = ?
--    LEFT JOIN subnet_points_adjustment spa ON up.wallet_address = spa.wallet_address AND spa.subnet_id = ?
--    WHERE up.wallet_address = ?
--
-- 4. 查询调整历史:
--    SELECT * FROM subnet_points_adjustment 
--    WHERE subnet_id = ? AND wallet_address = ?
--    ORDER BY created_at DESC
-- ============================================

