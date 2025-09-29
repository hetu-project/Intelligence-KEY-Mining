-- Fix total_points in user_profiles table by calculating from points_history
-- This script fixes the issue where NFT purchase bonuses and invitation rewards
-- were not reflected in the total_points field

-- Update total_points for all users based on their points_history
UPDATE user_profiles up
SET total_points = (
    SELECT COALESCE(SUM(ph.points), 0)
    FROM points_history ph
    WHERE ph.wallet_address = up.wallet_address
);

-- Verify the fix
SELECT 
    up.wallet_address,
    up.total_points as profile_total,
    COALESCE(SUM(ph.points), 0) as calculated_total
FROM user_profiles up
LEFT JOIN points_history ph ON ph.wallet_address = up.wallet_address
GROUP BY up.wallet_address, up.total_points
HAVING profile_total != calculated_total;
