# 主网-子网最佳实践与部署指南

## 12. 数据库设计

### 12.1 主网数据库表

```sql
-- 应用注册表
CREATE TABLE applications (
    app_id VARCHAR(100) PRIMARY KEY,
    name VARCHAR(200) NOT NULL,
    owner VARCHAR(42) NOT NULL,
    description TEXT,
    
    -- 状态
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    registered_at TIMESTAMP NOT NULL,
    last_submit_time TIMESTAMP,
    
    -- 协议
    flux_protocols JSON,
    credit_score DECIMAL(5,2) DEFAULT 0,
    
    -- 统计
    total_flux BIGINT DEFAULT 0,
    total_epochs INT DEFAULT 0,
    
    INDEX idx_status (status),
    INDEX idx_owner (owner),
    INDEX idx_credit_score (credit_score)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Epoch 提交表
CREATE TABLE epoch_submissions (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    app_id VARCHAR(100) NOT NULL,
    epoch_id BIGINT NOT NULL,
    submit_time TIMESTAMP NOT NULL,
    
    -- VLC 数据
    vlc_increment JSON NOT NULL,
    
    -- 工作证明
    task_count INT NOT NULL,
    user_count INT NOT NULL,
    work_proof JSON,
    
    -- 验证结果
    validated BOOLEAN DEFAULT FALSE,
    consensus_result VARCHAR(20),
    validator_votes JSON,
    
    -- 奖励
    flux_awarded BIGINT DEFAULT 0,
    
    UNIQUE KEY uk_app_epoch (app_id, epoch_id),
    INDEX idx_epoch_id (epoch_id),
    INDEX idx_submit_time (submit_time),
    FOREIGN KEY (app_id) REFERENCES applications(app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Flux 账户表
CREATE TABLE flux_accounts (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    app_id VARCHAR(100) NOT NULL,
    protocol VARCHAR(50) NOT NULL,
    
    balance BIGINT DEFAULT 0,
    earned BIGINT DEFAULT 0,
    spent BIGINT DEFAULT 0,
    
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    UNIQUE KEY uk_app_protocol (app_id, protocol),
    INDEX idx_balance (balance),
    FOREIGN KEY (app_id) REFERENCES applications(app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Credit 历史表
CREATE TABLE credit_history (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    app_id VARCHAR(100) NOT NULL,
    epoch_id BIGINT NOT NULL,
    
    score DECIMAL(5,2) NOT NULL,
    uptime_ratio DECIMAL(5,2),
    submit_on_time_ratio DECIMAL(5,2),
    consensus_rate DECIMAL(5,2),
    user_growth_rate DECIMAL(5,2),
    
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    INDEX idx_app_epoch (app_id, epoch_id),
    FOREIGN KEY (app_id) REFERENCES applications(app_id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Epoch 元数据表
CREATE TABLE epochs (
    epoch_id BIGINT PRIMARY KEY,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    
    total_submissions INT DEFAULT 0,
    approved_submissions INT DEFAULT 0,
    rejected_submissions INT DEFAULT 0,
    
    total_flux_minted BIGINT DEFAULT 0,
    
    status VARCHAR(20) DEFAULT 'active',
    
    INDEX idx_start_time (start_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

### 12.2 应用层数据库表

```sql
-- 应用 Round 表
CREATE TABLE app_rounds (
    round_id VARCHAR(100) PRIMARY KEY,
    epoch_id BIGINT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP,
    
    task_count INT DEFAULT 0,
    approved_count INT DEFAULT 0,
    rejected_count INT DEFAULT 0,
    
    vlc_before JSON,
    vlc_after JSON,
    vlc_delta JSON,
    
    flux_distributed BIGINT DEFAULT 0,
    
    INDEX idx_epoch_id (epoch_id),
    INDEX idx_start_time (start_time)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- 本地 Flux 账户
CREATE TABLE local_flux_accounts (
    user_wallet VARCHAR(42) PRIMARY KEY,
    
    balance BIGINT DEFAULT 0,
    earned BIGINT DEFAULT 0,
    spent BIGINT DEFAULT 0,
    
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    
    INDEX idx_balance (balance)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;

-- Epoch VLC 快照
CREATE TABLE epoch_vlc_snapshots (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    epoch_id BIGINT NOT NULL,
    start_time TIMESTAMP NOT NULL,
    end_time TIMESTAMP NOT NULL,
    
    final_vlc JSON NOT NULL,
    
    submitted_to_mainnet BOOLEAN DEFAULT FALSE,
    submit_time TIMESTAMP,
    
    INDEX idx_epoch_id (epoch_id),
    INDEX idx_submitted (submitted_to_mainnet)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
```

---

## 13. 安全性考虑

### 13.1 VLC 作弊防护

```go
// VLC 异常检测
type VLCAnomalyDetector struct {
    historicalData map[string][]int64  // participantID -> VLC history
    threshold      float64              // 异常阈值
}

func (d *VLCAnomalyDetector) DetectAnomaly(
    participantID string,
    currentIncrement int64,
) (bool, string) {
    history := d.historicalData[participantID]
    if len(history) < 10 {
        // 数据不足，暂不判断
        return false, ""
    }
    
    // 计算历史平均值和标准差
    avg, stdDev := calculateStats(history)
    
    // Z-score 检测
    zScore := math.Abs(float64(currentIncrement)-avg) / stdDev
    
    if zScore > d.threshold {
        return true, fmt.Sprintf(
            "VLC increment %d is %.2f standard deviations from mean %.2f",
            currentIncrement, zScore, avg,
        )
    }
    
    return false, ""
}

// 防止 Sybil 攻击
func DetectSybilAttack(submissions []EpochSubmission) []string {
    suspicious := make([]string, 0)
    
    // 检测相似的 VLC 模式
    for i := 0; i < len(submissions); i++ {
        for j := i + 1; j < len(submissions); j++ {
            similarity := calculateVLCSimilarity(
                submissions[i].VLCIncrement,
                submissions[j].VLCIncrement,
            )
            
            if similarity > 0.95 {
                suspicious = append(suspicious, 
                    submissions[i].AppID,
                    submissions[j].AppID,
                )
            }
        }
    }
    
    return suspicious
}
```

### 13.2 提交验证

```go
// 验证提交的完整性
func ValidateSubmissionIntegrity(sub *EpochSubmission) error {
    // 1. 签名验证
    if !verifySignature(sub) {
        return fmt.Errorf("invalid signature")
    }
    
    // 2. 时间戳验证
    now := time.Now()
    if sub.SubmitTime.After(now) {
        return fmt.Errorf("future timestamp")
    }
    
    if now.Sub(sub.SubmitTime) > 1*time.Hour {
        return fmt.Errorf("submission too old")
    }
    
    // 3. 数据完整性
    if len(sub.VLCIncrement) == 0 {
        return fmt.Errorf("empty VLC data")
    }
    
    if sub.TaskCount < 0 || sub.UserCount < 0 {
        return fmt.Errorf("invalid counts")
    }
    
    // 4. VLC 合理性
    totalVLC := int64(0)
    for _, v := range sub.VLCIncrement {
        totalVLC += v
    }
    
    if totalVLC > MaxVLCPerEpoch {
        return fmt.Errorf("VLC exceeds maximum")
    }
    
    return nil
}
```

---

## 14. 性能优化

### 14.1 VLC 存储优化

```go
// 使用压缩存储 VLC
type CompressedVLC struct {
    EpochID   int64
    Timestamp time.Time
    Data      []byte  // gzip 压缩的 JSON
}

func CompressVLC(vlc map[string]int64) ([]byte, error) {
    // 1. 序列化为 JSON
    jsonData, err := json.Marshal(vlc)
    if err != nil {
        return nil, err
    }
    
    // 2. gzip 压缩
    var buf bytes.Buffer
    gzipWriter := gzip.NewWriter(&buf)
    
    if _, err := gzipWriter.Write(jsonData); err != nil {
        return nil, err
    }
    
    if err := gzipWriter.Close(); err != nil {
        return nil, err
    }
    
    return buf.Bytes(), nil
}

func DecompressVLC(data []byte) (map[string]int64, error) {
    // 1. gzip 解压
    reader, err := gzip.NewReader(bytes.NewReader(data))
    if err != nil {
        return nil, err
    }
    defer reader.Close()
    
    jsonData, err := io.ReadAll(reader)
    if err != nil {
        return nil, err
    }
    
    // 2. 反序列化
    var vlc map[string]int64
    if err := json.Unmarshal(jsonData, &vlc); err != nil {
        return nil, err
    }
    
    return vlc, nil
}
```

### 14.2 批量处理优化

```go
// 批量验证 VLC
func BatchValidateVLC(submissions []EpochSubmission) []ValidationResult {
    results := make([]ValidationResult, len(submissions))
    
    // 使用 worker pool
    numWorkers := runtime.NumCPU()
    jobs := make(chan int, len(submissions))
    
    var wg sync.WaitGroup
    for w := 0; w < numWorkers; w++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for idx := range jobs {
                results[idx] = validateSingleVLC(submissions[idx])
            }
        }()
    }
    
    // 分发任务
    for i := range submissions {
        jobs <- i
    }
    close(jobs)
    
    wg.Wait()
    return results
}
```

---

## 15. 监控和告警

### 15.1 关键指标

```go
// Prometheus 指标
var (
    // Epoch 指标
    currentEpochGauge = promauto.NewGauge(prometheus.GaugeOpts{
        Name: "mainnet_current_epoch",
        Help: "Current epoch number",
    })
    
    epochSubmissionsCounter = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "mainnet_epoch_submissions_total",
            Help: "Total epoch submissions",
        },
        []string{"app_id", "status"},
    )
    
    // VLC 指标
    vlcIncrementHistogram = promauto.NewHistogramVec(
        prometheus.HistogramOpts{
            Name:    "mainnet_vlc_increment",
            Help:    "VLC increment distribution",
            Buckets: prometheus.ExponentialBuckets(1, 2, 20),
        },
        []string{"app_id"},
    )
    
    // Flux 指标
    fluxDistributedCounter = promauto.NewCounterVec(
        prometheus.CounterOpts{
            Name: "mainnet_flux_distributed_total",
            Help: "Total flux distributed",
        },
        []string{"app_id", "protocol"},
    )
    
    // Credit 指标
    creditScoreGauge = promauto.NewGaugeVec(
        prometheus.GaugeOpts{
            Name: "mainnet_credit_score",
            Help: "Application credit score",
        },
        []string{"app_id"},
    )
)

// 记录指标
func RecordEpochMetrics(result ConsensusResult) {
    currentEpochGauge.Set(float64(result.EpochID))
    
    for _, sub := range result.ApprovedSubmissions {
        epochSubmissionsCounter.WithLabelValues(sub.AppID, "approved").Inc()
        
        totalVLC := int64(0)
        for _, v := range sub.VLCIncrement {
            totalVLC += v
        }
        vlcIncrementHistogram.WithLabelValues(sub.AppID).Observe(float64(totalVLC))
    }
    
    for _, sub := range result.RejectedSubmissions {
        epochSubmissionsCounter.WithLabelValues(sub.AppID, "rejected").Inc()
    }
}
```

### 15.2 告警规则

```yaml
# Prometheus 告警规则
groups:
  - name: mainnet_alerts
    interval: 30s
    rules:
      # Epoch 延迟告警
      - alert: EpochDelayed
        expr: time() - mainnet_epoch_start_time > 86400 + 3600
        for: 5m
        labels:
          severity: warning
        annotations:
          summary: "Epoch processing delayed"
          description: "Epoch {{ $labels.epoch_id }} has been running for over 25 hours"
      
      # 应用未提交告警
      - alert: ApplicationNotSubmitting
        expr: |
          time() - mainnet_app_last_submit_time > 172800
        for: 10m
        labels:
          severity: critical
        annotations:
          summary: "Application {{ $labels.app_id }} not submitting"
          description: "No submission from {{ $labels.app_id }} for over 48 hours"
      
      # VLC 异常告警
      - alert: VLCAnomalyDetected
        expr: mainnet_vlc_increment > 1000000
        for: 1m
        labels:
          severity: warning
        annotations:
          summary: "Abnormal VLC increment"
          description: "{{ $labels.app_id }} reported VLC increment > 1M"
      
      # Credit 分数低告警
      - alert: LowCreditScore
        expr: mainnet_credit_score < 50
        for: 1h
        labels:
          severity: warning
        annotations:
          summary: "Low credit score for {{ $labels.app_id }}"
          description: "Credit score is {{ $value }}, below threshold of 50"
```

---

## 16. 部署架构

### 16.1 主网部署

```yaml
# docker-compose.yml for MainNet
version: '3.8'

services:
  mainnet-api:
    image: pocw-mainnet:latest
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=mainnet-db
      - DB_PORT=3306
      - DB_NAME=mainnet
      - EPOCH_DURATION=24h
      - FLUX_MINT_PER_EPOCH=100000
    depends_on:
      - mainnet-db
      - redis
    deploy:
      replicas: 3
      resources:
        limits:
          cpus: '2'
          memory: 4G
  
  mainnet-consensus:
    image: pocw-mainnet:latest
    command: ["consensus-worker"]
    environment:
      - DB_HOST=mainnet-db
      - REDIS_HOST=redis
    depends_on:
      - mainnet-db
      - redis
    deploy:
      replicas: 1
  
  mainnet-db:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=secure_password
      - MYSQL_DATABASE=mainnet
    volumes:
      - mainnet-data:/var/lib/mysql
    deploy:
      resources:
        limits:
          cpus: '4'
          memory: 8G
  
  redis:
    image: redis:7-alpine
    volumes:
      - redis-data:/data

volumes:
  mainnet-data:
  redis-data:
```

### 16.2 应用部署

```yaml
# docker-compose.yml for Application
version: '3.8'

services:
  app-pocw:
    image: pocw-app:latest
    ports:
      - "8081:8080"
    environment:
      - APP_ID=app-social-001
      - MAINNET_URL=https://mainnet.pocw.network
      - API_KEY=${APP_API_KEY}
      - DB_HOST=app-db
      - ROUND_INTERVAL=5m
      - EPOCH_DURATION=24h
    depends_on:
      - app-db
  
  app-db:
    image: mysql:8.0
    environment:
      - MYSQL_ROOT_PASSWORD=app_password
      - MYSQL_DATABASE=app_pocw
    volumes:
      - app-data:/var/lib/mysql

volumes:
  app-data:
```

---

## 17. 总结和建议

### 17.1 核心设计要点

1. **VLC 防溢出**
   - ✅ 使用 Epoch 周期性重置
   - ✅ 只存储相对增量
   - ✅ 历史数据压缩存储

2. **分层共识**
   - ✅ 应用层：每 5 分钟 PoCW Round
   - ✅ 主网层：每 24 小时 Epoch 共识
   - ✅ 独立运行，定期同步

3. **激励机制**
   - ✅ Flux：可转移的流动积分
   - ✅ Credit：不可转移的信用分
   - ✅ 基于工作量公平分配

4. **安全性**
   - ✅ VLC 异常检测
   - ✅ Sybil 攻击防护
   - ✅ 提交签名验证

### 17.2 实施路线图

**Phase 1: 核心基础设施 (2-3 个月)**
- [ ] 实现 Epoch VLC Manager
- [ ] 实现主网 Epoch Manager
- [ ] 实现应用 PoCW Engine
- [ ] 数据库设计和迁移

**Phase 2: 共识机制 (2-3 个月)**
- [ ] 主网 BFT 共识引擎
- [ ] 应用本地共识
- [ ] VLC 验证逻辑
- [ ] 工作证明生成

**Phase 3: 协议层 (1-2 个月)**
- [ ] Flux 协议实现
- [ ] Credit 评分系统
- [ ] 应用注册管理
- [ ] API 接口开发

**Phase 4: 监控和优化 (1 个月)**
- [ ] Prometheus 指标
- [ ] 告警系统
- [ ] 性能优化
- [ ] 压力测试

**Phase 5: 生产部署 (1 个月)**
- [ ] 主网部署
- [ ] 应用 SDK 发布
- [ ] 文档完善
- [ ] 社区支持

### 17.3 关键数值建议

```go
const (
    // Epoch 配置
    EpochDuration     = 24 * time.Hour
    MaxVLCPerEpoch    = 1_000_000
    
    // Round 配置
    RoundInterval     = 5 * time.Minute
    MaxTasksPerRound  = 1000
    
    // Flux 配置
    FluxMintPerEpoch  = 100_000
    MinFluxBalance    = 0
    
    // Credit 配置
    MaxCreditScore    = 100.0
    MinCreditScore    = 0.0
    CreditDecayRate   = 0.95  // 未提交时衰减
    
    // 验证配置
    BFTThreshold      = 0.5   // 50% 共识
    MinValidators     = 3
    MaxValidators     = 21
    
    // 性能配置
    MaxHistoryEpochs  = 100
    VLCCompressionEnabled = true
    BatchSize         = 100
)
```

### 17.4 下一步行动

1. **立即开始**：
   - 创建项目仓库结构
   - 设计详细的 API 规范
   - 搭建开发环境

2. **原型验证**：
   - 实现 VLC 防溢出原型
   - 测试 Epoch 切换逻辑
   - 验证性能指标

3. **社区建设**：
   - 发布技术白皮书
   - 开发者文档
   - 示例应用

---

**这个架构设计提供了一个完整的、可扩展的主网-子网 PoCW 系统，解决了 VLC 溢出、分层共识、激励分配等核心问题。**
