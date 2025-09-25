// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;

import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/security/ReentrancyGuard.sol";
import "@openzeppelin/contracts/utils/Counters.sol";

/**
 * @title SBTIdentity - Soulbound Token for User Identity
 * @dev Non-transferable NFT representing user identity and achievements
 * 
 * Features:
 * - Soulbound (non-transferable)
 * - Metadata stored on IPFS via Pinata
 * - Dynamic attributes via external_url API
 * - User registration and profile management
 */
contract SBTIdentity is ERC721, Ownable, ReentrancyGuard {
    using Counters for Counters.Counter;
    
    // Token ID counter
    Counters.Counter private _tokenIds;
    
    // Events
    event SBTMinted(address indexed to, uint256 indexed tokenId, string tokenURI);
    event MetadataUpdated(uint256 indexed tokenId, string newTokenURI);
    event UserRegistered(address indexed user, uint256 indexed tokenId);
    
    // User registration info
    struct UserInfo {
        string displayName;     
        address walletAddress;  
        address inviter;        
        uint256 registrationDate; 
        bool exists;           
    }
    
    // Mappings
    mapping(uint256 => string) private _tokenURIs;           // tokenId => IPFS URI
    mapping(address => uint256) public userToTokenId;        
    mapping(uint256 => UserInfo) public tokenIdToUserInfo;   
    mapping(address => bool) public authorizedMinters;       
    
    // Configuration
    string public baseExternalURL;  
    
    constructor(
        string memory name,
        string memory symbol,
        string memory _baseExternalURL
    ) ERC721(name, symbol) {
        baseExternalURL = _baseExternalURL;
    }
    
    /**
     * @dev Mint SBT for user registration
     * @param to User address
     * @param displayName User display name
     * @param inviter Inviter address (can be zero address)
     * @param tokenURI IPFS metadata URI
     */
    function mintSBT(
        address to,
        string memory displayName,
        address inviter,
        string memory tokenURI
    ) external nonReentrant returns (uint256) {
        require(authorizedMinters[msg.sender] || msg.sender == owner(), "Not authorized to mint");
        require(to != address(0), "Cannot mint to zero address");
        require(userToTokenId[to] == 0, "User already has SBT");
        require(bytes(displayName).length > 0, "Display name required");
        require(bytes(tokenURI).length > 0, "Token URI required");
        
        // Increment token ID
        _tokenIds.increment();
        uint256 newTokenId = _tokenIds.current();
        
        // Mint token
        _mint(to, newTokenId);
        
        // Set token URI
        _tokenURIs[newTokenId] = tokenURI;
        
        // Store user info
        userToTokenId[to] = newTokenId;
        tokenIdToUserInfo[newTokenId] = UserInfo({
            displayName: displayName,
            walletAddress: to,
            inviter: inviter,
            registrationDate: block.timestamp,
            exists: true
        });
        
        emit SBTMinted(to, newTokenId, tokenURI);
        emit UserRegistered(to, newTokenId);
        
        return newTokenId;
    }
    
    /**
     * @dev Update token URI (for metadata updates)
     * @param tokenId Token ID
     * @param newTokenURI New IPFS metadata URI
     */
    function updateTokenURI(uint256 tokenId, string memory newTokenURI) external {
        require(authorizedMinters[msg.sender] || msg.sender == owner(), "Not authorized");
        require(_exists(tokenId), "Token does not exist");
        require(bytes(newTokenURI).length > 0, "Token URI required");
        
        _tokenURIs[tokenId] = newTokenURI;
        emit MetadataUpdated(tokenId, newTokenURI);
    }
    
    /**
     * @dev Get token URI
     */
    function tokenURI(uint256 tokenId) public view override returns (string memory) {
        require(_exists(tokenId), "Token does not exist");
        return _tokenURIs[tokenId];
    }
    
    /**
     * @dev Get user info by token ID
     */
    function getUserInfo(uint256 tokenId) external view returns (UserInfo memory) {
        require(_exists(tokenId), "Token does not exist");
        return tokenIdToUserInfo[tokenId];
    }
    
    /**
     * @dev Get user info by address
     */
    function getUserInfoByAddress(address user) external view returns (UserInfo memory, uint256 tokenId) {
        tokenId = userToTokenId[user];
        require(tokenId != 0, "User does not have SBT");
        return (tokenIdToUserInfo[tokenId], tokenId);
    }
    
    /**
     * @dev Check if user has SBT
     */
    function hasSBT(address user) external view returns (bool) {
        return userToTokenId[user] != 0;
    }
    
    /**
     * @dev Get total supply
     */
    function totalSupply() external view returns (uint256) {
        return _tokenIds.current();
    }
    
    // ============ SOULBOUND FUNCTIONALITY ============
    
    /**
     * @dev Override transfer functions to make token soulbound
     */
    function _beforeTokenTransfer(
        address from,
        address to,
        uint256 tokenId,
        uint256 batchSize
    ) internal virtual override {
        require(from == address(0), "SBT: token is soulbound and cannot be transferred");
        super._beforeTokenTransfer(from, to, tokenId, batchSize);
    }
    
    /**
     * @dev Disable approve
     */
    function approve(address, uint256) public virtual override {
        revert("SBT: token is soulbound and cannot be approved");
    }
    
    /**
     * @dev Disable setApprovalForAll
     */
    function setApprovalForAll(address, bool) public virtual override {
        revert("SBT: token is soulbound and cannot be approved");
    }
    
    // ============ ADMIN FUNCTIONS ============
    
    /**
     * @dev Add authorized minter
     */
    function addAuthorizedMinter(address minter) external onlyOwner {
        authorizedMinters[minter] = true;
    }
    
    /**
     * @dev Remove authorized minter
     */
    function removeAuthorizedMinter(address minter) external onlyOwner {
        authorizedMinters[minter] = false;
    }
    
    /**
     * @dev Update base external URL
     */
    function updateBaseExternalURL(string memory newBaseURL) external onlyOwner {
        baseExternalURL = newBaseURL;
    }
    
    /**
     * @dev Emergency burn (only owner)
     */
    function emergencyBurn(uint256 tokenId) external onlyOwner {
        require(_exists(tokenId), "Token does not exist");
        
        address owner = ownerOf(tokenId);
        
        // Clear mappings
        userToTokenId[owner] = 0;
        delete tokenIdToUserInfo[tokenId];
        delete _tokenURIs[tokenId];
        
        // Burn token
        _burn(tokenId);
    }
}
