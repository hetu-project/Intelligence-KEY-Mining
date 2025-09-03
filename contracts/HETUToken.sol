// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "../lib/openzeppelin-contracts/contracts/token/ERC20/ERC20.sol";

/**
 * @title HETU Token
 * @dev Staking token for subnet registration. Fixed supply of 1,000,000 HETU.
 * Used for deposits: Miners need 500 HETU, Validators need 100 HETU each.
 */
contract HETUToken is ERC20 {
    uint256 public constant TOTAL_SUPPLY = 1_000_000 * 10**18;
    
    constructor() ERC20("HETU Token", "HETU") {
        _mint(msg.sender, TOTAL_SUPPLY);
    }
}