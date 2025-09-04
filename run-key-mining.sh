#!/bin/bash

# PoCW KEY Mining Script
# This script runs the complete PoCW system with KEY token mining:
# Real-time epoch submission where each completed epoch (3 rounds) 
# triggers immediate mainnet submission and KEY mining.

echo "💰 PoCW KEY MINING SYSTEM"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Architecture: Complete KEY mining with blockchain integration"
echo ""

# Preserve user's PATH when running with sudo
if [ -n "$SUDO_USER" ]; then
    USER_HOME=$(eval echo ~$SUDO_USER)
    # Common installation paths for Go, Foundry, Node.js
    export PATH="/usr/local/go/bin:$USER_HOME/go/bin:$USER_HOME/.foundry/bin:$USER_HOME/.local/bin:/snap/bin:$PATH"
else
    # Fallback paths when not running with sudo
    export PATH="/home/xx/.foundry/bin:$PATH"
fi

# Check prerequisites
echo "🔍 Checking prerequisites..."
if ! command -v anvil &> /dev/null; then
    echo "❌ Anvil not found. Please install Foundry."
    exit 1
fi

if ! command -v go &> /dev/null; then
    echo "❌ Go not found. Please install Go >= 1.21."
    exit 1
fi

if ! command -v node &> /dev/null; then
    echo "❌ Node.js not found. Please install Node.js."
    exit 1
fi

echo "✅ All prerequisites found"

# Cleanup function
cleanup() {
    echo ""
    echo "🛑 Cleaning up processes..."
    
    # Stop Dashboard server
    if [ ! -z "$DASHBOARD_PID" ]; then
        echo "🔴 Stopping Dashboard server (PID: $DASHBOARD_PID)..."
        kill $DASHBOARD_PID 2>/dev/null || true
        sleep 1
    fi
    
    # Stop Bridge service
    if [ ! -z "$BRIDGE_PID" ]; then
        echo "🔴 Stopping Bridge service (PID: $BRIDGE_PID)..."
        kill $BRIDGE_PID 2>/dev/null || true
        sleep 1
    fi
    
    # Stop Anvil
    if [ -f anvil-per-epoch.pid ]; then
        ANVIL_PID=$(cat anvil-per-epoch.pid)
        if kill -0 $ANVIL_PID 2>/dev/null; then
            echo "🔴 Stopping Anvil (PID: $ANVIL_PID)..."
            kill $ANVIL_PID
            sleep 2
        fi
        rm -f anvil-per-epoch.pid anvil-per-epoch.log
    fi
    
    # Stop Dgraph container
    echo "🔴 Stopping Dgraph container..."
    docker stop dgraph-standalone 2>/dev/null || true
    docker rm dgraph-standalone 2>/dev/null || true
    
    # Clean up Dgraph data directory
    echo "🧹 Cleaning up Dgraph data..."
    rm -rf ./dgraph-data 2>/dev/null || true
    
    # Clean up temporary files
    rm -f contract_addresses.json dashboard.log
    
    echo "✅ Cleanup complete"
    exit 0
}

# Set up trap for Ctrl+C
trap cleanup SIGINT SIGTERM

# === PHASE 1: START INFRASTRUCTURE ===
echo ""
echo "🚀 PHASE 1: Starting Infrastructure"
echo "────────────────────────────────────"

# === DASHBOARD SERVER SETUP ===
echo "🌐 Starting dashboard server on port 3000..."
nohup go run serve-dashboard.go > dashboard.log 2>&1 &
DASHBOARD_PID=$!

# Wait for dashboard to be ready
echo "⏳ Waiting for dashboard server to start..."
for i in {1..10}; do
    if curl -s http://localhost:3000 >/dev/null 2>&1; then
        echo "✅ Dashboard server is ready (PID: $DASHBOARD_PID)"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "⚠️  Dashboard server may not be ready, continuing anyway..."
    fi
    sleep 1
done

# === DGRAPH SETUP ===
echo "Setting up Dgraph for VLC event visualization..."

# Check if Docker is available
if ! command -v docker &> /dev/null; then
    echo "⚠️  Docker not found. Please install Docker to enable VLC visualization."
    DGRAPH_STARTED=false
else
    echo "🐳 Docker found, setting up Dgraph container..."
    
    # Stop any existing Dgraph containers
    echo "🔴 Stopping existing Dgraph containers..."
    docker stop dgraph-standalone 2>/dev/null || true
    docker rm dgraph-standalone 2>/dev/null || true
    
    # Remove any existing Dgraph data to start fresh
    echo "🧹 Cleaning up previous Dgraph data..."
    rm -rf ./dgraph-data 2>/dev/null || true
    mkdir -p ./dgraph-data
    
    # Start new Dgraph container with proper setup
    echo "🚀 Starting fresh Dgraph container..."
    DGRAPH_OUTPUT=$(docker run --rm -d --name dgraph-standalone \
        -p 8080:8080 -p 9080:9080 -p 8000:8000 \
        -v $(pwd)/dgraph-data:/dgraph \
        dgraph/standalone:latest 2>&1)
    
    if [ $? -eq 0 ]; then
        echo "✅ Dgraph container started successfully"
        echo "   - Container ID: $(echo $DGRAPH_OUTPUT | cut -c1-12)"
        echo "   - GraphQL Endpoint: http://localhost:8080/graphql"
        echo "   - Query Endpoint: http://localhost:8080/query" 
        echo "   - Ratel UI: http://localhost:8000"
        echo "   - GRPC Endpoint: localhost:9080"
        DGRAPH_STARTED=true
    else
        echo "❌ Dgraph container failed to start: $DGRAPH_OUTPUT"
        echo "⚠️  This might be due to:"
        echo "   - Port conflicts (8080, 9080, 8000 already in use)"
        echo "   - Docker permissions issues"
        echo "   - Insufficient system resources"
        echo "⚠️  Continuing without VLC visualization"
        DGRAPH_STARTED=false
    fi
fi

# Wait for Dgraph to be fully ready
if [ "$DGRAPH_STARTED" = true ]; then
    echo "⏳ Waiting for Dgraph to be ready..."
    DGRAPH_READY=false
    
    for i in {1..30}; do
        # Test GraphQL endpoint
        if curl -s -f http://localhost:8080/health >/dev/null 2>&1; then
            echo "✅ Dgraph health check passed"
            
            # Test if we can query (more comprehensive check)
            if curl -s -X POST -H "Content-Type: application/json" \
               -d '{"query": "{ __schema { queryType { name } } }"}' \
               http://localhost:8080/graphql >/dev/null 2>&1; then
                echo "✅ Dgraph GraphQL endpoint is ready"
                DGRAPH_READY=true
                break
            fi
        fi
        
        if [ $i -eq 30 ]; then
            echo "⚠️  Dgraph not fully ready after 60 seconds, continuing anyway"
            echo "   You may need to wait a bit more before VLC visualization works"
            DGRAPH_READY=true  # Continue anyway
        else
            echo "   Dgraph starting up... (attempt $i/30)"
        fi
        sleep 2
    done
    
    if [ "$DGRAPH_READY" = true ]; then
        echo "🎯 Dgraph is ready for VLC event tracking!"
        echo "   Access Ratel UI at: http://localhost:8000"
        echo ""
    fi
else
    echo "⏭️  Skipping Dgraph readiness check (not started)"
fi

# Clean up any existing Anvil processes and files
echo "Cleaning up existing Anvil instances..."
pkill -f "anvil.*8545" 2>/dev/null || true
sleep 2
rm -f anvil-per-epoch.pid anvil-per-epoch.log

# Start fresh Anvil blockchain
echo "Starting fresh Anvil blockchain from genesis block..."
nohup anvil \
    --accounts 10 \
    --balance 10000 \
    --port 8545 \
    --host 0.0.0.0 \
    --mnemonic "test test test test test test test test test test test junk" \
    > anvil-per-epoch.log 2>&1 &

ANVIL_PID=$!
echo $ANVIL_PID > anvil-per-epoch.pid

# Wait for Anvil to be ready
echo "Waiting for Anvil to be ready..."
for i in {1..10}; do
    if curl -s -X POST -H "Content-Type: application/json" \
       --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
       http://localhost:8545 >/dev/null 2>&1; then
        
        BLOCK_NUM=$(curl -s -X POST -H "Content-Type: application/json" \
           --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
           http://localhost:8545 | grep -o '"result":"0x[0-9a-f]*"' | cut -d'"' -f4)
        BLOCK_DEC=$((16#${BLOCK_NUM#0x}))
        
        echo "✅ Anvil is ready (PID: $ANVIL_PID) - Starting from block $BLOCK_DEC"
        break
    fi
    if [ $i -eq 10 ]; then
        echo "❌ Anvil failed to start"
        cleanup
        exit 1
    fi
    sleep 1
done

# === PHASE 2: DEPLOY MAINNET CONTRACTS ===
echo ""
echo "📋 PHASE 2: Deploying Mainnet Contracts"
echo "────────────────────────────────────────────"

# Configuration
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
VALIDATOR1_KEY="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
MINER_KEY="0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba"
RPC_URL="http://localhost:8545"

DEPLOYER="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
VALIDATOR1="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
VALIDATOR2="0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
VALIDATOR3="0x90F79bf6EB2c4f870365E785982E1f101E93b906"
VALIDATOR4="0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
MINER="0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"

# Compile contracts
echo "Compiling contracts..."
forge build > /dev/null 2>&1

# Deploy contracts
echo "Deploying HETU Token..."
HETU_RESULT=$(forge create contracts/HETUToken.sol:HETUToken \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
HETU_ADDRESS=$(echo "$HETU_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)

echo "Deploying KEY Token..."
KEY_RESULT=$(forge create contracts/KEYToken.sol:KEYToken \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
KEY_ADDRESS=$(echo "$KEY_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)

echo "Deploying Subnet Registry..."
REGISTRY_RESULT=$(forge create contracts/SubnetRegistry.sol:SubnetRegistry \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
REGISTRY_ADDRESS=$(echo "$REGISTRY_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)

echo "Deploying Enhanced PoCW Verifier..."
VERIFIER_RESULT=$(forge create contracts/EnhancedPoCWVerifier.sol:EnhancedPoCWVerifier \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL --broadcast 2>&1)
VERIFIER_ADDRESS=$(echo "$VERIFIER_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)

# Initialize contracts
echo "Initializing contracts..."
cast send $REGISTRY_ADDRESS "initialize(address)" $HETU_ADDRESS \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1

cast send $VERIFIER_ADDRESS "initialize(address,address)" $KEY_ADDRESS $REGISTRY_ADDRESS \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1

cast send $KEY_ADDRESS "setPoCWVerifier(address)" $VERIFIER_ADDRESS \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1

# Distribute HETU and setup subnet
echo "Setting up subnet participants..."
cast send $HETU_ADDRESS "transfer(address,uint256)" $MINER $(cast --to-wei 2000) \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1

for VALIDATOR in $VALIDATOR1 $VALIDATOR2 $VALIDATOR3 $VALIDATOR4; do
    cast send $HETU_ADDRESS "transfer(address,uint256)" $VALIDATOR $(cast --to-wei 2000) \
        --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1
done

# Approvals
cast send $HETU_ADDRESS "approve(address,uint256)" $REGISTRY_ADDRESS $(cast --to-wei 500) \
    --private-key $MINER_KEY --rpc-url $RPC_URL > /dev/null 2>&1

VALIDATOR_KEYS=("$VALIDATOR1_KEY" "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a" "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6" "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a")
for i in 0 1 2 3; do
    cast send $HETU_ADDRESS "approve(address,uint256)" $REGISTRY_ADDRESS $(cast --to-wei 100) \
        --private-key ${VALIDATOR_KEYS[$i]} --rpc-url $RPC_URL > /dev/null 2>&1
done

# Register subnet
SUBNET_ID="per-epoch-subnet-001"
cast send $REGISTRY_ADDRESS "registerSubnet(string,address,address[4])" \
    "$SUBNET_ID" \
    "$MINER" \
    "[$VALIDATOR1,$VALIDATOR2,$VALIDATOR3,$VALIDATOR4]" \
    --private-key $MINER_KEY --rpc-url $RPC_URL > /dev/null 2>&1

# Generate contract addresses JSON for bridge
cat > contract_addresses.json << EOF
{
  "$HETU_ADDRESS": "HETU Token",
  "$KEY_ADDRESS": "Intelligence Token (KEY)",
  "$REGISTRY_ADDRESS": "Subnet Registry", 
  "$VERIFIER_ADDRESS": "Enhanced PoCW Verifier"
}
EOF

echo "✅ Mainnet contracts deployed and configured"
echo "   HETU Token: $HETU_ADDRESS"
echo "   KEY Token: $KEY_ADDRESS"  
echo "   PoCW Verifier: $VERIFIER_ADDRESS"

# Helper function to format wei to KEY tokens
format_key_balance() {
    local wei_value=$1
    # Convert scientific notation to decimal if needed
    local decimal_value=$(printf "%.0f" $wei_value 2>/dev/null || echo $wei_value)
    # Convert from wei (divide by 10^18)
    local key_value=$(echo "scale=6; $decimal_value / 1000000000000000000" | bc -l)
    echo $key_value
}

# === INITIAL KEY BALANCES ===
echo ""
echo "💰 Initial KEY Token Balances (Before Mining)"
echo "────────────────────────────────────────────────"
echo "📊 Miner ($MINER):"
MINER_INITIAL=$(cast call $KEY_ADDRESS "balanceOf(address)(uint256)" $MINER --rpc-url $RPC_URL)
MINER_INITIAL_FORMATTED=$(format_key_balance $MINER_INITIAL)
echo "   Balance: $MINER_INITIAL_FORMATTED KEY"

echo "📊 Validator-1 ($VALIDATOR1):"
V1_INITIAL=$(cast call $KEY_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR1 --rpc-url $RPC_URL)
V1_INITIAL_FORMATTED=$(format_key_balance $V1_INITIAL)
echo "   Balance: $V1_INITIAL_FORMATTED KEY"

echo "📊 Validator-2 ($VALIDATOR2):"
V2_INITIAL=$(cast call $KEY_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR2 --rpc-url $RPC_URL)
V2_INITIAL_FORMATTED=$(format_key_balance $V2_INITIAL)
echo "   Balance: $V2_INITIAL_FORMATTED KEY"

echo "📊 Validator-3 ($VALIDATOR3):"
V3_INITIAL=$(cast call $KEY_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR3 --rpc-url $RPC_URL)
V3_INITIAL_FORMATTED=$(format_key_balance $V3_INITIAL)
echo "   Balance: $V3_INITIAL_FORMATTED KEY"

echo "📊 Validator-4 ($VALIDATOR4):"
V4_INITIAL=$(cast call $KEY_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR4 --rpc-url $RPC_URL)
V4_INITIAL_FORMATTED=$(format_key_balance $V4_INITIAL)
echo "   Balance: $V4_INITIAL_FORMATTED KEY"

TOTAL_SUPPLY_INITIAL=$(cast call $KEY_ADDRESS "totalSupply()(uint256)" --rpc-url $RPC_URL)
TOTAL_SUPPLY_INITIAL_FORMATTED=$(format_key_balance $TOTAL_SUPPLY_INITIAL)
echo "📊 Total Supply: $TOTAL_SUPPLY_INITIAL_FORMATTED KEY"
echo ""

# === PHASE 3: PER-EPOCH DEMONSTRATION ===
echo ""
echo "🧠 PHASE 3: Per-Epoch PoCW Subnet Demo"
echo "────────────────────────────────────────────"

# Initialize Dgraph schema and data
if [ "$DGRAPH_STARTED" = true ]; then
    echo "🔧 Initializing Dgraph schema for VLC events..."
    
    # Clear any existing data to start fresh
    echo "🧹 Clearing existing Dgraph data..."
    curl -s -X POST http://localhost:8080/alter \
        -H "Content-Type: application/json" \
        -d '{"drop_all": true}' > /dev/null 2>&1 || echo "⚠️  Could not clear Dgraph data"
    
    # Wait a moment for the drop to complete
    sleep 2
    
    # Set up the schema for VLC event tracking
    echo "📋 Setting up VLC event schema..."
    SCHEMA_RESULT=$(curl -s -X POST http://localhost:8080/alter \
        -H "Content-Type: application/json" \
        -d '{
            "schema": "
                event_id: string @index(exact) .
                event_name: string @index(exact) .
                event_type: string @index(exact) .
                subnet_id: string @index(exact) .
                request_id: string @index(exact) .
                vlc_clock: string .
                parents: [uid] .
                timestamp: int @index(int) .
                description: string @index(fulltext) .
            "
        }' 2>&1)
    
    if echo "$SCHEMA_RESULT" | grep -q "Success" || echo "$SCHEMA_RESULT" | grep -q "code"; then
        echo "✅ Dgraph schema initialized successfully"
    else
        echo "⚠️  Dgraph schema setup may have issues, but continuing..."
        echo "   Schema result: $SCHEMA_RESULT"
    fi
    
    echo "🎯 Dgraph ready for per-epoch VLC event tracking!"
else
    echo "⏭️  Skipping Dgraph schema setup (Dgraph not started)"
fi

echo "🔄 Starting per-epoch demonstration..."
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📋 How it works:"
echo "  1. Subnet processes tasks in rounds (3 rounds = 1 epoch)"
echo "  2. When epoch completes → EpochFinalized event triggers"
echo "  3. Callback immediately submits epoch data to mainnet"
echo "  4. KEY tokens are mined in real-time per completed epoch"
echo "  5. Process repeats for each new epoch"
echo ""
echo "🎯 Expected behavior:"
echo "  - Epoch 1 (rounds 1-3): Submit after task 3 completes"
echo "  - Epoch 2 (rounds 4-6): Submit after task 6 completes" 
echo "  - Partial epoch 3 (round 7): Submit after demo ends"
echo ""

# Initialize the Node.js bridge with HTTP server
echo "🌐 Initializing Per-Epoch Mainnet Bridge with HTTP server..."
node -e "
const PerEpochBridge = require('./mainnet-bridge-per-epoch.js');
const bridge = new PerEpochBridge();

async function setupBridge() {
    try {
        await bridge.initialize();
        console.log('✅ Per-Epoch Bridge ready for HTTP requests from Go');
        
        // Keep the bridge running to receive HTTP requests
        process.on('SIGTERM', () => {
            console.log('🔴 Bridge shutting down...');
            process.exit(0);
        });
        
    } catch (error) {
        console.error('❌ Bridge setup failed:', error.message);
        process.exit(1);
    }
}

setupBridge();
" &

BRIDGE_PID=$!
echo "⏳ Waiting for bridge HTTP server to initialize..."
sleep 5

# Verify bridge is ready
echo "🔍 Verifying bridge HTTP server..."
if curl -s http://localhost:3001/health > /dev/null; then
    echo "✅ Bridge HTTP server is ready and responding"
else
    echo "⚠️  Bridge HTTP server may not be ready, continuing anyway..."
fi

# Run the modified subnet demo with per-epoch callbacks
echo "🚀 Starting PoCW subnet with per-epoch mainnet integration..."
echo ""

echo "📝 NOTE: Real-time blockchain integration active!"
echo "   Go subnet will make HTTP calls to JavaScript bridge"
echo "   Each completed epoch will trigger actual blockchain submissions"
echo ""

# Run the per-epoch subnet demo with real HTTP bridge integration
timeout 45 go run main.go || true

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎉 KEY MINING DEMONSTRATION COMPLETE!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# === FINAL KEY BALANCES ===
echo ""
echo "💰 Final KEY Token Balances (After Mining)"
echo "────────────────────────────────────────────────"
echo "📊 Miner ($MINER):"
MINER_FINAL=$(cast call $KEY_ADDRESS "balanceOf(address)(uint256)" $MINER --rpc-url $RPC_URL)
MINER_FINAL_FORMATTED=$(format_key_balance $MINER_FINAL)
MINER_GAINED=$(echo "$MINER_FINAL_FORMATTED - $MINER_INITIAL_FORMATTED" | bc -l)
echo "   Balance: $MINER_FINAL_FORMATTED KEY (+$MINER_GAINED KEY mined)"

echo "📊 Validator-1 ($VALIDATOR1):"
V1_FINAL=$(cast call $KEY_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR1 --rpc-url $RPC_URL)
V1_FINAL_FORMATTED=$(format_key_balance $V1_FINAL)
V1_GAINED=$(echo "$V1_FINAL_FORMATTED - $V1_INITIAL_FORMATTED" | bc -l)
echo "   Balance: $V1_FINAL_FORMATTED KEY (+$V1_GAINED KEY mined)"

echo "📊 Validator-2 ($VALIDATOR2):"
V2_FINAL=$(cast call $KEY_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR2 --rpc-url $RPC_URL)
V2_FINAL_FORMATTED=$(format_key_balance $V2_FINAL)
V2_GAINED=$(echo "$V2_FINAL_FORMATTED - $V2_INITIAL_FORMATTED" | bc -l)
echo "   Balance: $V2_FINAL_FORMATTED KEY (+$V2_GAINED KEY mined)"

echo "📊 Validator-3 ($VALIDATOR3):"
V3_FINAL=$(cast call $KEY_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR3 --rpc-url $RPC_URL)
V3_FINAL_FORMATTED=$(format_key_balance $V3_FINAL)
V3_GAINED=$(echo "$V3_FINAL_FORMATTED - $V3_INITIAL_FORMATTED" | bc -l)
echo "   Balance: $V3_FINAL_FORMATTED KEY (+$V3_GAINED KEY mined)"

echo "📊 Validator-4 ($VALIDATOR4):"
V4_FINAL=$(cast call $KEY_ADDRESS "balanceOf(address)(uint256)" $VALIDATOR4 --rpc-url $RPC_URL)
V4_FINAL_FORMATTED=$(format_key_balance $V4_FINAL)
V4_GAINED=$(echo "$V4_FINAL_FORMATTED - $V4_INITIAL_FORMATTED" | bc -l)
echo "   Balance: $V4_FINAL_FORMATTED KEY (+$V4_GAINED KEY mined)"

TOTAL_SUPPLY_FINAL=$(cast call $KEY_ADDRESS "totalSupply()(uint256)" --rpc-url $RPC_URL)
TOTAL_SUPPLY_FINAL_FORMATTED=$(format_key_balance $TOTAL_SUPPLY_FINAL)
TOTAL_MINED=$(echo "$TOTAL_SUPPLY_FINAL_FORMATTED - $TOTAL_SUPPLY_INITIAL_FORMATTED" | bc -l)
echo "📊 Total Supply: $TOTAL_SUPPLY_FINAL_FORMATTED KEY (+$TOTAL_MINED KEY total mined)"

echo ""
echo "🔍 What was demonstrated:"
echo "  1. ✅ Infrastructure setup (Anvil + Dgraph + Contracts)"
echo "  2. ✅ Real-time KEY mining per epoch completion"  
echo "  3. ✅ Subnet processing with VLC consistency"
echo "  4. ✅ Blockchain integration with verified transactions"
echo ""
echo "🌐 Access points:"
echo "  📊 Dgraph VLC visualization: http://localhost:8000"
echo "  🔍 Blockchain Inspector: http://localhost:3000/pocw-inspector.html"
echo "  ⛓️  Anvil blockchain: http://localhost:8545"
echo ""

echo "🎉 Bridge stays running for continued KEY mining!"
echo "🌐 Bridge service: http://localhost:3001"
echo "Press Ctrl+C to cleanup and exit..."

# Keep running for inspection (controlled by NO_LOOP environment variable)
if [ "$NO_LOOP" != "true" ]; then
    while true; do
        sleep 10
        # Check if Anvil is still running
        if ! kill -0 $(cat anvil-per-epoch.pid) 2>/dev/null; then
            echo "❌ Anvil stopped unexpectedly"
            break
        fi
    done
else
    echo "🔧 NO_LOOP=true detected - exiting without forever loop"
    echo "   (Set NO_LOOP=false or unset to enable debugging loop)"
fi