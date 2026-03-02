ALTER TABLE withdrawal_requests ADD COLUMN tx_hash VARCHAR(100);
CREATE INDEX idx_withdrawal_requests_tx_hash ON withdrawal_requests(tx_hash);
