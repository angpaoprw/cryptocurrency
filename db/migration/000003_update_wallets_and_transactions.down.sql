-- Drop transaction columns
DROP INDEX IF EXISTS idx_transactions_matched;
DROP INDEX IF EXISTS idx_transactions_status;
DROP INDEX IF EXISTS idx_transactions_type;
DROP INDEX IF EXISTS idx_transactions_wallet_id;

ALTER TABLE transactions DROP COLUMN IF EXISTS confirmations;
ALTER TABLE transactions DROP COLUMN IF EXISTS gas_fee;
ALTER TABLE transactions DROP COLUMN IF EXISTS is_matched;
ALTER TABLE transactions DROP COLUMN IF EXISTS status;
ALTER TABLE transactions DROP COLUMN IF EXISTS transaction_type;
ALTER TABLE transactions DROP COLUMN IF EXISTS wallet_id;

-- Drop wallet columns
DROP INDEX IF EXISTS idx_wallets_address;
DROP INDEX IF EXISTS idx_wallets_active;
DROP INDEX IF EXISTS idx_wallets_token;
DROP INDEX IF EXISTS idx_wallets_network;
DROP INDEX IF EXISTS idx_wallets_type;

ALTER TABLE wallets DROP CONSTRAINT IF EXISTS unique_address;
ALTER TABLE wallets DROP COLUMN IF EXISTS last_sync_at;
ALTER TABLE wallets DROP COLUMN IF EXISTS is_active;
ALTER TABLE wallets DROP COLUMN IF EXISTS balance;
ALTER TABLE wallets DROP COLUMN IF EXISTS token;
ALTER TABLE wallets DROP COLUMN IF EXISTS wallet_type;
