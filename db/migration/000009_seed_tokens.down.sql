-- Remove seed data for tokens
DELETE FROM tokens WHERE (code, network_code) IN (
    -- Ethereum Mainnet
    ('ETH', 'ETH_MAINNET'),
    ('USDT', 'ETH_MAINNET'),
    ('USDC', 'ETH_MAINNET'),
    ('DAI', 'ETH_MAINNET'),
    ('WETH', 'ETH_MAINNET'),
    -- Ethereum Sepolia
    ('ETH', 'ETH_SEPOLIA'),
    ('USDT', 'ETH_SEPOLIA'),
    ('USDC', 'ETH_SEPOLIA'),
    -- Polygon Mainnet
    ('MATIC', 'POLYGON_MAINNET'),
    ('USDT', 'POLYGON_MAINNET'),
    ('USDC', 'POLYGON_MAINNET'),
    ('WMATIC', 'POLYGON_MAINNET'),
    ('WETH', 'POLYGON_MAINNET'),
    -- Polygon Amoy
    ('MATIC', 'POLYGON_AMOY'),
    ('USDT', 'POLYGON_AMOY'),
    ('USDC', 'POLYGON_AMOY')
);
