# Intelligence Money (KEY) Mining System

## Overview

This project implements a **Proof-of-Causal-Work (PoCW)** blockchain system for mining Intelligence Money (KEY tokens) through verifiable AI task completion. The system uses Vector Logical Clocks (VLC) to ensure causal ordering of distributed AI operations and rewards participants with soulbound KEY tokens.

## Key Concepts

### Proof-of-Causal-Work (PoCW)
A novel consensus mechanism where:
- Miners perform AI tasks and generate computational graphs
- Validators verify task completion and coordinate epoch submissions
- Rewards are distributed based on successful AI task completion
- Vector Logical Clocks ensure causal consistency in distributed operations

### Intelligence Money (KEY)
- **Soulbound tokens**: Non-transferable but redeemable for stablecoins
- **Maximum supply**: 21,000,000 KEY tokens
- **Mining mechanism**: Minted through verified AI task completion
- **Reward rate**: 100 KEY per successful AI task

### HETU Tokens
- **Staking tokens** for subnet registration and deposits
- **Fixed supply**: 1,000,000 HETU tokens
- **Deposit requirements**: 500 HETU (miner), 100 HETU per validator

## System Architecture

### Smart Contracts

#### 1. HETUToken.sol
```solidity
contract HETUToken is ERC20 {
    uint256 public constant TOTAL_SUPPLY = 1_000_000 * 10**18;
}
```
- Standard ERC20 token for subnet deposits
- Fixed supply of 1M HETU tokens
- Used for economic security in subnet registration

#### 2. KEYToken.sol
```solidity
contract KEYToken {
    uint256 public constant MAX_SUPPLY = 21_000_000 * 10**18;
    function mine(address recipient, uint256 amount, string memory reason) external onlyVerifier;
}
```
- Soulbound token implementation (non-transferable)
- Mining function restricted to authorized PoCW verifiers
- Tracks total supply against 21M maximum

#### 3. SubnetRegistry.sol
```solidity
struct Subnet {
    address miner;
    address[4] validators;
    bool isActive;
}
function registerSubnet(string memory subnetId, address miner, address[4] memory validators) external;
```
- Manages subnet registration and participant roles
- Requires HETU token deposits (500 from miner, 100 from each validator)
- Tracks subnet state and participant addresses

#### 4. EnhancedPoCWVerifier.sol
```solidity
struct EpochSubmission {
    bytes32 subnetId;
    uint256 epochNumber;
    bytes vlcGraphData;
    address[] successfulMiners;
    uint256 successfulTasks;
    uint256 failedTasks;
    uint256 timestamp;
    bool verified;
    bool rewardsDistributed;
    address submittingValidator;
    uint256 totalRewardDistributed;
}

struct MinerStats {
    address owner;
    uint256 successfulTasks;
    uint256 totalTasks;
    uint256 totalIntelligenceMined;
    uint256 reputationScore; // Percentage (0-100)
    uint256 lastActiveEpoch;
    uint256 joinedTimestamp;
    bool isActive;
}

function submitAndDistributeEpoch(
    string memory subnetId,
    bytes memory vlcGraphData,
    address[] memory successfulMiners,
    uint256 successfulTasks,
    uint256 failedTasks
) external;
```
- **Enhanced** PoCW verification with comprehensive tracking
- Maintains detailed statistics for miners and subnets
- Supports multiple miners per epoch (flexible architecture)
- Automatic reward calculation and distribution
- Reputation scoring system based on task success rate
- Complete audit trail of all epoch submissions

## System Flow

### 1. Subnet Registration
```
1. Deploy contracts (HETU, KEY, Registry, EpochSubmitter)
2. Initialize contract relationships
3. Distribute HETU tokens to participants
4. Participants approve HETU spending
5. Miner calls registerSubnet() with validator addresses
6. Deposits are locked, subnet becomes active
```

### 2. Epoch Submission and Mining
```
1. Miners perform AI tasks off-chain
2. VLC graph data is generated tracking causal relationships
3. Validator-1 submits epoch data to EnhancedPoCWVerifier:
   - Subnet ID
   - VLC graph data (comprehensive JSON as bytes)
   - Array of successful miner addresses
   - Count of successful tasks
   - Count of failed tasks
4. Contract verifies:
   - Subnet is active
   - Caller is validator-1
   - Addresses match registered participants
   - Automatic epoch numbering
5. KEY tokens are mined and statistics updated:
   - Miner reward = successful_tasks × 100 KEY per task
   - Validator reward = 20 KEY base + 10% of miner rewards
   - Miner stats: tasks completed, reputation score updated
   - Subnet info: epoch count, total tasks tracked
6. Enhanced events emitted with full audit trail
```

### 3. Enhanced Reward Distribution Example
```
Successful tasks: 5
Failed tasks: 0  
Miner reward: 5 × 100 KEY = 500 KEY
Validator base reward: 20 KEY
Validator percentage reward: 500 × 10% = 50 KEY
Total validator reward: 20 + 50 = 70 KEY
Total KEY mined: 570 KEY

Miner Statistics Updated:
- Total successful tasks: 5
- Total tasks attempted: 5
- Reputation score: 100% (perfect success rate)
- Total intelligence mined: 500 KEY
```

## Network Configuration

### Local Development (Anvil)
- **RPC URL**: http://localhost:8545
- **Block mining**: On-demand (only when transactions occur)
- **Accounts**: 10 pre-funded accounts with test mnemonic
- **Starting block**: Always 0 (fresh deployment each test)

### Account Roles
```
Deployer:    0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266 (Contract owner)
Miner:       0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc (AI task performer)
Validator-1: 0x70997970C51812dc3A010C7d01b50e0d17dc79C8 (Epoch submitter)
Validator-2: 0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC (Consensus participant)
Validator-3: 0x90F79bf6EB2c4f870365E785982E1f101E93b906 (Consensus participant)
Validator-4: 0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65 (Consensus participant)
```

## Deployment Scripts

### tests/test-enhanced-pocw.sh
**Primary testing pipeline** with enhanced features:
1. **Complete cleanup**: Stop existing Anvil, remove all artifacts
2. **Fresh Anvil start**: New instance from block 0, on-demand mining
3. **Smart contract compilation**: Build all enhanced contracts
4. **Sequential deployment**: Deploy all 4 core contracts
5. **Contract initialization**: Set up relationships and permissions
6. **Enhanced token distribution**: 2000 HETU to each participant (increased for visible balances)
7. **Approval setup**: Enable HETU spending for subnet deposits
8. **Subnet registration**: Register subnet with 1 miner + 4 validators
9. **Comprehensive epoch testing**: Submit VLC data and verify mining
10. **Statistics verification**: Decode and display miner statistics
11. **contract_addresses.json generation**: Support for dashboard.html
12. **Inspection mode**: Keep Anvil running for UI testing

### Key Features
- **Enhanced statistics**: Detailed miner performance tracking
- **Proper decoding**: Human-readable statistics display
- **Dashboard support**: Generates contract addresses JSON for UI
- **Inspection mode**: Anvil stays running for manual blockchain exploration
- **Increased balances**: 2000 HETU distribution for better UI visibility
- **Complete audit trail**: Full epoch submission tracking

## Graph Data Format

VLC graph data is submitted as comprehensive JSON containing detailed causal relationships:
```json
{
  "subnetId": "enhanced-subnet-001",
  "epochNumber": 1,
  "events": [
    {
      "id": "genesis_0",
      "name": "GenesisState", 
      "vlcClock": {},
      "parents": [],
      "timestamp": 1725404536000
    },
    {
      "id": "task_1",
      "name": "AITaskExecution",
      "vlcClock": {"miner1": 1, "validator1": 0},
      "parents": ["genesis_0"],
      "timestamp": 1725404536000,
      "taskId": "ai_inference_001",
      "result": "success"
    },
    {
      "id": "validation_1",
      "name": "TaskValidation", 
      "vlcClock": {"miner1": 2, "validator1": 1},
      "parents": ["task_2"],
      "timestamp": 1725404536000,
      "validatedTasks": ["ai_inference_001", "ai_inference_002"],
      "validationResult": "approved"
    }
  ],
  "miners": ["0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"],
  "validators": ["0x70997970C51812dc3A010C7d01b50e0d17dc79C8", "..."],
  "summary": {
    "totalTasks": 5,
    "successfulTasks": 5,
    "failedTasks": 0,
    "validationStatus": "complete"
  }
}
```

## Economic Model

### Token Distribution
- **Total KEY supply**: 21,000,000 tokens
- **Mining rate**: 100 KEY per successful AI task
- **Validator share**: 10% of all mining rewards
- **Max theoretical tasks**: 210,000 successful tasks to reach cap

### Deposit Requirements
- **Miner deposit**: 500 HETU tokens (locked during subnet operation)
- **Validator deposit**: 100 HETU tokens each (locked during subnet operation)
- **Total subnet cost**: 900 HETU tokens (0.09% of total supply per subnet)
- **Test distribution**: 2000 HETU tokens given to each participant for enhanced UI visibility

### Security Considerations
- **Soulbound tokens**: Prevents speculative trading
- **Validator-1 submission**: Single point reduces consensus complexity
- **Address verification**: Ensures rewards go to registered participants
- **Deposit slashing**: Economic penalty for malicious behavior (future enhancement)

## Testing Results

Latest enhanced test run with comprehensive tracking:
```
🎯 Subnet: enhanced-subnet-001 (ACTIVE) 
📊 Tasks: 5 successful, 0 failed
💰 KEY tokens mined:
   - Miner earned: 500.00 KEY (5 tasks × 100 KEY)
   - Validator-1 earned: 70.00 KEY (20 base + 50 percentage)
🔑 Total KEY supply: 570 KEY
📈 Epochs completed: 1

📋 Miner Statistics:
   - Owner: 0x9965507d1a55bcc2695c58ba16fb37d819b0a4dc
   - Successful Tasks: 5
   - Total Tasks: 5  
   - Intelligence Mined: 500.00 KEY
   - Reputation Score: 100%
   - Is Active: Yes

💰 Expected HETU Balances (after deposits):
   - Miner: 1,500 HETU (2,000 - 500 deposit)
   - Each Validator: 1,900 HETU (2,000 - 100 deposit)

🌐 UI Support:
   - Dashboard: http://localhost:3000 (with contract_addresses.json)
   - Blockchain: http://localhost:8545 (Anvil RPC)
   - Inspection Mode: Anvil runs until Ctrl+C
```

## Recent Enhancements ✅

### Completed Features
1. **Enhanced PoCW Verifier**: Comprehensive statistics tracking and reputation system
2. **Miner Statistics**: Detailed performance metrics with success rate calculation
3. **Improved UI Support**: contract_addresses.json generation for dashboard.html
4. **Statistics Decoding**: Human-readable display of blockchain data
5. **Enhanced Testing**: Inspection mode with increased HETU balances for visibility
6. **Better Reward System**: Base + percentage validator rewards (20 + 10%)
7. **Complete Audit Trail**: Full epoch submission tracking and verification
