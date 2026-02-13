# Cryptocurrency Wallet & Blockchain API

Internal cryptocurrency wallet management and blockchain transaction API server.

## Environment Variables

Required environment variables:

```bash
# Database
DATABASE_URL=postgresql://user:password@localhost:5432/dbname

# Server
PORT=5000
ENV=development  # or production, staging

# Alchemy Integration
ALCHEMY_NOTIFY_API_KEY=your_alchemy_api_key
ALCHEMY_WEBHOOK_SIGNING_KEY=your_webhook_signing_key
WEBHOOK_BASE_URL=https://your-domain.com

# Notification Service (Optional)
NOTIFICATION_SERVICE_URL=http://notification-service:5000
```

If `NOTIFICATION_SERVICE_URL` is not set, notifications will be disabled (logged as warnings).

## API Endpoints

### Internal Service Routes (No Authentication Required)

#### Create Deposit Request

```http
POST /v1/internal/deposit
Content-Type: application/json

{
  "customer_id": "customer-123",
  "network": "ethereum",
  "token": "USDT",
  "expected_amount": 100.0,
  "expiration_minutes": 3
}
```

**Response:**

```json
{
  "request_id": "uuid",
  "customer_id": "customer-123",
  "deposit_address": "0x...",
  "network": "ethereum",
  "token": "USDT",
  "expected_amount": 100.0,
  "status": "pending",
  "expires_at": "2026-02-12T10:05:00Z",
  "expires_in_seconds": 180,
  "created_at": "2026-02-12T10:02:00Z"
}
```

**Notification sent:** `deposit_created` event

---

#### Create Withdrawal Request

```http
POST /v1/internal/withdrawal
Content-Type: application/json

{
  "customer_id": "customer-123",
  "to_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
  "network": "ethereum",
  "token": "USDT",
  "amount": 50.0,
  "notes": "Customer withdrawal"
}
```

**Response:**

```json
{
  "request_id": "uuid",
  "status": "pending",
  "customer_id": "customer-123",
  "to_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
  "network": "ethereum",
  "token": "USDT",
  "amount": 50.0,
  "created_at": "2026-02-12T10:03:00Z"
}
```

**Processing:** Withdrawal is processed automatically in the background. No approval workflow.

**Notifications sent:**

- `withdrawal_created` - immediately after request creation
- `withdrawal_completed` - after blockchain transaction succeeds
- `withdrawal_failed` - if processing fails

---

#### Get Withdrawal Status

```http
GET /v1/internal/withdrawal/{id}
```

**Response:**

```json
{
  "request_id": "uuid",
  "status": "completed",
  "customer_id": "customer-123",
  "to_address": "0x742d35Cc6634C0532925a3b844Bc9e7595f0bEb",
  "network": "ethereum",
  "token": "USDT",
  "requested_amount": 50.0,
  "fee_amount": 0.5,
  "actual_amount": 49.5,
  "transaction_id": "uuid",
  "created_at": "2026-02-12T10:03:00Z",
  "completed_at": "2026-02-12T10:03:15Z"
}
```

### Public Deposit Routes

#### Create Deposit Request (Public)

```http
POST /v1/deposit/request
```

Same as internal deposit endpoint but without notifications.

#### Get Deposit Status

```http
GET /v1/deposit/request/{id}
```

Returns current deposit request status.

## Notification Service Integration

The system sends HTTP POST notifications to the configured notification service at:

```
POST {NOTIFICATION_SERVICE_URL}/api/v1/notify
```

**Payload Format:**

```json
{
  "customer_id": "customer-123",
  "event_type": "deposit_completed",
  "data": {
    "request_id": "uuid",
    "transaction_id": "uuid",
    "tx_hash": "0x...",
    "amount": "100.0",
    "network": "ethereum",
    "token": "USDT",
    "status": "completed"
  },
  "timestamp": "2026-02-12T10:05:00Z"
}
```

**Event Types:**

- `deposit_created` - New deposit request created
- `deposit_completed` - Deposit received and confirmed
- `deposit_expired` - Deposit request expired without payment
- `withdrawal_created` - New withdrawal request created
- `withdrawal_completed` - Withdrawal transaction confirmed on blockchain
- `withdrawal_failed` - Withdrawal processing failed

**Behavior:**

- Notifications are sent asynchronously (non-blocking)
- Failed notifications are logged but don't affect the main operation
- If `NOTIFICATION_SERVICE_URL` is not configured, notifications are skipped with warning logs

## Database

### New Tables

#### withdrawal_requests

Tracks customer withdrawal requests and their processing status.

**Fields:**

- `id` - UUID primary key
- `customer_id` - Customer identifier
- `from_wallet_id` - Hot wallet used for withdrawal
- `to_address` - Destination blockchain address
- `network` - Blockchain network (ethereum, polygon, bsc)
- `token` - Token type (ETH, USDT, USDC, etc.)
- `requested_amount` - Amount requested to withdraw
- `fee_amount` - Transaction fee amount
- `actual_amount` - Net amount sent (requested - fee)
- `status` - pending, processing, completed, failed
- `transaction_id` - Reference to blockchain transaction
- `notes` - Optional notes
- `error_message` - Error details if failed
- `created_at`, `completed_at`, `updated_at` - Timestamps

**Indexes:** customer_id, status, from_wallet_id, created_at

## Background Services

1. **Deposit Expiration Service** - Runs every 5 seconds
   - Expires pending/partial deposits past expiration time
   - Sends `deposit_expired` notifications

2. **Wallet Balance Sync Service** - Runs every 30 seconds
   - Syncs wallet balances with blockchain
   - Updates database balance records

3. **Withdrawal Processing** - Asynchronous per request
   - Processes withdrawals in background goroutines
   - Updates status and sends notifications

## Supported Networks

- Ethereum Mainnet (`ethereum`, `eth`)
- Polygon Mainnet (`polygon`, `matic`)
- BSC Mainnet (`bsc`, `bnb`)
- Sepolia Testnet (`sepolia`)

## Running the Application

```bash
# Install dependencies
go mod download

# Run migrations
make migrate-up

# Start server
go run main.go

# Or build and run
go build -o bin/app
./bin/app
```

## Development

```bash
# Generate SQL code with sqlc
sqlc generate

# Run tests
go test ./...

# Build
go build -o bin/app
```

## Architecture Notes

- **No Authentication**: Internal services are assumed to be on a trusted network
- **Automatic Withdrawals**: No approval workflow - withdrawals execute immediately if balance is sufficient
- **Hot Wallet Selection**: System automatically selects hot wallets with sufficient balance
- **Transaction Tracking**: All blockchain transactions are recorded for audit trail
- **Event-Driven Notifications**: All lifecycle events trigger notifications to the notification service
- **Webhook Processing**: Alchemy webhooks detect incoming deposits and update request status
