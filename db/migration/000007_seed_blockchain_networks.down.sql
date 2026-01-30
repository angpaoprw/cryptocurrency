-- Remove seed data for blockchain networks
DELETE FROM blockchain_networks WHERE code IN (
    'ETH_MAINNET',
    'ETH_SEPOLIA',
    'POLYGON_MAINNET',
    'POLYGON_AMOY'
);
