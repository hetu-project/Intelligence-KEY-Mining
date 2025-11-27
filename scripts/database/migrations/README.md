# Database Migrations

## PoCW Browser Migration (006_pocw_browser.sql)

### 概述
此迁移脚本创建 PoCW 浏览器所需的数据库表和视图。

### ⚠️ 重要提示
**此脚本会删除并重新创建 PoCW 相关的表！** 所有现有数据将丢失。

### 适用场景
- ✅ **初始安装**：第一次部署 PoCW 浏览器
- ✅ **开发环境**：需要重置表结构
- ✅ **表结构更新**：需要修改表定义时

### 不适用场景
- ❌ **生产环境有数据**：如果已有重要的 PoCW 数据，请先备份

---

## 使用方法

### 1. 执行迁移

```bash
# 方式1：使用 Docker Compose
docker-compose exec -T mysql mysql -upocw_user -ppocw_password pocw_db < scripts/database/migrations/006_pocw_browser.sql

# 方式2：直接连接 MySQL
mysql -h localhost -P 3306 -upocw_user -ppocw_password pocw_db < scripts/database/migrations/006_pocw_browser.sql
```

### 2. 验证安装

```bash
# 快速验证
bash scripts/verify-pocw-tables.sh

# 或手动检查
docker-compose exec mysql mysql -upocw_user -ppocw_password pocw_db -e "SHOW TABLES LIKE 'pocw%';"
```

### 3. 重启服务

```bash
# 重启 miner-gateway 以加载新的存储服务
docker-compose restart miner-gateway

# 查看日志确认正常运行
docker-compose logs -f miner-gateway | grep -i pocw
```

---

## 创建的对象

### 表 (3个)

1. **pocw_rounds** - 轮次数据
   - 主键: `round_id` (VARCHAR(100))
   - 存储：轮次生命周期、VLC 状态、共识结果

2. **pocw_votes** - 投票记录
   - 主键: `id` (BIGINT AUTO_INCREMENT)
   - 外键: `round_id` → `pocw_rounds.round_id`
   - 存储：验证者投票、质量分数、权重

3. **pocw_round_tasks** - 轮次任务关联
   - 主键: `id` (BIGINT AUTO_INCREMENT)
   - 外键: 
     - `round_id` → `pocw_rounds.round_id`
     - `task_id` → `tasks.id`
   - 存储：任务与轮次的关联、共识结果、积分

### 视图 (2个)

1. **pocw_round_summary** - 轮次汇总视图
   - 聚合轮次统计和投票数据

2. **pocw_validator_performance** - 验证者性能视图
   - 验证者投票统计和表现

---

## 字符集说明

所有表使用：
- **字符集**: `utf8mb4`
- **排序规则**: `utf8mb4_0900_ai_ci`

这与现有的 `tasks` 表保持一致，确保外键约束正常工作。

---

## 故障排除

### 问题1: 外键约束错误
```
ERROR 3780 (HY000): Referencing column and referenced column are incompatible
```

**原因**: 字符集或排序规则不匹配

**解决**: 
- 检查 `tasks` 表的字符集：`SHOW CREATE TABLE tasks\G`
- 确保迁移脚本使用相同的字符集

### 问题2: 表已存在
```
ERROR 1050 (42S01): Table 'pocw_rounds' already exists
```

**原因**: 之前执行过部分迁移

**解决**: 
- 脚本已包含 `DROP TABLE IF EXISTS`，重新执行即可
- 或手动删除：`DROP TABLE IF EXISTS pocw_rounds;`

### 问题3: 权限不足
```
ERROR 1142 (42000): DROP command denied
```

**原因**: 数据库用户权限不足

**解决**: 
- 使用 root 用户执行
- 或授予用户 DROP 权限

---

## 回滚

如果需要删除 PoCW 浏览器表：

```sql
-- 删除视图
DROP VIEW IF EXISTS pocw_validator_performance;
DROP VIEW IF EXISTS pocw_round_summary;

-- 删除表（注意顺序）
DROP TABLE IF EXISTS pocw_round_tasks;
DROP TABLE IF EXISTS pocw_votes;
DROP TABLE IF EXISTS pocw_rounds;
```

---

## 版本历史

- **v1.0** (2025-11-21)
  - 初始版本
  - 创建 3 个表和 2 个视图
  - 支持完整的 PoCW 数据持久化

---

## 相关文档

- [PoCW 浏览器实现指南](../../docs/pocw-browser-implementation-guide.md)
- [API 接口文档](../../services/miner-gateway/handlers/pocw_handler.go)
- [数据验证脚本](../verify-pocw-data.sh)
