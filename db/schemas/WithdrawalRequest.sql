CREATE TABLE IF NOT EXISTS withdrawal_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    customer_id TEXT NOT NULL,
    from_wallet_id UUID NOT NULL REFERENCES wallets(id),
    to_address TEXT NOT NULL,
    network TEXT NOT NULL,
    token TEXT NOT NULL,
    requested_amount DECIMAL(36, 18) NOT NULL,
    fee_amount DECIMAL(36, 18) DEFAULT 0,
    actual_amount DECIMAL(36, 18) DEFAULT 0,
    status TEXT NOT NULL CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    transaction_id UUID REFERENCES transactions(id),
    notes TEXT,
    error_message TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    completed_at TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
);
