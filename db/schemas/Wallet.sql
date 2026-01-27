CREATE TABLE "wallets" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "wallet_type" text NOT NULL CHECK (wallet_type IN ('hot', 'warm', 'cold')),
  "blockchain" text NOT NULL,
  "token" text NOT NULL,
  "address" text NOT NULL UNIQUE,
  "public_key" text NOT NULL,
  "private_key" text NOT NULL,
  "balance" decimal(36, 18) DEFAULT 0,
  "is_active" boolean DEFAULT true,
  "last_sync_at" timestamp,
  "created_at" timestamp NOT NULL DEFAULT (now()),
  "updated_at" timestamp NOT NULL DEFAULT (now())
);

-- Indexes for faster lookups
CREATE INDEX idx_wallets_type ON wallets(wallet_type);
CREATE INDEX idx_wallets_network ON wallets(blockchain);
CREATE INDEX idx_wallets_token ON wallets(token);
CREATE INDEX idx_wallets_active ON wallets(is_active);
CREATE INDEX idx_wallets_address ON wallets(address);

