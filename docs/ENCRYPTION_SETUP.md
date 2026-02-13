# Private Key Encryption Setup

This guide explains how to set up and use private key encryption in the cryptocurrency wallet system.

## Overview

Private keys are now encrypted using AES-256-GCM before being stored in the database. This adds an extra layer of security to protect wallet credentials at rest.

## Prerequisites

- OpenSSL (for generating encryption key)
- Go 1.x or higher
- Database backup (before running migration)

## Initial Setup

### 1. Generate Encryption Key

Generate a 32-byte (256-bit) encryption key:

```bash
openssl rand -hex 32
```

This will output a 64-character hex string. Example:

```
a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6q7r8s9t0u1v2w3x4y5z6a7b8c9d0e1f2
```

### 2. Add to Environment

Add the encryption key to your `.env` file:

```env
ENCRYPTION_KEY=your_64_character_hex_string_here
```

**⚠️ SECURITY WARNING:**

- Never commit the `.env` file with real encryption key to version control
- Use different keys for development, staging, and production
- Store production keys in a secure secrets manager (AWS Secrets Manager, HashiCorp Vault, etc.)

### 3. Encrypt Existing Private Keys (One-Time Migration)

If you have existing wallets with plain-text private keys in the database:

```bash
# IMPORTANT: Backup your database first!
pg_dump -U your_user -d your_database > backup_before_encryption.sql

# Run the migration script
go run tools/encrypt_private_keys.go
```

The script will:

- Read all wallets from the database
- Encrypt plain-text private keys
- Update the database with encrypted values
- Skip wallets that are already encrypted

**Output example:**

```
Found 150 wallets to encrypt
Wallet 123e4567-e89b-12d3-a456-426614174000: Encrypted successfully
Wallet 223e4567-e89b-12d3-a456-426614174001: Already encrypted, skipping
...
Migration completed:
  - Encrypted: 148 wallets
  - Skipped: 2 wallets
  - Total: 150 wallets
```

## How It Works

### Wallet Creation

1. New wallet is generated with private key
2. Private key is encrypted using AES-256-GCM
3. Encrypted value (base64-encoded) is stored in database
4. Original private key is discarded from memory

### Withdrawal Processing

1. Encrypted private key is retrieved from database
2. Key is decrypted on-demand in application memory
3. Transaction is signed using decrypted key
4. Decrypted key is discarded from memory after use

### Security Features

- **AES-256-GCM**: Industry-standard authenticated encryption
- **On-Demand Decryption**: Keys are only decrypted when needed
- **No Plain-Text Storage**: Database never contains plain-text private keys (after migration)
- **Memory Safety**: Decrypted keys are ephemeral and not persisted

## Production Recommendations

### Key Management Best Practices

1. **Use a Key Management Service (KMS)**
   - AWS KMS
   - Google Cloud KMS
   - Azure Key Vault
   - HashiCorp Vault

2. **Implement Key Rotation**
   - Periodically rotate encryption keys
   - Re-encrypt all private keys with new key
   - Maintain audit logs of key usage

3. **Access Control**
   - Limit who can access the encryption key
   - Log all private key decryption attempts
   - Monitor for unusual access patterns

4. **Hardware Security Module (HSM)**
   - For maximum security, consider using HSM for key storage
   - Keys never leave the HSM device

### Alternative Approaches (Even More Secure)

Instead of storing private keys in the database at all:

1. **Wallet-as-a-Service Providers**
   - Fireblocks
   - BitGo
   - Coinbase Cloud

2. **Multi-Party Computation (MPC)**
   - Distribute key shares across multiple parties
   - No single point of failure

3. **Hardware Wallets**
   - Ledger
   - Trezor
   - Integration via APIs

## Troubleshooting

### Application Won't Start

**Error:** "ENCRYPTION_KEY environment variable is required"

- **Solution:** Add ENCRYPTION_KEY to your .env file

**Error:** "Invalid ENCRYPTION_KEY format - must be hex encoded"

- **Solution:** Ensure key is valid hex (use `openssl rand -hex 32`)

**Error:** "Invalid ENCRYPTION_KEY length - must be 32 bytes"

- **Solution:** Key must be exactly 64 hex characters (32 bytes)

### Withdrawal Fails

**Error:** "Failed to decrypt private key"

- **Cause:** Wrong encryption key or corrupted data
- **Solution:** Verify ENCRYPTION_KEY matches the one used during encryption

### Migration Script Fails

**Error:** "Failed to update: constraint violation"

- **Cause:** Database transaction issues
- **Solution:** Check database connection, ensure proper permissions

## Key Rotation Procedure

1. Generate new encryption key: `openssl rand -hex 32`
2. Set as `NEW_ENCRYPTION_KEY` in environment
3. Run key rotation script (to be created):
   ```bash
   go run tools/rotate_encryption_key.go
   ```
4. Script decrypts with old key, re-encrypts with new key
5. Update `ENCRYPTION_KEY` to new value
6. Remove `NEW_ENCRYPTION_KEY`

## Monitoring & Auditing

### Recommended Logging

- All private key decryption attempts
- Failed decryption attempts (potential attack)
- Wallet creation and encryption events
- Key rotation events

### Metrics to Track

- Number of encrypted wallets
- Decryption success/failure rate
- Time since last key rotation
- Failed decryption attempts per hour

## Security Audit Checklist

- [ ] Encryption key is 32 bytes (256 bits)
- [ ] Different keys for dev/staging/production
- [ ] .env file is in .gitignore
- [ ] Database backup exists before migration
- [ ] All existing wallets are encrypted
- [ ] Decryption logs are monitored
- [ ] Key rotation schedule is defined
- [ ] Access to encryption key is restricted
- [ ] Consider migration to KMS/Vault

## Additional Resources

- [AES-GCM Documentation](https://en.wikipedia.org/wiki/Galois/Counter_Mode)
- [OWASP Cryptographic Storage](https://cheatsheetseries.owasp.org/cheatsheets/Cryptographic_Storage_Cheat_Sheet.html)
- [NIST Encryption Standards](https://csrc.nist.gov/projects/cryptographic-standards-and-guidelines)
