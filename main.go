package main

import (
	"log"

	"github.com/angpaoprw/cryptocurrency/cryptocurrency"
	"github.com/angpaoprw/cryptocurrency/i18n"
)

func main() {
	// Initialize i18n
	if err := i18n.Init(); err != nil {
		log.Fatal("Failed to initialize i18n:", err)
	}

	new_wallet, err := cryptocurrency.CreateWallet()
	if err != nil {
		log.Fatal("Failed to create wallet:", err)
	}
	log.Println("Address:", new_wallet.Address)
	log.Println("Private Key:", new_wallet.PrivateKeyHex())
	log.Println("Public Key:", new_wallet.PublicKeyHex())

	// Demo translations
	log.Println("=== English ===")
	log.Println(i18n.Translate("en", "welcome"))
	log.Println(i18n.Translate("en", "wallet.created"))

	log.Println("\n=== Thai ===")
	log.Println(i18n.Translate("th", "welcome"))
	log.Println(i18n.Translate("th", "wallet.created"))

	log.Println("\n=== Supported Languages ===")
	log.Println(i18n.SupportedLanguages())
}
