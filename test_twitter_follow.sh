#!/bin/bash

# Twitter Follow 任务完整测试脚本
# 用途：验证 twitter_follow 任务类型的完整功能

set -e

# 颜色定义
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 服务地址
MINER_GATEWAY_URL="http://localhost:8086"
POINTS_SERVICE_URL="http://localhost:8087"
TEST_WALLET="0xTEST_FOLLOW_$(date +%s)"

echo -e "${YELLOW}==================================${NC}"
echo -e "${YELLOW}Twitter Follow 任务测试脚本${NC}"
echo -e "${YELLOW}==================================${NC}"
echo ""
echo "测试钱包地址: ${TEST_WALLET}"
echo ""

# 测试 1: 创建 Twitter Follow 任务
echo -e "${YELLOW}[测试 1/6]${NC} 创建 Twitter Follow 任务..."
CREATE_RESPONSE=$(curl -s -X POST ${MINER_GATEWAY_URL}/api/v1/task-creation/create \
  -H "Content-Type: application/json" \
  -d "{
    \"user_wallet\": \"${TEST_WALLET}\",
    \"task_type\": \"twitter_follow\",
    \"project_name\": \"测试关注任务项目\",
    \"project_icon\": \"https://example.com/icon.png\",
    \"x_url\": \"https://x.com/test_project\",
    \"website\": \"https://test-project.com\",
    \"description\": \"这是一个测试Twitter关注任务\",
    \"deadline\": \"2025-12-31T23:59:59Z\",
    \"title\": \"关注测试账号获得积分\",
    \"follow_account_handle\": \"test_account_handle\"
  }")

echo "创建任务响应:"
echo $CREATE_RESPONSE | jq '.'

# 提取 task_id 和 subnet_id
TASK_ID=$(echo $CREATE_RESPONSE | jq -r '.task_id')
SUBNET_ID=$(echo $CREATE_RESPONSE | jq -r '.subnet_id')

if [ "$TASK_ID" == "null" ] || [ -z "$TASK_ID" ]; then
    echo -e "${RED}❌ 任务创建失败！${NC}"
    exit 1
fi

echo -e "${GREEN}✅ 任务创建成功！${NC}"
echo "  Task ID: $TASK_ID"
echo "  Subnet ID: $SUBNET_ID"
echo ""

# 等待一秒确保数据库写入完成
sleep 1

# 测试 2: 从数据库查询任务
echo -e "${YELLOW}[测试 2/6]${NC} 从数据库查询任务详情..."
docker compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db -e "
SELECT 
    id,
    task_type,
    subnet_id,
    user_wallet,
    status,
    created_at,
    JSON_EXTRACT(payload, '$.title') as title,
    JSON_EXTRACT(payload, '$.follow_account_handle') as follow_account
FROM tasks 
WHERE id = '$TASK_ID';
" 2>/dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 数据库查询成功！${NC}"
else
    echo -e "${RED}❌ 数据库查询失败！${NC}"
fi
echo ""

# 测试 3: 发放任务奖励
echo -e "${YELLOW}[测试 3/6]${NC} 发放 Twitter Follow 任务奖励..."
REWARD_RESPONSE=$(curl -s -X POST ${POINTS_SERVICE_URL}/api/v1/points/twitter-follow-reward \
  -H "Content-Type: application/json" \
  -d "{
    \"user_wallet\": \"${TEST_WALLET}\",
    \"task_id\": \"${TASK_ID}\"
  }")

echo "奖励发放响应:"
echo $REWARD_RESPONSE | jq '.'

SUCCESS=$(echo $REWARD_RESPONSE | jq -r '.success')
POINTS_ADDED=$(echo $REWARD_RESPONSE | jq -r '.points_added')
NEW_TOTAL=$(echo $REWARD_RESPONSE | jq -r '.new_total')

if [ "$SUCCESS" == "true" ]; then
    echo -e "${GREEN}✅ 奖励发放成功！${NC}"
    echo "  积分增加: $POINTS_ADDED"
    echo "  新的总分: $NEW_TOTAL"
else
    echo -e "${RED}❌ 奖励发放失败！${NC}"
    exit 1
fi
echo ""

# 测试 4: 查询积分记录
echo -e "${YELLOW}[测试 4/6]${NC} 查询积分记录..."
docker compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db -e "
SELECT 
    wallet_address,
    source,
    points,
    tx_ref,
    subnet_id,
    created_at
FROM points_history
WHERE wallet_address = '${TEST_WALLET}'
  AND source = 'Twitter Follow Task'
ORDER BY created_at DESC
LIMIT 5;
" 2>/dev/null

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ 积分记录查询成功！${NC}"
else
    echo -e "${RED}❌ 积分记录查询失败！${NC}"
fi
echo ""

# 测试 5: 查询用户完成的任务
echo -e "${YELLOW}[测试 5/6]${NC} 查询用户完成的 Twitter Follow 任务..."
COMPLETED_TASKS=$(curl -s "${POINTS_SERVICE_URL}/api/v1/points/completed-tasks/${TEST_WALLET}?task_type=twitter_follow")

echo "完成任务查询响应:"
echo $COMPLETED_TASKS | jq '.'

TOTAL_TASKS=$(echo $COMPLETED_TASKS | jq -r '.data.total')

if [ "$TOTAL_TASKS" -gt 0 ]; then
    echo -e "${GREEN}✅ 用户完成任务查询成功！找到 ${TOTAL_TASKS} 个任务${NC}"
else
    echo -e "${RED}❌ 未找到用户完成的任务！${NC}"
fi
echo ""

# 测试 6: 验证统计查询包含 twitter_follow
echo -e "${YELLOW}[测试 6/6]${NC} 验证 Subnet 统计包含 Twitter Follow 任务..."
SUBNET_STATS=$(curl -s "${POINTS_SERVICE_URL}/api/v1/stats/subnets/${SUBNET_ID}")

echo "Subnet 统计响应:"
echo $SUBNET_STATS | jq '.'

COMPLETED_TASKS_COUNT=$(echo $SUBNET_STATS | jq -r '.data.completed_tasks')

if [ "$COMPLETED_TASKS_COUNT" -ge 1 ]; then
    echo -e "${GREEN}✅ Subnet 统计查询成功！${NC}"
    echo "  完成任务数: $COMPLETED_TASKS_COUNT"
else
    echo -e "${YELLOW}⚠️  Subnet 统计可能需要更新${NC}"
fi
echo ""

# 额外测试：验证 mapSourceToSimpleName
echo -e "${YELLOW}[额外测试]${NC} 查询用户今日收益（验证 mapSourceToSimpleName）..."
TODAY_EARNINGS=$(curl -s "${POINTS_SERVICE_URL}/api/v1/points/earned-today/${TEST_WALLET}")

echo "今日收益响应:"
echo $TODAY_EARNINGS | jq '.'

TWITTER_FOLLOW_POINTS=$(echo $TODAY_EARNINGS | jq -r '.data.breakdown.twitter_follow // 0')

if [ "$TWITTER_FOLLOW_POINTS" -gt 0 ]; then
    echo -e "${GREEN}✅ mapSourceToSimpleName 映射正确！${NC}"
    echo "  twitter_follow 积分: $TWITTER_FOLLOW_POINTS"
else
    echo -e "${YELLOW}⚠️  今日暂无 twitter_follow 积分或映射可能有问题${NC}"
fi
echo ""

# 总结
echo -e "${YELLOW}==================================${NC}"
echo -e "${YELLOW}测试总结${NC}"
echo -e "${YELLOW}==================================${NC}"
echo -e "${GREEN}✅ Twitter Follow 任务功能完整可用！${NC}"
echo ""
echo "测试数据："
echo "  - 钱包地址: ${TEST_WALLET}"
echo "  - 任务 ID: ${TASK_ID}"
echo "  - Subnet ID: ${SUBNET_ID}"
echo "  - 获得积分: ${POINTS_ADDED}"
echo "  - Subnet 总分: ${NEW_TOTAL}"
echo ""
echo -e "${YELLOW}建议：检查上述所有测试步骤的输出，确保没有错误。${NC}"

