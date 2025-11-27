# 主网-子网 PoCW 共识架构设计

## 1. 系统概述

### 1.1 架构对比

| 特性 | Bittensor | 本系统 (PoCW Network) |
|------|-----------|---------------------|
| 主链 | TAO 区块链 | PoCW 主网 (非区块链) |
| 子网 | Subnet (挖矿网络) | Application (应用网络) |
| 共识 | Yuma Consensus | PoCW (Proof of Cognitive Work) |
| 激励 | TAO Token | Flux/Credit 协议 |
| 验证 | Validator Staking | VLC + BFT 投票 |

### 1.2 核心设计原则

1. **分层共识**：主网和子网独立运行 PoCW，定期同步
2. **VLC 防溢出**：使用周期性重置 + 相对时钟
3. **协议化激励**：Flux (流动积分) + Credit (信用积分)
4. **去中心化验证**：每个应用自治，主网协调

---

## 2. 整体架构

```
┌─────────────────────────────────────────────────────────────┐
│                      PoCW 主网 (MainNet)                      │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │ Epoch Manager│  │ VLC Aggregator│ │ Flux Registry│      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
│  ┌──────────────────────────────────────────────────┐      │
│  │         主网 PoCW Consensus (每 Epoch)            │      │
│  │  - 验证子网提交的 VLC 状态                         │
│  │  - 计算 Flux 分配                                 │
│  │  - 更新 Credit 信用分                             │
│  └──────────────────────────────────────────────────┘      │
└─────────────────────────────────────────────────────────────┘
                            ↕ (每 Epoch 同步)
┌─────────────────────────────────────────────────────────────┐
│                    应用层 (Applications)                      │
│                                                               │
│  ┌──────────────────┐  ┌──────────────────┐  ┌────────────┐│
│  │  App-1 (Social)  │  │  App-2 (AI)      │  │  App-N     ││
│  │  ┌────────────┐  │  │  ┌────────────┐  │  │  ┌──────┐  ││
│  │  │ Local PoCW │  │  │  │ Local PoCW │  │  │  │ PoCW │  ││
│  │  │ (每5分钟)   │  │  │  │ (每5分钟)   │  │  │  │      │  ││
│  │  └────────────┘  │  │  └────────────┘  │  │  └──────┘  ││
│  │  ┌────────────┐  │  │  ┌────────────┐  │  │            ││
│  │  │ VLC State  │  │  │  │ VLC State  │  │  │            ││
│  │  └────────────┘  │  │  └────────────┘  │  │            ││
│  └──────────────────┘  └──────────────────┘  └────────────┘│
└─────────────────────────────────────────────────────────────┘
```

---

## 3. VLC 防溢出方案

### 3.1 问题分析

VLC 持续累加会导致：
- **整数溢出**：int64 最大值 9,223,372,036,854,775,807
- **存储膨胀**：JSON 数据越来越大
- **比较困难**：大数值比较效率低

### 3.2 解决方案：周期性 VLC 重置 (Epoch-based VLC)

```go
type EpochVLC struct {
    EpochID       int64            // 当前 Epoch ID
    EpochStartVLC map[string]int64 // Epoch 开始时的 VLC 基准
    CurrentVLC    map[string]int64 // 当前 VLC 值
    RelativeVLC   map[string]int64 // 相对增量 (CurrentVLC - EpochStartVLC)
}

// Epoch 配置
const (
    EpochDuration = 24 * time.Hour  // 每 24 小时一个 Epoch
    MaxVLCValue   = 1_000_000       // VLC 单个 Epoch 最大值
)
```

### 3.3 VLC 重置流程

```
Epoch N                    Epoch N+1
├─────────────────────────┼─────────────────────────►
VLC: 0 → 50000            Reset to 0 → 30000
                          ↑
                    保存 Epoch N 的最终状态
                    新 Epoch 从 0 开始
```

**实现逻辑**：

```go
func (e *EpochVLC) StartNewEpoch(newEpochID int64) {
    // 1. 保存当前 Epoch 的最终状态到主网
    e.submitToMainNet(e.EpochID, e.RelativeVLC)
    
    // 2. 重置 VLC
    e.EpochID = newEpochID
    e.EpochStartVLC = make(map[string]int64)
    e.CurrentVLC = make(map[string]int64)
    e.RelativeVLC = make(map[string]int64)
}

func (e *EpochVLC) Increment(participantID string, delta int64) {
    e.CurrentVLC[participantID] += delta
    e.RelativeVLC[participantID] = e.CurrentVLC[participantID] - e.EpochStartVLC[participantID]
}
```

### 3.4 跨 Epoch 验证

```go
// 主网存储历史 VLC
type MainNetVLCHistory struct {
    AppID   string
    EpochID int64
    VLCData map[string]int64  // 该 Epoch 的 VLC 增量
}

// 验证时只需比较当前 Epoch 内的相对值
func ValidateVLC(current, expected map[string]int64) bool {
    for id, expectedVal := range expected {
        if current[id] != expectedVal {
            return false
        }
    }
    return true
}
```

---

## 4. 主网 PoCW 共识设计

### 4.1 主网职责

1. **应用注册管理**
2. **Epoch 周期管理**
3. **子网 VLC 验证**
4. **Flux 分配计算**
5. **Credit 信用评分**

### 4.2 主网 PoCW Round 流程

```
每个 Epoch (24小时) 执行一次主网共识：

1. 收集阶段 (Collection Phase)
   ├─ 收集所有应用提交的 VLC 状态
   ├─ 收集应用的工作证明数据
   └─ 验证提交的完整性

2. 验证阶段 (Validation Phase)
   ├─ 主网验证者检查 VLC 增量合理性
   ├─ 验证应用的工作量证明
   └─ 投票：approve / reject / abstain

3. 共识阶段 (Consensus Phase)
   ├─ BFT 投票聚合 (>50% 权重)
   ├─ 计算每个应用的信用分
   └─ 决定 Flux 分配比例

4. 分配阶段 (Distribution Phase)
   ├─ 根据工作量分配 Flux
   ├─ 更新应用 Credit 分数
   └─ 触发新 Epoch 开始
```

### 4.3 主网数据结构

```go
// 应用注册信息
type Application struct {
    AppID          string
    Name           string
    Owner          string
    RegisteredAt   time.Time
    Status         string  // active, suspended, inactive
    
    // 协议配置
    FluxProtocol   []string  // ["flux-compute", "flux-storage"]
    CreditScore    float64   // 0-100 信用分
    
    // 当前状态
    CurrentEpoch   int64
    TotalFlux      int64
    LastSubmitTime time.Time
}

// Epoch 提交数据
type EpochSubmission struct {
    AppID       string
    EpochID     int64
    SubmitTime  time.Time
    
    // VLC 状态
    VLCIncrement map[string]int64  // 本 Epoch 的 VLC 增量
    
    // 工作证明
    TaskCount    int
    UserCount    int
    WorkProof    WorkProofData
    
    // 验证结果
    Validated    bool
    ValidatorVotes []Vote
}
```

---

## 5. 子网 (应用) PoCW 设计

### 5.1 应用层架构

每个应用独立运行，包含：

```
Application Instance
├── Local PoCW Engine (每 5 分钟一个 Round)
│   ├── Task Collection
│   ├── VLC Verification
│   ├── Quality Voting
│   └── Consensus
│
├── VLC Manager (Epoch-based)
│   ├── Current Epoch VLC
│   ├── VLC Increment Tracking
│   └── Epoch Submission Preparation
│
├── Flux Accounting
│   ├── Local Flux Distribution
│   └── Flux Balance Management
│
└── MainNet Connector
    ├── Epoch Submission
    ├── Flux Claim
    └── Status Sync
```

### 5.2 应用 PoCW Round (每 5 分钟)

```go
type AppPoCWRound struct {
    AppID       string
    RoundID     string
    EpochID     int64
    StartTime   time.Time
    
    // 本地任务
    Tasks       []Task
    
    // 本地 VLC (Epoch 相对值)
    VLCBefore   map[string]int64
    VLCAfter    map[string]int64
    VLCDelta    map[string]int64
    
    // 共识结果
    Consensus   ConsensusResult
    
    // 本地 Flux 分配
    FluxAwarded map[string]int64
}

// Round 流程
func (app *Application) RunPoCWRound() {
    // 1. 收集待处理任务
    tasks := app.collectPendingTasks()
    
    // 2. VLC 验证 (使用 Epoch 相对值)
    vlcValid := app.verifyVLC(tasks)
    
    // 3. 质量投票 (本地验证者)
    votes := app.collectQualityVotes(tasks)
    
    // 4. BFT 共识
    consensus := app.calculateConsensus(votes)
    
    // 5. 分配本地 Flux
    app.distributeLocalFlux(consensus)
    
    // 6. 更新 Epoch VLC
    app.updateEpochVLC(consensus.VLCDelta)
}
```

### 5.3 Epoch 提交流程

```go
func (app *Application) SubmitEpoch(epochID int64) error {
    // 1. 准备提交数据
    submission := &EpochSubmission{
        AppID:        app.AppID,
        EpochID:      epochID,
        VLCIncrement: app.vlcManager.GetEpochIncrement(),
        TaskCount:    app.getEpochTaskCount(),
        UserCount:    app.getEpochUserCount(),
        WorkProof:    app.generateWorkProof(),
    }
    
    // 2. 提交到主网
    err := app.mainNetConnector.Submit(submission)
    
    // 3. 重置 VLC (开始新 Epoch)
    app.vlcManager.StartNewEpoch(epochID + 1)
    
    return err
}
```

---

## 6. Flux/Credit 协议设计

### 6.1 Flux 协议 (流动积分)

类似 ERC20，但基于 PoCW 共识分配：

```go
type FluxProtocol struct {
    Name        string  // "flux-compute", "flux-storage", "flux-social"
    TotalSupply int64   // 总供应量 (每 Epoch 增发)
    
    // 每 Epoch 分配规则
    EpochMint   int64   // 每 Epoch 新增 Flux
    
    // 分配权重
    Weights     map[string]float64  // AppID -> 权重
}

// Flux 账户
type FluxAccount struct {
    AppID       string
    Protocol    string
    Balance     int64
    Earned      int64  // 累计获得
    Spent       int64  // 累计消耗
}

// Flux 分配计算
func CalculateFluxDistribution(epoch int64, submissions []EpochSubmission) map[string]int64 {
    totalWork := 0
    for _, sub := range submissions {
        totalWork += sub.TaskCount
    }
    
    distribution := make(map[string]int64)
    for _, sub := range submissions {
        // 基于工作量比例分配
        share := float64(sub.TaskCount) / float64(totalWork)
        distribution[sub.AppID] = int64(share * float64(EpochMintAmount))
    }
    
    return distribution
}
```

### 6.2 Credit 协议 (信用积分)

不可转移，用于评估应用质量：

```go
type CreditScore struct {
    AppID           string
    Score           float64  // 0-100
    
    // 评分因子
    UptimeRatio     float64  // 在线率
    SubmitOnTime    float64  // 按时提交率
    ConsensusRate   float64  // 共识通过率
    UserGrowth      float64  // 用户增长率
    
    // 历史记录
    History         []CreditHistory
}

func CalculateCreditScore(app *Application, epoch int64) float64 {
    score := 0.0
    
    // 1. 在线率 (30%)
    score += app.getUptimeRatio() * 30
    
    // 2. 按时提交 (20%)
    score += app.getSubmitOnTimeRatio() * 20
    
    // 3. 共识通过率 (30%)
    score += app.getConsensusPassRate() * 30
    
    // 4. 用户增长 (20%)
    score += app.getUserGrowthRate() * 20
    
    return score
}
```

---

## 7. 系统运行流程

### 7.1 时间线

```
时间轴：
├─ 0:00 ────────────────────────────────────────── 24:00 ─────►
│  Epoch N 开始                                    Epoch N 结束
│                                                  Epoch N+1 开始
│
├─ 应用层：每 5 分钟一个 PoCW Round
│  Round 1 → Round 2 → ... → Round 288 (24h / 5min = 288)
│
└─ 主网层：每 24 小时一个 PoCW Epoch Round
   收集 → 验证 → 共识 → 分配
```

### 7.2 完整流程示例

```
Day 1, 00:00 - Epoch 1 开始
├─ App-Social: VLC 重置为 0
├─ App-AI: VLC 重置为 0
└─ App-Storage: VLC 重置为 0

Day 1, 00:05 - 各应用 Round 1
├─ App-Social: 处理 100 个任务，VLC +100
├─ App-AI: 处理 50 个任务，VLC +50
└─ App-Storage: 处理 200 个任务，VLC +200

... (每 5 分钟重复)

Day 1, 23:55 - 各应用 Round 288
├─ App-Social: 累计 VLC = 28,800
├─ App-AI: 累计 VLC = 14,400
└─ App-Storage: 累计 VLC = 57,600

Day 2, 00:00 - Epoch 1 结束，主网共识开始
├─ 1. 收集提交
│   ├─ App-Social 提交: VLC=28,800, Tasks=28,800
│   ├─ App-AI 提交: VLC=14,400, Tasks=14,400
│   └─ App-Storage 提交: VLC=57,600, Tasks=57,600
│
├─ 2. 主网验证者投票
│   ├─ Validator-1: 全部 approve
│   ├─ Validator-2: 全部 approve
│   └─ Validator-3: 全部 approve
│
├─ 3. BFT 共识通过
│
├─ 4. Flux 分配 (假设本 Epoch 总量 100,000 Flux)
│   ├─ App-Social: 28.8% = 28,800 Flux
│   ├─ App-AI: 14.4% = 14,400 Flux
│   └─ App-Storage: 57.6% = 57,600 Flux
│
└─ 5. Credit 更新
    ├─ App-Social: 95.2 (高在线率)
    ├─ App-AI: 88.5 (偶尔延迟)
    └─ App-Storage: 98.7 (完美表现)

Day 2, 00:00 - Epoch 2 开始
├─ 所有应用 VLC 重置为 0
└─ 开始新的 Round 循环...
```

---

## 8. 核心代码结构

### 8.1 主网服务

```
mainnet/
├── services/
│   ├── epoch_manager.go       # Epoch 周期管理
│   ├── vlc_aggregator.go      # VLC 聚合验证
│   ├── flux_registry.go       # Flux 协议注册
│   ├── credit_calculator.go   # Credit 计算
│   └── mainnet_consensus.go   # 主网 PoCW 共识
│
├── models/
│   ├── application.go         # 应用模型
│   ├── epoch_submission.go    # Epoch 提交
│   ├── flux_protocol.go       # Flux 协议
│   └── credit_score.go        # Credit 评分
│
└── handlers/
    ├── app_registry.go        # 应用注册 API
    ├── epoch_api.go           # Epoch 查询 API
    └── flux_api.go            # Flux 查询 API
```

### 8.2 应用层服务

```
application/
├── services/
│   ├── app_pocw_engine.go     # 应用 PoCW 引擎
│   ├── epoch_vlc_manager.go   # Epoch VLC 管理
│   ├── local_flux_manager.go  # 本地 Flux 管理
│   └── mainnet_connector.go   # 主网连接器
│
├── models/
│   ├── app_round.go           # 应用 Round
│   ├── epoch_vlc.go           # Epoch VLC
│   └── flux_account.go        # Flux 账户
│
└── handlers/
    ├── task_handler.go        # 任务处理
    └── stats_handler.go       # 统计查询
```

继续下一部分...
