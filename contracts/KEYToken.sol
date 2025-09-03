// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

/**
 * @title KEY Token - Intelligence Money
 * @dev Soulbound token mined through Proof-of-Causal-Work.
 * Non-transferable, represents verifiable AI contributions.
 * Max supply: 21 million KEY (like Bitcoin)
 */
contract KEYToken {
    mapping(address => uint256) public balanceOf;
    mapping(address => bool) public isMiner;
    
    address public pocwVerifier;
    uint256 public constant MAX_SUPPLY = 21_000_000 * 10**18; // 21 million KEY
    uint256 public totalSupply;
    uint256 public totalMined;
    
    event KEYMined(address indexed recipient, uint256 amount, string reason);
    event MinerRegistered(address indexed miner);
    event HalvingOccurred(uint256 newRewardRate);
    
    modifier onlyVerifier() {
        require(msg.sender == pocwVerifier, "Only PoCW verifier can mine KEY");
        _;
    }
    
    constructor() {
        pocwVerifier = msg.sender;
    }
    
    function setPoCWVerifier(address _verifier) external {
        require(pocwVerifier == msg.sender, "Only current verifier can update");
        pocwVerifier = _verifier;
    }
    
    /**
     * @dev Mine new KEY tokens for successful work
     * @param recipient Address to receive mined KEY
     * @param amount Amount of KEY to mine
     * @param reason Mining reason (e.g., "Miner: 5 successful rounds")
     */
    function mine(address recipient, uint256 amount, string memory reason) external onlyVerifier {
        require(totalSupply + amount <= MAX_SUPPLY, "Would exceed max supply of 21M KEY");
        
        balanceOf[recipient] += amount;
        totalSupply += amount;
        totalMined += amount;
        
        if (!isMiner[recipient]) {
            isMiner[recipient] = true;
            emit MinerRegistered(recipient);
        }
        
        emit KEYMined(recipient, amount, reason);
    }
    
    /**
     * @dev Get remaining minable KEY tokens
     */
    function remainingSupply() public view returns (uint256) {
        return MAX_SUPPLY - totalSupply;
    }
    
    /**
     * @dev Soulbound - transfers are disabled
     */
    function transfer(address, uint256) public pure returns (bool) {
        revert("KEY tokens are soulbound and cannot be transferred");
    }
    
    function transferFrom(address, address, uint256) public pure returns (bool) {
        revert("KEY tokens are soulbound and cannot be transferred");
    }
}