CREATE TABLE IF NOT EXISTS tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    code TEXT NOT NULL,
    name TEXT NOT NULL,
    symbol TEXT NOT NULL,
    decimals INTEGER NOT NULL DEFAULT 18,
    contract_address TEXT,
    network_code TEXT NOT NULL,
    token_type TEXT NOT NULL DEFAULT 'native',
    is_active BOOLEAN NOT NULL DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    
    CONSTRAINT unique_token_network UNIQUE (code, network_code)
);
