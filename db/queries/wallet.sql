-- name: CreateWallet :one
INSERT INTO wallets (blockchain, address, public_key, private_key)
VALUES ($1, $2, $3, $4)
RETURNING id, blockchain, address, public_key, private_key, created_at, updated_at;