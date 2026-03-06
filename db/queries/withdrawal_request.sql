-- name: CreateWithdrawalRequest :one
INSERT INTO withdrawal_requests (
    customer_id,
    from_wallet_id,
    to_address,
    network,
    token,
    requested_amount,
    fee_amount,
    actual_amount,
    status,
    notes
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8, $9, $10
) RETURNING *;

-- name: GetWithdrawalRequest :one
SELECT * FROM withdrawal_requests
WHERE id = $1 LIMIT 1;

-- name: ListWithdrawalRequestsByCustomer :many
SELECT * FROM withdrawal_requests
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: UpdateWithdrawalRequestStatus :one
UPDATE withdrawal_requests
SET 
    status = $2,
    transaction_id = COALESCE($3, transaction_id),
    error_message = COALESCE($4, error_message),
    tx_hash = COALESCE($5, tx_hash),
    completed_at = CASE 
        WHEN $2 IN ('completed', 'failed', 'cancelled') THEN CURRENT_TIMESTAMP 
        ELSE completed_at 
    END,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: UpdateWithdrawalRequestFees :one
UPDATE withdrawal_requests
SET 
    fee_amount = $2,
    actual_amount = $3,
    updated_at = CURRENT_TIMESTAMP
WHERE id = $1
RETURNING *;

-- name: ListPendingWithdrawals :many
SELECT * FROM withdrawal_requests
WHERE status = 'pending'
ORDER BY created_at ASC;

-- name: GetWithdrawalRequestsByWallet :many
SELECT * FROM withdrawal_requests
WHERE from_wallet_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetWithdrawalRequestByTxHash :one
SELECT * FROM withdrawal_requests
WHERE tx_hash = $1 LIMIT 1;
