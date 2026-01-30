-- name: CreateBlockchainNetwork :one
INSERT INTO blockchain_networks (
    code,
    name,
    chain_id,
    explorer,
    description,
    is_active
) VALUES (
    $1, $2, $3, $4, $5, $6
) RETURNING *;

-- name: GetBlockchainNetwork :one
SELECT * FROM blockchain_networks
WHERE id = $1 LIMIT 1;

-- name: GetBlockchainNetworkByCode :one
SELECT * FROM blockchain_networks
WHERE code = $1 LIMIT 1;

-- name: ListActiveBlockchainNetworks :many
SELECT * FROM blockchain_networks
WHERE is_active = true
ORDER BY name ASC;

-- name: ListAllBlockchainNetworks :many
SELECT * FROM blockchain_networks
ORDER BY name ASC;

-- name: UpdateBlockchainNetwork :one
UPDATE blockchain_networks
SET name = $2, chain_id = $3, explorer = $4, description = $5, is_active = $6, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateBlockchainNetwork :exec
UPDATE blockchain_networks
SET is_active = false, updated_at = NOW()
WHERE id = $1;
