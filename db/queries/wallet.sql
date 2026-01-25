-- name: CreateWallet :one
INSERT INTO wallets (id) VALUES ($1) RETURNING *;