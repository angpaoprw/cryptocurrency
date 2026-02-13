-- name: CreateWallet :one
INSERT INTO wallets (
  wallet_type,
  blockchain,
  token,
  address,
  public_key,
  private_key,
  balance,
  is_active
) VALUES (
  $1, $2, $3, $4, $5, $6, $7, $8
)
RETURNING *;

-- name: GetWallet :one
SELECT * FROM wallets
WHERE id = $1 LIMIT 1;

-- name: GetWalletByAddress :one
SELECT * FROM wallets
WHERE address = $1 LIMIT 1;

-- name: GetWalletsByType :many
SELECT * FROM wallets
WHERE wallet_type = $1 AND is_active = true
ORDER BY created_at DESC;

-- name: GetWalletByNetworkAndToken :one
SELECT * FROM wallets
WHERE blockchain = $1 AND token = $2 AND wallet_type = $3 AND is_active = true
LIMIT 1;

-- name: GetAvailableWalletByNetworkAndToken :one
SELECT w.* FROM wallets w
WHERE w.blockchain = $1 
  AND w.token = $2 
  AND w.wallet_type = $3 
  AND w.is_active = true
  AND NOT EXISTS (
    SELECT 1 FROM deposit_requests dr
    WHERE dr.wallet_id = w.id
      AND dr.status IN ('pending', 'partial')
      AND (dr.expires_at IS NULL OR dr.expires_at > NOW())
  )
ORDER BY w.created_at ASC
LIMIT 1;

-- name: GetWalletWithSufficientBalance :one
SELECT w.* FROM wallets w
WHERE w.blockchain = $1 
  AND w.token = $2 
  AND w.wallet_type = $3 
  AND w.is_active = true
  AND w.balance IS NOT NULL
  AND CAST(w.balance AS DECIMAL(36, 18)) >= $4
ORDER BY w.created_at ASC
LIMIT 1;

-- name: ListWallets :many
SELECT * FROM wallets
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;

-- name: UpdateWalletBalance :one
UPDATE wallets
SET balance = $2, last_sync_at = NOW(), updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: GetActiveWallets :many
SELECT * FROM wallets
WHERE is_active = true
ORDER BY created_at DESC;

-- name: UpdateWallet :one
UPDATE wallets
SET wallet_type = $2, is_active = $3, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateWallet :exec
UPDATE wallets
SET is_active = false, updated_at = NOW()
WHERE id = $1;

-- name: DeleteWallet :exec
DELETE FROM wallets
WHERE id = $1;

-- name: GetAllWallets :many
SELECT * FROM wallets
ORDER BY created_at ASC;

-- name: UpdateWalletPrivateKey :one
UPDATE wallets
SET private_key = $2, updated_at = NOW()
WHERE id = $1
RETURNING *;
