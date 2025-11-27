#!/bin/bash

# Create Test PoCW Data
# This script creates test tasks to trigger PoCW consensus

echo "🧪 Creating test PoCW data..."
echo ""

# 1. Create a test user profile if not exists
echo "1️⃣ Creating test user profile..."
docker-compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db -e "
INSERT IGNORE INTO user_profiles (
    wallet_address, 
    display_name, 
    registration_date, 
    token_uri, 
    ipfs_hash
) VALUES (
    '0xTEST1234567890ABCDEF',
    'Test User for PoCW',
    NOW(),
    'https://test.com/token',
    'QmTest123456789'
);
" 2>/dev/null

echo "✅ Test user created"
echo ""

# 2. Create test tasks
echo "2️⃣ Creating test Twitter retweet tasks..."

for i in {1..5}; do
    TASK_ID="test-task-$(date +%s)-$i"
    
    docker-compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db -e "
    INSERT INTO tasks (
        id,
        user_wallet,
        task_type,
        status,
        payload,
        vlc_clock,
        subnet_id,
        created_at,
        completed_at
    ) VALUES (
        '$TASK_ID',
        '0xTEST1234567890ABCDEF',
        'twitter_retweet',
        'VERIFIED',
        '{\"tweet_url\": \"https://twitter.com/test/status/123$i\"}',
        '{\"1\": 100$i}',
        'subnet-test',
        NOW(),
        NOW()
    );
    " 2>/dev/null
    
    echo "   ✓ Created task $i: $TASK_ID"
done

echo ""

# 3. Create user_task_completions records (这些会被 PoCW 处理)
echo "3️⃣ Creating user task completion records..."

for i in {1..5}; do
    TASK_ID="test-task-$(date +%s)-$i"
    
    docker-compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db -e "
    INSERT INTO user_task_completions (
        user_wallet,
        task_id,
        task_type,
        subnet_id,
        points_earned,
        completed_at
    ) VALUES (
        '0xTEST1234567890ABCDEF',
        '$TASK_ID',
        'twitter_retweet',
        'subnet-test',
        0,
        NOW()
    );
    " 2>/dev/null
    
    echo "   ✓ Created completion record $i"
done

echo ""
echo "✅ Test data created successfully!"
echo ""
echo "📊 Summary:"
docker-compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db -e "
SELECT 
    'Tasks' as type,
    COUNT(*) as count
FROM tasks 
WHERE user_wallet = '0xTEST1234567890ABCDEF'
UNION ALL
SELECT 
    'Completions (pending PoCW)' as type,
    COUNT(*) as count
FROM user_task_completions
WHERE user_wallet = '0xTEST1234567890ABCDEF'
AND points_earned = 0;
" 2>/dev/null

echo ""
echo "⏰ Next steps:"
echo "   1. Wait for the next round (every 5 minutes)"
echo "   2. Check round data: curl http://localhost:8080/api/v1/pocw/rounds?limit=1"
echo "   3. Check dashboard: curl http://localhost:8080/api/v1/pocw/stats/dashboard"
echo ""
echo "🔍 Monitor logs:"
echo "   docker-compose logs -f miner-gateway | grep -i 'round\\|pocw\\|test-task'"
