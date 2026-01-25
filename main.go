package main

import (
	"log"

	"github.com/angpaoprw/cryptocurrency/i18n"
)

func main() {
	// Initialize i18n
	if err := i18n.Init(); err != nil {
		log.Fatal("Failed to initialize i18n:", err)
	}

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
