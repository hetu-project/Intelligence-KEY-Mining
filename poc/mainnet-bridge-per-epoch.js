#!/usr/bin/env node

/**
 * Enhanced PoCW Subnet to Mainnet Bridge - Per-Epoch Submission
 * 
 * This module handles real-time epoch submission to mainnet as each epoch
 * completes (every 3 rounds), rather than batching all epochs at the end.
 * 
 * Architecture:
 * 1. Integrates with the Go subnet system via callback mechanism
 * 2. Submits each epoch immediately when EpochFinalized event occurs
 * 3. Provides real-time KEY token mining per completed epoch
 * 4. Maintains epoch tracking and statistics
 */

const { ethers } = require('ethers');
const { spawn } = require('child_process');
const fs = require('fs').promises;
const path = require('path');
const http = require('http');
const url = require('url');

class PerEpochMainnetBridge {
    constructor() {
        this.provider = null;
        this.contracts = {};
        this.wallets = {};
        this.subnetProcess = null;
        this.epochSubmissions = new Map(); // Track submitted epochs
        this.httpServer = null;
        
        // Network configuration
        this.RPC_URL = "http://localhost:8545";
        this.DGRAPH_URL = "http://localhost:8080";
        
        // Account configuration
        this.accounts = {
            deployer: {
                address: "0xf39Fd6e51aad88F6F4ce6aB8827279cffFb92266",
                privateKey: "0xac0974bec39a17e36ba4a6b4d238ff944bacb478cbed5efcae784d7bf4f2ff80"
            },
            validator1: {
                address: "0x70997970C51812dc3A010C7d01b50e0d17dc79C8", 
                privateKey: "0x59c6995e998f97a5a0044966f0945389dc9e86dae88c7a8412f4603b6b78690d"
            },
            miner: {
                address: "0x9965507D1a55bcC2695C58ba16FB37d819B0A4dc",
                privateKey: "0x8b3a350cf5c34c9194ca85829a2df0ec3153be0318b5e2d3348e872092edffba"
            }
        };
    }

    async initialize() {
        console.log("🌐 Per-Epoch PoCW Mainnet Bridge Initializing...");
        console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
        
        // Initialize ethers provider
        this.provider = new ethers.JsonRpcProvider(this.RPC_URL);
        
        // Create wallets
        this.wallets.validator1 = new ethers.Wallet(this.accounts.validator1.privateKey, this.provider);
        this.wallets.miner = new ethers.Wallet(this.accounts.miner.privateKey, this.provider);
        
        // Load contract addresses and ABIs
        await this.loadContracts();
        
        // Start HTTP server for Go integration
        await this.startHttpServer();
        
        console.log("✅ Bridge initialized successfully!");
        console.log(`📍 Validator-1: ${this.accounts.validator1.address}`);
        console.log(`⛏️  Miner: ${this.accounts.miner.address}`);
    }

    async loadContracts() {
        try {
            // Load contract addresses
            const addressesPath = path.join(__dirname, 'contract_addresses.json');
            const addressesData = await fs.readFile(addressesPath, 'utf8');
            const addresses = JSON.parse(addressesData);
            
            // Find contract addresses by searching for known patterns
            let hetuAddress, keyAddress, registryAddress, verifierAddress;
            
            for (const [address, name] of Object.entries(addresses)) {
                if (name.includes('HETU Token')) hetuAddress = address;
                else if (name.includes('Intelligence Token') || name.includes('KEY')) keyAddress = address;
                else if (name.includes('Subnet Registry')) registryAddress = address;
                else if (name.includes('Enhanced PoCW') || name.includes('Verifier')) verifierAddress = address;
            }
            
            if (!hetuAddress || !keyAddress || !registryAddress || !verifierAddress) {
                throw new Error('Could not find all required contract addresses');
            }
            
            // Load ABIs
            const verifierABI = [
                "function submitAndDistributeEpoch(string memory subnetId, bytes memory vlcGraphData, address[] memory successfulMiners, uint256 successfulTasks, uint256 failedTasks) external",
                "function getMinerStats(address miner) external view returns (tuple(address owner, uint256 successfulTasks, uint256 totalTasks, uint256 totalIntelligenceMined, uint256 reputationScore, uint256 lastActiveEpoch, uint256 joinedTimestamp, bool isActive))",
                "function subnetIdToHash(string memory subnetId) external view returns (bytes32)"
            ];
            
            const keyABI = [
                "function balanceOf(address account) external view returns (uint256)",
                "function totalSupply() external view returns (uint256)"
            ];
            
            // Create contract instances
            this.contracts.verifier = new ethers.Contract(verifierAddress, verifierABI, this.wallets.validator1);
            this.contracts.key = new ethers.Contract(keyAddress, keyABI, this.provider);
            
            console.log(`📋 Loaded contracts:`);
            console.log(`  EnhancedPoCWVerifier: ${verifierAddress}`);
            console.log(`  KEY Token: ${keyAddress}`);
            
        } catch (error) {
            throw new Error(`Failed to load contracts: ${error.message}`);
        }
    }

    // Callback function to handle epoch finalized events from the Go subnet
    async handleEpochFinalized(epochNumber, subnetId, epochData) {
        try {
            console.log(`\n🚀 EPOCH ${epochNumber} FINALIZED - IMMEDIATE MAINNET SUBMISSION`);
            console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
            console.log(`📊 Subnet: ${subnetId}`);
            console.log(`📈 Epoch: ${epochNumber}`);
            console.log(`🔗 Completed Rounds: ${epochData.CompletedRounds.length}`);
            console.log(`⏰ VLC Clock State:`, epochData.VLCClockState);
            
            // Check if already submitted (prevent duplicates)
            const epochKey = `${subnetId}-${epochNumber}`;
            if (this.epochSubmissions.has(epochKey)) {
                console.log(`⚠️  Epoch ${epochNumber} already submitted, skipping...`);
                return;
            }
            
            // Extract current epoch data from Dgraph
            const { vlcGraphData, successfulTasks, failedTasks, miners } = await this.extractCurrentEpochData(subnetId, epochNumber);
            
            // Submit to mainnet
            const result = await this.submitEpochToMainnet(subnetId, vlcGraphData, miners, successfulTasks, failedTasks);
            
            // Mark as submitted
            this.epochSubmissions.set(epochKey, {
                epochNumber,
                subnetId,
                txHash: result.txHash,
                blockNumber: result.blockNumber,
                keyMined: result.keyMined,
                timestamp: Date.now()
            });
            
            console.log(`✅ Epoch ${epochNumber} submitted successfully!`);
            console.log(`💰 KEY Mined: ${result.keyMined} KEY tokens`);
            console.log(`📤 Transaction: ${result.txHash}`);
            console.log(`📦 Block: ${result.blockNumber}`);
            
        } catch (error) {
            console.error(`❌ Failed to submit epoch ${epochNumber}:`, error.message);
        }
    }

    // Extract VLC data for the current completed epoch
    async extractCurrentEpochData(subnetId, epochNumber) {
        try {
            console.log(`📊 Extracting VLC data for epoch ${epochNumber}...`);
            
            // Query Dgraph for events from this specific epoch
            const query = `
            {
                events(func: has(event_id)) @filter(eq(subnet_id, "${subnetId}")) {
                    uid
                    event_id
                    event_name
                    event_type
                    vlc_clock
                    parents {
                        uid
                        event_id
                    }
                    timestamp
                    description
                    request_id
                }
            }`;
            
            const response = await fetch(`${this.DGRAPH_URL}/query`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({ query })
            });
            
            if (!response.ok) {
                throw new Error('Failed to query Dgraph');
            }
            
            const data = await response.json();
            const events = data.data.events || [];
            
            console.log(`✅ Extracted ${events.length} events from Dgraph`);
            
            // Filter events for current epoch (rough estimation based on timing or event patterns)
            // In a real implementation, you would track epoch boundaries more precisely
            const epochEvents = this.filterEventsForCurrentEpoch(events, epochNumber);
            
            // Count successful and failed tasks from the epoch
            const { successfulTasks, failedTasks } = this.analyzeEpochTasks(epochEvents);
            
            // Generate comprehensive VLC graph data for this epoch
            const vlcGraphData = this.generateEpochVLCData(subnetId, epochNumber, epochEvents, successfulTasks, failedTasks);
            
            console.log(`📈 Epoch ${epochNumber}: ${successfulTasks} successful, ${failedTasks} failed tasks`);
            
            return {
                vlcGraphData,
                successfulTasks,
                failedTasks,
                miners: [this.accounts.miner.address]
            };
            
        } catch (error) {
            console.error("❌ Error extracting epoch data:", error.message);
            // Fallback to simulated data for this epoch
            return this.generateSimulatedEpochData(subnetId, epochNumber);
        }
    }

    // Filter events that belong to the current epoch
    filterEventsForCurrentEpoch(events, epochNumber) {
        // For simplicity, assume the most recent events belong to the current epoch
        // In a production system, you would have explicit epoch boundaries
        const eventsPerEpoch = 10; // Approximate events per epoch (3 rounds * ~3 events per round + epoch events)
        const startIndex = Math.max(0, events.length - (eventsPerEpoch * (4 - epochNumber)));
        const endIndex = events.length - (eventsPerEpoch * (3 - epochNumber));
        
        return events.slice(startIndex, Math.min(endIndex, events.length));
    }

    // Analyze epoch events to count successful/failed tasks
    analyzeEpochTasks(epochEvents) {
        const successfulTasks = epochEvents.filter(e => 
            e.event_name === 'RoundSuccess' || 
            e.description?.includes('OUTPUT DELIVERED TO USER')
        ).length;
        
        const failedTasks = epochEvents.filter(e => 
            e.event_name === 'RoundFailed' || 
            e.description?.includes('OUTPUT REJECTED')
        ).length;
        
        return { successfulTasks, failedTasks };
    }

    // Generate VLC graph data for a specific epoch
    generateEpochVLCData(subnetId, epochNumber, epochEvents, successfulTasks, failedTasks) {
        return {
            subnetId,
            epochNumber,
            events: epochEvents.map(event => ({
                id: event.event_id || `epoch_${epochNumber}_${event.uid}`,
                name: event.event_name || 'Unknown',
                vlcClock: event.vlc_clock || {},
                parents: (event.parents || []).map(p => p.event_id || p.uid),
                timestamp: event.timestamp || Date.now(),
                description: event.description || `Epoch ${epochNumber} event`,
                requestId: event.request_id || null
            })),
            miners: [this.accounts.miner.address],
            validators: [
                this.accounts.validator1.address,
                "0x3C44CdDdB6a900fa2b585dd299e03d12FA4293BC",
                "0x90F79bf6EB2c4f870365E785982E1f101E93b906", 
                "0x15d34AAf54267DB7D7c367839AAf71A00a2C6A65"
            ],
            summary: {
                epochNumber,
                totalTasks: successfulTasks + failedTasks,
                successfulTasks,
                failedTasks,
                validationStatus: "complete",
                consensusReached: true
            },
            statistics: {
                epochProcessingTime: 7500, // 3 rounds * ~2500ms per round
                eventsInEpoch: epochEvents.length,
                avgEventsPerRound: epochEvents.length / 3
            }
        };
    }

    // Generate simulated epoch data as fallback
    generateSimulatedEpochData(subnetId, epochNumber) {
        console.log(`🎭 Generating simulated data for epoch ${epochNumber}...`);
        
        const currentTime = Date.now();
        const events = [];
        
        // Simulate 3 rounds (1 per round for this epoch)
        const tasksInEpoch = 1; // Typically 1 task per epoch in our 7-task demo
        for (let round = 1; round <= 3; round++) {
            const baseTime = currentTime - (3000 - round * 1000);
            const globalTaskId = (epochNumber - 1) * 1 + 1; // Map to global task sequence
            
            if (globalTaskId <= 7) { // Only if within our 7-task demo
                // User input
                events.push({
                    id: `user_input_epoch_${epochNumber}_round_${round}`,
                    name: "UserInput",
                    vlcClock: { 1: globalTaskId-1, 2: globalTaskId },
                    parents: events.length > 0 ? [events[events.length-1].id] : [],
                    timestamp: baseTime,
                    description: `Epoch ${epochNumber} Round ${round}: User submits task`,
                    requestId: `req-${subnetId}-${globalTaskId}`
                });
                
                // Miner output
                events.push({
                    id: `miner_output_epoch_${epochNumber}_round_${round}`,
                    name: "MinerOutput", 
                    vlcClock: { 1: globalTaskId, 2: globalTaskId },
                    parents: [`user_input_epoch_${epochNumber}_round_${round}`],
                    timestamp: baseTime + 500,
                    description: `Epoch ${epochNumber} Round ${round}: Miner provides solution`,
                    requestId: `req-${subnetId}-${globalTaskId}`
                });
                
                // Round success
                events.push({
                    id: `round_${epochNumber}_${round}_complete`,
                    name: "RoundSuccess",
                    vlcClock: { 1: globalTaskId, 2: globalTaskId + 1 },
                    parents: [`miner_output_epoch_${epochNumber}_round_${round}`],
                    timestamp: baseTime + 1000,
                    description: `Epoch ${epochNumber} Round ${round}: OUTPUT DELIVERED TO USER`,
                    requestId: `req-${subnetId}-${globalTaskId}`
                });
            }
        }
        
        const vlcGraphData = this.generateEpochVLCData(subnetId, epochNumber, events, 1, 0);
        
        return {
            vlcGraphData,
            successfulTasks: 1,
            failedTasks: 0,
            miners: [this.accounts.miner.address]
        };
    }

    // Submit epoch data to mainnet
    async submitEpochToMainnet(subnetId, vlcGraphData, miners, successfulTasks, failedTasks) {
        console.log(`\n⚡ Submitting epoch ${vlcGraphData.epochNumber} to mainnet...`);
        console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
        
        // Convert VLC data to bytes
        const vlcDataString = JSON.stringify(vlcGraphData);
        const vlcDataBytes = ethers.toUtf8Bytes(vlcDataString);
        
        console.log(`📊 Submitting epoch data:`);
        console.log(`  Subnet: ${subnetId}`);
        console.log(`  Epoch: ${vlcGraphData.epochNumber}`);
        console.log(`  Tasks: ${successfulTasks} successful, ${failedTasks} failed`);
        console.log(`  VLC Events: ${vlcGraphData.events.length}`);
        console.log(`  Data Size: ${vlcDataBytes.length} bytes`);
        
        // Get pre-submission balances
        const minerBalanceBefore = await this.contracts.key.balanceOf(this.accounts.miner.address);
        const validator1BalanceBefore = await this.contracts.key.balanceOf(this.accounts.validator1.address);
        
        try {
            // Submit epoch and mine KEY tokens
            console.log(`\n🚀 Validator-1 posting epoch ${vlcGraphData.epochNumber} to mainnet...`);
            const tx = await this.contracts.verifier.submitAndDistributeEpoch(
                vlcGraphData.subnetId,
                vlcDataBytes,
                miners,
                successfulTasks,
                failedTasks
            );
            
            console.log(`📤 Transaction submitted: ${tx.hash}`);
            console.log("⏳ Waiting for confirmation...");
            
            const receipt = await tx.wait();
            console.log(`✅ Transaction confirmed in block: ${receipt.blockNumber}`);
            
            // Check post-submission balances
            const minerBalanceAfter = await this.contracts.key.balanceOf(this.accounts.miner.address);
            const validator1BalanceAfter = await this.contracts.key.balanceOf(this.accounts.validator1.address);
            
            const minerEarned = ethers.formatEther(minerBalanceAfter - minerBalanceBefore);
            const validator1Earned = ethers.formatEther(validator1BalanceAfter - validator1BalanceBefore);
            const totalMined = ethers.formatEther((minerBalanceAfter - minerBalanceBefore) + (validator1BalanceAfter - validator1BalanceBefore));
            
            console.log(`\n🎉 EPOCH ${vlcGraphData.epochNumber} KEY MINING COMPLETE!`);
            console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
            console.log(`💰 Miner earned: ${minerEarned} KEY`);
            console.log(`🏆 Validator-1 earned: ${validator1Earned} KEY`);
            console.log(`🔑 Total KEY mined: ${totalMined} KEY`);
            
            return {
                txHash: tx.hash,
                blockNumber: receipt.blockNumber,
                keyMined: totalMined,
                gasUsed: receipt.gasUsed.toString()
            };
            
        } catch (error) {
            console.error(`❌ Epoch submission failed: ${error.message}`);
            throw error;
        }
    }

    // Get summary of all submitted epochs
    getSubmissionSummary() {
        console.log(`\n📊 EPOCH SUBMISSION SUMMARY`);
        console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
        console.log(`📈 Total Epochs Submitted: ${this.epochSubmissions.size}`);
        
        let totalKeyMined = 0;
        for (const [epochKey, submission] of this.epochSubmissions.entries()) {
            console.log(`  Epoch ${submission.epochNumber}: ${submission.keyMined} KEY (tx: ${submission.txHash.substring(0, 10)}...)`);
            totalKeyMined += parseFloat(submission.keyMined);
        }
        
        console.log(`💰 Total KEY Mined: ${totalKeyMined} KEY across all epochs`);
        console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
    }

    // Start HTTP server to receive epoch data from Go
    async startHttpServer() {
        const PORT = 3001;
        
        this.httpServer = http.createServer((req, res) => {
            // Handle CORS
            res.setHeader('Access-Control-Allow-Origin', '*');
            res.setHeader('Access-Control-Allow-Methods', 'GET, POST, OPTIONS');
            res.setHeader('Access-Control-Allow-Headers', 'Content-Type');
            
            if (req.method === 'OPTIONS') {
                res.writeHead(200);
                res.end();
                return;
            }
            
            const parsedUrl = url.parse(req.url, true);
            
            if (req.method === 'POST' && parsedUrl.pathname === '/submit-epoch') {
                this.handleEpochSubmission(req, res);
            } else if (req.method === 'GET' && parsedUrl.pathname === '/health') {
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ status: 'healthy', service: 'per-epoch-bridge' }));
            } else {
                res.writeHead(404, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ error: 'Not Found' }));
            }
        });

        return new Promise((resolve, reject) => {
            this.httpServer.listen(PORT, (err) => {
                if (err) {
                    reject(err);
                } else {
                    console.log(`🌐 HTTP server listening on port ${PORT}`);
                    console.log(`📡 Ready to receive epoch data from Go at http://localhost:${PORT}/submit-epoch`);
                    resolve();
                }
            });
        });
    }

    // Handle epoch submission from Go
    async handleEpochSubmission(req, res) {
        let body = '';
        
        req.on('data', chunk => {
            body += chunk.toString();
        });
        
        req.on('end', async () => {
            try {
                const epochData = JSON.parse(body);
                console.log(`\n🚀 RECEIVED EPOCH SUBMISSION FROM GO`);
                console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`);
                console.log(`📊 Epoch: ${epochData.epochNumber}`);
                console.log(`🌐 Subnet: ${epochData.subnetId}`);
                console.log(`⏰ Timestamp: ${new Date(epochData.timestamp * 1000).toISOString()}`);
                console.log(`🔗 Rounds: ${epochData.completedRounds.length}`);
                console.log(`🔍 Detailed Rounds: ${epochData.detailedRounds ? epochData.detailedRounds.length : 'undefined'}`);
                console.log(`🕘 VLC State: ${JSON.stringify(epochData.vlcClockState)}`);
                
                // Debug detailed round data
                if (epochData.detailedRounds && epochData.detailedRounds.length > 0) {
                    console.log(`🔍 DEBUG - Detailed rounds received:`);
                    epochData.detailedRounds.forEach((round, index) => {
                        console.log(`   Round ${index + 1}: ${round.userInput ? round.userInput.substring(0, 40) + '...' : 'No input'}`);
                    });
                } else {
                    console.log(`❌ DEBUG - No detailed rounds in payload`);
                }
                
                // Submit to blockchain
                await this.submitEpochToBlockchain(epochData);
                
                res.writeHead(200, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ 
                    success: true, 
                    epochNumber: epochData.epochNumber,
                    message: 'Epoch submitted successfully'
                }));
                
            } catch (error) {
                console.error('❌ Error handling epoch submission:', error.message);
                res.writeHead(500, { 'Content-Type': 'application/json' });
                res.end(JSON.stringify({ 
                    success: false, 
                    error: error.message 
                }));
            }
        });
    }

    // Submit epoch data to blockchain using the received data
    async submitEpochToBlockchain(epochData) {
        try {
            console.log(`📤 Submitting Epoch ${epochData.epochNumber} to blockchain...`);
            
            // Convert Go epoch data to blockchain format
            const vlcGraphData = this.encodeVLCGraphData(epochData);
            
            const successfulMiners = [this.accounts.miner.address];
            
            // Use actual successful/failed counts from detailed round data
            let successfulTasks = (epochData.detailedRounds || []).filter(r => r.success).length;
            let failedTasks = (epochData.detailedRounds || []).filter(r => !r.success).length;
            
            console.log(`📊 Task breakdown: ${successfulTasks} successful, ${failedTasks} failed (total: ${successfulTasks + failedTasks})`);
            
            // Verify we have the expected task counts
            const totalRoundsInEpoch = successfulTasks + failedTasks;
            if (totalRoundsInEpoch === 0) {
                console.log(`⚠️  WARNING: No detailed round data available, falling back to completedRounds count`);
                // Fallback to legacy method if detailed rounds are unavailable
                successfulTasks = epochData.completedRounds ? epochData.completedRounds.length : 0;
                failedTasks = 0;
            }
            
            // Submit to contract
            const tx = await this.contracts.verifier.submitAndDistributeEpoch(
                epochData.subnetId,
                vlcGraphData,
                successfulMiners,
                successfulTasks,
                failedTasks
            );
            
            console.log(`📤 Transaction submitted: ${tx.hash}`);
            const receipt = await tx.wait();
            console.log(`✅ Transaction confirmed in block ${receipt.blockNumber}`);
            
            // Track submission
            const submissionKey = `${epochData.subnetId}-epoch-${epochData.epochNumber}`;
            this.epochSubmissions.set(submissionKey, {
                epochNumber: epochData.epochNumber,
                subnetId: epochData.subnetId,
                txHash: tx.hash,
                blockNumber: receipt.blockNumber,
                timestamp: epochData.timestamp,
                keyMined: "0" // Would calculate from logs
            });
            
            console.log(`🎉 Epoch ${epochData.epochNumber} submitted successfully!`);
            console.log(`━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━`);
            
            return {
                txHash: tx.hash,
                blockNumber: receipt.blockNumber,
                gasUsed: receipt.gasUsed.toString()
            };
            
        } catch (error) {
            console.error(`❌ Blockchain submission failed: ${error.message}`);
            throw error;
        }
    }

    // Encode VLC graph data for blockchain submission
    encodeVLCGraphData(epochData) {
        // Create a structured representation of the VLC graph for this epoch
        const vlcGraph = {
            epochNumber: epochData.epochNumber,
            vlcClockState: epochData.vlcClockState,
            detailedRounds: epochData.detailedRounds || [],  // Include detailed round data
            epochEventId: epochData.epochEventId || '',
            parentRoundEventId: epochData.parentRoundEventId || '',
            timestamp: Math.floor(Date.now() / 1000)
            // Removed redundant fields: completedRounds, totalRounds, successfulRounds, failedRounds
            // These are now calculated and passed separately to smart contract
        };

        // Convert to hex-encoded bytes for smart contract
        const jsonString = JSON.stringify(vlcGraph);
        const hexData = '0x' + Buffer.from(jsonString, 'utf8').toString('hex');
        
        console.log(`🔗 Encoded VLC graph data: ${jsonString.length} bytes`);
        console.log(`📊 Epoch summary: ${vlcGraph.totalRounds} rounds (${vlcGraph.successfulRounds} success, ${vlcGraph.failedRounds} failed)`);
        
        // Log detailed round information
        if (epochData.detailedRounds && epochData.detailedRounds.length > 0) {
            console.log(`📋 Round details:`);
            epochData.detailedRounds.forEach(round => {
                const status = round.success ? '✅' : '❌';
                const inputPreview = round.userInput.length > 40 ? round.userInput.substring(0, 40) + '...' : round.userInput;
                console.log(`   Round ${round.roundNumber}: ${status} "${inputPreview}"`);
            });
        }
        
        return hexData;
    }
}

// Export the class for use in integration scripts
module.exports = PerEpochMainnetBridge;

// If run directly, start in interactive mode
if (require.main === module) {
    const bridge = new PerEpochMainnetBridge();
    
    async function main() {
        try {
            await bridge.initialize();
            console.log(`\n🔄 Per-Epoch Bridge Ready!`);
            console.log("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━");
            console.log("To use this bridge:");
            console.log("1. Set up epoch callback in subnet coordinator");  
            console.log("2. Each completed epoch (3 rounds) triggers immediate submission");
            console.log("3. KEY tokens are mined in real-time per epoch");
            console.log("");
            console.log("Example usage:");
            console.log("const bridge = new PerEpochMainnetBridge();");
            console.log("coordinator.GraphAdapter.SetEpochFinalizedCallback(bridge.handleEpochFinalized.bind(bridge));");
            
        } catch (error) {
            console.error("❌ Bridge initialization failed:", error.message);
        }
    }
    
    main().catch(console.error);
}