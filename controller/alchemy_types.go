package controller

// AlchemyWebhook represents the root webhook payload
type AlchemyWebhook struct {
	WebhookID string       `json:"webhookId"`
	ID        string       `json:"id"`
	CreatedAt string       `json:"createdAt"`
	Type      string       `json:"type"`
	Event     AlchemyEvent `json:"event"`
}

// AlchemyEvent represents the event data in the webhook
type AlchemyEvent struct {
	Network  string            `json:"network"`
	Activity []AlchemyActivity `json:"activity"`
	Source   string            `json:"source"`
}

// AlchemyActivity represents a single activity/transaction
type AlchemyActivity struct {
	FromAddress    string             `json:"fromAddress"`
	ToAddress      string             `json:"toAddress"`
	BlockNum       string             `json:"blockNum"`
	Hash           string             `json:"hash"`
	Value          float64            `json:"value"`
	Asset          string             `json:"asset"`
	Category       string             `json:"category"`
	RawContract    AlchemyRawContract `json:"rawContract"`
	Log            AlchemyLog         `json:"log"`
	BlockTimestamp string             `json:"blockTimestamp"`
}

// AlchemyRawContract represents the contract details
type AlchemyRawContract struct {
	RawValue string `json:"rawValue"`
	Address  string `json:"address"`
	Decimals int    `json:"decimals"`
}

// AlchemyLog represents the log details
type AlchemyLog struct {
	Address          string   `json:"address"`
	Topics           []string `json:"topics"`
	Data             string   `json:"data"`
	BlockNumber      string   `json:"blockNumber"`
	TransactionHash  string   `json:"transactionHash"`
	TransactionIndex string   `json:"transactionIndex"`
	BlockHash        string   `json:"blockHash"`
	BlockTimestamp   string   `json:"blockTimestamp"`
	LogIndex         string   `json:"logIndex"`
	Removed          bool     `json:"removed"`
}
