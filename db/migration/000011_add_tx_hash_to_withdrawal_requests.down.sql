DROP INDEX IF EXISTS idx_withdrawal_requests_tx_hash;
ALTER TABLE withdrawal_requests DROP COLUMN tx_hash;
