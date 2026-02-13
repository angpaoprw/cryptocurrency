package cryptocurrency

import (
	"math/big"
	"testing"
)

const wallet_address = "0xa1cc6701a88Cca21a1F694A1e081440c7cEbE073"

// TestGetBalance tests getting native token balance (ETH/MATIC/BNB)
func TestGetBalance(t *testing.T) {
	t.Skip("Skipping - requires RPC endpoint. Remove t.Skip() to run with real network")

	// Connect to Polygon mainnet
	client, err := NewClient(PolygonMainnet)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test with a known address (Polygon USDT contract address)
	address := wallet_address

	balance, err := client.GetBalance(address)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}

	if balance == nil {
		t.Fatal("Balance is nil")
	}

	t.Logf("Balance (Wei): %s", balance.String())

	// Balance should be non-negative
	if balance.Sign() < 0 {
		t.Error("Balance should not be negative")
	}
}

// TestGetBalanceEther tests getting balance in Ether format
func TestGetBalanceEther(t *testing.T) {
	t.Skip("Skipping - requires RPC endpoint. Remove t.Skip() to run with real network")

	client, err := NewClient(EthereumMainnet)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Vitalik's address (public example with known balance)
	address := wallet_address

	balance, err := client.GetBalanceEther(address)
	if err != nil {
		t.Fatalf("Failed to get balance in ether: %v", err)
	}

	t.Logf("Balance (ETH): %f", balance)

	// Vitalik's address should have some ETH
	if balance < 0 {
		t.Error("Balance should not be negative")
	}
}

// TestGetBalanceMultipleNetworks tests balance across different networks
func TestGetBalanceMultipleNetworks(t *testing.T) {
	t.Skip("Skipping - requires RPC endpoint. Remove t.Skip() to run with real network")

	// Create a test wallet
	wallet, err := CreateWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	t.Logf("Testing wallet: %s", wallet.Address)

	networks := []Network{EthereumMainnet, PolygonMainnet, BSCMainnet}

	for _, network := range networks {
		t.Run(network.Name, func(t *testing.T) {
			client, err := NewClient(network)
			if err != nil {
				t.Fatalf("Failed to create client for %s: %v", network.Name, err)
			}
			defer client.Close()

			balance, err := client.GetBalanceEther(wallet.Address)
			if err != nil {
				t.Fatalf("Failed to get balance on %s: %v", network.Name, err)
			}

			t.Logf("%s balance: %f", network.Name, balance)

			// New wallet should have 0 balance
			if balance != 0 {
				t.Logf("Warning: New wallet has non-zero balance on %s", network.Name)
			}
		})
	}
}

// TestGetUSDTBalance tests getting USDT token balance
func TestGetUSDTBalance(t *testing.T) {
	t.Skip("Skipping - requires RPC endpoint. Remove t.Skip() to run with real network")

	client, err := NewClient(PolygonMainnet)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test with a known address that might have USDT
	address := wallet_address

	balance, err := client.GetUSDTBalance(address)
	if err != nil {
		t.Fatalf("Failed to get USDT balance: %v", err)
	}

	if balance == nil {
		t.Fatal("USDT balance is nil")
	}

	t.Logf("USDT Balance (raw): %s", balance.String())

	if balance.Sign() < 0 {
		t.Error("USDT balance should not be negative")
	}
}

// TestGetUSDTBalanceFloat tests getting USDT balance as float
func TestGetUSDTBalanceFloat(t *testing.T) {
	// Testing on Polygon network

	client, err := NewClient(PolygonMainnet)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Polygon USDT contract address (this address likely has USDT)
	address := wallet_address

	balance, err := client.GetUSDTBalanceFloat(address)
	if err != nil {
		t.Fatalf("Failed to get USDT balance: %v", err)
	}

	t.Logf("USDT Balance: %f", balance)

	if balance < 0 {
		t.Error("USDT balance should not be negative")
	}
}

// TestGetPOLBalance tests getting POL (MATIC) token balance on Polygon
func TestGetPOLBalance(t *testing.T) {
	// Testing on Polygon network

	client, err := NewClient(PolygonMainnet)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// Test address
	address := wallet_address

	balance, err := client.GetBalanceEther(address)
	if err != nil {
		t.Fatalf("Failed to get POL balance: %v", err)
	}

	t.Logf("POL/MATIC Balance: %f", balance)

	if balance < 0 {
		t.Error("POL balance should not be negative")
	}
}

// TestTransferNative tests transferring native tokens (POL/MATIC)
func TestTransferNative(t *testing.T) {
	t.Skip("⚠️  WARNING: This test will spend real money! Remove t.Skip() and update private key to run")

	// Connect to Polygon
	client, err := NewClient(PolygonMainnet)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// IMPORTANT: Replace with your private key (DO NOT commit to git!)
	privateKey := "your-private-key-here"
	wallet, err := ImportWalletFromPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to import wallet: %v", err)
	}

	t.Logf("From wallet: %s", wallet.Address)

	// Check balance before transfer
	balanceBefore, err := client.GetBalanceEther(wallet.Address)
	if err != nil {
		t.Fatalf("Failed to get balance: %v", err)
	}
	t.Logf("Balance before: %f POL", balanceBefore)

	if balanceBefore < 0.01 {
		t.Fatal("Insufficient balance. Need at least 0.01 POL for test")
	}

	// Recipient address (replace with your test address)
	toAddress := "0xRecipientAddressHere"

	// Transfer 0.001 POL (1000000000000000 Wei)
	amount := big.NewInt(1000000000000000)

	t.Logf("Transferring 0.001 POL to %s", toAddress)

	txHash, err := client.TransferNative(wallet, toAddress, amount)
	if err != nil {
		t.Fatalf("Failed to transfer: %v", err)
	}

	t.Logf("Transaction hash: %s", txHash)
	t.Logf("View on PolygonScan: https://polygonscan.com/tx/%s", txHash)

	// Wait a bit for transaction to be mined
	t.Log("Waiting for transaction to be mined...")
	// Note: In production, you should poll for receipt instead of sleeping
}

// TestTransferUSDT tests transferring USDT tokens
func TestTransferUSDT(t *testing.T) {
	// ⚠️  WARNING: This test will spend real money!
	// Make sure to update private key and recipient address below

	// Connect to Polygon
	client, err := NewClient(PolygonMainnet)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	// IMPORTANT: Replace with your private key (DO NOT commit to git!)
	privateKey := "" // TODO: Add your private key here
	if privateKey == "" {
		t.Fatal("Please set your private key in the test")
	}

	wallet, err := ImportWalletFromPrivateKey(privateKey)
	if err != nil {
		t.Fatalf("Failed to import wallet: %v", err)
	}

	t.Logf("From wallet: %s", wallet.Address)

	// Check USDT balance before transfer
	usdtBefore, err := client.GetUSDTBalanceFloat(wallet.Address)
	if err != nil {
		t.Fatalf("Failed to get USDT balance: %v", err)
	}
	t.Logf("USDT balance before: %f", usdtBefore)

	if usdtBefore < 1.0 {
		t.Fatal("Insufficient USDT balance. Need at least 1 USDT for test")
	}

	// Check POL balance for gas
	polBalance, err := client.GetBalanceEther(wallet.Address)
	if err != nil {
		t.Fatalf("Failed to get POL balance: %v", err)
	}
	t.Logf("POL balance (for gas): %f", polBalance)

	if polBalance < 0.01 {
		t.Fatal("Insufficient POL for gas fees. Need at least 0.01 POL")
	}

	// Recipient address (replace with your test address)
	toAddress := "" // TODO: Add recipient address here
	if toAddress == "" {
		t.Fatal("Please set recipient address in the test")
	}

	// Transfer amount in USDT (you can change this)
	usdtAmount := 0.1 // Change this to any amount you want to send

	// Convert USDT amount to smallest unit (USDT has 6 decimals)
	// 1 USDT = 1,000,000 smallest units
	amount := USDTToSmallestUnit(usdtAmount)

	t.Logf("Transferring %f USDT to %s", usdtAmount, toAddress)

	txHash, err := client.TransferUSDT(wallet, toAddress, amount)
	if err != nil {
		t.Fatalf("Failed to transfer USDT: %v", err)
	}

	t.Logf("Transaction hash: %s", txHash)
	t.Logf("View on PolygonScan: https://polygonscan.com/tx/%s", txHash)

	// Wait a bit for transaction to be mined
	t.Log("Waiting for transaction to be mined...")
}

// TestGetChainID tests getting chain ID
func TestGetChainID(t *testing.T) {
	t.Skip("Skipping - requires RPC endpoint. Remove t.Skip() to run with real network")

	tests := []struct {
		name     string
		network  Network
		expected int64
	}{
		{"Ethereum", EthereumMainnet, 1},
		{"Polygon", PolygonMainnet, 137},
		{"BSC", BSCMainnet, 56},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client, err := NewClient(tt.network)
			if err != nil {
				t.Fatalf("Failed to create client: %v", err)
			}
			defer client.Close()

			chainID, err := client.GetChainID()
			if err != nil {
				t.Fatalf("Failed to get chain ID: %v", err)
			}

			if chainID.Int64() != tt.expected {
				t.Errorf("Chain ID mismatch. Expected %d, got %d", tt.expected, chainID.Int64())
			}

			t.Logf("%s Chain ID: %d", tt.name, chainID.Int64())
		})
	}
}

// TestGetBlockNumber tests getting the latest block number
func TestGetBlockNumber(t *testing.T) {
	t.Skip("Skipping - requires RPC endpoint. Remove t.Skip() to run with real network")

	client, err := NewClient(EthereumMainnet)
	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}
	defer client.Close()

	blockNumber, err := client.GetBlockNumber()
	if err != nil {
		t.Fatalf("Failed to get block number: %v", err)
	}

	t.Logf("Latest block number: %d", blockNumber)

	// Block number should be greater than 0
	if blockNumber == 0 {
		t.Error("Block number should be greater than 0")
	}
}

// TestBalanceConversion tests Wei to Ether conversion
func TestBalanceConversion(t *testing.T) {
	tests := []struct {
		wei      string
		expected float64
	}{
		{"1000000000000000000", 1.0}, // 1 ETH
		{"500000000000000000", 0.5},  // 0.5 ETH
		{"100000000000000000", 0.1},  // 0.1 ETH
		{"1000000000000000", 0.001},  // 0.001 ETH
		{"0", 0},                     // 0 ETH
	}

	for _, tt := range tests {
		t.Run(tt.wei, func(t *testing.T) {
			wei := new(big.Int)
			wei.SetString(tt.wei, 10)

			fbalance, _ := new(big.Float).SetString(wei.String())
			ethValue := new(big.Float).Quo(fbalance, big.NewFloat(1e18))
			result, _ := ethValue.Float64()

			if result != tt.expected {
				t.Errorf("Conversion failed. Expected %f, got %f", tt.expected, result)
			}
		})
	}
}

// TestUSDTConversion tests USDT smallest unit to float conversion
func TestUSDTConversion(t *testing.T) {
	tests := []struct {
		raw      string
		expected float64
	}{
		{"1000000", 1.0},   // 1 USDT
		{"500000", 0.5},    // 0.5 USDT
		{"100000", 0.1},    // 0.1 USDT
		{"10000000", 10.0}, // 10 USDT
		{"0", 0},           // 0 USDT
	}

	for _, tt := range tests {
		t.Run(tt.raw, func(t *testing.T) {
			raw := new(big.Int)
			raw.SetString(tt.raw, 10)

			fbalance, _ := new(big.Float).SetString(raw.String())
			usdtValue := new(big.Float).Quo(fbalance, big.NewFloat(1e6))
			result, _ := usdtValue.Float64()

			if result != tt.expected {
				t.Errorf("Conversion failed. Expected %f, got %f", tt.expected, result)
			}
		})
	}
}
