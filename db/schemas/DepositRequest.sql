CREATE TABLE "deposit_requests" (
  "id" uuid PRIMARY KEY DEFAULT (gen_random_uuid()),
  "customer_id" text NOT NULL,
  "wallet_id" uuid NOT NULL REFERENCES wallets(id),
  "assigned_address" text NOT NULL,
  "network" text NOT NULL,
  "token" text NOT NULL,
  "expected_amount" decimal(36, 18),
  "received_amount" decimal(36, 18) DEFAULT 0,
  "status" text NOT NULL DEFAULT 'pending' CHECK (status IN ('pending', 'partial', 'completed', 'expired', 'cancelled')),
  "transaction_id" uuid REFERENCES transactions(id),
  "expires_at" timestamp,
  "completed_at" timestamp,
  "created_at" timestamp NOT NULL DEFAULT (now()),
  "updated_at" timestamp NOT NULL DEFAULT (now())
);

-- Indexes
CREATE INDEX idx_deposit_requests_customer ON deposit_requests(customer_id);
CREATE INDEX idx_deposit_requests_wallet ON deposit_requests(wallet_id);
CREATE INDEX idx_deposit_requests_status ON deposit_requests(status);
CREATE INDEX idx_deposit_requests_address ON deposit_requests(assigned_address);
