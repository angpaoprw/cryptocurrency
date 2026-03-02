package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/angpaoprw/cryptocurrency/api"
	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/logger"
	"github.com/google/uuid"
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
	if s.store == nil {
		logger.Error("Store is nil in expireRequests")
		return
	}

	expiredRequests, err := s.store.ExpireDepositRequests(ctx)
	if err != nil {
		logger.Error("Failed to expire deposit requests", zap.Error(err))
		return
	}

	// Send notifications for expired deposits
	if s.notificationClient != nil && len(expiredRequests) > 0 {
		logger.Info("Expired deposit requests", zap.Int("count", len(expiredRequests)))

		for _, req := range expiredRequests {
			body := map[string]interface{}{
				"request_id":      req.ID.String(),
				"deposit_address": req.AssignedAddress,
				"network":         req.Network,
				"token":           req.Token,
				"status":          "expired",
				"expired_at":      time.Now().Format(time.RFC3339),
			}
			if req.RefID.Valid {
				body["ref_id"] = req.RefID.String
			}
			err = s.notificationClient.SendNotification(req.CustomerID, api.EventDepositExpired, body)
			if err != nil {
				logger.Warn("Failed to send deposit expiration notification",
					zap.String("customer_id", req.CustomerID),
					zap.String("request_id", req.ID.String()),
					zap.Error(err),
				)

				body_byte, marshalErr := json.Marshal(body)
				if marshalErr != nil {
					logger.Error("Failed to marshal notification body",
						zap.Any("body", body),
						zap.Error(marshalErr),
					)
					continue
				}

				create_failed_api_req_param := db.CreateFailedAPIRequestParams{
					DepositRequestID: uuid.NullUUID{Valid: true, UUID: req.ID},
					Body:             body_byte,
					Error:            err.Error(),
				}

				if _, createErr := s.store.CreateFailedAPIRequest(ctx, create_failed_api_req_param); createErr != nil {
					logger.Error("Failed to create failed API request record",
						zap.String("request_id", req.ID.String()),
						zap.Error(createErr),
					)
				}
			}
		}
	}
}
