package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type NotificationClient struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

type NotificationPayload struct {
	CustomerID string                 `json:"customer_id"`
	EventType  string                 `json:"event_type"`
	Data       map[string]interface{} `json:"data"`
	Timestamp  time.Time              `json:"timestamp"`
}

// Event types
const (
	EventDepositCreated      = "deposit_created"
	EventDepositCompleted    = "deposit_completed"
	EventDepositExpired      = "deposit_expired"
	EventDepositCancelled    = "deposit_cancelled"
	EventWithdrawalCreated   = "withdrawal_created"
	EventWithdrawalCompleted = "withdrawal_completed"
	EventWithdrawalFailed    = "withdrawal_failed"
)

// NewNotificationClient creates a new notification service client
func NewNotificationClient(baseURL string, logger *zap.Logger) *NotificationClient {
	return &NotificationClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		logger: logger,
	}
}

// SendNotification sends a notification to the notification service
// This function is non-blocking and logs errors instead of returning them
func (nc *NotificationClient) SendNotification(customerID, eventType string, data map[string]interface{}) {
	go func() {
		if nc.baseURL == "" {
			nc.logger.Warn("Notification service URL not configured, skipping notification",
				zap.String("event_type", eventType),
				zap.String("customer_id", customerID),
			)
			return
		}

		payload := NotificationPayload{
			CustomerID: customerID,
			EventType:  eventType,
			Data:       data,
			Timestamp:  time.Now(),
		}

		jsonData, err := json.Marshal(payload)
		if err != nil {
			nc.logger.Error("Failed to marshal notification payload",
				zap.Error(err),
				zap.String("event_type", eventType),
				zap.String("customer_id", customerID),
			)
			return
		}

		url := fmt.Sprintf("%s/api/v1/notify", nc.baseURL)
		req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
		if err != nil {
			nc.logger.Error("Failed to create notification request",
				zap.Error(err),
				zap.String("url", url),
			)
			return
		}

		req.Header.Set("Content-Type", "application/json")

		resp, err := nc.httpClient.Do(req)
		if err != nil {
			nc.logger.Error("Failed to send notification",
				zap.Error(err),
				zap.String("url", url),
				zap.String("event_type", eventType),
			)
			return
		}
		defer resp.Body.Close()

		if resp.StatusCode >= 400 {
			nc.logger.Warn("Notification service returned error status",
				zap.Int("status_code", resp.StatusCode),
				zap.String("event_type", eventType),
				zap.String("customer_id", customerID),
			)
			return
		}

		nc.logger.Info("Notification sent successfully",
			zap.String("event_type", eventType),
			zap.String("customer_id", customerID),
		)
	}()
}
