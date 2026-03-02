CREATE TABLE IF NOT EXISTS failed_api_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    deposit_request_id uuid REFERENCES deposit_requests(id), 
    withdrawal_request_id uuid REFERENCES withdrawal_requests(id),
    body jsonb not null,
    error text not null default '',
    last_attempt TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_failed_api_request_created_at_desc ON failed_api_requests(created_at desc);