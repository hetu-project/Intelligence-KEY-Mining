package services

import (
	"context"
	"crypto/ecdsa"
	"fmt"
	"log"
	"math/big"
	"os"
	"strings"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/accounts/abi"
	"github.com/ethereum/go-ethereum/accounts/abi/bind"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/ethereum/go-ethereum/ethclient"
)

type BlockchainService struct {
	client          *ethclient.Client
	contractAddress common.Address
	contractABI     abi.ABI
	defaultSigner   *bind.TransactOpts
}

func NewBlockchainService() (*BlockchainService, error) {
	// Connect to Ethereum node
	rpcURL := os.Getenv("ETH_RPC_URL")
	if rpcURL == "" {
		return nil, fmt.Errorf("ETH_RPC_URL environment variable is required")
	}

	client, err := ethclient.Dial(rpcURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Ethereum client: %v", err)
	}

	// Get contract address
	contractAddr := os.Getenv("SBT_CONTRACT_ADDRESS")
	if contractAddr == "" {
		return nil, fmt.Errorf("SBT_CONTRACT_ADDRESS environment variable is required")
	}

	// Parse contract ABI
	contractABI, err := parseContractABI()
	if err != nil {
		return nil, fmt.Errorf("failed to parse contract ABI: %v", err)
	}

	// Set up default signer (for gas fee sponsorship)
	defaultSigner, err := setupDefaultSigner(client)
	if err != nil {
		return nil, fmt.Errorf("failed to setup default signer: %v", err)
	}

	service := &BlockchainService{
		client:          client,
		contractAddress: common.HexToAddress(contractAddr),
		contractABI:     contractABI,
		defaultSigner:   defaultSigner,
	}

	return service, nil
}

// CheckContractDeployed checks if the contract is deployed
func (bs *BlockchainService) CheckContractDeployed(ctx context.Context) error {
	code, err := bs.client.CodeAt(ctx, bs.contractAddress, nil)
	if err != nil {
		return fmt.Errorf("failed to get contract code: %v", err)
	}

	if len(code) == 0 {
		return fmt.Errorf("contract not deployed at address %s. Please deploy the SBT contract first", bs.contractAddress.Hex())
	}

	log.Printf("✅ SBT contract verified at address: %s", bs.contractAddress.Hex())
	return nil
}

// MintSBT mints an SBT token
func (bs *BlockchainService) MintSBT(ctx context.Context, toAddress, tokenURI string) (*big.Int, error) {
	// Check user balance
	userAddr := common.HexToAddress(toAddress)
	balance, err := bs.client.BalanceAt(ctx, userAddr, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get user balance: %v", err)
	}

	// Estimate gas cost
	gasLimit := uint64(200000) // Estimated gas limit
	gasPrice, err := bs.client.SuggestGasPrice(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get gas price: %v", err)
	}

	estimatedCost := new(big.Int).Mul(gasPrice, big.NewInt(int64(gasLimit)))

	var signer *bind.TransactOpts
	if balance.Cmp(estimatedCost) < 0 {
		// User balance insufficient, use default signer for gas sponsorship
		log.Printf("User balance insufficient (%s ETH), using default signer for gas", balance.String())
		signer = bs.defaultSigner
	} else {
		// User balance sufficient, let user pay gas themselves
		// Note: This requires user's private key, in real scenarios should use frontend wallet signing
		log.Printf("User has sufficient balance (%s ETH), but using default signer for simplicity", balance.String())
		signer = bs.defaultSigner
	}

	// Prepare contract call data
	_, err = bs.contractABI.Pack("mintSBT", userAddr, tokenURI)
	if err != nil {
		return nil, fmt.Errorf("failed to pack contract call: %v", err)
	}

	// Create transaction
	nonce, err := bs.client.PendingNonceAt(ctx, signer.From)
	if err != nil {
		return nil, fmt.Errorf("failed to get nonce: %v", err)
	}

	tx := &bind.TransactOpts{
		From:     signer.From,
		Nonce:    big.NewInt(int64(nonce)),
		Value:    big.NewInt(0),
		GasLimit: gasLimit,
		GasPrice: gasPrice,
		Signer:   signer.Signer,
		Context:  ctx,
	}

	// Call contract
	boundContract := bind.NewBoundContract(bs.contractAddress, bs.contractABI, bs.client, bs.client, bs.client)
	transaction, err := boundContract.Transact(tx, "mintSBT", userAddr, tokenURI)
	if err != nil {
		return nil, fmt.Errorf("failed to mint SBT: %v", err)
	}

	log.Printf("SBT minting transaction sent: %s", transaction.Hash().Hex())

	// Wait for transaction confirmation
	receipt, err := bind.WaitMined(ctx, bs.client, transaction)
	if err != nil {
		return nil, fmt.Errorf("failed to wait for transaction confirmation: %v", err)
	}

	if receipt.Status != 1 {
		return nil, fmt.Errorf("transaction failed with status: %d", receipt.Status)
	}

	// Parse event to get tokenId
	tokenId, err := bs.parseTokenIdFromReceipt(receipt)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token ID from receipt: %v", err)
	}

	log.Printf("✅ SBT minted successfully. Token ID: %s, Transaction: %s", tokenId.String(), transaction.Hash().Hex())
	return tokenId, nil
}

// GetTokenURI gets the URI of a token
func (bs *BlockchainService) GetTokenURI(ctx context.Context, tokenId *big.Int) (string, error) {
	// Prepare contract call
	input, err := bs.contractABI.Pack("tokenURI", tokenId)
	if err != nil {
		return "", fmt.Errorf("failed to pack tokenURI call: %v", err)
	}

	// Call contract
	result, err := bs.client.CallContract(ctx, ethereum.CallMsg{
		To:   &bs.contractAddress,
		Data: input,
	}, nil)
	if err != nil {
		return "", fmt.Errorf("failed to call tokenURI: %v", err)
	}

	// Parse result
	var tokenURI string
	err = bs.contractABI.UnpackIntoInterface(&tokenURI, "tokenURI", result)
	if err != nil {
		return "", fmt.Errorf("failed to unpack tokenURI result: %v", err)
	}

	return tokenURI, nil
}

// parseContractABI parses the contract ABI
func parseContractABI() (abi.ABI, error) {
	// Key ABI definitions for SBT contract
	abiJSON := `[
    {
      "inputs": [
        {
          "internalType": "string",
          "name": "name",
          "type": "string"
        },
        {
          "internalType": "string",
          "name": "symbol",
          "type": "string"
        },
        {
          "internalType": "string",
          "name": "_baseExternalURL",
          "type": "string"
        }
      ],
      "stateMutability": "nonpayable",
      "type": "constructor"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "internalType": "address",
          "name": "owner",
          "type": "address"
        },
        {
          "indexed": true,
          "internalType": "address",
          "name": "approved",
          "type": "address"
        },
        {
          "indexed": true,
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        }
      ],
      "name": "Approval",
      "type": "event"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "internalType": "address",
          "name": "owner",
          "type": "address"
        },
        {
          "indexed": true,
          "internalType": "address",
          "name": "operator",
          "type": "address"
        },
        {
          "indexed": false,
          "internalType": "bool",
          "name": "approved",
          "type": "bool"
        }
      ],
      "name": "ApprovalForAll",
      "type": "event"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        },
        {
          "indexed": false,
          "internalType": "string",
          "name": "newTokenURI",
          "type": "string"
        }
      ],
      "name": "MetadataUpdated",
      "type": "event"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "internalType": "address",
          "name": "previousOwner",
          "type": "address"
        },
        {
          "indexed": true,
          "internalType": "address",
          "name": "newOwner",
          "type": "address"
        }
      ],
      "name": "OwnershipTransferred",
      "type": "event"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "internalType": "address",
          "name": "to",
          "type": "address"
        },
        {
          "indexed": true,
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        },
        {
          "indexed": false,
          "internalType": "string",
          "name": "tokenURI",
          "type": "string"
        }
      ],
      "name": "SBTMinted",
      "type": "event"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "internalType": "address",
          "name": "from",
          "type": "address"
        },
        {
          "indexed": true,
          "internalType": "address",
          "name": "to",
          "type": "address"
        },
        {
          "indexed": true,
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        }
      ],
      "name": "Transfer",
      "type": "event"
    },
    {
      "anonymous": false,
      "inputs": [
        {
          "indexed": true,
          "internalType": "address",
          "name": "user",
          "type": "address"
        },
        {
          "indexed": true,
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        }
      ],
      "name": "UserRegistered",
      "type": "event"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "minter",
          "type": "address"
        }
      ],
      "name": "addAuthorizedMinter",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "",
          "type": "address"
        },
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "name": "approve",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "",
          "type": "address"
        }
      ],
      "name": "authorizedMinters",
      "outputs": [
        {
          "internalType": "bool",
          "name": "",
          "type": "bool"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "owner",
          "type": "address"
        }
      ],
      "name": "balanceOf",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "baseExternalURL",
      "outputs": [
        {
          "internalType": "string",
          "name": "",
          "type": "string"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        }
      ],
      "name": "emergencyBurn",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        }
      ],
      "name": "getApproved",
      "outputs": [
        {
          "internalType": "address",
          "name": "",
          "type": "address"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        }
      ],
      "name": "getUserInfo",
      "outputs": [
        {
          "components": [
            {
              "internalType": "string",
              "name": "displayName",
              "type": "string"
            },
            {
              "internalType": "address",
              "name": "walletAddress",
              "type": "address"
            },
            {
              "internalType": "address",
              "name": "inviter",
              "type": "address"
            },
            {
              "internalType": "uint256",
              "name": "registrationDate",
              "type": "uint256"
            },
            {
              "internalType": "bool",
              "name": "exists",
              "type": "bool"
            }
          ],
          "internalType": "struct SBTIdentity.UserInfo",
          "name": "",
          "type": "tuple"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "user",
          "type": "address"
        }
      ],
      "name": "getUserInfoByAddress",
      "outputs": [
        {
          "components": [
            {
              "internalType": "string",
              "name": "displayName",
              "type": "string"
            },
            {
              "internalType": "address",
              "name": "walletAddress",
              "type": "address"
            },
            {
              "internalType": "address",
              "name": "inviter",
              "type": "address"
            },
            {
              "internalType": "uint256",
              "name": "registrationDate",
              "type": "uint256"
            },
            {
              "internalType": "bool",
              "name": "exists",
              "type": "bool"
            }
          ],
          "internalType": "struct SBTIdentity.UserInfo",
          "name": "",
          "type": "tuple"
        },
        {
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "user",
          "type": "address"
        }
      ],
      "name": "hasSBT",
      "outputs": [
        {
          "internalType": "bool",
          "name": "",
          "type": "bool"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "owner",
          "type": "address"
        },
        {
          "internalType": "address",
          "name": "operator",
          "type": "address"
        }
      ],
      "name": "isApprovedForAll",
      "outputs": [
        {
          "internalType": "bool",
          "name": "",
          "type": "bool"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "to",
          "type": "address"
        },
        {
          "internalType": "string",
          "name": "displayName",
          "type": "string"
        },
        {
          "internalType": "address",
          "name": "inviter",
          "type": "address"
        },
        {
          "internalType": "string",
          "name": "tokenURI",
          "type": "string"
        }
      ],
      "name": "mintSBT",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "name",
      "outputs": [
        {
          "internalType": "string",
          "name": "",
          "type": "string"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "owner",
      "outputs": [
        {
          "internalType": "address",
          "name": "",
          "type": "address"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        }
      ],
      "name": "ownerOf",
      "outputs": [
        {
          "internalType": "address",
          "name": "",
          "type": "address"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "minter",
          "type": "address"
        }
      ],
      "name": "removeAuthorizedMinter",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "renounceOwnership",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "from",
          "type": "address"
        },
        {
          "internalType": "address",
          "name": "to",
          "type": "address"
        },
        {
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        }
      ],
      "name": "safeTransferFrom",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "from",
          "type": "address"
        },
        {
          "internalType": "address",
          "name": "to",
          "type": "address"
        },
        {
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        },
        {
          "internalType": "bytes",
          "name": "data",
          "type": "bytes"
        }
      ],
      "name": "safeTransferFrom",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "",
          "type": "address"
        },
        {
          "internalType": "bool",
          "name": "",
          "type": "bool"
        }
      ],
      "name": "setApprovalForAll",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "bytes4",
          "name": "interfaceId",
          "type": "bytes4"
        }
      ],
      "name": "supportsInterface",
      "outputs": [
        {
          "internalType": "bool",
          "name": "",
          "type": "bool"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "symbol",
      "outputs": [
        {
          "internalType": "string",
          "name": "",
          "type": "string"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "name": "tokenIdToUserInfo",
      "outputs": [
        {
          "internalType": "string",
          "name": "displayName",
          "type": "string"
        },
        {
          "internalType": "address",
          "name": "walletAddress",
          "type": "address"
        },
        {
          "internalType": "address",
          "name": "inviter",
          "type": "address"
        },
        {
          "internalType": "uint256",
          "name": "registrationDate",
          "type": "uint256"
        },
        {
          "internalType": "bool",
          "name": "exists",
          "type": "bool"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        }
      ],
      "name": "tokenURI",
      "outputs": [
        {
          "internalType": "string",
          "name": "",
          "type": "string"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [],
      "name": "totalSupply",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "from",
          "type": "address"
        },
        {
          "internalType": "address",
          "name": "to",
          "type": "address"
        },
        {
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        }
      ],
      "name": "transferFrom",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "newOwner",
          "type": "address"
        }
      ],
      "name": "transferOwnership",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "string",
          "name": "newBaseURL",
          "type": "string"
        }
      ],
      "name": "updateBaseExternalURL",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "uint256",
          "name": "tokenId",
          "type": "uint256"
        },
        {
          "internalType": "string",
          "name": "newTokenURI",
          "type": "string"
        }
      ],
      "name": "updateTokenURI",
      "outputs": [],
      "stateMutability": "nonpayable",
      "type": "function"
    },
    {
      "inputs": [
        {
          "internalType": "address",
          "name": "",
          "type": "address"
        }
      ],
      "name": "userToTokenId",
      "outputs": [
        {
          "internalType": "uint256",
          "name": "",
          "type": "uint256"
        }
      ],
      "stateMutability": "view",
      "type": "function"
    }
  ]`

	return abi.JSON(strings.NewReader(abiJSON))
}

// setupDefaultSigner sets up default signer
func setupDefaultSigner(client *ethclient.Client) (*bind.TransactOpts, error) {
	privateKeyHex := os.Getenv("DEFAULT_SIGNER_PRIVATE_KEY")
	if privateKeyHex == "" {
		return nil, fmt.Errorf("DEFAULT_SIGNER_PRIVATE_KEY environment variable is required")
	}

	// Remove 0x prefix
	if strings.HasPrefix(privateKeyHex, "0x") {
		privateKeyHex = privateKeyHex[2:]
	}

	privateKey, err := crypto.HexToECDSA(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("invalid private key: %v", err)
	}

	publicKey := privateKey.Public()
	publicKeyECDSA, ok := publicKey.(*ecdsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("cannot assert type: publicKey is not of type *ecdsa.PublicKey")
	}

	fromAddress := crypto.PubkeyToAddress(*publicKeyECDSA)
	chainID, err := client.ChainID(context.Background())
	if err != nil {
		return nil, fmt.Errorf("failed to get chain ID: %v", err)
	}

	auth, err := bind.NewKeyedTransactorWithChainID(privateKey, chainID)
	if err != nil {
		return nil, fmt.Errorf("failed to create transactor: %v", err)
	}

	log.Printf("Default signer configured: %s (Chain ID: %s)", fromAddress.Hex(), chainID.String())
	return auth, nil
}

// parseTokenIdFromReceipt parses tokenId from transaction receipt
func (bs *BlockchainService) parseTokenIdFromReceipt(receipt *types.Receipt) (*big.Int, error) {
	// Find Transfer event
	transferEventSig := crypto.Keccak256Hash([]byte("Transfer(address,address,uint256)"))

	for _, vLog := range receipt.Logs {
		if len(vLog.Topics) > 0 && vLog.Topics[0] == transferEventSig {
			// The third topic of Transfer event is tokenId
			if len(vLog.Topics) >= 4 {
				tokenId := new(big.Int).SetBytes(vLog.Topics[3].Bytes())
				return tokenId, nil
			}
		}
	}

	return nil, fmt.Errorf("Transfer event not found in receipt")
}

// Close closes connection
func (bs *BlockchainService) Close() {
	if bs.client != nil {
		bs.client.Close()
	}
}
