#!/bin/bash

# PoCW Browser Deployment Script
# This script deploys the PoCW browser database tables and restarts services

set -e  # Exit on error

echo "🚀 Starting PoCW Browser Deployment..."
echo ""

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Step 1: Check if Docker is running
echo "📋 Step 1: Checking Docker..."
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running. Please start Docker first.${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Docker is running${NC}"
echo ""

# Step 2: Run database migration
echo "📋 Step 2: Running database migration..."
if docker-compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db < scripts/database/migrations/006_pocw_browser.sql; then
    echo -e "${GREEN}✅ Database migration completed${NC}"
else
    echo -e "${RED}❌ Database migration failed${NC}"
    echo -e "${YELLOW}💡 Tip: Make sure MySQL container is running: docker-compose ps mysql${NC}"
    exit 1
fi
echo ""

# Step 3: Verify tables created
echo "📋 Step 3: Verifying tables..."
TABLE_COUNT=$(docker-compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db -e "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='pocw_db' AND table_name IN ('pocw_rounds', 'pocw_votes', 'pocw_round_tasks');" -s -N)

if [ "$TABLE_COUNT" -eq "3" ]; then
    echo -e "${GREEN}✅ All 3 tables created successfully${NC}"
else
    echo -e "${YELLOW}⚠️  Expected 3 tables, found $TABLE_COUNT${NC}"
fi
echo ""

# Step 4: Rebuild and restart miner-gateway
echo "📋 Step 4: Rebuilding miner-gateway service..."
docker-compose stop miner-gateway
docker-compose build miner-gateway
docker-compose up -d miner-gateway
echo -e "${GREEN}✅ Miner-gateway restarted${NC}"
echo ""

# Step 5: Rebuild and restart points-service (for Chat task limit change)
echo "📋 Step 5: Rebuilding points-service..."
docker-compose stop points-service
docker-compose build points-service
docker-compose up -d points-service
echo -e "${GREEN}✅ Points-service restarted${NC}"
echo ""

# Step 6: Wait for services to be healthy
echo "📋 Step 6: Waiting for services to be healthy..."
sleep 5

# Check miner-gateway health
if docker-compose ps miner-gateway | grep -q "Up"; then
    echo -e "${GREEN}✅ Miner-gateway is running${NC}"
else
    echo -e "${RED}❌ Miner-gateway failed to start${NC}"
fi

# Check points-service health
if docker-compose ps points-service | grep -q "Up"; then
    echo -e "${GREEN}✅ Points-service is running${NC}"
else
    echo -e "${RED}❌ Points-service failed to start${NC}"
fi
echo ""

# Step 7: Show sample queries
echo "📋 Step 7: Sample verification queries"
echo ""
echo -e "${YELLOW}Run these commands to verify data is being saved:${NC}"
echo ""
echo "# Check recent rounds:"
echo "docker-compose exec mysql mysql -upocw_user -ppocw_password pocw_db -e \"SELECT round_id, start_time, phase, task_count FROM pocw_rounds ORDER BY start_time DESC LIMIT 5;\""
echo ""
echo "# Check votes:"
echo "docker-compose exec mysql mysql -upocw_user -ppocw_password pocw_db -e \"SELECT round_id, validator_id, vote, COUNT(*) as count FROM pocw_votes GROUP BY round_id, validator_id, vote LIMIT 10;\""
echo ""
echo "# Check round tasks:"
echo "docker-compose exec mysql mysql -upocw_user -ppocw_password pocw_db -e \"SELECT round_id, COUNT(*) as task_count FROM pocw_round_tasks GROUP BY round_id LIMIT 5;\""
echo ""

# Step 8: Show logs
echo "📋 Step 8: Monitoring logs..."
echo -e "${YELLOW}Press Ctrl+C to stop log monitoring${NC}"
echo ""
sleep 2

docker-compose logs -f --tail=50 miner-gateway | grep -i "round\|pocw\|saved\|consensus"
