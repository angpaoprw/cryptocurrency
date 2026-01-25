package cryptocurrency

import (
	"encoding/hex"
	"fmt"
	"log"
)

// ExampleCreateWallet demonstrates how to create a new Ethereum wallet
func ExampleCreateWallet() {
	// Create a new wallet
	wallet, err := CreateWallet()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Address: %s\n", wallet.Address)
	fmt.Printf("Private Key: %s\n", wallet.PrivateKeyHex())
	fmt.Printf("Public Key: %s\n", wallet.PublicKeyHex())

	// Store the private key securely!
	// Never share or expose your private key
}

// ExampleImportWallet demonstrates how to import a wallet from a private key
func ExampleImportWallet() {
	// Example private key (DO NOT use in production!)
	privateKeyHex := "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"

	// Import the wallet
	wallet, err := ImportWalletFromPrivateKey(privateKeyHex)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Imported wallet address: %s\n", wallet.Address)
}

// ExampleSignAndVerify demonstrates message signing and verification
func ExampleSignAndVerify() {
	// Create a wallet
	wallet, err := CreateWallet()
	if err != nil {
		log.Fatal(err)
	}

	// Message to sign
	message := []byte("This is a test message")

	// Sign the message
	signature, err := wallet.SignMessage(message)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Message signed successfully\n")
	fmt.Printf("Signature length: %d bytes\n", len(signature))

	// Verify the signature
	signatureHex := hex.EncodeToString(signature)
	valid, err := VerifySignature(wallet.PublicKeyHex(), message, signatureHex)
	if err != nil {
		log.Fatal(err)
	}

	if valid {
		fmt.Println("Signature is valid!")
	} else {
		fmt.Println("Signature is invalid!")
	}
}
