package cryptocurrency

import (
	"context"
	"fmt"
	"math/big"

	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/ethereum/go-ethereum/ethclient"
)

// Network represents a blockchain network configuration
type Network struct {
	Name         string
	ChainID      int64
	RPCURL       string
	USDTContract string
}

// Predefined networks
var (
	EthereumMainnet = Network{
		Name:         "Ethereum Mainnet",
		ChainID:      1,
		RPCURL:       "https://eth.llamarpc.com",
		USDTContract: "0xdAC17F958D2ee523a2206206994597C13D831ec7",
	}

	PolygonMainnet = Network{
		Name:    "Polygon Mainnet",
		ChainID: 137,
		//its either https://polygon-rpc.com or https://1rpc.io/matic
		RPCURL:       "https://1rpc.io/matic",
		USDTContract: "0xc2132D05D31c914a87C6611C10748AEb04B58e8F",
	}

	BSCMainnet = Network{
		Name:         "BNB Smart Chain",
		ChainID:      56,
		RPCURL:       "https://bsc-dataseed.binance.org",
		USDTContract: "0x55d398326f99059fF775485246999027B3197955",
	}

	// Testnets
	SepoliaTestnet = Network{
		Name:         "Sepolia Testnet",
		ChainID:      11155111,
		RPCURL:       "https://rpc.sepolia.org",
		USDTContract: "", // USDT may not be available on testnet
	}

	PolygonMumbai = Network{
		Name:         "Polygon Mumbai",
		ChainID:      80001,
		RPCURL:       "https://rpc-mumbai.maticvigil.com",
		USDTContract: "",
	}
)

// USDTToSmallestUnit converts USDT amount to smallest unit (6 decimals)
// Example: 1.5 USDT -> 1500000
func USDTToSmallestUnit(usdt float64) *big.Int {
	// USDT has 6 decimals, so multiply by 1,000,000
	multiplier := big.NewFloat(1e6)
	amount := big.NewFloat(usdt)
	result := new(big.Float).Mul(amount, multiplier)

	// Convert to big.Int
	resultInt := new(big.Int)
	result.Int(resultInt)
	return resultInt
}

// EtherToWei converts ETH/POL/BNB amount to Wei (18 decimals)
// Example: 0.001 ETH -> 1000000000000000
func EtherToWei(ether float64) *big.Int {
	// ETH/POL/BNB has 18 decimals
	multiplier := big.NewFloat(1e18)
	amount := big.NewFloat(ether)
	result := new(big.Float).Mul(amount, multiplier)

	// Convert to big.Int
	resultInt := new(big.Int)
	result.Int(resultInt)
	return resultInt
}

// Client represents a blockchain client for a specific network
type Client struct {
	network *Network
	client  *ethclient.Client
}

// NewClient creates a new blockchain client for a specific network
func NewClient(network Network) (*Client, error) {
	client, err := ethclient.Dial(network.RPCURL)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %w", network.Name, err)
	}

	return &Client{
		network: &network,
		client:  client,
	}, nil
}

// Close closes the client connection
func (c *Client) Close() {
	c.client.Close()
}

// GetBalance returns the native token balance (ETH/MATIC/BNB) in Wei
func (c *Client) GetBalance(address string) (*big.Int, error) {
	account := common.HexToAddress(address)
	balance, err := c.client.BalanceAt(context.Background(), account, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get balance: %w", err)
	}
	return balance, nil
}

// GetBalanceEther returns the balance in Ether/MATIC/BNB as a float64
func (c *Client) GetBalanceEther(address string) (float64, error) {
	balance, err := c.GetBalance(address)
	if err != nil {
		return 0, err
	}

	fbalance, _ := new(big.Float).SetString(balance.String())
	ethValue := new(big.Float).Quo(fbalance, big.NewFloat(1e18))
	result, _ := ethValue.Float64()
	return result, nil
}

// GetUSDTBalance returns the USDT balance for an address
func (c *Client) GetUSDTBalance(address string) (*big.Int, error) {
	if c.network.USDTContract == "" {
		return nil, fmt.Errorf("USDT contract not available on %s", c.network.Name)
	}

	// ERC20 balanceOf function selector: 0x70a08231
	data := common.Hex2Bytes("70a08231000000000000000000000000" + address[2:])

	msg := ethereum.CallMsg{
		To:   &common.Address{},
		Data: data,
	}
	copy(msg.To[:], common.HexToAddress(c.network.USDTContract).Bytes())

	result, err := c.client.CallContract(context.Background(), msg, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get USDT balance: %w", err)
	}

	balance := new(big.Int).SetBytes(result)
	return balance, nil
}

// GetUSDTBalanceFloat returns USDT balance as a float64 (with 6 decimals for USDT)
func (c *Client) GetUSDTBalanceFloat(address string) (float64, error) {
	balance, err := c.GetUSDTBalance(address)
	if err != nil {
		return 0, err
	}

	// USDT has 6 decimals
	fbalance, _ := new(big.Float).SetString(balance.String())
	usdtValue := new(big.Float).Quo(fbalance, big.NewFloat(1e6))
	result, _ := usdtValue.Float64()
	return result, nil
}

// TransferUSDT transfers USDT tokens to a recipient
func (c *Client) TransferUSDT(wallet *Wallet, toAddress string, amount *big.Int) (string, error) {
	if c.network.USDTContract == "" {
		return "", fmt.Errorf("USDT contract not available on %s", c.network.Name)
	}

	// Get nonce
	fromAddress := common.HexToAddress(wallet.Address)
	nonce, err := c.client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := c.client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %w", err)
	}

	// ERC20 transfer function: transfer(address,uint256)
	// Function selector: 0xa9059cbb
	toAddr := common.HexToAddress(toAddress)
	data := common.Hex2Bytes("a9059cbb")

	// Pad address to 32 bytes
	paddedAddr := common.LeftPadBytes(toAddr.Bytes(), 32)
	data = append(data, paddedAddr...)

	// Pad amount to 32 bytes
	paddedAmount := common.LeftPadBytes(amount.Bytes(), 32)
	data = append(data, paddedAmount...)

	// Estimate gas
	gasLimit := uint64(100000) // Standard ERC20 transfer

	// Create transaction
	contractAddress := common.HexToAddress(c.network.USDTContract)
	tx := types.NewTransaction(
		nonce,
		contractAddress,
		big.NewInt(0), // value is 0 for token transfer
		gasLimit,
		gasPrice,
		data,
	)

	// Sign transaction
	chainID := big.NewInt(c.network.ChainID)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), wallet.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = c.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	return signedTx.Hash().Hex(), nil
}

// TransferNative transfers native tokens (ETH/MATIC/BNB)
func (c *Client) TransferNative(wallet *Wallet, toAddress string, amount *big.Int) (string, error) {
	fromAddress := common.HexToAddress(wallet.Address)

	// Get nonce
	nonce, err := c.client.PendingNonceAt(context.Background(), fromAddress)
	if err != nil {
		return "", fmt.Errorf("failed to get nonce: %w", err)
	}

	// Get gas price
	gasPrice, err := c.client.SuggestGasPrice(context.Background())
	if err != nil {
		return "", fmt.Errorf("failed to get gas price: %w", err)
	}

	// Gas limit for simple transfer
	gasLimit := uint64(21000)

	// Create transaction
	toAddr := common.HexToAddress(toAddress)
	tx := types.NewTransaction(
		nonce,
		toAddr,
		amount,
		gasLimit,
		gasPrice,
		nil,
	)

	// Sign transaction
	chainID := big.NewInt(c.network.ChainID)
	signedTx, err := types.SignTx(tx, types.NewEIP155Signer(chainID), wallet.PrivateKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign transaction: %w", err)
	}

	// Send transaction
	err = c.client.SendTransaction(context.Background(), signedTx)
	if err != nil {
		return "", fmt.Errorf("failed to send transaction: %w", err)
	}

	return signedTx.Hash().Hex(), nil
}

// GetTransactionReceipt gets the receipt of a transaction
func (c *Client) GetTransactionReceipt(txHash string) (*types.Receipt, error) {
	hash := common.HexToHash(txHash)
	receipt, err := c.client.TransactionReceipt(context.Background(), hash)
	if err != nil {
		return nil, fmt.Errorf("failed to get transaction receipt: %w", err)
	}
	return receipt, nil
}

// GetChainID returns the chain ID of the network
func (c *Client) GetChainID() (*big.Int, error) {
	return c.client.ChainID(context.Background())
}

// GetBlockNumber returns the latest block number
func (c *Client) GetBlockNumber() (uint64, error) {
	return c.client.BlockNumber(context.Background())
}
