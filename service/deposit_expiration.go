package service

import (
	"context"
	"time"

	"github.com/angpaoprw/cryptocurrency/api"
	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/logger"
	"go.uber.org/zap"
)

type DepositExpirationService struct {
	store              *db.Store
	notificationClient *api.NotificationClient
}

func NewDepositExpirationService(store *db.Store, notificationClient *api.NotificationClient) *DepositExpirationService {
	return &DepositExpirationService{
		store:              store,
		notificationClient: notificationClient,
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
	expiredRequests, err := s.store.ExpireDepositRequests(ctx)
	if err != nil {
		logger.Error("Failed to expire deposit requests", zap.Error(err))
		return
	}

	// Send notifications for expired deposits
	if s.notificationClient != nil && len(expiredRequests) > 0 {
		logger.Info("Expired deposit requests", zap.Int("count", len(expiredRequests)))

		for _, req := range expiredRequests {
			s.notificationClient.SendNotification(req.CustomerID, api.EventDepositExpired, map[string]interface{}{
				"request_id":      req.ID.String(),
				"deposit_address": req.AssignedAddress,
				"network":         req.Network,
				"token":           req.Token,
				"status":          "expired",
				"expired_at":      time.Now().Format(time.RFC3339),
			})
		}
	}
}
