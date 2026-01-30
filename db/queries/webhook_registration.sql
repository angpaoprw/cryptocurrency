-- name: CreateWebhookRegistration :one
INSERT INTO webhook_registrations (
    wallet_id,
    alchemy_webhook_id,
    address,
    network,
    is_active
) VALUES (
    $1, $2, $3, $4, $5
) RETURNING *;

-- name: GetWebhookRegistrationByWalletID :one
SELECT * FROM webhook_registrations
WHERE wallet_id = $1 AND is_active = true
LIMIT 1;

-- name: GetWebhookRegistrationByAddress :one
SELECT * FROM webhook_registrations
WHERE address = $1 AND is_active = true
LIMIT 1;

-- name: GetAllActiveWebhookRegistrations :many
SELECT * FROM webhook_registrations
WHERE is_active = true
ORDER BY registered_at DESC;

-- name: GetWebhookRegistrationsByWebhookID :many
SELECT * FROM webhook_registrations
WHERE alchemy_webhook_id = $1 AND is_active = true
ORDER BY registered_at DESC;

-- name: CountWebhookRegistrationsByWebhookID :one
SELECT COUNT(*) FROM webhook_registrations
WHERE alchemy_webhook_id = $1 AND is_active = true;

-- name: GetWalletsWithoutWebhookRegistration :many
SELECT w.* FROM wallets w
LEFT JOIN webhook_registrations wr ON w.id = wr.wallet_id AND wr.is_active = true
WHERE wr.id IS NULL AND w.wallet_type = $1 AND w.is_active = true
ORDER BY w.created_at DESC;

-- name: DeactivateWebhookRegistration :exec
UPDATE webhook_registrations
SET is_active = false, updated_at = CURRENT_TIMESTAMP
WHERE id = $1;

-- name: DeactivateWebhookRegistrationByWalletID :exec
UPDATE webhook_registrations
SET is_active = false, updated_at = CURRENT_TIMESTAMP
WHERE wallet_id = $1;
