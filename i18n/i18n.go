package i18n

import (
	"embed"
	"encoding/json"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"golang.org/x/text/language"
)

//go:embed locales/*.json
var localesFS embed.FS

var bundle *i18n.Bundle

// Init initializes the i18n bundle with available translations
func Init() error {
	bundle = i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)

	// Load all locale files
	locales := []string{"en", "th"}
	for _, locale := range locales {
		if _, err := bundle.LoadMessageFileFS(localesFS, "locales/"+locale+".json"); err != nil {
			return err
		}
	}

	return nil
}

// GetLocalizer returns a localizer for the given language
func GetLocalizer(lang string) *i18n.Localizer {
	if bundle == nil {
		// Initialize if not already done
		if err := Init(); err != nil {
			panic(err)
		}
	}

	return i18n.NewLocalizer(bundle, lang)
}

// Translate translates a message ID to the given language
func Translate(lang, messageID string) string {
	localizer := GetLocalizer(lang)

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID: messageID,
	})
	if err != nil {
		return messageID // Return the message ID if translation fails
	}

	return msg
}

// TranslateWithData translates a message ID with template data
func TranslateWithData(lang, messageID string, templateData map[string]interface{}) string {
	localizer := GetLocalizer(lang)

	msg, err := localizer.Localize(&i18n.LocalizeConfig{
		MessageID:    messageID,
		TemplateData: templateData,
	})
	if err != nil {
		return messageID
	}

	return msg
}

// SupportedLanguages returns a list of supported languages
func SupportedLanguages() []string {
	return []string{"en", "th"}
}
