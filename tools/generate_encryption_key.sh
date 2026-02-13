#!/bin/bash

# Generate a secure 32-byte encryption key for private key encryption
# Run this once and add the output to your .env file as ENCRYPTION_KEY

echo "Generating 32-byte (256-bit) encryption key..."
echo ""

KEY=$(openssl rand -hex 32)

echo "Your encryption key:"
echo "===================="
echo "$KEY"
echo "===================="
echo ""
echo "Add this to your .env file:"
echo "ENCRYPTION_KEY=$KEY"
echo ""
echo "⚠️  IMPORTANT SECURITY NOTES:"
echo "  - Never commit this key to version control"
echo "  - Use different keys for dev/staging/production"
echo "  - Store production keys in a secrets manager"
echo "  - Keep a secure backup of this key"
echo ""
