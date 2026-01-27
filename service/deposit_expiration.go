package service

import (
	"context"
	"time"

	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/logger"
	"go.uber.org/zap"
)

type DepositExpirationService struct {
	store *db.Store
}

func NewDepositExpirationService(store *db.Store) *DepositExpirationService {
	return &DepositExpirationService{
		store: store,
	}
}

// Start begins the expiration checker (runs every 5 seconds)
func (s *DepositExpirationService) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	s.expireRequests(ctx)

	for {
		select {
		case <-ticker.C:
			s.expireRequests(ctx)
		case <-ctx.Done():
			logger.Info("Deposit expiration service stopped")
			return
		}
	}
}

func (s *DepositExpirationService) expireRequests(ctx context.Context) {
	err := s.store.ExpireDepositRequests(ctx)
	if err != nil {
		logger.Error("Failed to expire deposit requests", zap.Error(err))
		return
	}
}
