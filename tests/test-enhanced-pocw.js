// Enhanced PoCW Verifier Test - JavaScript Version
const { ethers } = require("ethers");

// Contract addresses (will be updated dynamically or from deployment)
const CONTRACTS = {
    HETU: "0x5FbDB2315678afecb367f032d93F642f64180aa3",
    KEY: "0xe7f1725E7734CE288F8367e1Bb143E90bb3F0512", 
    Registry: "0x9fE46736679d2D9a65F0992F2272dE9f3c7fa6e0",
    EnhancedVerifier: "0xCf7Ed3AccA5a467e9e704C703E8D87F634fB0Fc9"
};

// Anvil test accounts with proper key mapping
const ACCOUNTS = {
    deployer: {
        address: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
        privateKey: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
    },
    validator1: {
        address: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8",
        privateKey: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
    },
    validator2: {
        address: "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC",
        privateKey: "0x5de4111afa1a4b94908f83103eb1f1706367c2e68ca870fc3fb9a804cdab365a"
    },
    validator3: {
        address: "0x90F79bf6EB2c4f870365E785982E1f101E93b906",
        privateKey: "0x7c852118294e51e653712a81e05800f419141751be58f605c371e15141b007a6"
    },
    validator4: {
        address: "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65",
        privateKey: "0x47e179ec197488593b187f80a00eb0da91f1b9d0b13f8733639f19c30a34926a"
    },
    miner: {
        address: "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc",
        privateKey: "0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba"
    }
};

async function main() {
    console.log("🔷 ENHANCED PoCW VERIFIER TEST (JavaScript)");
    console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
    console.log("Testing EnhancedPoCWVerifier with comprehensive VLC data");
    console.log("");
    
    // Connect to local Anvil
    const provider = new ethers.JsonRpcProvider("http://localhost:8545");
    
    // Setup signers
    const deployerSigner = new ethers.Wallet(ACCOUNTS.deployer.privateKey, provider);
    const validator1Signer = new ethers.Wallet(ACCOUNTS.validator1.privateKey, provider);
    const minerSigner = new ethers.Wallet(ACCOUNTS.miner.privateKey, provider);
    
    // Contract ABIs for enhanced contracts
    const hetuABI = [
        "function balanceOf(address) view returns (uint256)",
        "function totalSupply() view returns (uint256)",
        "function transfer(address to, uint256 amount) returns (bool)",
        "function approve(address spender, uint256 amount) returns (bool)"
    ];
    
    const keyABI = [
        "function balanceOf(address) view returns (uint256)",
        "function totalSupply() view returns (uint256)",
        "function MAX_SUPPLY() view returns (uint256)"
    ];
    
    const registryABI = [
        "function MINER_DEPOSIT() view returns (uint256)",
        "function VALIDATOR_DEPOSIT() view returns (uint256)",
        "function registerSubnet(string memory subnetId, address miner, address[4] memory validators)",
        "function getSubnet(string memory subnetId) view returns (address, address[4] memory, bool)",
        "function isSubnetActive(string memory subnetId) view returns (bool)"
    ];
    
    const enhancedVerifierABI = [
        "function MINER_REWARD_PER_TASK() view returns (uint256)",
        "function VALIDATOR_BASE_REWARD() view returns (uint256)",
        "function VALIDATOR_PERCENTAGE() view returns (uint256)",
        "function submitAndDistributeEpoch(string subnetId, bytes vlcGraphData, address[] successfulMiners, uint256 successfulTasks, uint256 failedTasks)",
        "function getMinerStats(address miner) view returns (tuple(address owner, uint256 successfulTasks, uint256 totalTasks, uint256 totalIntelligenceMined, uint256 reputationScore, uint256 lastActiveEpoch, uint256 joinedTimestamp, bool isActive))",
        "function getSubnetInfoByString(string subnetId) view returns (tuple(uint256 epochCount, uint256 totalTasksCompleted, uint256 totalRewardsDistributed, uint256 lastEpochTimestamp, bool isActive))",
        "function getEpochCount(string subnetId) view returns (uint256)"
    ];
    
    // Connect to contracts
    const hetuToken = new ethers.Contract(CONTRACTS.HETU, hetuABI, deployerSigner);
    const keyToken = new ethers.Contract(CONTRACTS.KEY, keyABI, deployerSigner);
    const registry = new ethers.Contract(CONTRACTS.Registry, registryABI, deployerSigner);
    const enhancedVerifier = new ethers.Contract(CONTRACTS.EnhancedVerifier, enhancedVerifierABI, validator1Signer);
    
    console.log("📊 Initial State:");
    console.log("────────────────");
    
    // Check initial balances and constants
    const deployerHetuBalance = await hetuToken.balanceOf(ACCOUNTS.deployer.address);
    console.log(`Deployer HETU: ${ethers.formatEther(deployerHetuBalance)} HETU`);
    
    const keyTotalSupply = await keyToken.totalSupply();
    console.log(`KEY Total Supply: ${ethers.formatEther(keyTotalSupply)} KEY`);
    
    const minerDeposit = await registry.MINER_DEPOSIT();
    const validatorDeposit = await registry.VALIDATOR_DEPOSIT();
    console.log(`Required deposits - Miner: ${ethers.formatEther(minerDeposit)} HETU, Validator: ${ethers.formatEther(validatorDeposit)} HETU`);
    
    const minerRewardPerTask = await enhancedVerifier.MINER_REWARD_PER_TASK();
    const validatorBaseReward = await enhancedVerifier.VALIDATOR_BASE_REWARD();
    const validatorPercentage = await enhancedVerifier.VALIDATOR_PERCENTAGE();
    console.log(`Enhanced rewards - Miner: ${ethers.formatEther(minerRewardPerTask)} KEY/task, Validator: ${ethers.formatEther(validatorBaseReward)} KEY base + ${validatorPercentage}%`);
    
    // STEP 1: Transfer HETU to participants
    console.log("\\n💸 Step 1: Distributing HETU for deposits");
    console.log("────────────────────────────────────");
    
    // Transfer HETU to miner
    let tx = await hetuToken.transfer(ACCOUNTS.miner.address, minerDeposit);
    let receipt = await tx.wait();
    console.log(`✅ Transferred ${ethers.formatEther(minerDeposit)} HETU to miner (gas: ${receipt.gasUsed})`);
    
    // Transfer HETU to validators
    for (let i = 1; i <= 4; i++) {
        const validatorKey = `validator${i}`;
        const validatorAccount = ACCOUNTS[validatorKey];
        tx = await hetuToken.transfer(validatorAccount.address, validatorDeposit);
        receipt = await tx.wait();
        console.log(`✅ Transferred ${ethers.formatEther(validatorDeposit)} HETU to validator ${i} (gas: ${receipt.gasUsed})`);
    }
    
    // STEP 2: Approve HETU spending for registry
    console.log("\\n🔓 Step 2: Approving HETU spending");
    console.log("───────────────────────────────────");
    
    // Miner approves
    const minerHetuToken = hetuToken.connect(minerSigner);
    tx = await minerHetuToken.approve(CONTRACTS.Registry, minerDeposit);
    receipt = await tx.wait();
    console.log(`✅ Miner approved ${ethers.formatEther(minerDeposit)} HETU (gas: ${receipt.gasUsed})`);
    
    // Validators approve
    for (let i = 1; i <= 4; i++) {
        const validatorKey = `validator${i}`;
        const validatorSigner = new ethers.Wallet(ACCOUNTS[validatorKey].privateKey, provider);
        const validatorHetuToken = hetuToken.connect(validatorSigner);
        tx = await validatorHetuToken.approve(CONTRACTS.Registry, validatorDeposit);
        receipt = await tx.wait();
        console.log(`✅ Validator ${i} approved ${ethers.formatEther(validatorDeposit)} HETU (gas: ${receipt.gasUsed})`);
    }
    
    // STEP 3: Register subnet
    console.log("\\n🏗️ Step 3: Registering Subnet");
    console.log("──────────────────────────────");
    
    const subnetId = "enhanced-subnet-001";
    const validators = [
        ACCOUNTS.validator1.address,
        ACCOUNTS.validator2.address,
        ACCOUNTS.validator3.address,
        ACCOUNTS.validator4.address
    ];
    
    // Miner registers the subnet
    const minerRegistry = registry.connect(minerSigner);
    tx = await minerRegistry.registerSubnet(subnetId, ACCOUNTS.miner.address, validators);
    receipt = await tx.wait();
    console.log(`✅ Subnet registered: ${subnetId} (gas: ${receipt.gasUsed})`);
    
    // Verify registration
    const isActive = await registry.isSubnetActive(subnetId);
    console.log(`✅ Subnet active: ${isActive}`);
    
    // STEP 4: Create comprehensive VLC data
    console.log("\\n📊 Step 4: Creating Enhanced VLC Graph Data");
    console.log("──────────────────────────────────────────");
    
    const timestamp = Date.now();
    const vlcGraphData = {
        subnetId: subnetId,
        epochNumber: 1,
        events: [
            {
                id: "genesis_0",
                name: "GenesisState",
                vlcClock: {},
                parents: [],
                timestamp: timestamp
            },
            {
                id: "task_1",
                name: "AITaskExecution",
                vlcClock: { "miner1": 1, "validator1": 0 },
                parents: ["genesis_0"],
                timestamp: timestamp + 1000,
                taskId: "ai_inference_001",
                result: "success"
            },
            {
                id: "task_2",
                name: "AITaskExecution", 
                vlcClock: { "miner1": 2, "validator1": 0 },
                parents: ["task_1"],
                timestamp: timestamp + 2000,
                taskId: "ai_inference_002",
                result: "success"
            },
            {
                id: "task_3",
                name: "AITaskExecution",
                vlcClock: { "miner1": 3, "validator1": 0 },
                parents: ["task_2"], 
                timestamp: timestamp + 3000,
                taskId: "ai_inference_003",
                result: "success"
            },
            {
                id: "task_4",
                name: "AITaskExecution",
                vlcClock: { "miner1": 4, "validator1": 0 },
                parents: ["task_3"],
                timestamp: timestamp + 4000,
                taskId: "ai_inference_004", 
                result: "success"
            },
            {
                id: "task_5",
                name: "AITaskExecution",
                vlcClock: { "miner1": 5, "validator1": 0 },
                parents: ["task_4"],
                timestamp: timestamp + 5000,
                taskId: "ai_inference_005",
                result: "success"
            },
            {
                id: "validation_1",
                name: "TaskValidation",
                vlcClock: { "miner1": 5, "validator1": 1 },
                parents: ["task_5"],
                timestamp: timestamp + 6000,
                validatedTasks: ["ai_inference_001", "ai_inference_002", "ai_inference_003", "ai_inference_004", "ai_inference_005"],
                validationResult: "approved"
            }
        ],
        miners: [ACCOUNTS.miner.address],
        validators: [ACCOUNTS.validator1.address, ACCOUNTS.validator2.address, ACCOUNTS.validator3.address, ACCOUNTS.validator4.address],
        summary: {
            totalTasks: 5,
            successfulTasks: 5,
            failedTasks: 0,
            validationStatus: "complete"
        }
    };
    
    console.log(`📋 VLC Data Structure (${vlcGraphData.events.length} events):`);
    console.log(`   Subnet: ${vlcGraphData.subnetId}`);
    console.log(`   Epoch: ${vlcGraphData.epochNumber}`); 
    console.log(`   Tasks: ${vlcGraphData.summary.successfulTasks} successful, ${vlcGraphData.summary.failedTasks} failed`);
    console.log(`   Participants: ${vlcGraphData.miners.length} miner, ${vlcGraphData.validators.length} validators`);
    
    // Convert to bytes for contract call
    const vlcGraphDataBytes = ethers.toUtf8Bytes(JSON.stringify(vlcGraphData));
    console.log(`📦 VLC data size: ${vlcGraphDataBytes.length} bytes`);
    
    // STEP 5: Submit enhanced epoch
    console.log("\\n⚡ Step 5: Testing Enhanced Epoch Submission");
    console.log("──────────────────────────────────────────────");
    
    // Check balances before
    const minerKeyBefore = await keyToken.balanceOf(ACCOUNTS.miner.address);
    const validator1KeyBefore = await keyToken.balanceOf(ACCOUNTS.validator1.address);
    
    console.log("KEY balances before mining:");
    console.log(`  Miner: ${ethers.formatEther(minerKeyBefore)} KEY`);
    console.log(`  Validator1: ${ethers.formatEther(validator1KeyBefore)} KEY`);
    
    // Submit epoch with enhanced data
    console.log("\\n📊 Submitting enhanced epoch with 5 successful tasks...");
    const successfulMiners = [ACCOUNTS.miner.address];
    const successfulTasks = 5;
    const failedTasks = 0;
    
    tx = await enhancedVerifier.submitAndDistributeEpoch(
        subnetId,
        vlcGraphDataBytes,
        successfulMiners, 
        successfulTasks,
        failedTasks
    );
    receipt = await tx.wait();
    
    console.log(`✅ Enhanced epoch submitted successfully!`);
    console.log(`   Transaction: ${receipt.hash}`);
    console.log(`   Block: ${receipt.blockNumber}`);
    console.log(`   Gas used: ${receipt.gasUsed}`);
    
    // STEP 6: Check enhanced rewards and statistics
    console.log("\\n💰 Step 6: Checking Enhanced Rewards & Statistics");
    console.log("────────────────────────────────────────────────");
    
    // Check balances after
    const minerKeyAfter = await keyToken.balanceOf(ACCOUNTS.miner.address);
    const validator1KeyAfter = await keyToken.balanceOf(ACCOUNTS.validator1.address);
    
    console.log("\\nKEY balances after mining:");
    console.log(`  Miner: ${ethers.formatEther(minerKeyAfter)} KEY`);
    console.log(`  Validator1: ${ethers.formatEther(validator1KeyAfter)} KEY`);
    
    // Calculate mined amounts
    const minerMined = minerKeyAfter - minerKeyBefore;
    const validator1Mined = validator1KeyAfter - validator1KeyBefore;
    
    console.log("\\n💰 KEY tokens mined:");
    console.log(`  Miner earned: ${ethers.formatEther(minerMined)} KEY`);
    console.log(`  Validator1 earned: ${ethers.formatEther(validator1Mined)} KEY`);
    
    // Check enhanced miner statistics
    console.log("\\n📈 Enhanced Miner Statistics:");
    console.log("────────────────────────────");
    
    try {
        const minerStats = await enhancedVerifier.getMinerStats(ACCOUNTS.miner.address);
        console.log(`  Owner: ${minerStats.owner}`);
        console.log(`  Successful tasks: ${minerStats.successfulTasks.toString()}`);
        console.log(`  Total tasks: ${minerStats.totalTasks.toString()}`); 
        console.log(`  Total KEY mined: ${ethers.formatEther(minerStats.totalIntelligenceMined)} KEY`);
        console.log(`  Reputation score: ${minerStats.reputationScore.toString()}%`);
        console.log(`  Last active epoch: ${minerStats.lastActiveEpoch.toString()}`);
        console.log(`  Is active: ${minerStats.isActive}`);
    } catch (error) {
        console.log(`  Error fetching miner stats: ${error.message}`);
    }
    
    // Check subnet information
    console.log("\\n📊 Enhanced Subnet Information:");
    console.log("─────────────────────────────");
    
    try {
        const subnetInfo = await enhancedVerifier.getSubnetInfoByString(subnetId);
        console.log(`  Epoch count: ${subnetInfo.epochCount.toString()}`);
        console.log(`  Total tasks completed: ${subnetInfo.totalTasksCompleted.toString()}`);
        console.log(`  Total rewards distributed: ${ethers.formatEther(subnetInfo.totalRewardsDistributed)} KEY`);
        console.log(`  Last epoch timestamp: ${new Date(Number(subnetInfo.lastEpochTimestamp) * 1000).toISOString()}`);
        console.log(`  Is active: ${subnetInfo.isActive}`);
    } catch (error) {
        console.log(`  Error fetching subnet info: ${error.message}`);
    }
    
    // Final state
    const keySupplyAfter = await keyToken.totalSupply();
    const totalMined = keySupplyAfter - keyTotalSupply;
    
    console.log("\\n🔑 Final State:");
    console.log("───────────────");
    console.log(`Total KEY supply: ${ethers.formatEther(keySupplyAfter)} KEY`);
    console.log(`Total KEY mined this test: ${ethers.formatEther(totalMined)} KEY`);
    
    // Check current block
    const currentBlock = await provider.getBlockNumber();
    console.log(`Final block number: ${currentBlock}`);
    
    console.log("\\n━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
    console.log("🎉 ENHANCED PoCW VERIFIER TEST COMPLETE!");
    console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
}

// Error handling
main()
    .then(() => process.exit(0))
    .catch((error) => {
        console.error("\\n❌ Test failed:");
        console.error(error);
        process.exit(1);
    });