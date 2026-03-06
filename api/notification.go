package api

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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
	EventWithdrawalCancelled = "withdrawal_cancelled"
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

// SendNotification sends a notification to the internal callback service
// This function is synchronous and blocks until the callback completes
func (nc *NotificationClient) SendNotification(customerID, eventType string, data map[string]interface{}) error {
	if nc.baseURL == "" {
		nc.logger.Warn("Notification service URL not configured, skipping notification",
			zap.String("event_type", eventType),
			zap.String("customer_id", customerID),
		)
		return nil
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
		return err
	}

	// Map event types to internal callback routes
	var endpoint string
	switch eventType {
	case EventDepositCreated, EventDepositCompleted, EventDepositCancelled:
		endpoint = "/v1/cryptocurrency/internal/callback/deposit"
	case EventDepositExpired:
		endpoint = "/v1/cryptocurrency/internal/callback/expire"
	case EventWithdrawalCreated, EventWithdrawalCompleted, EventWithdrawalFailed:
		endpoint = "/v1/cryptocurrency/internal/callback/withdrawal"
	default:
		endpoint = "/v1/cryptocurrency/internal/callback/unknown-transaction"
	}

	url := fmt.Sprintf("%s%s", nc.baseURL, endpoint)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		nc.logger.Error("Failed to create notification request",
			zap.Error(err),
			zap.String("url", url),
		)
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := nc.httpClient.Do(req)
	if err != nil {
		nc.logger.Error("Failed to send notification",
			zap.Error(err),
			zap.String("url", url),
			zap.String("event_type", eventType),
		)
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		nc.logger.Warn("Notification service returned error status",
			zap.Int("status_code", resp.StatusCode),
			zap.String("event_type", eventType),
			zap.String("customer_id", customerID),
		)
		resp_body, err := io.ReadAll(resp.Body)
		if err != nil {
			return err
		}
		return errors.New(string(resp_body))
	}

	nc.logger.Info("Notification sent successfully",
		zap.String("event_type", eventType),
		zap.String("customer_id", customerID),
	)
	return nil
}
