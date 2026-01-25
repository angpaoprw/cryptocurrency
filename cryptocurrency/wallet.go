package cryptocurrency

import (
	"crypto/ecdsa"
	"encoding/hex"
	"fmt"

	"github.com/ethereum/go-ethereum/crypto"
)

// Wallet represents an Ethereum wallet
type Wallet struct {
	PrivateKey *ecdsa.PrivateKey
	PublicKey  []byte
	Address    string
}

// CreateWallet generates a new Ethereum wallet with private key, public key, and address
func CreateWallet() (*Wallet, error) {
	// Generate a new private key
	privateKey, err := crypto.GenerateKey()
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	// Extract the public key from private key
	publicKey := crypto.FromECDSAPub(&privateKey.PublicKey)

	// Generate the Ethereum address from public key
	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
	}, nil
}

// PrivateKeyHex returns the private key as a hex string
func (w *Wallet) PrivateKeyHex() string {
	return hex.EncodeToString(crypto.FromECDSA(w.PrivateKey))
}

// PublicKeyHex returns the public key as a hex string
func (w *Wallet) PublicKeyHex() string {
	return hex.EncodeToString(w.PublicKey)
}

// ImportWalletFromPrivateKey imports a wallet from a private key hex string
func ImportWalletFromPrivateKey(privateKeyHex string) (*Wallet, error) {
	// Remove "0x" prefix if present
	if len(privateKeyHex) >= 2 && privateKeyHex[:2] == "0x" {
		privateKeyHex = privateKeyHex[2:]
	}

	// Decode the hex string
	privateKeyBytes, err := hex.DecodeString(privateKeyHex)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	// Convert to ECDSA private key
	privateKey, err := crypto.ToECDSA(privateKeyBytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Extract the public key
	publicKey := crypto.FromECDSAPub(&privateKey.PublicKey)

	// Generate the address
	address := crypto.PubkeyToAddress(privateKey.PublicKey).Hex()

	return &Wallet{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		Address:    address,
	}, nil
}

// GetAddressFromPublicKey derives an Ethereum address from a public key hex string
func GetAddressFromPublicKey(publicKeyHex string) (string, error) {
	// Remove "0x" prefix if present
	if len(publicKeyHex) >= 2 && publicKeyHex[:2] == "0x" {
		publicKeyHex = publicKeyHex[2:]
	}

	// Decode the hex string
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return "", fmt.Errorf("failed to decode public key: %w", err)
	}

	// Convert to ECDSA public key
	publicKey, err := crypto.UnmarshalPubkey(publicKeyBytes)
	if err != nil {
		return "", fmt.Errorf("failed to unmarshal public key: %w", err)
	}

	// Generate the address
	address := crypto.PubkeyToAddress(*publicKey).Hex()

	return address, nil
}

// SignMessage signs a message with the wallet's private key
func (w *Wallet) SignMessage(message []byte) ([]byte, error) {
	// Hash the message using Keccak256
	hash := crypto.Keccak256Hash(message)

	// Sign the hash
	signature, err := crypto.Sign(hash.Bytes(), w.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to sign message: %w", err)
	}

	return signature, nil
}

// VerifySignature verifies a signature against a message and public key
func VerifySignature(publicKeyHex string, message []byte, signatureHex string) (bool, error) {
	// Remove "0x" prefix if present
	if len(publicKeyHex) >= 2 && publicKeyHex[:2] == "0x" {
		publicKeyHex = publicKeyHex[2:]
	}
	if len(signatureHex) >= 2 && signatureHex[:2] == "0x" {
		signatureHex = signatureHex[2:]
	}

	// Decode public key
	publicKeyBytes, err := hex.DecodeString(publicKeyHex)
	if err != nil {
		return false, fmt.Errorf("failed to decode public key: %w", err)
	}

	// Decode signature
	signature, err := hex.DecodeString(signatureHex)
	if err != nil {
		return false, fmt.Errorf("failed to decode signature: %w", err)
	}

	// Hash the message
	hash := crypto.Keccak256Hash(message)

	// Remove recovery ID from signature (last byte)
	if len(signature) == 65 {
		signature = signature[:64]
	}

	// Verify the signature
	return crypto.VerifySignature(publicKeyBytes, hash.Bytes(), signature), nil
}
