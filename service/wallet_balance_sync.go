package service

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/angpaoprw/cryptocurrency/cryptocurrency"
	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/logger"
	"github.com/shopspring/decimal"
	"go.uber.org/zap"
)

type WalletBalanceSyncService struct {
	store *db.Store
}

func NewWalletBalanceSyncService(store *db.Store) *WalletBalanceSyncService {
	return &WalletBalanceSyncService{
		store: store,
	}
}

// Start begins the balance sync service (runs every 30 seconds)
func (s *WalletBalanceSyncService) Start(ctx context.Context) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	logger.Info("Wallet balance sync service starting...")

	// Run immediately on start
	s.syncAllWallets(ctx)

	for {
		select {
		case <-ticker.C:
			s.syncAllWallets(ctx)
		case <-ctx.Done():
			logger.Info("Wallet balance sync service stopped")
			return
		}
	}
}

// syncAllWallets fetches and updates balances for all active wallets
func (s *WalletBalanceSyncService) syncAllWallets(ctx context.Context) {
	wallets, err := s.store.GetActiveWallets(ctx)
	if err != nil {
		logger.Error("Failed to get active wallets", zap.Error(err))
		return
	}

	if len(wallets) == 0 {
		logger.Debug("No active wallets to sync")
		return
	}

	logger.Info("Starting balance sync", zap.Int("wallet_count", len(wallets)))

	successCount := 0
	errorCount := 0

	for _, wallet := range wallets {
		if err := s.syncWalletBalance(ctx, wallet); err != nil {
			logger.Error("Failed to sync wallet balance",
				zap.String("wallet_id", wallet.ID.String()),
				zap.String("address", wallet.Address),
				zap.String("network", wallet.Blockchain),
				zap.Error(err),
			)
			errorCount++
		} else {
			successCount++
		}
	}

	logger.Info("Balance sync completed",
		zap.Int("success", successCount),
		zap.Int("errors", errorCount),
		zap.Int("total", len(wallets)),
	)
}

// syncWalletBalance fetches the real balance from blockchain and updates the database
func (s *WalletBalanceSyncService) syncWalletBalance(ctx context.Context, wallet db.Wallet) error {
	// Get the network configuration
	network, err := s.getNetworkConfig(wallet.Blockchain)
	if err != nil {
		return fmt.Errorf("unsupported network %s: %w", wallet.Blockchain, err)
	}

	// Create blockchain client
	client, err := cryptocurrency.NewClient(network)
	if err != nil {
		return fmt.Errorf("failed to create blockchain client: %w", err)
	}
	defer client.Close()

	// Fetch balance based on token type
	var balance decimal.Decimal
	if wallet.Token == "USDT" {
		balanceFloat, err := client.GetUSDTBalanceFloat(wallet.Address)
		if err != nil {
			return fmt.Errorf("failed to get USDT balance: %w", err)
		}
		balance = decimal.NewFromFloat(balanceFloat)
	} else {
		// Native token (ETH, MATIC, BNB, etc.)
		balanceFloat, err := client.GetBalanceEther(wallet.Address)
		if err != nil {
			return fmt.Errorf("failed to get native token balance: %w", err)
		}
		balance = decimal.NewFromFloat(balanceFloat)
	}

	// Check if balance has changed
	currentBalance := decimal.Zero
	if wallet.Balance.Valid {
		currentBalance, _ = decimal.NewFromString(wallet.Balance.String)
	}

	// Update database if balance changed
	if !balance.Equal(currentBalance) {
		_, err = s.store.UpdateWalletBalance(ctx, db.UpdateWalletBalanceParams{
			ID:      wallet.ID,
			Balance: sql.NullString{String: balance.String(), Valid: true},
		})
		if err != nil {
			return fmt.Errorf("failed to update wallet balance: %w", err)
		}

		logger.Info("Wallet balance updated",
			zap.String("wallet_id", wallet.ID.String()),
			zap.String("address", wallet.Address),
			zap.String("network", wallet.Blockchain),
			zap.String("token", wallet.Token),
			zap.String("old_balance", currentBalance.String()),
			zap.String("new_balance", balance.String()),
		)
	} else {
		logger.Debug("Wallet balance unchanged",
			zap.String("wallet_id", wallet.ID.String()),
			zap.String("address", wallet.Address),
			zap.String("balance", balance.String()),
		)
	}

	return nil
}

// getNetworkConfig returns the blockchain network configuration
func (s *WalletBalanceSyncService) getNetworkConfig(networkCode string) (cryptocurrency.Network, error) {
	switch networkCode {
	case "ETH_MAINNET", "ETH", "ethereum":
		return cryptocurrency.EthereumMainnet, nil
	case "POLYGON_MAINNET", "MATIC", "polygon":
		return cryptocurrency.PolygonMainnet, nil
	case "BSC_MAINNET", "BSC", "bnb":
		return cryptocurrency.BSCMainnet, nil
	case "ETH_SEPOLIA", "sepolia":
		return cryptocurrency.SepoliaTestnet, nil
	case "POLYGON_AMOY", "POLYGON_MUMBAI", "mumbai", "amoy":
		return cryptocurrency.PolygonMumbai, nil
	default:
		return cryptocurrency.Network{}, fmt.Errorf("unsupported network: %s", networkCode)
	}
}
