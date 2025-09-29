#!/bin/bash
echo "=== 检查所有相关表结构 ==="
echo ""

echo "1. tasks 表结构 (应该有 subnet_id, expires_at 字段):"
docker-compose exec mysql mysql -u pocw_user -ppocw_password -e "DESCRIBE tasks;" pocw_db
echo ""

echo "2. subnets 表结构:"
docker-compose exec mysql mysql -u pocw_user -ppocw_password -e "DESCRIBE subnets;" pocw_db
echo ""

echo "3. user_task_completions 表结构:"
docker-compose exec mysql mysql -u pocw_user -ppocw_password -e "DESCRIBE user_task_completions;" pocw_db
echo ""

echo "4. daily_distribution_log 表结构:"
docker-compose exec mysql mysql -u pocw_user -ppocw_password -e "DESCRIBE daily_distribution_log;" pocw_db
echo ""

echo "5. nft_ownership_cache 表结构:"
docker-compose exec mysql mysql -u pocw_user -ppocw_password -e "DESCRIBE nft_ownership_cache;" pocw_db
echo ""

echo "6. invitation_rewards 表结构:"
docker-compose exec mysql mysql -u pocw_user -ppocw_password -e "DESCRIBE invitation_rewards;" pocw_db
echo ""

echo "7. points_history 表结构 (检查是否有新字段):"
docker-compose exec mysql mysql -u pocw_user -ppocw_password -e "DESCRIBE points_history;" pocw_db
echo ""

echo "8. user_profiles 表结构 (检查基础结构):"
docker-compose exec mysql mysql -u pocw_user -ppocw_password -e "DESCRIBE user_profiles;" pocw_db
echo ""

echo "=== 检查完成 ==="

