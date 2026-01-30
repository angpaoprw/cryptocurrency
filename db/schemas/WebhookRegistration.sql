CREATE TABLE IF NOT EXISTS webhook_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE CASCADE,
    alchemy_webhook_id TEXT NOT NULL,
    address TEXT NOT NULL,
    network TEXT NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT true,
    registered_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_wallet_webhook UNIQUE (wallet_id, alchemy_webhook_id)
);
