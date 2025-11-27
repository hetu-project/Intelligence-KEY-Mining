# 主网-子网实现指南

## 9. VLC 防溢出详细实现

### 9.1 VLC Manager 实现

```go
package vlc

import (
    "sync"
    "time"
)

// EpochVLCManager 管理 Epoch 级别的 VLC
type EpochVLCManager struct {
    mu            sync.RWMutex
    currentEpoch  int64
    epochStart    time.Time
    epochDuration time.Duration
    
    // VLC 数据
    baseVLC       map[string]int64  // Epoch 开始时的基准
    currentVLC    map[string]int64  // 当前值
    
    // 历史记录 (用于查询)
    history       []EpochVLCSnapshot
    maxHistory    int
}

type EpochVLCSnapshot struct {
    EpochID   int64
    StartTime time.Time
    EndTime   time.Time
    FinalVLC  map[string]int64
}

func NewEpochVLCManager(epochDuration time.Duration) *EpochVLCManager {
    return &EpochVLCManager{
        currentEpoch:  1,
        epochStart:    time.Now(),
        epochDuration: epochDuration,
        baseVLC:       make(map[string]int64),
        currentVLC:    make(map[string]int64),
        history:       make([]EpochVLCSnapshot, 0),
        maxHistory:    100, // 保留最近 100 个 Epoch
    }
}

// Increment VLC (自动检查是否需要切换 Epoch)
func (m *EpochVLCManager) Increment(participantID string, delta int64) error {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // 检查是否需要切换 Epoch
    if time.Since(m.epochStart) >= m.epochDuration {
        m.rotateEpoch()
    }
    
    // 增加 VLC
    m.currentVLC[participantID] += delta
    
    // 防止单个 Epoch 内溢出
    if m.currentVLC[participantID] > MaxVLCPerEpoch {
        return fmt.Errorf("VLC overflow in epoch %d for %s", m.currentEpoch, participantID)
    }
    
    return nil
}

// GetRelativeVLC 获取相对于 Epoch 开始的增量
func (m *EpochVLCManager) GetRelativeVLC() map[string]int64 {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    relative := make(map[string]int64)
    for id, current := range m.currentVLC {
        base := m.baseVLC[id]
        relative[id] = current - base
    }
    return relative
}

// RotateEpoch 切换到新 Epoch
func (m *EpochVLCManager) rotateEpoch() {
    // 1. 保存当前 Epoch 快照
    snapshot := EpochVLCSnapshot{
        EpochID:   m.currentEpoch,
        StartTime: m.epochStart,
        EndTime:   time.Now(),
        FinalVLC:  m.GetRelativeVLC(),
    }
    
    // 2. 添加到历史 (限制大小)
    m.history = append(m.history, snapshot)
    if len(m.history) > m.maxHistory {
        m.history = m.history[1:] // 删除最老的
    }
    
    // 3. 重置 VLC
    m.currentEpoch++
    m.epochStart = time.Now()
    m.baseVLC = make(map[string]int64)
    m.currentVLC = make(map[string]int64)
    
    log.Printf("Rotated to Epoch %d, saved snapshot for Epoch %d", 
        m.currentEpoch, snapshot.EpochID)
}

// GetEpochHistory 获取历史 Epoch 数据
func (m *EpochVLCManager) GetEpochHistory(epochID int64) (*EpochVLCSnapshot, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    
    for _, snap := range m.history {
        if snap.EpochID == epochID {
            return &snap, nil
        }
    }
    return nil, fmt.Errorf("epoch %d not found in history", epochID)
}
```

### 9.2 VLC 验证逻辑

```go
// ValidateVLCIncrement 验证 VLC 增量的合理性
func ValidateVLCIncrement(
    participantID string,
    increment int64,
    taskCount int,
    expectedRange VLCRange,
) error {
    // 1. 检查增量是否为正
    if increment < 0 {
        return fmt.Errorf("negative VLC increment")
    }
    
    // 2. 检查是否在合理范围内
    if increment < expectedRange.Min || increment > expectedRange.Max {
        return fmt.Errorf("VLC increment %d out of range [%d, %d]",
            increment, expectedRange.Min, expectedRange.Max)
    }
    
    // 3. 检查与任务数量的比例
    avgPerTask := float64(increment) / float64(taskCount)
    if avgPerTask < 0.5 || avgPerTask > 2.0 {
        return fmt.Errorf("abnormal VLC/task ratio: %.2f", avgPerTask)
    }
    
    return nil
}

type VLCRange struct {
    Min int64
    Max int64
}

// CalculateExpectedVLCRange 根据任务数量计算预期 VLC 范围
func CalculateExpectedVLCRange(taskCount int) VLCRange {
    // 假设每个任务 VLC +1，允许 ±20% 误差
    base := int64(taskCount)
    return VLCRange{
        Min: int64(float64(base) * 0.8),
        Max: int64(float64(base) * 1.2),
    }
}
```

---

## 10. 主网 Epoch 共识实现

### 10.1 Epoch Manager

```go
package mainnet

import (
    "context"
    "time"
)

type EpochManager struct {
    currentEpoch  int64
    epochDuration time.Duration
    
    // 服务依赖
    vlcAggregator  *VLCAggregator
    fluxRegistry   *FluxRegistry
    creditCalc     *CreditCalculator
    consensusEngine *MainNetConsensus
    
    // 应用管理
    applications   map[string]*Application
    submissions    map[int64][]EpochSubmission
}

func NewEpochManager(duration time.Duration) *EpochManager {
    return &EpochManager{
        currentEpoch:  1,
        epochDuration: duration,
        applications:  make(map[string]*Application),
        submissions:   make(map[int64][]EpochSubmission),
    }
}

// Start 启动 Epoch 管理器
func (em *EpochManager) Start(ctx context.Context) error {
    ticker := time.NewTicker(em.epochDuration)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        case <-ticker.C:
            // 执行 Epoch 共识
            if err := em.runEpochConsensus(ctx); err != nil {
                log.Errorf("Epoch consensus failed: %v", err)
            }
        }
    }
}

// RunEpochConsensus 执行一次 Epoch 共识
func (em *EpochManager) runEpochConsensus(ctx context.Context) error {
    epochID := em.currentEpoch
    log.Printf("Starting Epoch %d consensus", epochID)
    
    // 1. 收集阶段
    submissions := em.collectSubmissions(epochID)
    if len(submissions) == 0 {
        log.Printf("No submissions for Epoch %d", epochID)
        em.currentEpoch++
        return nil
    }
    
    // 2. 验证阶段
    validatedSubs, err := em.validateSubmissions(ctx, submissions)
    if err != nil {
        return fmt.Errorf("validation failed: %w", err)
    }
    
    // 3. 共识阶段
    consensusResult, err := em.consensusEngine.RunConsensus(ctx, validatedSubs)
    if err != nil {
        return fmt.Errorf("consensus failed: %w", err)
    }
    
    // 4. 分配阶段
    if err := em.distributeRewards(ctx, consensusResult); err != nil {
        return fmt.Errorf("distribution failed: %w", err)
    }
    
    // 5. 更新 Credit
    if err := em.updateCreditScores(ctx, consensusResult); err != nil {
        return fmt.Errorf("credit update failed: %w", err)
    }
    
    // 6. 切换到下一个 Epoch
    em.currentEpoch++
    log.Printf("Epoch %d consensus completed", epochID)
    
    return nil
}

// CollectSubmissions 收集所有应用的提交
func (em *EpochManager) collectSubmissions(epochID int64) []EpochSubmission {
    submissions := em.submissions[epochID]
    
    // 检查是否所有活跃应用都提交了
    for appID, app := range em.applications {
        if app.Status != "active" {
            continue
        }
        
        found := false
        for _, sub := range submissions {
            if sub.AppID == appID {
                found = true
                break
            }
        }
        
        if !found {
            log.Warnf("Application %s did not submit for Epoch %d", appID, epochID)
            // 降低 Credit 分数
            app.CreditScore *= 0.95
        }
    }
    
    return submissions
}

// ValidateSubmissions 验证提交的数据
func (em *EpochManager) validateSubmissions(
    ctx context.Context,
    submissions []EpochSubmission,
) ([]EpochSubmission, error) {
    validated := make([]EpochSubmission, 0)
    
    for _, sub := range submissions {
        // 1. 验证 VLC 增量
        if err := em.vlcAggregator.ValidateVLC(sub); err != nil {
            log.Errorf("VLC validation failed for %s: %v", sub.AppID, err)
            continue
        }
        
        // 2. 验证工作证明
        if err := em.validateWorkProof(sub); err != nil {
            log.Errorf("Work proof validation failed for %s: %v", sub.AppID, err)
            continue
        }
        
        // 3. 验证时间戳
        if err := em.validateTimestamp(sub); err != nil {
            log.Errorf("Timestamp validation failed for %s: %v", sub.AppID, err)
            continue
        }
        
        validated = append(validated, sub)
    }
    
    return validated, nil
}

// DistributeRewards 分配 Flux 奖励
func (em *EpochManager) distributeRewards(
    ctx context.Context,
    result ConsensusResult,
) error {
    // 计算总工作量
    totalWork := 0
    for _, sub := range result.ApprovedSubmissions {
        totalWork += sub.TaskCount
    }
    
    if totalWork == 0 {
        return nil
    }
    
    // 本 Epoch 的 Flux 总量
    epochFlux := em.fluxRegistry.GetEpochMintAmount()
    
    // 按工作量比例分配
    for _, sub := range result.ApprovedSubmissions {
        share := float64(sub.TaskCount) / float64(totalWork)
        fluxAmount := int64(float64(epochFlux) * share)
        
        // 分配 Flux
        if err := em.fluxRegistry.Mint(sub.AppID, fluxAmount); err != nil {
            log.Errorf("Failed to mint flux for %s: %v", sub.AppID, err)
            continue
        }
        
        log.Printf("Distributed %d Flux to %s (%.2f%% of total)",
            fluxAmount, sub.AppID, share*100)
    }
    
    return nil
}
```

### 10.2 主网共识引擎

```go
type MainNetConsensus struct {
    validators []*MainNetValidator
    threshold  float64  // BFT 阈值 (0.5 = 50%)
}

type MainNetValidator struct {
    ID     string
    Weight float64
}

type ConsensusResult struct {
    EpochID             int64
    ApprovedSubmissions []EpochSubmission
    RejectedSubmissions []EpochSubmission
    ValidatorVotes      map[string][]Vote
}

func (c *MainNetConsensus) RunConsensus(
    ctx context.Context,
    submissions []EpochSubmission,
) (ConsensusResult, error) {
    result := ConsensusResult{
        ValidatorVotes: make(map[string][]Vote),
    }
    
    // 1. 每个验证者对每个提交投票
    for _, validator := range c.validators {
        for _, sub := range submissions {
            vote := validator.VoteOnSubmission(sub)
            result.ValidatorVotes[sub.AppID] = append(
                result.ValidatorVotes[sub.AppID],
                vote,
            )
        }
    }
    
    // 2. 计算每个提交的共识结果
    for _, sub := range submissions {
        votes := result.ValidatorVotes[sub.AppID]
        
        // 计算加权投票
        approveWeight := 0.0
        totalWeight := 0.0
        
        for _, vote := range votes {
            totalWeight += vote.Weight
            if vote.Decision == "approve" {
                approveWeight += vote.Weight
            }
        }
        
        // BFT 共识：>50% 权重同意
        if approveWeight/totalWeight > c.threshold {
            result.ApprovedSubmissions = append(result.ApprovedSubmissions, sub)
        } else {
            result.RejectedSubmissions = append(result.RejectedSubmissions, sub)
        }
    }
    
    return result, nil
}

// VoteOnSubmission 验证者对提交投票
func (v *MainNetValidator) VoteOnSubmission(sub EpochSubmission) Vote {
    // 验证逻辑
    score := 0.0
    
    // 1. VLC 合理性 (30%)
    if v.checkVLCReasonable(sub) {
        score += 0.3
    }
    
    // 2. 任务数量合理性 (30%)
    if v.checkTaskCountReasonable(sub) {
        score += 0.3
    }
    
    // 3. 工作证明有效性 (40%)
    if v.checkWorkProofValid(sub) {
        score += 0.4
    }
    
    decision := "reject"
    if score >= 0.7 {
        decision = "approve"
    }
    
    return Vote{
        ValidatorID: v.ID,
        Decision:    decision,
        Score:       score,
        Weight:      v.Weight,
        Timestamp:   time.Now(),
    }
}
```

---

## 11. 应用层实现

### 11.1 应用 PoCW 引擎

```go
package application

type AppPoCWEngine struct {
    appID         string
    vlcManager    *EpochVLCManager
    fluxManager   *LocalFluxManager
    mainNetConn   *MainNetConnector
    
    // Round 配置
    roundInterval time.Duration
    
    // 本地验证者
    validators    []*LocalValidator
}

func NewAppPoCWEngine(appID string) *AppPoCWEngine {
    return &AppPoCWEngine{
        appID:         appID,
        vlcManager:    NewEpochVLCManager(24 * time.Hour),
        fluxManager:   NewLocalFluxManager(),
        roundInterval: 5 * time.Minute,
        validators:    make([]*LocalValidator, 0),
    }
}

// Start 启动应用 PoCW 引擎
func (e *AppPoCWEngine) Start(ctx context.Context) error {
    // 1. 启动 Round 循环
    go e.runRoundLoop(ctx)
    
    // 2. 启动 Epoch 提交循环
    go e.runEpochSubmitLoop(ctx)
    
    return nil
}

// RunRoundLoop Round 循环 (每 5 分钟)
func (e *AppPoCWEngine) runRoundLoop(ctx context.Context) {
    ticker := time.NewTicker(e.roundInterval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := e.runRound(ctx); err != nil {
                log.Errorf("Round failed: %v", err)
            }
        }
    }
}

// RunRound 执行一个 Round
func (e *AppPoCWEngine) runRound(ctx context.Context) error {
    roundID := fmt.Sprintf("round_%d", time.Now().Unix())
    
    // 1. 收集待处理任务
    tasks, err := e.collectPendingTasks()
    if err != nil {
        return err
    }
    
    if len(tasks) == 0 {
        log.Debug("No tasks to process in this round")
        return nil
    }
    
    // 2. VLC 验证
    vlcBefore := e.vlcManager.GetRelativeVLC()
    validTasks, err := e.verifyTasksVLC(tasks)
    if err != nil {
        return err
    }
    
    // 3. 质量投票
    votes, err := e.collectQualityVotes(validTasks)
    if err != nil {
        return err
    }
    
    // 4. BFT 共识
    consensus := e.calculateConsensus(votes)
    
    // 5. 更新 VLC
    for _, task := range consensus.ApprovedTasks {
        e.vlcManager.Increment(task.UserWallet, 1)
    }
    
    vlcAfter := e.vlcManager.GetRelativeVLC()
    
    // 6. 分配本地 Flux
    e.distributeLocalFlux(consensus.ApprovedTasks)
    
    // 7. 保存 Round 数据
    e.saveRound(roundID, tasks, vlcBefore, vlcAfter, consensus)
    
    log.Printf("Round %s completed: %d tasks approved",
        roundID, len(consensus.ApprovedTasks))
    
    return nil
}

// RunEpochSubmitLoop Epoch 提交循环 (每 24 小时)
func (e *AppPoCWEngine) runEpochSubmitLoop(ctx context.Context) {
    ticker := time.NewTicker(24 * time.Hour)
    defer ticker.Stop()
    
    for {
        select {
        case <-ctx.Done():
            return
        case <-ticker.C:
            if err := e.submitEpoch(ctx); err != nil {
                log.Errorf("Epoch submission failed: %v", err)
            }
        }
    }
}

// SubmitEpoch 提交 Epoch 数据到主网
func (e *AppPoCWEngine) submitEpoch(ctx context.Context) error {
    // 1. 获取当前 Epoch 数据
    epochID := e.vlcManager.GetCurrentEpoch()
    vlcIncrement := e.vlcManager.GetRelativeVLC()
    
    // 2. 生成工作证明
    workProof := e.generateWorkProof()
    
    // 3. 创建提交
    submission := &EpochSubmission{
        AppID:        e.appID,
        EpochID:      epochID,
        SubmitTime:   time.Now(),
        VLCIncrement: vlcIncrement,
        TaskCount:    workProof.TaskCount,
        UserCount:    workProof.UserCount,
        WorkProof:    workProof,
    }
    
    // 4. 提交到主网
    if err := e.mainNetConn.SubmitEpoch(ctx, submission); err != nil {
        return fmt.Errorf("failed to submit epoch: %w", err)
    }
    
    log.Printf("Submitted Epoch %d to MainNet", epochID)
    
    // 5. VLC Manager 会自动切换到新 Epoch
    
    return nil
}
```

### 11.2 主网连接器

```go
type MainNetConnector struct {
    mainNetURL string
    appID      string
    apiKey     string
    client     *http.Client
}

// SubmitEpoch 提交 Epoch 到主网
func (c *MainNetConnector) SubmitEpoch(
    ctx context.Context,
    submission *EpochSubmission,
) error {
    url := fmt.Sprintf("%s/api/v1/mainnet/epochs/submit", c.mainNetURL)
    
    body, err := json.Marshal(submission)
    if err != nil {
        return err
    }
    
    req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
    if err != nil {
        return err
    }
    
    req.Header.Set("Content-Type", "application/json")
    req.Header.Set("X-App-ID", c.appID)
    req.Header.Set("X-API-Key", c.apiKey)
    
    resp, err := c.client.Do(req)
    if err != nil {
        return err
    }
    defer resp.Body.Close()
    
    if resp.StatusCode != http.StatusOK {
        return fmt.Errorf("submission failed: %s", resp.Status)
    }
    
    return nil
}

// ClaimFlux 从主网领取 Flux
func (c *MainNetConnector) ClaimFlux(ctx context.Context, epochID int64) (int64, error) {
    url := fmt.Sprintf("%s/api/v1/mainnet/flux/claim?epoch=%d", c.mainNetURL, epochID)
    
    req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
    if err != nil {
        return 0, err
    }
    
    req.Header.Set("X-App-ID", c.appID)
    req.Header.Set("X-API-Key", c.apiKey)
    
    resp, err := c.client.Do(req)
    if err != nil {
        return 0, err
    }
    defer resp.Body.Close()
    
    var result struct {
        FluxAmount int64 `json:"flux_amount"`
    }
    
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return 0, err
    }
    
    return result.FluxAmount, nil
}
```

继续下一部分...
