package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/angpaoprw/cryptocurrency/api"
	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/angpaoprw/cryptocurrency/logger"
	"go.uber.org/zap"
)

type FailedAPIRetryService struct {
	store              *db.Store
	notificationClient *api.NotificationClient
}

func NewFailedAPIRetryService(store *db.Store, notificationClient *api.NotificationClient) *FailedAPIRetryService {
	return &FailedAPIRetryService{
		store:              store,
		notificationClient: notificationClient,
	}
}

// Start begins the retry service (runs every 5 seconds)
func (s *FailedAPIRetryService) Start(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	// Run immediately on start
	s.retryFailedRequests(ctx)

	for {
		select {
		case <-ticker.C:
			s.retryFailedRequests(ctx)
		case <-ctx.Done():
			logger.Info("Failed API retry service stopped")
			return
		}
	}
}

func (s *FailedAPIRetryService) retryFailedRequests(ctx context.Context) {
	if s.store == nil {
		logger.Error("Store is nil in retryFailedRequests")
		return
	}

	if s.notificationClient == nil {
		logger.Warn("Notification client is nil, skipping retry")
		return
	}

	// Get pending failed requests
	failedRequests, err := s.store.GetPendingFailedAPIRequests(ctx)
	if err != nil {
		logger.Error("Failed to get pending failed API requests", zap.Error(err))
		return
	}

	if len(failedRequests) == 0 {
		return
	}

	logger.Info("Processing failed API requests", zap.Int("count", len(failedRequests)))

	for _, failedReq := range failedRequests {
		// Update last_attempt timestamp
		_, err := s.store.UpdateFailedAPIRequestAttempt(ctx, failedReq.ID)
		if err != nil {
			logger.Error("Failed to update failed API request attempt time",
				zap.String("failed_request_id", failedReq.ID.String()),
				zap.Error(err),
			)
			continue
		}

		// Parse the notification body
		var body map[string]interface{}
		if err := json.Unmarshal(failedReq.Body, &body); err != nil {
			logger.Error("Failed to unmarshal notification body",
				zap.String("failed_request_id", failedReq.ID.String()),
				zap.Error(err),
			)
			continue
		}

		// Determine customer ID and event type from the request
		var customerID string
		var eventType string

		// Try to get deposit request details if this is a deposit-related notification
		if failedReq.DepositRequestID.Valid {
			depositReq, err := s.store.GetDepositRequest(ctx, failedReq.DepositRequestID.UUID)
			if err != nil {
				logger.Error("Failed to get deposit request for retry",
					zap.String("failed_request_id", failedReq.ID.String()),
					zap.String("deposit_request_id", failedReq.DepositRequestID.UUID.String()),
					zap.Error(err),
				)
				continue
			}
			customerID = depositReq.CustomerID

			// Determine event type based on status in body
			if status, ok := body["status"].(string); ok {
				switch status {
				case "expired":
					eventType = api.EventDepositExpired
				case "cancelled":
					eventType = api.EventDepositCancelled
				case "pending":
					eventType = api.EventDepositCreated
				case "completed":
					eventType = api.EventDepositCompleted
				default:
					eventType = api.EventDepositCreated
				}
			}
		} else if failedReq.WithdrawalRequestID.Valid {
			// Handle withdrawal requests similarly
			withdrawalReq, err := s.store.GetWithdrawalRequest(ctx, failedReq.WithdrawalRequestID.UUID)
			if err != nil {
				logger.Error("Failed to get withdrawal request for retry",
					zap.String("failed_request_id", failedReq.ID.String()),
					zap.String("withdrawal_request_id", failedReq.WithdrawalRequestID.UUID.String()),
					zap.Error(err),
				)
				continue
			}
			customerID = withdrawalReq.CustomerID

			// Determine event type based on status
			if status, ok := body["status"].(string); ok {
				switch status {
				case "completed":
					eventType = api.EventWithdrawalCompleted
				case "failed":
					eventType = api.EventWithdrawalFailed
				default:
					eventType = api.EventWithdrawalCreated
				}
			}
		} else {
			logger.Warn("Failed API request has no associated deposit or withdrawal",
				zap.String("failed_request_id", failedReq.ID.String()),
			)
			continue
		}

		// Retry sending the notification
		err = s.notificationClient.SendNotification(customerID, eventType, body)
		if err != nil {
			logger.Warn("Failed to retry notification, will try again later",
				zap.String("failed_request_id", failedReq.ID.String()),
				zap.String("customer_id", customerID),
				zap.String("event_type", eventType),
				zap.Error(err),
			)
			continue
		}

		// Successfully sent, delete the failed request record
		err = s.store.DeleteFailedAPIRequest(ctx, failedReq.ID)
		if err != nil {
			logger.Error("Failed to delete successfully retried API request",
				zap.String("failed_request_id", failedReq.ID.String()),
				zap.Error(err),
			)
			continue
		}

		logger.Info("Successfully retried failed API request",
			zap.String("failed_request_id", failedReq.ID.String()),
			zap.String("customer_id", customerID),
			zap.String("event_type", eventType),
		)
	}
}
