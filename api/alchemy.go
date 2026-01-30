package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type Alchemy struct {
	APIKey        string
	BaseURL       string
	NotifyAPIKey  string
	NotifyBaseURL string
	httpClient    *http.Client
}

func NewAlchemy(apiKey string) *Alchemy {
	return &Alchemy{
		APIKey:        apiKey,
		BaseURL:       "https://eth-mainnet.alchemyapi.io/v2/",
		NotifyBaseURL: "https://dashboard.alchemy.com/api",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func NewAlchemyWithNotify(apiKey, notifyAPIKey string) *Alchemy {
	return &Alchemy{
		APIKey:        apiKey,
		BaseURL:       "https://eth-mainnet.alchemyapi.io/v2/",
		NotifyAPIKey:  notifyAPIKey,
		NotifyBaseURL: "https://dashboard.alchemy.com/api",
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (a *Alchemy) GetURL() string {
	return a.BaseURL + a.APIKey
}

// Webhook structures for Alchemy Notify API
type CreateWebhookRequest struct {
	Network     string   `json:"network"`
	WebhookType string   `json:"webhook_type"`
	WebhookURL  string   `json:"webhook_url"`
	Addresses   []string `json:"addresses,omitempty"`
}

type WebhookResponse struct {
	Data WebhookData `json:"data"`
}

type WebhookData struct {
	ID          string   `json:"id"`
	Network     string   `json:"network"`
	WebhookType string   `json:"webhook_type"`
	WebhookURL  string   `json:"webhook_url"`
	IsActive    bool     `json:"is_active"`
	TimeCreated int64    `json:"time_created"`
	SigningKey  string   `json:"signing_key"`
	Version     string   `json:"version"`
	Addresses   []string `json:"addresses,omitempty"`
}

type AddAddressesRequest struct {
	WebhookID      string   `json:"webhook_id"`
	AddressesToAdd []string `json:"addresses_to_add"`
}

type AddAddressesResponse struct {
	Data struct {
		Message string `json:"message"`
	} `json:"data"`
}

type GetWebhookDetailsResponse struct {
	Data WebhookDetailsData `json:"data"`
}

type WebhookDetailsData struct {
	ID          string   `json:"id"`
	Network     string   `json:"network"`
	WebhookType string   `json:"webhook_type"`
	WebhookURL  string   `json:"webhook_url"`
	IsActive    bool     `json:"is_active"`
	Addresses   []string `json:"addresses"`
	Version     string   `json:"version"`
}

// convertToAlchemyNetwork converts our network codes to Alchemy's expected format
func convertToAlchemyNetwork(network string) string {
	switch network {
	case "ETH_MAINNET":
		return "ETH_MAINNET"
	case "ETH_SEPOLIA":
		return "ETH_SEPOLIA"
	case "POLYGON_MAINNET":
		return "MATIC_MAINNET"
	case "POLYGON_AMOY":
		return "MATIC_AMOY"
	default:
		return network // Return as-is if not mapped
	}
}

// GetOrCreateWebhook finds an existing webhook or creates a new Address Activity webhook
func (a *Alchemy) GetOrCreateWebhook(webhookURL, network string, addresses []string) (*WebhookData, error) {
	if a.NotifyAPIKey == "" {
		return nil, fmt.Errorf("notify API key not configured")
	}

	// Convert network code to Alchemy format
	alchemyNetwork := convertToAlchemyNetwork(network)

	// First, try to get existing webhooks
	existingWebhook, err := a.getExistingWebhook(webhookURL, alchemyNetwork)
	if err == nil && existingWebhook != nil {
		return existingWebhook, nil
	}

	// Create new webhook if none exists (requires at least one address)
	if len(addresses) == 0 {
		return nil, fmt.Errorf("at least one address is required to create a new webhook")
	}
	return a.createWebhook(webhookURL, alchemyNetwork, addresses)
}

// getExistingWebhook attempts to find an existing webhook with the same URL and network
func (a *Alchemy) getExistingWebhook(webhookURL, network string) (*WebhookData, error) {
	url := fmt.Sprintf("%s/team-webhooks", a.NotifyBaseURL)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Alchemy-Token", a.NotifyAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhooks: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get webhooks: %s", string(body))
	}

	var result struct {
		Data []WebhookData `json:"data"`
	}

	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Find matching webhook
	for _, webhook := range result.Data {
		if webhook.WebhookURL == webhookURL && webhook.Network == network && webhook.WebhookType == "ADDRESS_ACTIVITY" {
			return &webhook, nil
		}
	}

	return nil, fmt.Errorf("no matching webhook found")
}

// createWebhook creates a new Address Activity webhook
func (a *Alchemy) createWebhook(webhookURL, network string, addresses []string) (*WebhookData, error) {
	url := fmt.Sprintf("%s/create-webhook", a.NotifyBaseURL)

	createReq := CreateWebhookRequest{
		Network:     network,
		WebhookType: "ADDRESS_ACTIVITY",
		WebhookURL:  webhookURL,
		Addresses:   addresses,
	}

	jsonData, err := json.Marshal(createReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Debug: log the request being sent
	fmt.Printf("DEBUG: Creating webhook with URL=%s, Network=%s, Request=%s\n", url, network, string(jsonData))

	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Alchemy-Token", a.NotifyAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create webhook: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		return nil, fmt.Errorf("failed to create webhook (status %d): %s", resp.StatusCode, string(body))
	}

	var result WebhookResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	return &result.Data, nil
}

// AddAddressesToWebhook adds addresses to an existing webhook
func (a *Alchemy) AddAddressesToWebhook(webhookID string, addresses []string) error {
	if a.NotifyAPIKey == "" {
		return fmt.Errorf("notify API key not configured")
	}

	if len(addresses) == 0 {
		return fmt.Errorf("no addresses provided")
	}

	url := fmt.Sprintf("%s/update-webhook-addresses", a.NotifyBaseURL)

	addReq := AddAddressesRequest{
		WebhookID:      webhookID,
		AddressesToAdd: addresses,
	}

	jsonData, err := json.Marshal(addReq)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequest("PATCH", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Alchemy-Token", a.NotifyAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to add addresses: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("failed to add addresses (status %d): %s", resp.StatusCode, string(body))
	}

	return nil
}

// GetWebhookDetails retrieves detailed information about a webhook including address count
func (a *Alchemy) GetWebhookDetails(webhookID string) (*WebhookDetailsData, error) {
	if a.NotifyAPIKey == "" {
		return nil, fmt.Errorf("notify API key not configured")
	}

	url := fmt.Sprintf("%s/webhook-addresses?webhook_id=%s", a.NotifyBaseURL, webhookID)

	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Alchemy-Token", a.NotifyAPIKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get webhook details: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get webhook details (status %d): %s", resp.StatusCode, string(body))
	}

	// Debug: log the response
	fmt.Printf("DEBUG: GetWebhookDetails response: %s\n", string(body))

	// The API returns an array of addresses in data field, not a webhook object
	var result struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Return a minimal WebhookDetailsData with just the addresses
	return &WebhookDetailsData{
		ID:        webhookID,
		Addresses: result.Data,
	}, nil
}
