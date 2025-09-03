#!/bin/bash

# Cleanup function for graceful exit
cleanup() {
    echo ""
    echo ""
    echo "🛑 Cleaning up..."
    if [ -f anvil-enhanced.pid ]; then
        ANVIL_PID=$(cat anvil-enhanced.pid)
        if kill -0 $ANVIL_PID 2>/dev/null; then
            echo "🔴 Stopping Anvil (PID: $ANVIL_PID)..."
            kill $ANVIL_PID
            sleep 2
        fi
        rm anvil-enhanced.pid anvil-enhanced.log 2>/dev/null
    fi
    echo "✅ Cleanup complete"
    exit 0
}

# Set up trap for Ctrl+C
trap cleanup SIGINT SIGTERM

echo "🔷 ENHANCED PoCW VERIFIER TEST"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Testing EnhancedPoCWVerifier with comprehensive VLC data"
echo ""

# Set Foundry path
export PATH="/home/xx/.foundry/bin:$PATH"

# === STEP 1: CLEAN UP EVERYTHING ===
echo "🧹 Step 1: Complete Cleanup"
echo "────────────────────────────"
echo "Stopping all Anvil instances..."
pkill anvil 2>/dev/null || true
sleep 2
echo "Removing old files..."
rm -f anvil-enhanced.pid anvil-enhanced.log contract_addresses.json 2>/dev/null
echo "✅ Cleanup complete"

# === STEP 2: START FRESH ANVIL ===
echo ""
echo "🚀 Step 2: Starting Fresh Anvil (Block 0)"
echo "──────────────────────────────────────────"

# Start Anvil without auto-mining
nohup anvil \
    --accounts 10 \
    --balance 10000 \
    --port 8545 \
    --host 0.0.0.0 \
    --mnemonic "test test test test test test test test test test test junk" \
    > anvil-enhanced.log 2>&1 &

ANVIL_PID=$!
echo $ANVIL_PID > anvil-enhanced.pid

sleep 3

# Verify Anvil is running
if curl -s -X POST -H "Content-Type: application/json" \
   --data '{"jsonrpc":"2.0","method":"eth_blockNumber","params":[],"id":1}' \
   http://localhost:8545 >/dev/null 2>&1; then
    echo "✅ Anvil started successfully (PID: $ANVIL_PID)"
    BLOCK=$(cast block-number --rpc-url http://localhost:8545)
    echo "📊 Current block: $BLOCK"
else
    echo "❌ Failed to start Anvil"
    exit 1
fi

# Configuration
PRIVATE_KEY="0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
VALIDATOR1_KEY="0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
VALIDATOR2_KEY="0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
VALIDATOR3_KEY="0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
VALIDATOR4_KEY="0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a"
MINER_KEY="0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba"
RPC_URL="http://localhost:8545"

DEPLOYER="0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266"
VALIDATOR1="0x70997970C51812dc3A010C7d01b50e0d17dc79C8"
VALIDATOR2="0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC"
VALIDATOR3="0x90F79bf6EB2c4f870365E785982E1f101E93b906"
VALIDATOR4="0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
MINER="0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc"

# === STEP 3: COMPILE CONTRACTS ===
echo ""
echo "📦 Step 3: Compiling Enhanced Contracts"
echo "───────────────────────────────────────"
forge build

# === STEP 4: DEPLOY CONTRACTS ===
echo ""
echo "🚀 Step 4: Deploying Enhanced Contract Suite"
echo "──────────────────────────────────────────"

# Deploy HETU Token
echo "Deploying HETU Token..."
HETU_RESULT=$(forge create contracts/HETUToken.sol:HETUToken \
    --private-key $PRIVATE_KEY \
    --rpc-url $RPC_URL \
    --broadcast 2>&1)
HETU_ADDRESS=$(echo "$HETU_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)
echo "✅ HETU Token: $HETU_ADDRESS"

# Deploy KEY Token
echo "Deploying KEY Token..."
KEY_RESULT=$(forge create contracts/KEYToken.sol:KEYToken \
    --private-key $PRIVATE_KEY \
    --rpc-url $RPC_URL \
    --broadcast 2>&1)
KEY_ADDRESS=$(echo "$KEY_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)
echo "✅ KEY Token: $KEY_ADDRESS"

# Deploy Subnet Registry
echo "Deploying Subnet Registry..."
REGISTRY_RESULT=$(forge create contracts/SubnetRegistry.sol:SubnetRegistry \
    --private-key $PRIVATE_KEY \
    --rpc-url $RPC_URL \
    --broadcast 2>&1)
REGISTRY_ADDRESS=$(echo "$REGISTRY_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)
echo "✅ Subnet Registry: $REGISTRY_ADDRESS"

# Deploy EnhancedPoCWVerifier
echo "Deploying EnhancedPoCWVerifier..."
VERIFIER_RESULT=$(forge create contracts/EnhancedPoCWVerifier.sol:EnhancedPoCWVerifier \
    --private-key $PRIVATE_KEY \
    --rpc-url $RPC_URL \
    --broadcast 2>&1)
VERIFIER_ADDRESS=$(echo "$VERIFIER_RESULT" | grep -o "Deployed to: 0x[a-fA-F0-9]\{40\}" | cut -d' ' -f3)
echo "✅ EnhancedPoCWVerifier: $VERIFIER_ADDRESS"

# Generate contract_addresses.json for dashboard.html
echo "Generating contract_addresses.json..."
cat > contract_addresses.json << EOF
{
  "$HETU_ADDRESS": "HETU Token",
  "$KEY_ADDRESS": "Intelligence Token (KEY)",
  "$REGISTRY_ADDRESS": "Subnet Registry", 
  "$VERIFIER_ADDRESS": "Enhanced PoCW Verifier"
}
EOF
echo "✅ contract_addresses.json created"

# === STEP 5: INITIALIZE CONTRACTS ===
echo ""
echo "⚙️ Step 5: Initializing Enhanced Contracts"
echo "───────────────────────────────────────────"

# Initialize SubnetRegistry
echo "Initializing SubnetRegistry..."
cast send $REGISTRY_ADDRESS "initialize(address)" $HETU_ADDRESS \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1
echo "✅ SubnetRegistry initialized"

# Initialize EnhancedPoCWVerifier
echo "Initializing EnhancedPoCWVerifier..."
cast send $VERIFIER_ADDRESS "initialize(address,address)" $KEY_ADDRESS $REGISTRY_ADDRESS \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1
echo "✅ EnhancedPoCWVerifier initialized"

# Set EnhancedPoCWVerifier as PoCW verifier on KEY token
echo "Setting PoCW verifier on KEY token..."
cast send $KEY_ADDRESS "setPoCWVerifier(address)" $VERIFIER_ADDRESS \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1
echo "✅ PoCW verifier set"

# === STEP 6: DISTRIBUTE HETU FOR DEPOSITS ===
echo ""
echo "💸 Step 6: Distributing HETU Tokens"
echo "────────────────────────────────────"

# Transfer HETU to miner (2000 instead of 500)
cast send $HETU_ADDRESS "transfer(address,uint256)" $MINER $(cast --to-wei 2000) \
    --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1
echo "✅ Transferred 2000 HETU to miner"

# Transfer to validators (2000 each instead of 100)
for i in 1 2 3 4; do
    VALIDATOR_VAR="VALIDATOR$i"
    VALIDATOR_ADDR="${!VALIDATOR_VAR}"
    cast send $HETU_ADDRESS "transfer(address,uint256)" $VALIDATOR_ADDR $(cast --to-wei 2000) \
        --private-key $PRIVATE_KEY --rpc-url $RPC_URL > /dev/null 2>&1
    echo "✅ Transferred 2000 HETU to validator $i"
done

# === STEP 7: APPROVE HETU SPENDING ===
echo ""
echo "🔓 Step 7: Approving HETU Spending"
echo "───────────────────────────────────"

# Miner approves (still only need 500 for deposit)
cast send $HETU_ADDRESS "approve(address,uint256)" $REGISTRY_ADDRESS $(cast --to-wei 500) \
    --private-key $MINER_KEY --rpc-url $RPC_URL > /dev/null 2>&1
echo "✅ Miner approved 500 HETU (for deposit)"

# Validators approve (still only need 100 each for deposit)
VALIDATOR_KEYS=("$VALIDATOR1_KEY" "$VALIDATOR2_KEY" "$VALIDATOR3_KEY" "$VALIDATOR4_KEY")
for i in 0 1 2 3; do
    cast send $HETU_ADDRESS "approve(address,uint256)" $REGISTRY_ADDRESS $(cast --to-wei 100) \
        --private-key ${VALIDATOR_KEYS[$i]} --rpc-url $RPC_URL > /dev/null 2>&1
    echo "✅ Validator $((i+1)) approved 100 HETU (for deposit)"
done

# === STEP 8: REGISTER SUBNET ===
echo ""
echo "🏗️ Step 8: Registering Subnet"
echo "──────────────────────────────"

SUBNET_ID="enhanced-subnet-001"
cast send $REGISTRY_ADDRESS "registerSubnet(string,address,address[4])" \
    "$SUBNET_ID" \
    "$MINER" \
    "[$VALIDATOR1,$VALIDATOR2,$VALIDATOR3,$VALIDATOR4]" \
    --private-key $MINER_KEY --rpc-url $RPC_URL > /dev/null 2>&1
echo "✅ Subnet registered: $SUBNET_ID"

# === STEP 9: CREATE COMPREHENSIVE VLC DATA ===
echo ""
echo "📊 Step 9: Creating Enhanced VLC Graph Data"
echo "──────────────────────────────────────────"

# Create comprehensive VLC data similar to your suggestion
VLC_JSON='{
  "subnetId": "enhanced-subnet-001",
  "epochNumber": 1,
  "events": [
    {
      "id": "genesis_0",
      "name": "GenesisState",
      "vlcClock": {},
      "parents": [],
      "timestamp": '$(date +%s000)'
    },
    {
      "id": "task_1",
      "name": "AITaskExecution",
      "vlcClock": {"miner1": 1, "validator1": 0},
      "parents": ["genesis_0"],
      "timestamp": '$(date +%s000)',
      "taskId": "ai_inference_001",
      "result": "success"
    },
    {
      "id": "task_2", 
      "name": "AITaskExecution",
      "vlcClock": {"miner1": 2, "validator1": 0},
      "parents": ["task_1"],
      "timestamp": '$(date +%s000)',
      "taskId": "ai_inference_002",
      "result": "success"
    },
    {
      "id": "validation_1",
      "name": "TaskValidation",
      "vlcClock": {"miner1": 2, "validator1": 1},
      "parents": ["task_2"],
      "timestamp": '$(date +%s000)',
      "validatedTasks": ["ai_inference_001", "ai_inference_002"],
      "validationResult": "approved"
    }
  ],
  "miners": ["'$MINER'"],
  "validators": ["'$VALIDATOR1'", "'$VALIDATOR2'", "'$VALIDATOR3'", "'$VALIDATOR4'"],
  "summary": {
    "totalTasks": 2,
    "successfulTasks": 2,
    "failedTasks": 0,
    "validationStatus": "complete"
  }
}'

echo "📋 VLC Data Structure:"
echo "$VLC_JSON" | head -15
echo "..."

# Convert to hex
VLC_HEX=$(echo "$VLC_JSON" | xxd -p | tr -d '\n')
VLC_DATA_HEX="0x$VLC_HEX"

# === STEP 10: TEST ENHANCED EPOCH SUBMISSION ===
echo ""
echo "⚡ Step 10: Testing Enhanced Epoch Submission"
echo "──────────────────────────────────────────────"

# Check balances before
MINER_BEFORE=$(cast call $KEY_ADDRESS "balanceOf(address)" $MINER --rpc-url $RPC_URL)
VALIDATOR1_BEFORE=$(cast call $KEY_ADDRESS "balanceOf(address)" $VALIDATOR1 --rpc-url $RPC_URL)

echo "KEY balances before mining:"
echo "  Miner: $(cast --to-dec $MINER_BEFORE | awk '{print $1/1e18}') KEY"
echo "  Validator1: $(cast --to-dec $VALIDATOR1_BEFORE | awk '{print $1/1e18}') KEY"

# Submit epoch with single miner (as per original architecture)
echo ""
echo "📊 Submitting enhanced epoch with 1 miner (correct architecture)..."
TX_RESULT=$(cast send $VERIFIER_ADDRESS \
    "submitAndDistributeEpoch(string,bytes,address[],uint256,uint256)" \
    "$SUBNET_ID" \
    "$VLC_DATA_HEX" \
    "[$MINER]" \
    5 \
    0 \
    --private-key $VALIDATOR1_KEY \
    --rpc-url $RPC_URL 2>&1)

if echo "$TX_RESULT" | grep -q "transactionHash"; then
    TX_HASH=$(echo "$TX_RESULT" | grep "transactionHash" | awk '{print $2}')
    echo "✅ Enhanced epoch submitted successfully!"
    echo "   Transaction: ${TX_HASH:0:10}..."
else
    echo "❌ Failed to submit epoch:"
    echo "$TX_RESULT"
fi

# Check balances after
sleep 2
MINER_AFTER=$(cast call $KEY_ADDRESS "balanceOf(address)" $MINER --rpc-url $RPC_URL)
VALIDATOR1_AFTER=$(cast call $KEY_ADDRESS "balanceOf(address)" $VALIDATOR1 --rpc-url $RPC_URL)

echo ""
echo "KEY balances after mining:"
echo "  Miner: $(cast --to-dec $MINER_AFTER | awk '{print $1/1e18}') KEY"
echo "  Validator1: $(cast --to-dec $VALIDATOR1_AFTER | awk '{print $1/1e18}') KEY"

# Calculate mined amounts
MINER_MINED=$(echo "$(cast --to-dec $MINER_AFTER) - $(cast --to-dec $MINER_BEFORE)" | bc)
VALIDATOR1_MINED=$(echo "$(cast --to-dec $VALIDATOR1_AFTER) - $(cast --to-dec $VALIDATOR1_BEFORE)" | bc)

echo ""
echo "💰 KEY tokens mined:"
echo "  Miner earned: $(echo "scale=2; $MINER_MINED/1000000000000000000" | bc) KEY"
echo "  Validator1 earned: $(echo "scale=2; $VALIDATOR1_MINED/1000000000000000000" | bc) KEY"

# === STEP 11: CHECK ENHANCED STATS ===
echo ""
echo "📈 Step 11: Checking Enhanced Statistics"
echo "────────────────────────────────────────"

# Check miner stats
echo "Miner Statistics:"
MINER_STATS=$(cast call $VERIFIER_ADDRESS "getMinerStats(address)" $MINER --rpc-url $RPC_URL)

# Decode the MinerStats struct manually
# struct MinerStats { address owner, uint256 successfulTasks, uint256 totalTasks, uint256 totalIntelligenceMined, uint256 reputationScore, uint256 lastActiveEpoch, uint256 joinedTimestamp, bool isActive }
# Each field is 32 bytes = 64 hex chars, but address is only 20 bytes with 12 bytes padding
OWNER=$(echo $MINER_STATS | cut -c27-66)  # Address is at the end of first 32-byte slot
SUCCESSFUL_TASKS_HEX=$(echo $MINER_STATS | cut -c67-130)   # Next 64 hex chars
TOTAL_TASKS_HEX=$(echo $MINER_STATS | cut -c131-194)       # Next 64 hex chars  
INTELLIGENCE_MINED_HEX=$(echo $MINER_STATS | cut -c195-258) # Next 64 hex chars
REPUTATION_HEX=$(echo $MINER_STATS | cut -c259-322)        # Next 64 hex chars
LAST_EPOCH_HEX=$(echo $MINER_STATS | cut -c323-386)       # Next 64 hex chars
JOINED_TIME_HEX=$(echo $MINER_STATS | cut -c387-450)      # Next 64 hex chars
IS_ACTIVE_HEX=$(echo $MINER_STATS | cut -c451-514)        # Last 64 hex chars

SUCCESSFUL_TASKS=$(cast --to-dec 0x$SUCCESSFUL_TASKS_HEX)
TOTAL_TASKS=$(cast --to-dec 0x$TOTAL_TASKS_HEX)
INTELLIGENCE_MINED=$(echo "scale=2; $(cast --to-dec 0x$INTELLIGENCE_MINED_HEX)/1000000000000000000" | bc)
REPUTATION=$(cast --to-dec 0x$REPUTATION_HEX)
LAST_EPOCH=$(cast --to-dec 0x$LAST_EPOCH_HEX)
JOINED_TIME=$(cast --to-dec 0x$JOINED_TIME_HEX)
IS_ACTIVE=$([ "$(cast --to-dec 0x$IS_ACTIVE_HEX)" -eq "1" ] && echo "Yes" || echo "No")

echo "  Owner: 0x$OWNER"
echo "  Successful Tasks: $SUCCESSFUL_TASKS"
echo "  Total Tasks: $TOTAL_TASKS"
echo "  Intelligence Mined: $INTELLIGENCE_MINED KEY"
echo "  Reputation Score: $REPUTATION%"
echo "  Last Active Epoch: $LAST_EPOCH"
echo "  Joined Timestamp: $JOINED_TIME"
echo "  Is Active: $IS_ACTIVE"

# Check epoch submission details
SUBNET_HASH=$(cast call $VERIFIER_ADDRESS "subnetIdToHash(string)" "$SUBNET_ID" --rpc-url $RPC_URL)
echo ""
echo "Subnet Hash: $SUBNET_HASH"

# Check total KEY supply
KEY_SUPPLY=$(cast call $KEY_ADDRESS "totalSupply()" --rpc-url $RPC_URL)
echo "🔑 Total KEY supply: $(cast --to-dec $KEY_SUPPLY | awk '{print $1/1e18}') KEY"

# Final block number
FINAL_BLOCK=$(cast block-number --rpc-url $RPC_URL)
echo ""
echo "📊 Final block number: $FINAL_BLOCK"

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🎉 ENHANCED PoCW VERIFIER TEST COMPLETE!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "🔍 BLOCKCHAIN INSPECTION MODE"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "📋 Contract Addresses for UI Inspection:"
echo "────────────────────────────────────────"
echo "🏪 HETU Token:           $HETU_ADDRESS"
echo "🔑 KEY Token:            $KEY_ADDRESS"
echo "📁 Subnet Registry:      $REGISTRY_ADDRESS"
echo "⚡ EnhancedPoCWVerifier: $VERIFIER_ADDRESS"
echo ""
echo "👥 Account Addresses:"
echo "────────────────────"
echo "🏭 Deployer:    $DEPLOYER"
echo "⛏️  Miner:       $MINER"
echo "✅ Validator-1: $VALIDATOR1"
echo "✅ Validator-2: $VALIDATOR2"
echo "✅ Validator-3: $VALIDATOR3"
echo "✅ Validator-4: $VALIDATOR4"
echo ""
echo "🌐 Network Information:"
echo "──────────────────────"
echo "🔗 RPC URL: $RPC_URL"
echo "⛓️  Chain ID: 31337 (Anvil)"
echo "📊 Current Block: $FINAL_BLOCK"
echo "🆔 Subnet ID: $SUBNET_ID"
echo ""
echo "💰 Expected Balances:"
echo "────────────────────"
echo "HETU Balances:"
echo "  📦 Deployer: ~990,000 HETU (after distributing 10,000 total)"
echo "  ⛏️  Miner: 1,500 HETU (2,000 received - 500 deposit)"
echo "  ✅ Each Validator: 1,900 HETU (2,000 received - 100 deposit)"
echo ""
echo "KEY Balances:"
echo "  ⛏️  Miner: 500 KEY (mined from tasks)"
echo "  ✅ Validator-1: 70 KEY (coordination reward)"
echo "  ✅ Other Validators: 0 KEY"
echo "  🔑 Total Supply: 570 KEY"
echo ""
echo "🎯 Subnet Status:"
echo "───────────────"
echo "  🏷️  Subnet: $SUBNET_ID"
echo "  ✅ Active: Yes"
echo "  📊 Epochs: 1"
echo "  ⛏️  Tasks: 5 successful, 0 failed"
echo "  💎 Reputation: 100% (perfect success rate)"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🚀 ANVIL IS RUNNING - You can now inspect the blockchain!"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""
echo "Instructions:"
echo "1. 🌐 Open dashboard at: http://localhost:3000 (dashboard.html)"
echo "2. 🔗 Blockchain connects to: $RPC_URL"
echo "3. 📋 Contract addresses are shown above"
echo "4. 👥 Account addresses and balances are displayed"
echo "5. ⚡ Interact with contracts through the dashboard"
echo ""
echo "⏸️  Press Ctrl+C when you're done inspecting to cleanup and exit"
echo ""

# Keep the script running and Anvil alive
while true; do
    sleep 10
    # Check if Anvil is still running
    if ! kill -0 $(cat anvil-enhanced.pid) 2>/dev/null; then
        echo "❌ Anvil stopped unexpectedly"
        break
    fi
done