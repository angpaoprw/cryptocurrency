-- Seed data for blockchain networks
INSERT INTO blockchain_networks (code, name, chain_id, explorer, description, is_active) VALUES
('ETH_MAINNET', 'Ethereum Mainnet', 1, 'https://etherscan.io', 'Ethereum main network', true),
('ETH_SEPOLIA', 'Ethereum Sepolia Testnet', 11155111, 'https://sepolia.etherscan.io', 'Ethereum Sepolia test network', true),
('POLYGON_MAINNET', 'Polygon Mainnet', 137, 'https://polygonscan.com', 'Polygon (Matic) main network', true),
('POLYGON_AMOY', 'Polygon Amoy Testnet', 80002, 'https://amoy.polygonscan.com', 'Polygon Amoy test network', true)
ON CONFLICT (code) DO NOTHING;
