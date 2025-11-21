#!/bin/bash

# PoCW Data Verification Script
# Quick script to check if PoCW data is being saved correctly

echo "🔍 Verifying PoCW Browser Data..."
echo ""

# Check if tables exist
echo "1️⃣ Checking tables..."
docker-compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db -e "
SELECT 
    table_name,
    table_rows
FROM information_schema.tables 
WHERE table_schema='pocw_db' 
AND table_name IN ('pocw_rounds', 'pocw_votes', 'pocw_round_tasks');
"
echo ""

# Check recent rounds
echo "2️⃣ Recent rounds (last 5)..."
docker-compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db -e "
SELECT 
    round_id,
    start_time,
    phase,
    task_count,
    consensus_decision,
    ROUND(duration_seconds, 2) as duration_sec
FROM pocw_rounds 
ORDER BY start_time DESC 
LIMIT 5;
"
echo ""

# Check vote statistics
echo "3️⃣ Vote statistics..."
docker-compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db -e "
SELECT 
    validator_id,
    vote,
    COUNT(*) as count,
    AVG(quality_score) as avg_score
FROM pocw_votes 
GROUP BY validator_id, vote
ORDER BY validator_id, vote;
"
echo ""

# Check round task statistics
echo "4️⃣ Round task statistics..."
docker-compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db -e "
SELECT 
    round_id,
    COUNT(*) as task_count,
    COUNT(DISTINCT user_wallet) as unique_users,
    SUM(points_awarded) as total_points
FROM pocw_round_tasks 
GROUP BY round_id
ORDER BY round_id DESC
LIMIT 5;
"
echo ""

echo "✅ Verification complete!"
