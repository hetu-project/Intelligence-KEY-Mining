// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./KEYToken.sol";
import "./SubnetRegistry.sol";

/**
 * @title Enhanced PoCW Verifier
 * @dev Advanced epoch submission system with comprehensive tracking and statistics
 */
contract EnhancedPoCWVerifier {
    KEYToken public keyToken;
    SubnetRegistry public subnetRegistry;
    
    // Reward constants
    uint256 public constant MINER_REWARD_PER_TASK = 100 * 10**18; // 100 KEY per successful task
    uint256 public constant VALIDATOR_BASE_REWARD = 20 * 10**18;  // 20 KEY base validator reward
    uint256 public constant VALIDATOR_PERCENTAGE = 10; // 10% of miner rewards
    
    // Enhanced epoch submission structure
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
    
    // Miner statistics and reputation
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
    
    // Subnet enhanced info
    struct SubnetInfo {
        uint256 epochCount;
        uint256 totalTasksCompleted;
        uint256 totalRewardsDistributed;
        uint256 lastEpochTimestamp;
        bool isActive;
    }
    
    // Mappings
    mapping(bytes32 => mapping(uint256 => EpochSubmission)) public epochSubmissions;
    mapping(address => MinerStats) public minerStats;
    mapping(bytes32 => SubnetInfo) public subnetInfo;
    mapping(string => bytes32) public subnetIdToHash;
    
    // Events
    event EpochSubmitted(bytes32 indexed subnetId, uint256 indexed epochNumber, address indexed validator);
    event EpochVerified(bytes32 indexed subnetId, uint256 indexed epochNumber, bool verified);
    event IntelligenceMoneyMined(address indexed recipient, uint256 amount, bytes32 indexed subnetId, uint256 indexed epochNumber);
    event MinerStatsUpdated(address indexed miner, uint256 successfulTasks, uint256 totalTasks, uint256 reputationScore);
    event ReputationUpdated(address indexed miner, uint256 oldScore, uint256 newScore);
    
    // Security
    address public owner;
    bool public initialized;
    mapping(address => bool) public paused;
    
    modifier onlyOwner() {
        require(msg.sender == owner, "Only owner");
        _;
    }
    
    modifier whenNotPaused() {
        require(!paused[address(this)], "Contract paused");
        _;
    }
    
    modifier nonReentrant() {
        require(!paused[msg.sender], "Reentrant call");
        paused[msg.sender] = true;
        _;
        paused[msg.sender] = false;
    }

    constructor() {
        owner = msg.sender;
    }

    function initialize(address _keyToken, address _subnetRegistry) external onlyOwner {
        require(!initialized, "Already initialized");
        keyToken = KEYToken(_keyToken);
        subnetRegistry = SubnetRegistry(_subnetRegistry);
        initialized = true;
    }
    
    /**
     * @dev Enhanced epoch submission with comprehensive tracking
     */
    function submitAndDistributeEpoch(
        string memory subnetIdString,
        bytes calldata vlcGraphData,
        address[] calldata successfulMiners,
        uint256 successfulTasks,
        uint256 failedTasks
    ) external nonReentrant whenNotPaused {
        require(initialized, "Contract not initialized");
        
        bytes32 subnetId = _getOrCreateSubnetHash(subnetIdString);
        require(subnetRegistry.isSubnetActive(subnetIdString), "Subnet not active");
        require(successfulTasks > 0, "No successful tasks");
        require(vlcGraphData.length > 0, "Graph data cannot be empty");
        require(successfulMiners.length > 0, "No successful miners");
        require(successfulMiners.length <= 10, "Too many miners (max 10)");
        
        // Verify caller is authorized validator
        (address registeredMiner, address[4] memory validators, ) = subnetRegistry.getSubnet(subnetIdString);
        require(_isSubnetValidator(validators, msg.sender), "Not authorized subnet validator");
        
        // Get next epoch number
        uint256 epochNumber = subnetInfo[subnetId].epochCount + 1;
        require(epochSubmissions[subnetId][epochNumber].timestamp == 0, "Epoch already submitted");
        
        // Process epoch submission
        _processEpochSubmission(
            subnetId, 
            epochNumber, 
            vlcGraphData, 
            successfulMiners, 
            successfulTasks,
            failedTasks
        );
    }
    
    function _processEpochSubmission(
        bytes32 subnetId,
        uint256 epochNumber,
        bytes calldata vlcGraphData,
        address[] calldata successfulMiners,
        uint256 successfulTasks,
        uint256 failedTasks
    ) private {
        // Store epoch data
        epochSubmissions[subnetId][epochNumber] = EpochSubmission({
            subnetId: subnetId,
            epochNumber: epochNumber,
            vlcGraphData: vlcGraphData,
            successfulMiners: successfulMiners,
            successfulTasks: successfulTasks,
            failedTasks: failedTasks,
            timestamp: block.timestamp,
            verified: true,
            rewardsDistributed: true,
            submittingValidator: msg.sender,
            totalRewardDistributed: 0
        });
        
        // Distribute rewards and update stats
        uint256 totalRewards = _distributeEpochRewards(subnetId, epochNumber, successfulMiners, successfulTasks);
        epochSubmissions[subnetId][epochNumber].totalRewardDistributed = totalRewards;
        
        // Update subnet info
        subnetInfo[subnetId].epochCount = epochNumber;
        subnetInfo[subnetId].totalTasksCompleted += successfulTasks;
        subnetInfo[subnetId].totalRewardsDistributed += totalRewards;
        subnetInfo[subnetId].lastEpochTimestamp = block.timestamp;
        subnetInfo[subnetId].isActive = true;
        
        // Emit events
        emit EpochSubmitted(subnetId, epochNumber, msg.sender);
        emit EpochVerified(subnetId, epochNumber, true);
    }
    
    function _distributeEpochRewards(
        bytes32 subnetId,
        uint256 epochNumber,
        address[] calldata successfulMiners,
        uint256 successfulTasks
    ) private returns (uint256 totalRewards) {
        
        // Calculate rewards per miner (distribute total tasks among successful miners)
        uint256 tasksPerMiner = successfulTasks / successfulMiners.length;
        uint256 remainingTasks = successfulTasks % successfulMiners.length;
        
        // Reward miners and update their stats
        for (uint256 i = 0; i < successfulMiners.length; i++) {
            address miner = successfulMiners[i];
            
            // First miner gets extra tasks if there's a remainder
            uint256 minerTasks = tasksPerMiner + (i == 0 ? remainingTasks : 0);
            uint256 minerReward = minerTasks * MINER_REWARD_PER_TASK;
            
            // Mine tokens for miner
            keyToken.mine(miner, minerReward, "Miner: AI tasks completed");
            totalRewards += minerReward;
            
            emit IntelligenceMoneyMined(miner, minerReward, subnetId, epochNumber);
            
            // Update miner statistics
            _updateMinerStats(miner, minerTasks, epochNumber);
        }
        
        // Calculate and distribute validator reward
        uint256 validatorReward = VALIDATOR_BASE_REWARD + (totalRewards * VALIDATOR_PERCENTAGE) / 100;
        keyToken.mine(msg.sender, validatorReward, "Validator: Epoch coordination");
        totalRewards += validatorReward;
        
        emit IntelligenceMoneyMined(msg.sender, validatorReward, subnetId, epochNumber);
        
        return totalRewards;
    }
    
    function _updateMinerStats(address miner, uint256 newSuccessfulTasks, uint256 epochNumber) private {
        MinerStats storage stats = minerStats[miner];
        
        // Initialize if new miner
        if (stats.owner == address(0)) {
            stats.owner = miner;
            stats.joinedTimestamp = block.timestamp;
            stats.isActive = true;
        }
        
        uint256 oldReputation = stats.reputationScore;
        
        // Update task counts
        stats.successfulTasks += newSuccessfulTasks;
        stats.totalTasks += newSuccessfulTasks; // Assuming all tasks assigned were successful for simplicity
        stats.totalIntelligenceMined += newSuccessfulTasks * MINER_REWARD_PER_TASK;
        
        // Update reputation score (success rate percentage)
        if (stats.totalTasks > 0) {
            stats.reputationScore = (stats.successfulTasks * 100) / stats.totalTasks;
        }
        
        stats.lastActiveEpoch = epochNumber;
        stats.isActive = true;
        
        emit MinerStatsUpdated(miner, stats.successfulTasks, stats.totalTasks, stats.reputationScore);
        
        if (oldReputation != stats.reputationScore) {
            emit ReputationUpdated(miner, oldReputation, stats.reputationScore);
        }
    }
    
    function _isSubnetValidator(address[4] memory validators, address validator) private pure returns (bool) {
        for (uint256 i = 0; i < 4; i++) {
            if (validators[i] == validator) {
                return true;
            }
        }
        return false;
    }
    
    function _getOrCreateSubnetHash(string memory subnetIdString) private returns (bytes32) {
        bytes32 hash = subnetIdToHash[subnetIdString];
        if (hash == bytes32(0)) {
            hash = keccak256(abi.encodePacked(subnetIdString));
            subnetIdToHash[subnetIdString] = hash;
        }
        return hash;
    }
    
    // View functions
    function getEpochSubmission(bytes32 subnetId, uint256 epochNumber) external view returns (EpochSubmission memory) {
        return epochSubmissions[subnetId][epochNumber];
    }
    
    function getMinerStats(address miner) external view returns (MinerStats memory) {
        return minerStats[miner];
    }
    
    function getSubnetInfo(bytes32 subnetId) external view returns (SubnetInfo memory) {
        return subnetInfo[subnetId];
    }
    
    function getSubnetInfoByString(string memory subnetIdString) external view returns (SubnetInfo memory) {
        bytes32 subnetId = keccak256(abi.encodePacked(subnetIdString));
        return subnetInfo[subnetId];
    }
    
    function getEpochCount(string memory subnetIdString) external view returns (uint256) {
        bytes32 subnetId = keccak256(abi.encodePacked(subnetIdString));
        return subnetInfo[subnetId].epochCount;
    }
    
    // Admin functions
    function pauseContract() external onlyOwner {
        paused[address(this)] = true;
    }
    
    function unpauseContract() external onlyOwner {
        paused[address(this)] = false;
    }
}