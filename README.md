# PoCW (Proof-of-Causal-Work) Subnet Demonstration

A distributed AI consensus system where specialized agents (miners) process user tasks while validators ensure quality through Byzantine Fault Tolerant consensus and Vector Logical Clock consistency.

## 🚀 Quick Start

### Prerequisites

- **Go** >= 1.21
- **Docker** (for graph visualization)
- **Git** (for cloning repository)

### Setup & Run

```bash
# 1. Setup environment (starts Dgraph container)
./setup.sh

# 2. Run the subnet demonstration
go run main.go

# 3. Cleanup when finished
./teardown.sh
```

### Accessing Graph Visualization

Once setup is complete, access the interactive graph visualization:
- **Ratel UI**: http://localhost:8000 
- **Dgraph API**: http://localhost:8080
- **GraphQL**: http://localhost:8080/graphql

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
PoCW/
├── main.go              # Entry point - subnet demonstration
├── setup.sh            # Environment setup script  
├── teardown.sh         # Cleanup script
├── go.mod              # Go module dependencies
├── subnet/             # Core subnet implementation
│   ├── core_miner.go   # AI miner agents
│   ├── core_validator.go # BFT validators  
│   ├── messages.go     # Protocol messages
│   └── demo/           # Demonstration scenarios
├── vlc/                # Vector Logical Clock implementation
├── dgraph/             # Graph database integration
└── integration/        # Mainnet bridge components
```

---

**🌟 Intelligence Money represents the next evolution of digital assets - from energy-based mining to intelligence-based value creation. This subnet demonstration shows how AI agents can collaborate, compete, and be fairly compensated in a trustless, merit-based economy.**
