package main

import (
	"context"
	"encoding/hex"
	"log"
	"os"

	"github.com/angpaoprw/cryptocurrency/crypto"
	db "github.com/angpaoprw/cryptocurrency/db/sqlc"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

// This script encrypts existing plain-text private keys in the database
// IMPORTANT: Run this ONCE before deploying the new encryption feature
// BACKUP YOUR DATABASE BEFORE RUNNING THIS SCRIPT

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("Warning: Could not load .env file:", err)
	}

	// Get encryption key
	encryptionKeyHex := os.Getenv("ENCRYPTION_KEY")
	if encryptionKeyHex == "" {
		log.Fatal("ENCRYPTION_KEY environment variable is required")
	}

	encryptionKey, err := hex.DecodeString(encryptionKeyHex)
	if err != nil {
		log.Fatal("Invalid ENCRYPTION_KEY format:", err)
	}

	if len(encryptionKey) != 32 {
		log.Fatalf("Invalid ENCRYPTION_KEY length: %d bytes (need 32)", len(encryptionKey))
	}

	// Connect to database
	dbConn := db.Connect()
	defer dbConn.Close()

	// Get all wallets
	queries := db.New(dbConn)
	wallets, err := queries.GetAllWallets(context.Background())
	if err != nil {
		log.Fatal("Failed to fetch wallets:", err)
	}

	log.Printf("Found %d wallets to encrypt\n", len(wallets))

	// Start transaction
	tx, err := dbConn.Begin()
	if err != nil {
		log.Fatal("Failed to start transaction:", err)
	}
	defer tx.Rollback()

	txQueries := queries.WithTx(tx)
	encryptedCount := 0
	skippedCount := 0

	for _, wallet := range wallets {
		// Skip if private key is empty
		if wallet.PrivateKey == "" {
			log.Printf("Wallet %s: Skipping (empty private key)\n", wallet.ID)
			skippedCount++
			continue
		}

		// Check if already encrypted (encrypted keys are base64, private keys are hex starting with 0x or just hex)
		// Simple heuristic: if it starts with 0x and is 66 chars, it's likely unencrypted
		isLikelyUnencrypted := (len(wallet.PrivateKey) == 66 && wallet.PrivateKey[:2] == "0x") ||
			(len(wallet.PrivateKey) == 64 && isHex(wallet.PrivateKey))

		if !isLikelyUnencrypted {
			// Try to decrypt - if it works, it's already encrypted
			_, err := crypto.Decrypt(wallet.PrivateKey, encryptionKey)
			if err == nil {
				log.Printf("Wallet %s: Already encrypted, skipping\n", wallet.ID)
				skippedCount++
				continue
			}
		}

		// Encrypt the private key
		encryptedKey, err := crypto.Encrypt(wallet.PrivateKey, encryptionKey)
		if err != nil {
			log.Printf("Wallet %s: Failed to encrypt: %v\n", wallet.ID, err)
			continue
		}

		// Update database
		_, err = txQueries.UpdateWalletPrivateKey(context.Background(), db.UpdateWalletPrivateKeyParams{
			ID:         wallet.ID,
			PrivateKey: encryptedKey,
		})
		if err != nil {
			log.Printf("Wallet %s: Failed to update: %v\n", wallet.ID, err)
			continue
		}

		log.Printf("Wallet %s: Encrypted successfully\n", wallet.ID)
		encryptedCount++
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		log.Fatal("Failed to commit transaction:", err)
	}

	log.Printf("\nMigration completed:")
	log.Printf("  - Encrypted: %d wallets\n", encryptedCount)
	log.Printf("  - Skipped: %d wallets\n", skippedCount)
	log.Printf("  - Total: %d wallets\n", len(wallets))
}

func isHex(s string) bool {
	_, err := hex.DecodeString(s)
	return err == nil
}
