package cryptocurrency

import (
	"encoding/hex"
	"strings"
	"testing"
)

func TestCreateWallet(t *testing.T) {
	wallet, err := CreateWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	if wallet == nil {
		t.Fatal("Wallet is nil")
	}

	if wallet.PrivateKey == nil {
		t.Fatal("Private key is nil")
	}

	if len(wallet.PublicKey) == 0 {
		t.Fatal("Public key is empty")
	}

	if !strings.HasPrefix(wallet.Address, "0x") {
		t.Errorf("Address should start with 0x, got: %s", wallet.Address)
	}
	if len(wallet.Address) != 42 {
		t.Errorf("Address should be 42 characters long, got: %d", len(wallet.Address))
	}

	t.Logf("Created wallet with address: %s", wallet.Address)
}

func TestPrivateKeyHex(t *testing.T) {
	wallet, err := CreateWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	privateKeyHex := wallet.PrivateKeyHex()

	if len(privateKeyHex) != 64 {
		t.Errorf("Private key hex should be 64 characters, got: %d", len(privateKeyHex))
	}

	t.Logf("Private key (hex): %s", privateKeyHex)
}

func TestPublicKeyHex(t *testing.T) {
	wallet, err := CreateWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	publicKeyHex := wallet.PublicKeyHex()

	if len(publicKeyHex) != 130 {
		t.Errorf("Public key hex should be 130 characters, got: %d", len(publicKeyHex))
	}

	t.Logf("Public key (hex): %s", publicKeyHex)
}

func TestImportWalletFromPrivateKey(t *testing.T) {
	originalWallet, err := CreateWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	privateKeyHex := originalWallet.PrivateKeyHex()

	importedWallet, err := ImportWalletFromPrivateKey(privateKeyHex)
	if err != nil {
		t.Fatalf("Failed to import wallet: %v", err)
	}

	if originalWallet.Address != importedWallet.Address {
		t.Errorf("Addresses don't match. Original: %s, Imported: %s",
			originalWallet.Address, importedWallet.Address)
	}

	if originalWallet.PublicKeyHex() != importedWallet.PublicKeyHex() {
		t.Error("Public keys don't match")
	}

	t.Logf("Successfully imported wallet with address: %s", importedWallet.Address)
}

func TestImportWalletWithPrefix(t *testing.T) {
	originalWallet, err := CreateWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	privateKeyHex := "0x" + originalWallet.PrivateKeyHex()

	importedWallet, err := ImportWalletFromPrivateKey(privateKeyHex)
	if err != nil {
		t.Fatalf("Failed to import wallet with 0x prefix: %v", err)
	}

	if originalWallet.Address != importedWallet.Address {
		t.Errorf("Addresses don't match with 0x prefix")
	}
}

func TestGetAddressFromPublicKey(t *testing.T) {
	wallet, err := CreateWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	publicKeyHex := wallet.PublicKeyHex()

	address, err := GetAddressFromPublicKey(publicKeyHex)
	if err != nil {
		t.Fatalf("Failed to get address from public key: %v", err)
	}

	if wallet.Address != address {
		t.Errorf("Addresses don't match. Original: %s, Derived: %s",
			wallet.Address, address)
	}

	t.Logf("Successfully derived address: %s", address)
}

func TestSignMessage(t *testing.T) {
	wallet, err := CreateWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	message := []byte("Hello, Ethereum!")

	signature, err := wallet.SignMessage(message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	if len(signature) != 65 {
		t.Errorf("Signature should be 65 bytes, got: %d", len(signature))
	}

	t.Logf("Signature length: %d bytes", len(signature))
}

func TestVerifySignature(t *testing.T) {
	wallet, err := CreateWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	message := []byte("Hello, Ethereum!")

	signature, err := wallet.SignMessage(message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	publicKeyHex := wallet.PublicKeyHex()
	signatureHex := hex.EncodeToString(signature)

	valid, err := VerifySignature(publicKeyHex, message, signatureHex)
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}

	if !valid {
		t.Error("Signature verification failed")
	}

	t.Log("Signature verified successfully")
}

func TestVerifySignatureInvalid(t *testing.T) {
	wallet, err := CreateWallet()
	if err != nil {
		t.Fatalf("Failed to create wallet: %v", err)
	}

	message := []byte("Hello, Ethereum!")
	wrongMessage := []byte("Wrong message")

	signature, err := wallet.SignMessage(message)
	if err != nil {
		t.Fatalf("Failed to sign message: %v", err)
	}

	signatureHex := hex.EncodeToString(signature)
	publicKeyHex := wallet.PublicKeyHex()

	valid, err := VerifySignature(publicKeyHex, wrongMessage, signatureHex)
	if err != nil {
		t.Fatalf("Failed to verify signature: %v", err)
	}

	if valid {
		t.Error("Signature should not be valid for wrong message")
	}

	t.Log("Correctly rejected invalid signature")
}
