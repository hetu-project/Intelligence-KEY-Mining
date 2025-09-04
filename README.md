# Intelligence KEY Mining - Proof-of-Causal-Work System

A blockchain-based consensus mechanism that mines soulbound **KEY tokens** through verified AI intelligence work, implementing Vector Logical Clocks (VLC) for causal ordering and Byzantine Fault Tolerant consensus.

## Overview

This system demonstrates a novel approach to cryptocurrency mining where tokens are earned through actual valuable AI work rather than computational waste. The **KEY token** represents verifiable intelligence contributions and is non-transferable (soulbound) but redeemable.

### Key Features

- 🧠 **Intelligence Mining**: Earn KEY tokens through actual AI task completion
- 🔗 **Vector Logical Clocks**: Causal ordering of distributed consensus events  
- 🏛️ **Byzantine Fault Tolerant**: 4-validator consensus with quality assessment
- 💎 **Soulbound Tokens**: Non-transferable but redeemable KEY tokens
- 📊 **Real-time Visualization**: VLC event graph via Dgraph
- ⛓️ **Blockchain Integration**: Smart contracts on Anvil/Ethereum

## Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   Subnet        │    │   Bridge         │    │   Blockchain    │
│                 │    │                  │    │                 │
│ Miner + 4       │◄──►│ HTTP Server      │◄──►│ Smart Contracts │
│ Validators      │    │ (Port 3001)      │    │ (Anvil)         │
│                 │    │                  │    │                 │
│ VLC Consensus   │    │ Per-epoch        │    │ KEY Mining      │
└─────────────────┘    │ Submission       │    │ Token Rewards   │
         │              └──────────────────┘    └─────────────────┘
         ▼
┌─────────────────┐
│   Dgraph        │
│   VLC Graph     │
│   Visualization │
└─────────────────┘
```

## Two Execution Modes

### 1. Subnet-Only Mode 🔹

**Purpose**: Pure subnet consensus demonstration with VLC visualization (no blockchain integration)

**What it does**:
- ✅ Runs distributed consensus with 1 miner + 4 validators
- ✅ Processes 7 rounds of AI tasks with quality assessment  
- ✅ Generates VLC event graph for causal analysis
- ✅ Provides interactive exploration of consensus behavior
- ❌ No blockchain integration or KEY token mining

**Run Command**:
```bash
./run-subnet-only.sh
```

**Access Points**:
- 📊 VLC Graph: `http://localhost:8000` (Dgraph Ratel UI)
- 📋 Event Query: `http://localhost:8080/graphql`

### 2. KEY Mining Mode 💰

**Purpose**: Complete PoCW system with real KEY token mining and blockchain integration

**What it does**:
- ✅ Everything from subnet-only mode PLUS:
- ✅ Deploys smart contracts (KEY Token, PoCW Verifier, etc.)
- ✅ Real-time KEY mining per epoch (every 3 rounds)
- ✅ Blockchain transactions with verified rewards
- ✅ Bridge service for epoch submission
- ✅ Complete before/after balance tracking

**Run Command**:
```bash
./run-key-mining.sh
```

**Access Points**:
- 📊 VLC Graph: `http://localhost:8000`
- 🔍 Blockchain Inspector: `http://localhost:3000/pocw-inspector.html`
- ⛓️ Blockchain RPC: `http://localhost:8545`
- 🌐 Bridge API: `http://localhost:3001`

## Prerequisites

### Required Software
```bash
# Install Foundry (for Anvil and Cast)
curl -L https://foundry.paradigm.xyz | bash
foundryup

# Install Go >= 1.21
# Install Node.js >= 22
# Install Docker (for Dgraph)
# Install bc (for calculations)
sudo apt install bc
```

### System Requirements
- **Ports**: 3000, 3001, 8000, 8080, 8545, 9080 (must be available)
- **Docker**: Required for Dgraph container
- **Sudo Access**: Required for Docker operations

## Quick Start

### Option 1: Subnet-Only Demo
```bash
# Clean run of subnet consensus with VLC visualization
sudo ./run-subnet-only.sh

# Access VLC graph at http://localhost:8000
# Press Ctrl+C when done exploring
```

### Option 2: Full KEY Mining Demo  
```bash
# Complete PoCW system with blockchain integration
sudo ./run-key-mining.sh

# Watch KEY tokens being mined in real-time
# Explore blockchain inspector at http://localhost:3000/pocw-inspector.html
# Bridge stays active for continued mining
# Press Ctrl+C when done
```

## Smart Contracts

| Contract | Purpose | Features |
|----------|---------|----------|
| **KEYToken** | Soulbound intelligence tokens | Non-transferable, 21M max supply |
| **HETUToken** | Staking for subnet registration | ERC20, 1M total supply |
| **SubnetRegistry** | Manages subnet participants | Deposit requirements, validation |
| **EnhancedPoCWVerifier** | Consensus verification & mining | Per-epoch KEY distribution |

## Expected Output

### Subnet-Only Mode
```
🔹 PoCW SUBNET-ONLY DEMONSTRATION
Architecture: Pure subnet consensus with VLC graph visualization

✅ 7 rounds processed with Byzantine consensus
✅ VLC events committed to Dgraph
🔄 Interactive mode - explore at http://localhost:8000
```

### KEY Mining Mode
```
💰 PoCW KEY MINING SYSTEM  
Architecture: Complete KEY mining with blockchain integration

💰 Initial KEY Token Balances (Before Mining)
📊 Miner: 0.000000 KEY
📊 Validator-1: 0.000000 KEY
...

[CONSENSUS & MINING HAPPENS]

💰 Final KEY Token Balances (After Mining)
📊 Miner: 400.000000 KEY (+400.000000 KEY mined)
📊 Validator-1: 25.000000 KEY (+25.000000 KEY mined)
📊 Total Supply: 500.000000 KEY (+500.000000 KEY total mined)

🎉 Bridge stays running for continued KEY mining!
```

## Key Concepts

### Epochs & Rounds
- **Round**: Single AI task with miner output + validator consensus
- **Epoch**: 3 consecutive rounds
- **Mining Trigger**: Each completed epoch triggers KEY token mining

### VLC (Vector Logical Clocks)
- Ensures causal ordering of distributed events
- Tracks happened-before relationships
- Prevents Byzantine inconsistencies
- Visualized as directed graphs in Dgraph

### KEY Token Economics
- **Mining Rate**: Based on successful task completion
- **Distribution**: 80% to miner, 20% split among validators
- **Soulbound**: Cannot be transferred, but can be redeemed
- **Max Supply**: 21 million KEY (like Bitcoin)

## Troubleshooting

### Common Issues

**Port Conflicts**:
```bash
# Check what's using required ports
netstat -tlnp | grep -E ":(3000|3001|8000|8080|8545|9080)"
# Kill conflicting processes if needed
```

**Docker Permissions**:
```bash
# Run the scripts with sudo to handle Docker operations
sudo ./run-subnet-only.sh
sudo ./run-key-mining.sh
```

**Bridge Connection Errors**:
- Ensure Node.js dependencies are installed: `npm install`
- Bridge starts after contract deployment in key-mining mode
- Check bridge health: `curl http://localhost:3001/health`

**Dgraph Issues**:
- Wait 30+ seconds for Dgraph container to fully start
- Try accessing Ratel UI directly: `http://localhost:8000`
- Check container: `docker ps | grep dgraph`

### Clean Restart
```bash
# If anything gets stuck, clean everything:
pkill anvil
sudo docker stop dgraph-standalone
sudo docker rm dgraph-standalone  
sudo rm -rf ./dgraph-data
rm -f contract_addresses.json *.log *.pid
```

## 🏗️ Architecture Overview

### Core Components

**🤖 CoreMiner**
- AI agents that process user tasks and requests
- Maintain Vector Logical Clock (VLC) consistency
- Generate responses and request additional context when needed
- Process 7 different test scenarios demonstrating various interaction patterns

**🛡️ CoreValidator** 
- Quality assessment nodes that vote on miner outputs
- Two specialized roles:
  - **UserInterfaceValidator**: Handles user interaction and info requests
  - **ConsensusValidator**: Performs distributed quality voting
- Byzantine Fault Tolerant consensus with 0.25 weight per validator
- Validates VLC sequences and ensures causal consistency

**⏰ Vector Logical Clocks (VLC)**
- Ensures causal ordering of all operations
- Tracks dependencies between events across the network
- Prevents out-of-order execution and maintains consistency
- Critical for distributed consensus and event validation

**📊 Graph Visualization**
- Real-time VLC event tracking via Dgraph database
- Interactive visualization of causal relationships
- Event categorization: UserInput, MinerOutput, InfoRequest, RoundSuccess, etc.
- Complete audit trail of all network interactions

## 🎯 Demonstration Scenarios

The subnet demo processes **7 test scenarios** that showcase all aspects of the PoCW protocol:

### Standard Processing (Scenarios 1, 2, 5, 7)
- **User Input**: "Analyze market trends for Q4"
- **Miner Response**: Direct solution generation
- **Validator Assessment**: Quality voting and consensus
- **User Feedback**: Acceptance confirmation
- **Result**: `OUTPUT DELIVERED TO USER`

### Information Request Flow (Scenarios 3, 6)  
- **User Input**: "Create optimization strategy for resource allocation"
- **Miner Behavior**: Requests additional context
- **Validator Mediation**: Facilitates user-miner communication
- **Enhanced Processing**: Solution with additional context
- **Advanced Validation**: Quality assessment of refined output

### Rejection Scenarios (Scenario 4)
- **Validator Rejection**: Low-quality output rejected by consensus (0.45 quality score)
- **Result**: `OUTPUT REJECTED BY VALIDATORS`

### User Override (Scenario 6)
- **Validator Acceptance**: Output passes validator consensus
- **User Rejection**: User rejects despite validator approval
- **Result**: `OUTPUT REJECTED BY USER (despite validator acceptance)`

## 📊 Visualizing Event Graphs

After running the demonstration, visualize the complete VLC event graph:

### Access Steps
1. **Open Ratel UI**: http://localhost:8000
2. **Verify Connection**: Ensure connection shows `localhost:8080`
3. **Query Events**: Use this GraphQL query to view all subnet events:

```graphql
{
  events(func: has(event_id)) {
    uid
    event_id
    event_name  
    event_type
    subnet_id
    request_id
    vlc_clock
    parents {
      uid
      event_id
      event_name
    }
    timestamp
    description
  }
}
```

### Event Types in Subnet Demo
- **🎯 UserInput**: User task submissions (7 scenarios)
- **🤖 MinerOutput**: AI agent responses and solutions
- **❓ InfoRequest**: Miner requests for additional context  
- **💬 InfoResponse**: User-provided clarifications
- **✅ RoundSuccess**: Successful consensus rounds
- **❌ RoundFailed**: Failed validation or user rejection
- **🏁 EpochFinalized**: Subnet epoch completion markers
- **⭐ GenesisState**: Initial subnet state

### Understanding VLC Relationships
- **Parent Links**: Show causal dependencies between events
- **VLC Clocks**: Demonstrate proper ordering (format: `{miner:X, validator:Y}`)

The graph provides a complete audit trail showing how each user request flows through the subnet, demonstrating the causal consistency guarantees of the PoCW protocol.

## 🧠 Intelligence Money & PoCW Protocol

### Proof-of-Causal-Work Consensus

PoCW extends traditional blockchain consensus by focusing on **causal relationships** rather than just computational work:

**🔗 Causal Consistency**
- Every event must reference its causal dependencies
- Vector Logical Clocks ensure proper ordering across distributed nodes
- Invalid causal relationships are automatically rejected
- Creates immutable audit trail of decision-making processes

**🏛️ Byzantine Fault Tolerant Consensus**  
- 4 validators with 0.25 weight each (total weight = 1.0)
- Requires majority consensus for output acceptance
- Handles up to 1 Byzantine (malicious) validator
- Quality threshold-based voting (accept if quality > 0.5)

**⚡ Event-Driven Architecture**
- Real-time processing of user requests
- Dynamic info request/response cycles
- Asynchronous validator consensus
- Complete traceability of all interactions 

### Blockchain Integration Architecture

PoCW subnets can be bridged to mainnet blockchains for broader ecosystem integration:

```
🔄 Subnet to Mainnet Flow:
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│   PoCW Subnet   │───▶│  Epoch Finalize  │───▶│  Mainnet Post   │
│                 │    │                  │    │                 │
│ • Miners        │    │ • VLC State      │    │ • Smart Contract│
│ • Validators    │    │ • Consensus      │    │ • TOKEN Mining  │
│ • VLC Graph     │    │                  │    │ • Verification  │
└─────────────────┘    └──────────────────┘    └─────────────────┘
```

## 💰 Intelligence Money (KEY Tokens)

### Revolutionary Digital Asset Class

Intelligence Money represents **verifiable units of intelligent work** - the first digital asset derived from provable AI contributions rather than energy consumption.

**🎯 Core Principles**
```
Traditional Crypto:  Energy → Computational Work → Token Value
Intelligence Money:   AI Work → Verified Contribution → KEY Value
```

### Mining Through Value Creation

**🏗️ The Mining Process**
1. **Query Initiation**: User submits complex problem to subnet
2. **AI Collaboration**: Miners and validators work together via PoCW
3. **Quality Validation**: BFT consensus ensures solution quality  
4. **Value Attribution**: Successful contributions mine new KEY tokens
5. **Cryptographic Proof**: VLC graph provides immutable work evidence

**⚡ Real-Time Mining**
- Every accepted user solution mines new tokens
- Quality multipliers affect mining rewards
- Validator consensus participation earns rewards
- Failed outputs generate no tokens (merit-based system)

### Soulbound Token Economics

**🔒 Non-Transferable Design**
- **Soulbound**: KEY tokens cannot be transferred between addresses
- **Reputation-Based**: Tokens represent earned capability and track record
- **Anti-Speculation**: Prevents market manipulation and speculation bubbles
- **Cryptographic Resume**: Immutable proof of AI agent competence

**💵 Liquidity Bridge**
- **Redemption Pool**: One-way bridge to stablecoins (USDC/USDT)
- **Burn Mechanism**: KEY tokens are destroyed when redeemed
- **Market Valuation**: Exchange rate determined by supply/demand
- **Utility Preservation**: Core reputation asset remains non-transferable

### Economic Model

```
🔄 VALUE FLOW:
User Problem ──▶ AI Solution ──▶ Quality Validation ──▶ KEY Mining ──▶ Stablecoin Redemption
     ▲                                      │                              │
     └─────── Economic Feedback Loop ───────┴──────────────────────────────┘
```

This creates a **merit-only economy** where value flows directly from problem-solving capability to economic rewards, eliminating speculative intermediaries and ensuring AI agents are compensated based purely on their verifiable contributions to human knowledge and productivity.

## 🛠️ Development & Contributing

### Project Structure
```
Intelligence-KEY-Mining/
├── main.go                     # Entry point - subnet demonstration
├── go.mod                      # Go module dependencies
├── run-subnet-only.sh          # Subnet consensus only mode
├── run-key-mining.sh           # Full KEY mining mode
├── run-per-epoch-integration.sh # Alternative KEY mining script
├── serve-dashboard.go          # Web UI server (integrated into scripts)
├── mainnet-bridge-per-epoch.js # HTTP bridge service (port 3001)
├── pocw-inspector.html # Blockchain inspector UI
├── contracts/                  # Solidity smart contracts
│   ├── KEYToken.sol           # Soulbound KEY tokens
│   ├── HETUToken.sol          # Staking token
│   ├── SubnetRegistry.sol     # Subnet management
│   └── EnhancedPoCWVerifier.sol # Consensus verification
├── subnet/                    # Go consensus implementation  
│   ├── core_miner.go         # AI miner agents
│   ├── core_validator.go     # BFT validators
│   ├── graph_adapter.go      # VLC graph & HTTP bridge integration
│   ├── messages.go           # Protocol message definitions
│   └── demo/                 # Demo scenarios & coordination
│       ├── demo_coordinator.go      # Demo orchestration
│       ├── demo_task_processor.go   # Task processing logic
│       ├── demo_quality_assessor.go # Quality assessment
│       └── demo_user_interaction.go # User interface simulation
├── vlc/                      # Vector Logical Clock library
│   └── vlc.go               # VLC implementation
├── dgraph/                   # Graph database integration
│   ├── connection.go        # Dgraph client connection
│   └── init.go              # Dgraph initialization
├── models/                   # Data models
│   └── event.go             # Event structure definitions
└── tests/                    # Test utilities
    ├── test-enhanced-pocw.js  # JavaScript contract tests
    └── test-enhanced-pocw.sh  # Shell test runner
```

### Adding New Features
1. **Subnet Logic**: Modify files in `subnet/` directory
2. **Smart Contracts**: Update contracts in `contracts/` directory  
3. **Bridge Logic**: Enhance `mainnet-bridge-per-epoch.js`
4. **UI Components**: Update `pocw-inspector.html`

## License

MIT License - See [LICENSE](LICENSE) for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Test with both `./run-subnet-only.sh` and `./run-key-mining.sh`
4. Submit a pull request

---

🌟 **Start with `./run-subnet-only.sh` to understand the consensus, then try `./run-key-mining.sh` for the full KEY mining experience!**

**Intelligence Money represents the next evolution of digital assets - from energy-based mining to intelligence-based value creation. This system shows how AI agents can collaborate, compete, and be fairly compensated in a trustless, merit-based economy.**
