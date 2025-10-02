// SPDX-License-Identifier: MIT
pragma solidity ^0.8.19;
import "@openzeppelin/contracts/token/ERC721/ERC721.sol";
import "@openzeppelin/contracts/token/ERC721/extensions/ERC721URIStorage.sol";
import "@openzeppelin/contracts/token/ERC721/extensions/ERC721Enumerable.sol";
import "@openzeppelin/contracts/access/Ownable.sol";
import "@openzeppelin/contracts/utils/Strings.sol";
contract FluxNFT is ERC721, ERC721URIStorage, ERC721Enumerable, Ownable {
    using Strings for uint256;
    uint256 private _tokenIdCounter;
    // Base URI
    string private _baseTokenURI;
    // Maximum supply
    uint256 public constant MAX_SUPPLY = 10000;
    // Mint price
    uint256 public mintPrice = 0;
    // Number of tokens minted by each address (kept for statistics, no longer restricted)
    mapping(address => uint256) public mintedCount;
    // Metadata URI mapping
    mapping(uint256 => string) private _tokenURIs;
    // Events
    event Minted(address indexed to, uint256 indexed tokenId, string tokenURI);
    event PriceUpdated(uint256 oldPrice, uint256 newPrice);
    event BaseURIUpdated(string oldURI, string newURI);
    constructor(
        string memory name,
        string memory symbol,
        string memory baseTokenURI
    ) ERC721(name, symbol) Ownable() {
        _baseTokenURI = baseTokenURI;
    }
    // Public mint function
    function mint(string memory _tokenURI) public payable {
        require(totalSupply() < MAX_SUPPLY, "Max supply reached");
        require(msg.value >= mintPrice, "Insufficient payment");
        uint256 tokenId = _tokenIdCounter;
        _tokenIdCounter++;
        _safeMint(msg.sender, tokenId);
        _setTokenURI(tokenId, _tokenURI);
        mintedCount[msg.sender]++;
        emit Minted(msg.sender, tokenId, _tokenURI);
    }
    // Batch mint function
    function mintBatch(string[] memory tokenURIs) public payable {
        require(totalSupply() + tokenURIs.length <= MAX_SUPPLY, "Exceeds max supply");
        require(msg.value >= mintPrice * tokenURIs.length, "Insufficient payment");
        for (uint256 i = 0; i < tokenURIs.length; i++) {
            uint256 tokenId = _tokenIdCounter;
            _tokenIdCounter++;
            _safeMint(msg.sender, tokenId);
            _setTokenURI(tokenId, tokenURIs[i]);
            emit Minted(msg.sender, tokenId, tokenURIs[i]);
        }
        mintedCount[msg.sender] += tokenURIs.length;
    }
    // Admin free mint function
    function adminMint(address to, string memory _tokenURI) public onlyOwner {
        require(totalSupply() < MAX_SUPPLY, "Max supply reached");
        uint256 tokenId = _tokenIdCounter;
        _tokenIdCounter++;
        _safeMint(to, tokenId);
        _setTokenURI(tokenId, _tokenURI);
        emit Minted(to, tokenId, _tokenURI);
    }
    // Set mint price
    function setMintPrice(uint256 newPrice) public onlyOwner {
        uint256 oldPrice = mintPrice;
        mintPrice = newPrice;
        emit PriceUpdated(oldPrice, newPrice);
    }
    // Set base URI
    function setBaseURI(string memory newBaseURI) public onlyOwner {
        string memory oldURI = _baseTokenURI;
        _baseTokenURI = newBaseURI;
        emit BaseURIUpdated(oldURI, newBaseURI);
    }
    // Withdraw contract balance
    function withdraw() public onlyOwner {
        uint256 balance = address(this).balance;
        require(balance > 0, "No funds to withdraw");
        (bool success, ) = payable(owner()).call{value: balance}("");
        require(success, "Withdrawal failed");
    }
    // Get contract balance
    function getBalance() public view returns (uint256) {
        return address(this).balance;
    }
    // Get current token ID
    function getCurrentTokenId() public view returns (uint256) {
        return _tokenIdCounter;
    }
    // Get user minted count
    function getUserMintedCount(address user) public view returns (uint256) {
        return mintedCount[user];
    }
    // Override functions to support multiple inheritance
    function _baseURI() internal view override returns (string memory) {
        return _baseTokenURI;
    }
    function tokenURI(uint256 tokenId) public view override(ERC721, ERC721URIStorage) returns (string memory) {
        return super.tokenURI(tokenId);
    }
    function supportsInterface(bytes4 interfaceId) public view override(ERC721, ERC721Enumerable, ERC721URIStorage) returns (bool) {
        return super.supportsInterface(interfaceId);
    }
    function _beforeTokenTransfer(address from, address to, uint256 firstTokenId, uint256 batchSize) internal override(ERC721, ERC721Enumerable) {
        super._beforeTokenTransfer(from, to, firstTokenId, batchSize);
    }
    function _burn(uint256 tokenId) internal override(ERC721, ERC721URIStorage) {
        super._burn(tokenId);
    }
}