# KEY Identity Hub - Intelligence Mining Platform

## Introduction
Intelligence-KEY-Mining is a decentralized mining system that implements Proof of Cognitive Work (PoCW) consensus mechanism. The system consists of multiple microservices including miner-gateway, validators, points service, and SBT service, all working together to validate social tasks (like Twitter retweets) and distribute rewards.

## Quick Start

### 1. Navigate to Project Directory
```bash
cd /path/to/Intelligence-KEY-Mining
```

### 2. Generate Environment Configuration File
```bash
# Interactive mode (recommended)
bash scripts/setup-env.sh interactive

# Or simple mode
bash scripts/setup-env.sh create
```

### 3. Configure Environment Variables
Edit the generated `.env` file with your actual parameters. The script will automatically highlight required modifications.

### 4. Start Core Services
```bash
# One-click startup script
bash scripts/start-core-services.sh
```

The startup script automatically handles:
- Dependency checks
- Infrastructure startup (MySQL, Redis, Dgraph)
- Database migrations (if needed)
- Core service initialization
- Health checks

### Service Management Commands

**Rebuild and restart after code changes:**
```bash
docker-compose stop miner-gateway
docker-compose build miner-gateway  
docker-compose up -d miner-gateway
```

**Real-time log monitoring:**
```bash
# Twitter-related logs only
docker-compose logs -f miner-gateway | grep -i twitter

# Verification-related logs
docker-compose logs -f miner-gateway | grep -i "verif\|batch"

# Check task creation
docker-compose logs miner-gateway | grep "Task created"

# Check task status
docker-compose logs miner-gateway | grep "PENDING_VERIFICATION"
```

## Environment Variables Configuration

### 1. Database Configuration
*Purpose: All services require MySQL connection for storing tasks, points, and user data*
```bash
MYSQL_ROOT_PASSWORD=your_secure_root_password
MYSQL_DATABASE=pocw_db
MYSQL_USER=pocw_user
MYSQL_PASSWORD=pocw_password
DATABASE_URL=mysql://pocw_user:pocw_password@localhost:3306/pocw_db
```

### 2. Blockchain Configuration
*Purpose: SBT service for Ethereum interaction and SBT management*
```bash
ETH_RPC_URL=https://sepolia.infura.io/v3/your_project_id
SBT_CONTRACT_ADDRESS=0x...
SBT_CONTRACT_PRIVATE_KEY=0x...
```

### 3. IPFS Configuration
*Purpose: SBT service for uploading metadata and images to IPFS*
```bash
PINATA_API_KEY=your_pinata_api_key
PINATA_SECRET_KEY=your_pinata_secret_key
```

### 4. Private Key Configuration
*Purpose: Each service uses its own private key for authentication and message signing*
```bash
MINER_PRIVATE_KEY=0x1111...
VALIDATOR_1_PRIVATE_KEY=0x2222...
VALIDATOR_2_PRIVATE_KEY=0x3333...
VALIDATOR_3_PRIVATE_KEY=0x4444...
VALIDATOR_4_PRIVATE_KEY=0x5555...
```

### 5. Validator Endpoints Configuration
*Purpose: Miner-gateway connects to validator services based on this configuration*
```bash
VALIDATOR_ENDPOINTS=[{"id":"validator-1","role":"ui_validator","url":"http://validator-ui:8080","weight":0.40,"priority":1},...]
```

### 6. Twitter Verification
*Purpose: Miner-gateway for Twitter retweet task validation (currently using third-party API)*
```bash
TWITTER_RETWEET_CHECK_URL=http://144.91.78.212:8000/api/v1/twitter/retweet-check
```

### 7. Third-party Configuration
*Purpose: SBT service for fetching user referral relationships*
```bash
REFERRAL_API_URL=https://api.hetuverse.com/subnet/api/v1/referral/user/sbt
```

### 8. Timing Configuration
*Purpose: Miner-gateway controls task verification frequency and PoCW consensus timing*
```bash
VALIDATOR_POLL_INTERVAL_SECONDS=600     # 10 minutes (default 2 hours)
CONSENSUS_DELAY_SECONDS=300             # 5 minutes
```

### 9. Service URL Configuration
*Purpose: HTTP communication addresses between services*
```bash
POINTS_SERVICE_URL=http://points-service:8080
SBT_SERVICE_BASE_URL=http://localhost:8086
DGRAPH_URL=dgraph-alpha:9080
REDIS_URL=redis://redis:6379
```

### 10. Points System Configuration
*Purpose: Points service configuration using direct VLC-to-points mapping*
```bash
POINTS_HISTORY_LIMIT=1000
```

### 11. Logging Configuration
*Purpose: Controls log verbosity for all services*
```bash
LOG_LEVEL=info  # debug, info, warn, error
```

## Service Architecture

### Project Structure Comparison
```
Intelligence-KEY-Mining/
├── subnet/                    # Original Demo Code
│   ├── core_miner.go         # Original Miner Implementation
│   ├── core_validator.go     # Original Validator Implementation  
│   ├── demo/                 # Demo Coordinator
│   └── messages.go           # Original Message Definitions
├── services/                  # New Architecture Core Services
│   ├── miner-gateway/        # Miner Service
│   ├── validator/            # Validator Service (Independent Nodes)
│   ├── points-service/       # Points Management Service
│   └── sbt-service/          # SBT Identity Service
```

### Service Communication

**Protocol:** HTTP REST API + JSON via Docker Compose network

| From | To | Purpose | Endpoint |
|------|----|---------|---------| 
| miner-gateway | validators (4x) | PoCW consensus voting | `POST /api/v1/validate` |
| miner-gateway | points-service | Points distribution | `POST /api/v1/points/distribute` |
| sbt-service | points-service | Get user points | `GET /api/v1/points/user/{wallet}` |
| sbt-service | External API | Referral data | `GET api.hetuverse.com/...` |

### Task Processing Overview

**TaskCreation**: Direct validation → VLC++ → 50 points → VERIFIED  
**Twitter Tasks**: API verification → VLC++ → Batch PoCW → Points distribution

## PoCW Shared Components

### 1. VLC (Vector Clock) Management
- `pkg/vlc/vector_clock.go` (Shared VLC implementation)
- `miner-gateway/services/vlc_service.go`
- `validator/services/vlc_service.go`

### 2. Protocol Message Definitions
- `pkg/protocol/messages.go` (MinerOutputRequest, ValidatorVoteResponse)

### 3. Quality Assessment Interface
- `validator/plugins/quality_assessor.go` (QualityAssessor interface)
- Original demo: `subnet/demo/demo_quality_assessor.go`

### 4. BFT Consensus Logic
- `miner-gateway/services/coordinator.go` (RoundCoordinator)
- Original demo: `subnet/demo/demo_coordinator.go`

## Current System Architecture

### Core Services Overview

| Service | Port | Purpose | Key Components |
|---------|------|---------|----------------|
| **miner-gateway** | 8081 | Task processing & PoCW coordination | `BatchVerifier`, `RoundCoordinator`, `ValidatorScheduler` |
| **validator-ui** | 8082 | VLC validation & UI interaction | `UIValidator`, `VLCService` |
| **validator-format-1/2** | 8083/8084 | Format validation | `FormatValidator`, `TwitterQualityAssessor` |
| **validator-semantic** | 8085 | Semantic validation | `SemanticValidator`, deep analysis |
| **points-service** | 8087 | Points calculation & distribution | Direct VLC-to-points mapping |
| **sbt-service** | 8086 | Soulbound token management | Dynamic metadata, referral integration |

### Task Processing Flow

#### TaskCreation Tasks
```
User submits task → ValidatorScheduler → Simple validation → VLC++ → 50 points → VERIFIED
```

#### Twitter Tasks  
```
User submits task → BatchVerifier → API verification → VLC++ → Batch PoCW → Points distribution
```

### PoCW Consensus Architecture

**Coordinator**: `RoundCoordinator` (miner-gateway)
- Manages round lifecycle
- Collects validator votes via HTTP
- Implements BFT consensus with weighted voting

**Validators**: 4 specialized validators
- **UI Validator**: VLC sequence validation + basic format check
- **Format Validators** (2x): Twitter/Tweet ID format validation  
- **Semantic Validator**: Deep consistency analysis

**Quality Assessment**: Role-based evaluation
- Each validator type has specialized `QualityAssessor`
- Scores combined with validator weights for BFT consensus
- Threshold: >50% weighted votes for approval

## Quality Assessor System

### Base Interface
**File:** `validator/plugins/quality_assessor.go`

```go
type QualityAssessor interface {
    AssessQuality(minerOutput *models.MinerOutput) (*QualityAssessment, error)
    GetRole() ValidatorRole
}

type ValidatorRole string
const (
    RoleUI       ValidatorRole = "ui"        // User Interface Validation
    RoleFormat   ValidatorRole = "format"    // Format Validation  
    RoleSemantic ValidatorRole = "semantic"  // Semantic Validation
)
```

### Validator Role Specialization

| Validator | Role | Weight | Primary Assessment |
|-----------|------|--------|-------------------|
| validator-ui | UI Validator | 0.40 | VLC validation + basic format |
| validator-format-1/2 | Format Validator | 0.20 each | Data format compliance |
| validator-semantic | Semantic Validator | 0.20 | Deep consistency analysis |

### Quality Assessment Process
1. Each validator assesses `MinerOutput` based on their specialized role
2. Returns `QualityResult{Accept, Score, Reason}` with score 0.0-1.0
3. Scores combined with validator weights for BFT consensus
4. Consensus threshold: >50% weighted approval for task acceptance

## BFT Consensus Mechanism

### Weighted Voting Process
1. **Vote Collection**: `RoundCoordinator` sends `MinerOutput` to 4 validators via HTTP
2. **Quality Assessment**: Each validator returns vote + quality score + weight  
3. **Consensus Calculation**: Weighted BFT algorithm
   - Total validator weight: 4.0 (0.40 + 0.20 + 0.20 + 0.20)
   - Consensus achieved when >50% weight participates (>2.0)
   - Task approved when >50% of participating weight votes "approve"
4. **Result**: Batch tasks approved/rejected based on weighted consensus

## Dgraph Usage

### Core Services DGraph Storage
**Location:** `services/miner-gateway/services/`

**Stored Content:**
1. **Batch Verification Rounds:**
   - round_id, start_time, end_time
   - Participating Twitter task list
   - Verification result summary

2. **PoCW Consensus Records:**
   - Each round's participants
   - Voting results and BFT status
   - Final consensus results

3. **VLC History:**
   - Each user's VLC change trajectory
   - Task completion corresponding VLC increments
   - Cross-service VLC synchronization status

**Current Implementation:**
- `services/miner-gateway/services/coordinator.go` (consensus round recording)
- `services/batch_verifier.go` (batch verification recording)
- `pkg/graph/graph_client.go` (DGraph client wrapper)

## System Flow Diagram

### Twitter Task Processing Flow

```mermaid
sequenceDiagram
    participant U as User
    participant MG as Miner-Gateway
    participant BV as BatchVerifier
    participant API as Twitter API
    participant V1 as Validator-UI
    participant V2 as Validator-Format
    participant V3 as Validator-Semantic
    participant RC as RoundCoordinator
    participant PS as Points-Service

    U->>MG: Submit Twitter task
    MG->>BV: Queue task for verification
    
    Note over BV,API: Single Task Verification Phase
    BV->>API: Verify retweet via external API
    API-->>BV: Verification result
    BV->>BV: VLC++ (user VLC increment)
    BV->>BV: Task status: PENDING_VERIFICATION
    
    Note over BV,RC: Batch Collection Phase  
    BV->>BV: Collect verified tasks into batch
    BV->>RC: Send completed batch round
    
    Note over RC,V3: PoCW Consensus Phase
    RC->>V1: Request validation (VLC + format check)
    V1-->>RC: Vote + Quality score + Weight
    
    RC->>V2: Request validation (format validation)
    V2-->>RC: Vote + Quality score + Weight
    
    RC->>V3: Request validation (semantic analysis)
    V3-->>RC: Vote + Quality score + Weight
    
    RC->>RC: BFT consensus calculation
    Note over RC: Weighted voting: >50% weight for approval
    
    Note over RC,PS: Points Distribution Phase
    RC->>PS: Distribute points for approved batch
    PS-->>RC: Distribution result
    RC->>RC: Update task status: VERIFIED
```

## Core Architecture & Technologies

This project builds a PoCW-based identity verification and reward system featuring:

- **Vector Clocks (VLC)** for causal ordering and user progress tracking
- **Multi-validator quality assessment** with specialized roles (UI, Format, Semantic)
- **Weighted BFT consensus** for Byzantine fault-tolerant voting
- **Microservices architecture** with HTTP REST API communication
- **Batch processing** for efficient multi-task consensus
- **Direct VLC-to-points mapping** for transparent reward distribution

The architecture ensures reliable social task validation while maintaining decentralization through coordinated specialized validators.