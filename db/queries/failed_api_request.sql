-- name: CreateFailedAPIRequest :one
insert into failed_api_requests (
    deposit_request_id,
    withdrawal_request_id,
    body,
    error) values ($1,$2,$3,$4) 
    returning *;

-- name: GetPendingFailedAPIRequests :many
SELECT * FROM failed_api_requests;

-- name: UpdateFailedAPIRequestAttempt :one
UPDATE failed_api_requests
SET last_attempt = NOW()
WHERE id = $1
RETURNING *;

-- name: DeleteFailedAPIRequest :exec
DELETE FROM failed_api_requests
WHERE id = $1;