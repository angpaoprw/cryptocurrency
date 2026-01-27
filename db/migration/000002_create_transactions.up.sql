CREATE TABLE "transactions" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
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
  "contract_address" text,
  "decimals" int,
  "raw_value" text,
  "block_timestamp" text NOT NULL,
  "created_at" timestamp NOT NULL DEFAULT (now()),
  "updated_at" timestamp NOT NULL DEFAULT (now())
);

-- Index for faster lookups
CREATE INDEX idx_transactions_to_address ON transactions(to_address);
CREATE INDEX idx_transactions_from_address ON transactions(from_address);
CREATE INDEX idx_transactions_hash ON transactions(transaction_hash);
CREATE INDEX idx_transactions_event_id ON transactions(event_id);
