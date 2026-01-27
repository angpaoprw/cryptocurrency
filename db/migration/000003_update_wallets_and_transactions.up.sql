-- Update wallets table
ALTER TABLE wallets ADD COLUMN "wallet_type" text NOT NULL DEFAULT 'hot' CHECK (wallet_type IN ('hot', 'warm', 'cold'));
ALTER TABLE wallets ADD COLUMN "token" text NOT NULL DEFAULT 'USDT';
ALTER TABLE wallets ADD COLUMN "balance" decimal(36, 18) DEFAULT 0;
ALTER TABLE wallets ADD COLUMN "is_active" boolean DEFAULT true;
ALTER TABLE wallets ADD COLUMN "last_sync_at" timestamp;
ALTER TABLE wallets ADD CONSTRAINT unique_address UNIQUE (address);

CREATE INDEX idx_wallets_type ON wallets(wallet_type);
CREATE INDEX idx_wallets_network ON wallets(blockchain);
CREATE INDEX idx_wallets_token ON wallets(token);
CREATE INDEX idx_wallets_active ON wallets(is_active);
CREATE INDEX idx_wallets_address ON wallets(address);

-- Update transactions table
ALTER TABLE transactions ADD COLUMN "wallet_id" uuid REFERENCES wallets(id);
ALTER TABLE transactions ADD COLUMN "transaction_type" text CHECK (transaction_type IN ('deposit', 'withdrawal', 'internal'));
ALTER TABLE transactions ADD COLUMN "status" text DEFAULT 'confirmed' CHECK (status IN ('pending', 'confirmed', 'failed'));
ALTER TABLE transactions ADD COLUMN "is_matched" boolean DEFAULT false;
ALTER TABLE transactions ADD COLUMN "gas_fee" decimal(36, 18);
ALTER TABLE transactions ADD COLUMN "confirmations" int DEFAULT 1;

CREATE INDEX idx_transactions_wallet_id ON transactions(wallet_id);
CREATE INDEX idx_transactions_type ON transactions(transaction_type);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_matched ON transactions(is_matched);
