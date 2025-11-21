# PoCW 浏览器实现指南

## ✅ 已完成的工作

### 1. Chat 任务每日限制已注释
**文件**: `services/points-service/handlers/nft_handler.go`

已将每日限制检查代码注释掉，现在用户可以一天内多次完成 Chat 任务。

```go
// NOTE: Daily limit check is temporarily disabled to allow multiple chat tasks per day
// completed, err := nh.pointsService.CheckUserChatTaskToday(...)
```

### 2. 数据库表结构已创建
**文件**: `scripts/database/migrations/006_pocw_browser.sql`

创建了三个核心表：
- `pocw_rounds`: 存储轮次数据
- `pocw_votes`: 存储验证者投票
- `pocw_round_tasks`: 存储轮次-任务关联

### 3. PoCW 存储服务已实现
**文件**: `services/miner-gateway/services/pocw_storage.go`

实现了完整的数据持久化服务，包括：
- `SaveRoundStart()`: 保存轮次开始
- `SaveVote()`: 保存投票记录
- `SaveRoundTasks()`: 保存任务关联
- `UpdateRoundConsensus()`: 更新共识结果
- `CompleteRound()`: 完成轮次
- `GetRecentRounds()`: 查询最近轮次

### 4. RoundCoordinator 已集成存储
**文件**: `services/miner-gateway/services/coordinator.go`

在关键位置添加了数据保存：
- ✅ `startRound()`: 保存轮次开始和任务列表
- ✅ `qualityVotingPhase()`: 保存投票记录
- ✅ `consensusPhase()`: 保存共识结果
- ✅ `completeRound()`: 保存轮次完成状态

---

## 📋 下一步操作

### 第一步：运行数据库迁移

```bash
# 方式1：使用 Docker
docker-compose exec mysql mysql -upocw_user -ppocw_password pocw_db < scripts/database/migrations/006_pocw_browser.sql

# 方式2：直接连接
mysql -h localhost -P 3306 -upocw_user -ppocw_password pocw_db < scripts/database/migrations/006_pocw_browser.sql
```

### 第二步：重启服务

```bash
# 重新构建并启动 miner-gateway
docker-compose stop miner-gateway
docker-compose build miner-gateway
docker-compose up -d miner-gateway

# 重启 points-service (Chat 任务限制修改)
docker-compose stop points-service
docker-compose build points-service
docker-compose up -d points-service

# 查看日志确认启动成功
docker-compose logs -f miner-gateway | grep -i "round\|pocw"
```

### 第三步：验证数据保存

等待一个 Round 完成后，查询数据库：

```sql
-- 查看最近的轮次
SELECT * FROM pocw_rounds ORDER BY start_time DESC LIMIT 5;

-- 查看投票记录
SELECT * FROM pocw_votes LIMIT 10;

-- 查看轮次任务
SELECT * FROM pocw_round_tasks LIMIT 10;

-- 使用视图查看汇总数据
SELECT * FROM pocw_round_summary LIMIT 5;
```

---

## 🎯 后续开发步骤

### 阶段 2：创建查询 API（预计1周）

需要创建的文件：
```
services/miner-gateway/handlers/pocw_handler.go  # API 处理器
services/miner-gateway/services/pocw_query.go    # 查询服务
```

需要实现的 API：
```
GET /api/v1/pocw/rounds                    # 轮次列表
GET /api/v1/pocw/rounds/:round_id          # 轮次详情
GET /api/v1/pocw/rounds/:round_id/votes    # 轮次投票
GET /api/v1/pocw/stats/dashboard           # Dashboard 统计
GET /api/v1/pocw/stats/validators          # 验证者性能
```

### 阶段 3：前端开发（预计2周）

页面结构：
```
pocw-browser/
├── pages/
│   ├── Dashboard.tsx          # 首页总览
│   ├── RoundList.tsx          # 轮次列表
│   ├── RoundDetail.tsx        # 轮次详情
│   ├── ValidatorList.tsx      # 验证者列表
│   └── VLCExplorer.tsx        # VLC 浏览器
├── components/
│   ├── RoundCard.tsx          # 轮次卡片
│   ├── VoteTable.tsx          # 投票表格
│   ├── ConsensusChart.tsx     # 共识图表
│   └── VLCTimeline.tsx        # VLC 时间线
└── api/
    └── pocw.ts                # API 调用封装
```

---

## 📊 数据流程图

```
用户完成任务
    ↓
BatchVerifier 验证
    ↓
RoundCoordinator.startRound()
    ├─→ SaveRoundStart() ✅ 保存到 pocw_rounds
    └─→ SaveRoundTasks() ✅ 保存到 pocw_round_tasks
    ↓
qualityVotingPhase()
    └─→ SaveVote() ✅ 保存到 pocw_votes
    ↓
consensusPhase()
    └─→ UpdateRoundConsensus() ✅ 更新 pocw_rounds
    ↓
completeRound()
    └─→ CompleteRound() ✅ 完成 pocw_rounds
    ↓
数据库持久化完成 ✅
    ↓
PoCW 浏览器可查询 📊
```

---

## 🔍 调试和监控

### 查看 PoCW 日志

```bash
# 查看轮次相关日志
docker-compose logs -f miner-gateway | grep "Round"

# 查看数据保存日志
docker-compose logs -f miner-gateway | grep "Saved\|Failed to save"

# 查看共识日志
docker-compose logs -f miner-gateway | grep "Consensus"
```

### 监控数据增长

```sql
-- 每小时轮次数
SELECT 
    DATE_FORMAT(start_time, '%Y-%m-%d %H:00:00') as hour,
    COUNT(*) as round_count,
    SUM(task_count) as total_tasks
FROM pocw_rounds
WHERE start_time >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
GROUP BY hour
ORDER BY hour DESC;

-- 验证者投票统计
SELECT 
    validator_id,
    COUNT(*) as vote_count,
    AVG(quality_score) as avg_score
FROM pocw_votes
WHERE vote_timestamp >= DATE_SUB(NOW(), INTERVAL 24 HOUR)
GROUP BY validator_id;
```

---

## ⚠️ 注意事项

1. **数据库性能**
   - 轮次数据会持续增长，建议定期归档旧数据
   - 考虑添加分区表（按月份）

2. **错误处理**
   - 数据保存失败不会中断 Round 流程
   - 所有错误都会记录到日志

3. **事务处理**
   - `SaveRoundTasks` 使用事务批量插入
   - 其他操作独立提交

4. **查询优化**
   - 已创建必要的索引
   - 使用视图简化复杂查询

---

## 📞 联系和支持

如果遇到问题：
1. 查看日志：`docker-compose logs miner-gateway`
2. 检查数据库：连接 MySQL 查询表数据
3. 验证迁移：确认表已创建

下一步需要帮助时，请告诉我你在哪个阶段！
