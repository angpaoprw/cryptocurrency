-- name: CreateTransaction :one
INSERT INTO transactions (
  wallet_id,
  webhook_id,
  event_id,
  network,
  transaction_hash,
  block_number,
  from_address,
  to_address,
  value,
  asset,
  category,
  transaction_type,
  status,
  is_matched,
  gas_fee,
  confirmations,
  contract_address,
  decimals,
  raw_value,
  block_timestamp
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
)
RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions
WHERE id = $1 LIMIT 1;

-- name: GetTransactionByEventId :one
SELECT * FROM transactions
WHERE event_id = $1 LIMIT 1;

-- name: GetTransactionByHash :one
SELECT * FROM transactions
WHERE transaction_hash = $1 LIMIT 1;

-- name: ListTransactionsByAddress :many
SELECT * FROM transactions
WHERE to_address = $1 OR from_address = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTransactionsByToAddress :many
SELECT * FROM transactions
WHERE to_address = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTransactionsByFromAddress :many
SELECT * FROM transactions
WHERE from_address = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListTransactionsByWallet :many
SELECT * FROM transactions
WHERE wallet_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetDepositsByWallet :many
SELECT * FROM transactions
WHERE wallet_id = $1 AND transaction_type = 'deposit'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetWithdrawalsByWallet :many
SELECT * FROM transactions
WHERE wallet_id = $1 AND transaction_type = 'withdrawal'
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: ListUnmatchedTransactions :many
SELECT * FROM transactions
WHERE is_matched = false AND transaction_type = 'deposit'
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateTransactionStatus :one
UPDATE transactions
SET status = $2, confirmations = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteTransaction :exec
DELETE FROM transactions
WHERE id = $1;

