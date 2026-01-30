-- name: CreateToken :one
INSERT INTO tokens (
    code,
    name,
    symbol,
    decimals,
    contract_address,
    network_code,
    token_type,
    is_active
) VALUES (
    $1, $2, $3, $4, $5, $6, $7, $8
) RETURNING *;

-- name: GetToken :one
SELECT * FROM tokens
WHERE id = $1 LIMIT 1;

-- name: GetTokenByCodeAndNetwork :one
SELECT * FROM tokens
WHERE code = $1 AND network_code = $2 LIMIT 1;

-- name: ListTokensByNetwork :many
SELECT * FROM tokens
WHERE network_code = $1 AND is_active = true
ORDER BY name ASC;

-- name: ListAllActiveTokens :many
SELECT * FROM tokens
WHERE is_active = true
ORDER BY network_code ASC, name ASC;

-- name: ListAllTokens :many
SELECT * FROM tokens
ORDER BY network_code ASC, name ASC;

-- name: UpdateToken :one
UPDATE tokens
SET name = $2, symbol = $3, decimals = $4, contract_address = $5, token_type = $6, is_active = $7, updated_at = NOW()
WHERE id = $1
RETURNING *;

-- name: DeactivateToken :exec
UPDATE tokens
SET is_active = false, updated_at = NOW()
WHERE id = $1;
