// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "./HETUToken.sol";

/**
 * @title Subnet Registry
 * @dev Manages subnet registration with HETU token deposits
 * Miner: 500 HETU deposit
 * Each Validator: 100 HETU deposit
 */
contract SubnetRegistry {
    HETUToken public hetuToken;
    
    struct Subnet {
        string subnetId;
        address miner;
        address[4] validators;
        uint256 minerDeposit;
        uint256 validatorDeposit;
        bool isActive;
        uint256 registeredAt;
    }
    
    mapping(string => Subnet) public subnets;
    mapping(address => string) public participantToSubnet;
    
    uint256 public constant MINER_DEPOSIT = 500 * 10**18; // 500 HETU
    uint256 public constant VALIDATOR_DEPOSIT = 100 * 10**18; // 100 HETU per validator
    
    event SubnetRegistered(string subnetId, address miner, address[4] validators);
    event SubnetDeactivated(string subnetId);
    
    address public owner;
    bool public initialized;

    constructor() {
        owner = msg.sender;
    }

    function initialize(address _hetuToken) external {
        require(msg.sender == owner, "Only owner can initialize");
        require(!initialized, "Already initialized");
        hetuToken = HETUToken(_hetuToken);
        initialized = true;
    }
    
    /**
     * @dev Register a new subnet with deposits
     * @param subnetId Unique subnet identifier
     * @param miner Address of the miner
     * @param validators Array of 4 validator addresses
     */
    function registerSubnet(
        string memory subnetId,
        address miner,
        address[4] memory validators
    ) external {
        require(initialized, "Contract not initialized");
        require(bytes(subnets[subnetId].subnetId).length == 0, "Subnet already exists");
        require(miner != address(0), "Invalid miner address");
        
        // Check all validators are unique and valid
        for (uint i = 0; i < 4; i++) {
            require(validators[i] != address(0), "Invalid validator address");
            require(validators[i] != miner, "Miner cannot be validator");
            for (uint j = i + 1; j < 4; j++) {
                require(validators[i] != validators[j], "Duplicate validator");
            }
        }
        
        // Collect miner deposit
        require(
            hetuToken.transferFrom(miner, address(this), MINER_DEPOSIT),
            "Miner deposit failed"
        );
        
        // Collect validator deposits
        for (uint i = 0; i < 4; i++) {
            require(
                hetuToken.transferFrom(validators[i], address(this), VALIDATOR_DEPOSIT),
                "Validator deposit failed"
            );
        }
        
        // Register subnet
        subnets[subnetId] = Subnet({
            subnetId: subnetId,
            miner: miner,
            validators: validators,
            minerDeposit: MINER_DEPOSIT,
            validatorDeposit: VALIDATOR_DEPOSIT * 4,
            isActive: true,
            registeredAt: block.timestamp
        });
        
        // Map participants to subnet
        participantToSubnet[miner] = subnetId;
        for (uint i = 0; i < 4; i++) {
            participantToSubnet[validators[i]] = subnetId;
        }
        
        emit SubnetRegistered(subnetId, miner, validators);
    }
    
    /**
     * @dev Check if a subnet is active
     */
    function isSubnetActive(string memory subnetId) external view returns (bool) {
        return subnets[subnetId].isActive;
    }
    
    /**
     * @dev Get subnet details
     */
    function getSubnet(string memory subnetId) external view returns (
        address miner,
        address[4] memory validators,
        bool isActive
    ) {
        Subnet memory subnet = subnets[subnetId];
        return (subnet.miner, subnet.validators, subnet.isActive);
    }
}