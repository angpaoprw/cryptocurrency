CREATE TABLE "transactions" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "wallet_id" uuid REFERENCES wallets(id),
  "webhook_id" text NOT NULL,
  "event_id" text NOT NULL UNIQUE,
  "network" text NOT NULL,
  "transaction_hash" text NOT NULL,
  "block_number" text NOT NULL,
  "from_address" text NOT NULL,
  "to_address" text NOT NULL,
  "value" decimal(36, 18) NOT NULL,
  "asset" text NOT NULL,
  "category" text NOT NULL,
  "transaction_type" text CHECK (transaction_type IN ('deposit', 'withdrawal', 'internal')),
  "status" text DEFAULT 'confirmed' CHECK (status IN ('pending', 'confirmed', 'failed')),
  "is_matched" boolean DEFAULT false,
  "gas_fee" decimal(36, 18),
  "confirmations" int DEFAULT 1,
  "contract_address" text,
  "decimals" int,
  "raw_value" text,
  "block_timestamp" text NOT NULL,
  "created_at" timestamp NOT NULL DEFAULT (now()),
  "updated_at" timestamp NOT NULL DEFAULT (now())
);

-- Indexes for faster lookups
CREATE INDEX idx_transactions_wallet_id ON transactions(wallet_id);
CREATE INDEX idx_transactions_to_address ON transactions(to_address);
CREATE INDEX idx_transactions_from_address ON transactions(from_address);
CREATE INDEX idx_transactions_hash ON transactions(transaction_hash);
CREATE INDEX idx_transactions_event_id ON transactions(event_id);
CREATE INDEX idx_transactions_type ON transactions(transaction_type);
CREATE INDEX idx_transactions_status ON transactions(status);
CREATE INDEX idx_transactions_matched ON transactions(is_matched);
