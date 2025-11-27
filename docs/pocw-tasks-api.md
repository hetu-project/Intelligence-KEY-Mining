# PoCW 任务 API 文档

## 新增接口

### 1. 获取任务列表

**接口**: `GET /api/v1/pocw/tasks`

**描述**: 获取已进入 PoCW 共识的任务列表，支持分页和多种筛选条件

**请求参数**:

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码 |
| limit | int | 否 | 20 | 每页数量 (最大100) |
| round_id | string | 否 | - | 按轮次ID筛选 |
| user_wallet | string | 否 | - | 按用户钱包地址筛选 |
| task_type | string | 否 | - | 按任务类型筛选 |
| subnet_id | string | 否 | - | 按子网ID筛选 |
| verdict | string | 否 | - | 按共识状态筛选: `awaiting`, `approved`, `rejected` |

**响应示例**:

```json
{
  "success": true,
  "data": {
    "tasks": [
      {
        "task_id": "task-abc-123",
        "user_wallet": "0x123...456",
        "task_type": "twitter_retweet",
        "subnet_id": "subnet-linkol",
        "created_at": "2025-11-21T10:00:00Z",
        "round_id": "round_1732161600",
        "verdict": "approved",
        "credit": 1,
        "new_vlc": 1001,
        "processed_at": "2025-11-21T10:00:05Z"
      },
      {
        "task_id": "task-def-456",
        "user_wallet": "0x789...012",
        "task_type": "twitter_retweet",
        "subnet_id": "subnet-linkol",
        "created_at": "2025-11-21T09:55:00Z",
        "round_id": "round_1732161300",
        "verdict": "awaiting",
        "credit": 1,
        "new_vlc": 0,
        "processed_at": null
      }
    ],
    "total": 150,
    "page": 1,
    "limit": 20,
    "total_pages": 8
  }
}
```

**字段说明**:

- `task_id`: 任务唯一标识符
- `user_wallet`: 用户钱包地址
- `task_type`: 任务类型 (如 `twitter_retweet`, `telegram_task` 等)
- `subnet_id`: 所属子网ID
- `created_at`: 任务创建时间
- `round_id`: 所属轮次ID
- `verdict`: 共识状态
  - `awaiting`: 等待共识（已进入轮次但还未完成共识）
  - `approved`: 共识通过
  - `rejected`: 共识拒绝
- `credit`: VLC 增量（通常为 1）
- `new_vlc`: 用户在此任务后的 VLC 值
- `processed_at`: 任务处理时间

**使用示例**:

```bash
# 1. 获取所有任务（分页）
curl "http://localhost:8080/api/v1/pocw/tasks?page=1&limit=20"

# 2. 获取特定用户的任务
curl "http://localhost:8080/api/v1/pocw/tasks?user_wallet=0x123...456"

# 3. 获取特定轮次的任务
curl "http://localhost:8080/api/v1/pocw/tasks?round_id=round_1732161600"

# 4. 获取已通过共识的任务
curl "http://localhost:8080/api/v1/pocw/tasks?verdict=approved"

# 5. 组合筛选：特定子网的等待共识任务
curl "http://localhost:8080/api/v1/pocw/tasks?subnet_id=subnet-linkol&verdict=awaiting"

# 6. 按任务类型筛选
curl "http://localhost:8080/api/v1/pocw/tasks?task_type=twitter_retweet&limit=50"
```

---

### 2. 获取任务详情

**接口**: `GET /api/v1/pocw/tasks/:task_id`

**描述**: 获取特定任务的详细信息，包括完整的投票记录

**路径参数**:

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| task_id | string | 是 | 任务ID |

**响应示例**:

```json
{
  "success": true,
  "data": {
    "task_id": "task-abc-123",
    "user_wallet": "0x123...456",
    "task_type": "twitter_retweet",
    "subnet_id": "subnet-linkol",
    "status": "COMPLETED",
    "created_at": "2025-11-21T10:00:00Z",
    "completed_at": "2025-11-21T10:00:02Z",
    
    "round_id": "round_1732161600",
    "processed_at": "2025-11-21T10:00:05Z",
    
    "verdict": "approved",
    "vlc_increment": 1,
    "vlc_before": 1000,
    "vlc_after": 1001,
    "vlc_snapshot": {
      "1": 1001
    },
    "points_awarded": 1,
    
    "payload": {
      "tweet_url": "https://twitter.com/user/status/123456",
      "action": "retweet"
    },
    "proof": {
      "screenshot_url": "https://...",
      "verified_at": "2025-11-21T10:00:02Z"
    },
    
    "votes": [
      {
        "validator_id": "validator-1",
        "validator_role": "ui_validator",
        "task_id": "task-abc-123",
        "vote": "approve",
        "quality_score": 0.95,
        "weight": 0.40,
        "reasoning": "Task meets quality standards",
        "vote_timestamp": "2025-11-21T10:00:03Z"
      },
      {
        "validator_id": "validator-2",
        "validator_role": "format_validator",
        "task_id": "task-abc-123",
        "vote": "approve",
        "quality_score": 0.88,
        "weight": 0.20,
        "vote_timestamp": "2025-11-21T10:00:03Z"
      },
      {
        "validator_id": "validator-3",
        "validator_role": "semantic_validator",
        "task_id": "task-abc-123",
        "vote": "approve",
        "quality_score": 0.92,
        "weight": 0.40,
        "vote_timestamp": "2025-11-21T10:00:04Z"
      }
    ],
    "total_votes": 3,
    "approve_votes": 3,
    "reject_votes": 0
  }
}
```

**字段说明**:

**基本信息**:
- `task_id`: 任务ID
- `user_wallet`: 用户钱包地址
- `task_type`: 任务类型
- `subnet_id`: 子网ID
- `status`: 任务状态 (`PENDING`, `VERIFIED`, `COMPLETED`, `FAILED`)
- `created_at`: 创建时间
- `completed_at`: 完成时间

**PoCW Round 信息**:
- `round_id`: 所属轮次ID
- `processed_at`: 在轮次中的处理时间

**共识信息**:
- `verdict`: 共识结果
  - `approved`: 通过
  - `rejected`: 拒绝
  - `awaiting`: 等待中
  - `not_in_consensus`: 未进入共识
- `vlc_increment`: VLC 增量
- `vlc_before`: 处理前的 VLC 值
- `vlc_after`: 处理后的 VLC 值
- `vlc_snapshot`: VLC 快照（完整状态）
- `points_awarded`: 获得的积分

**任务数据**:
- `payload`: 任务载荷（原始任务数据）
- `proof`: 任务证明（验证数据）

**投票详情**:
- `votes`: 验证者投票列表
  - `validator_id`: 验证者ID
  - `validator_role`: 验证者角色
  - `vote`: 投票结果 (`approve`, `reject`, `abstain`)
  - `quality_score`: 质量分数 (0-1)
  - `weight`: 投票权重
  - `reasoning`: 投票理由
  - `vote_timestamp`: 投票时间
- `total_votes`: 总投票数
- `approve_votes`: 同意票数
- `reject_votes`: 拒绝票数

**使用示例**:

```bash
# 获取任务详情
curl "http://localhost:8080/api/v1/pocw/tasks/task-abc-123"

# 使用 jq 格式化输出
curl "http://localhost:8080/api/v1/pocw/tasks/task-abc-123" | jq

# 只查看投票信息
curl "http://localhost:8080/api/v1/pocw/tasks/task-abc-123" | jq '.data.votes'

# 查看共识结果
curl "http://localhost:8080/api/v1/pocw/tasks/task-abc-123" | jq '.data | {verdict, vlc_increment, points_awarded}'
```

---

## 数据流说明

### 任务生命周期

```
1. 用户完成任务
   ↓
2. 任务创建 (tasks 表)
   - status: PENDING
   - created_at: 当前时间
   ↓
3. BatchVerifier 验证
   - status: VERIFIED
   - completed_at: 验证时间
   ↓
4. 进入 PoCW Round (pocw_round_tasks 表)
   - round_id: 分配的轮次
   - verdict: null (等待共识)
   ↓
5. VLC 验证
   - vlc_snapshot: 记录 VLC 状态
   ↓
6. 验证者投票 (pocw_votes 表)
   - 多个验证者投票
   - 记录 quality_score, weight
   ↓
7. BFT 共识
   - consensus_result: approved/rejected
   - points_awarded: 分配积分
   - verdict: approved/rejected
   ↓
8. 完成
   - processed_at: 处理完成时间
```

### Verdict 状态说明

| Verdict | 含义 | 数据库状态 |
|---------|------|-----------|
| `not_in_consensus` | 未进入共识 | `pocw_round_tasks` 中无记录 |
| `awaiting` | 等待共识 | `consensus_result` 为 NULL |
| `approved` | 共识通过 | `consensus_result` = 'approved' |
| `rejected` | 共识拒绝 | `consensus_result` = 'rejected' |

### VLC 计算逻辑

```javascript
// 任务列表中的 new_vlc
new_vlc = vlc_snapshot["1"]  // 从快照中提取用户的 VLC

// 任务详情中的 VLC 计算
vlc_after = vlc_snapshot["1"]
vlc_before = vlc_after - vlc_increment
```

**注意**: 当前实现假设 miner ID 为 "1"，实际应该根据 `user_wallet` 动态获取对应的 miner ID。

---

## 前端集成示例

### React/TypeScript

```typescript
// api/pocw-tasks.ts
import axios from 'axios';

const BASE_URL = 'http://localhost:8080/api/v1/pocw';

export interface TaskListItem {
  task_id: string;
  user_wallet: string;
  task_type: string;
  subnet_id?: string;
  created_at: string;
  round_id?: string;
  verdict: string;
  credit: number;
  new_vlc: number;
  processed_at?: string;
}

export interface TaskDetail extends TaskListItem {
  status: string;
  completed_at?: string;
  vlc_increment: number;
  vlc_before: number;
  vlc_after: number;
  vlc_snapshot?: Record<string, any>;
  points_awarded: number;
  payload?: Record<string, any>;
  proof?: Record<string, any>;
  votes: Vote[];
  total_votes: number;
  approve_votes: number;
  reject_votes: number;
}

export interface Vote {
  validator_id: string;
  validator_role?: string;
  task_id: string;
  vote: string;
  quality_score: number;
  weight: number;
  reasoning?: string;
  vote_timestamp: string;
}

export const tasksApi = {
  // 获取任务列表
  getTasksList: async (params: {
    page?: number;
    limit?: number;
    round_id?: string;
    user_wallet?: string;
    task_type?: string;
    subnet_id?: string;
    verdict?: string;
  }) => {
    const response = await axios.get(`${BASE_URL}/tasks`, { params });
    return response.data;
  },

  // 获取任务详情
  getTaskDetail: async (taskId: string) => {
    const response = await axios.get(`${BASE_URL}/tasks/${taskId}`);
    return response.data;
  },
};
```

### 使用示例

```typescript
// TasksList.tsx
import { useEffect, useState } from 'react';
import { tasksApi, TaskListItem } from './api/pocw-tasks';

function TasksList() {
  const [tasks, setTasks] = useState<TaskListItem[]>([]);
  const [page, setPage] = useState(1);
  const [total, setTotal] = useState(0);

  useEffect(() => {
    tasksApi.getTasksList({ page, limit: 20 }).then(res => {
      if (res.success) {
        setTasks(res.data.tasks);
        setTotal(res.data.total);
      }
    });
  }, [page]);

  return (
    <div>
      <h1>PoCW Tasks</h1>
      <table>
        <thead>
          <tr>
            <th>Task ID</th>
            <th>User</th>
            <th>Type</th>
            <th>Subnet</th>
            <th>Round</th>
            <th>Verdict</th>
            <th>Credit</th>
            <th>New VLC</th>
          </tr>
        </thead>
        <tbody>
          {tasks.map(task => (
            <tr key={task.task_id}>
              <td>{task.task_id}</td>
              <td>{task.user_wallet.slice(0, 10)}...</td>
              <td>{task.task_type}</td>
              <td>{task.subnet_id}</td>
              <td>{task.round_id}</td>
              <td>
                <span className={`badge ${task.verdict}`}>
                  {task.verdict}
                </span>
              </td>
              <td>{task.credit}</td>
              <td>{task.new_vlc}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <div>
        Total: {total} | Page: {page}
      </div>
    </div>
  );
}
```

```typescript
// TaskDetail.tsx
import { useEffect, useState } from 'react';
import { useParams } from 'react-router-dom';
import { tasksApi, TaskDetail } from './api/pocw-tasks';

function TaskDetailPage() {
  const { taskId } = useParams<{ taskId: string }>();
  const [task, setTask] = useState<TaskDetail | null>(null);

  useEffect(() => {
    if (taskId) {
      tasksApi.getTaskDetail(taskId).then(res => {
        if (res.success) {
          setTask(res.data);
        }
      });
    }
  }, [taskId]);

  if (!task) return <div>Loading...</div>;

  return (
    <div>
      <h1>Task Detail: {task.task_id}</h1>
      
      <section>
        <h2>Basic Info</h2>
        <p>User: {task.user_wallet}</p>
        <p>Type: {task.task_type}</p>
        <p>Subnet: {task.subnet_id}</p>
        <p>Status: {task.status}</p>
      </section>

      <section>
        <h2>Consensus Info</h2>
        <p>Round: {task.round_id}</p>
        <p>Verdict: <span className={task.verdict}>{task.verdict}</span></p>
        <p>VLC: {task.vlc_before} → {task.vlc_after} (+{task.vlc_increment})</p>
        <p>Points: {task.points_awarded}</p>
      </section>

      <section>
        <h2>Votes ({task.total_votes})</h2>
        <p>Approve: {task.approve_votes} | Reject: {task.reject_votes}</p>
        <table>
          <thead>
            <tr>
              <th>Validator</th>
              <th>Role</th>
              <th>Vote</th>
              <th>Quality</th>
              <th>Weight</th>
            </tr>
          </thead>
          <tbody>
            {task.votes.map((vote, idx) => (
              <tr key={idx}>
                <td>{vote.validator_id}</td>
                <td>{vote.validator_role}</td>
                <td>{vote.vote}</td>
                <td>{vote.quality_score.toFixed(2)}</td>
                <td>{vote.weight.toFixed(2)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </section>
    </div>
  );
}
```

---

## 注意事项

### 1. VLC Snapshot 中的 Miner ID

当前实现中，`new_vlc` 是从 `vlc_snapshot["1"]` 中提取的，假设 miner ID 为 "1"。

**TODO**: 需要根据 `user_wallet` 动态映射到对应的 miner ID。

### 2. 任务列表只显示已进入 PoCW 的任务

任务列表查询的是 `pocw_round_tasks` 表，只包含已经进入 PoCW 共识流程的任务。

如果需要查看所有任务（包括未进入共识的），需要额外的接口。

### 3. Verdict 状态的判断逻辑

```sql
CASE
  WHEN consensus_result IS NULL THEN 'awaiting'
  WHEN consensus_result = 'approved' THEN 'approved'
  WHEN consensus_result = 'rejected' THEN 'rejected'
END
```

### 4. 性能考虑

- 任务列表支持多种筛选条件，建议在数据库中添加相应索引
- 大量数据时建议使用分页
- 任务详情包含投票信息，查询较重，建议添加缓存

---

## 总结

✅ **已实现**:
1. 任务列表接口 - 支持分页和多种筛选
2. 任务详情接口 - 包含完整投票信息
3. 路由注册
4. API 文档

📝 **待优化**:
1. VLC Snapshot 中的 Miner ID 动态映射
2. 添加数据库索引优化查询性能
3. 考虑添加缓存机制
4. 前端组件完善
