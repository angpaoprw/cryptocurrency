-- name: CreateDepositRequest :one
INSERT INTO deposit_requests (
  customer_id,
  wallet_id,
  assigned_address,
  network,
  token,
  expected_amount,
  expires_at
) VALUES (
  $1, $2, $3, $4, $5, $6, $7
)
RETURNING *;

-- name: GetDepositRequest :one
SELECT * FROM deposit_requests
WHERE id = $1 LIMIT 1;

-- name: GetDepositRequestByCustomer :many
SELECT * FROM deposit_requests
WHERE customer_id = $1
ORDER BY created_at DESC
LIMIT $2 OFFSET $3;

-- name: GetPendingDepositByAddress :one
SELECT * FROM deposit_requests
WHERE assigned_address = $1 
  AND status IN ('pending', 'partial')
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY created_at DESC
LIMIT 1;

-- name: UpdateDepositRequestStatus :one
UPDATE deposit_requests
SET 
  status = $2,
  received_amount = $3,
  transaction_id = $4,
  completed_at = CASE WHEN $2 = 'completed' THEN NOW() ELSE completed_at END,
  updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: ListPendingDeposits :many
SELECT * FROM deposit_requests
WHERE status IN ('pending', 'partial')
  AND (expires_at IS NULL OR expires_at > NOW())
ORDER BY created_at DESC;

-- name: ExpireDepositRequests :exec
UPDATE deposit_requests
SET status = 'expired', updated_at = NOW()
WHERE status IN ('pending', 'partial')
  AND expires_at IS NOT NULL
  AND expires_at < NOW();

-- name: CancelDepositRequest :one
UPDATE deposit_requests
SET status = 'cancelled', updated_at = NOW()
WHERE id = $1
RETURNING *;
