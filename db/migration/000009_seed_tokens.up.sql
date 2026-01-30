-- Seed data for tokens
-- Ethereum Mainnet tokens
INSERT INTO tokens (code, name, symbol, decimals, contract_address, network_code, token_type, is_active) VALUES
('ETH', 'Ethereum', 'ETH', 18, NULL, 'ETH_MAINNET', 'native', true),
('USDT', 'Tether USD', 'USDT', 6, '0xdac17f958d2ee523a2206206994597c13d831ec7', 'ETH_MAINNET', 'erc20', true),
('USDC', 'USD Coin', 'USDC', 6, '0xa0b86991c6218b36c1d19d4a2e9eb0ce3606eb48', 'ETH_MAINNET', 'erc20', true),
('DAI', 'Dai Stablecoin', 'DAI', 18, '0x6b175474e89094c44da98b954eedeac495271d0f', 'ETH_MAINNET', 'erc20', true),
('WETH', 'Wrapped Ether', 'WETH', 18, '0xc02aaa39b223fe8d0a0e5c4f27ead9083c756cc2', 'ETH_MAINNET', 'erc20', true)
ON CONFLICT (code, network_code) DO NOTHING;

-- Ethereum Sepolia tokens
INSERT INTO tokens (code, name, symbol, decimals, contract_address, network_code, token_type, is_active) VALUES
('ETH', 'Ethereum', 'ETH', 18, NULL, 'ETH_SEPOLIA', 'native', true),
('USDT', 'Tether USD (Testnet)', 'USDT', 6, '0x7169D38820dfd117C3FA1f22a697dBA58d90BA06', 'ETH_SEPOLIA', 'erc20', true),
('USDC', 'USD Coin (Testnet)', 'USDC', 6, '0x94a9D9AC8a22534E3FaCa9F4e7F2E2cf85d5E4C8', 'ETH_SEPOLIA', 'erc20', true)
ON CONFLICT (code, network_code) DO NOTHING;

-- Polygon Mainnet tokens
INSERT INTO tokens (code, name, symbol, decimals, contract_address, network_code, token_type, is_active) VALUES
('MATIC', 'Polygon', 'MATIC', 18, NULL, 'POLYGON_MAINNET', 'native', true),
('USDT', 'Tether USD', 'USDT', 6, '0xc2132d05d31c914a87c6611c10748aeb04b58e8f', 'POLYGON_MAINNET', 'erc20', true),
('USDC', 'USD Coin', 'USDC', 6, '0x2791bca1f2de4661ed88a30c99a7a9449aa84174', 'POLYGON_MAINNET', 'erc20', true),
('WMATIC', 'Wrapped Matic', 'WMATIC', 18, '0x0d500b1d8e8ef31e21c99d1db9a6444d3adf1270', 'POLYGON_MAINNET', 'erc20', true),
('WETH', 'Wrapped Ether', 'WETH', 18, '0x7ceb23fd6bc0add59e62ac25578270cff1b9f619', 'POLYGON_MAINNET', 'erc20', true)
ON CONFLICT (code, network_code) DO NOTHING;

-- Polygon Amoy tokens
INSERT INTO tokens (code, name, symbol, decimals, contract_address, network_code, token_type, is_active) VALUES
('MATIC', 'Polygon', 'MATIC', 18, NULL, 'POLYGON_AMOY', 'native', true),
('USDT', 'Tether USD (Testnet)', 'USDT', 6, '0xf9F3AB0527bc2d0726b37D6AB6389dD81F2D7e5c', 'POLYGON_AMOY', 'erc20', true),
('USDC', 'USD Coin (Testnet)', 'USDC', 6, '0x41e94eb019c0762f9bfcf9fb1e58725bfb0e7582', 'POLYGON_AMOY', 'erc20', true)
ON CONFLICT (code, network_code) DO NOTHING;
