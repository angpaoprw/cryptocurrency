package i18n_test

import (
	"testing"

	"github.com/angpaoprw/cryptocurrency/i18n"
)

func TestTranslate(t *testing.T) {
	// Initialize i18n
	if err := i18n.Init(); err != nil {
		t.Fatalf("Failed to initialize i18n: %v", err)
	}

	tests := []struct {
		name      string
		lang      string
		messageID string
		want      string
	}{
		{
			name:      "English welcome message",
			lang:      "en",
			messageID: "welcome",
			want:      "Welcome to Cryptocurrency Platform",
		},
		{
			name:      "Thai welcome message",
			lang:      "th",
			messageID: "welcome",
			want:      "ยินดีต้อนรับสู่แพลตฟอร์มคริปโตเคอเรนซี",
		},
		{
			name:      "English wallet created",
			lang:      "en",
			messageID: "wallet.created",
			want:      "Wallet created successfully",
		},
		{
			name:      "Thai wallet created",
			lang:      "th",
			messageID: "wallet.created",
			want:      "สร้างกระเป๋าเงินสำเร็จ",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := i18n.Translate(tt.lang, tt.messageID)
			if got != tt.want {
				t.Errorf("Translate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSupportedLanguages(t *testing.T) {
	langs := i18n.SupportedLanguages()

	if len(langs) != 2 {
		t.Errorf("Expected 2 supported languages, got %d", len(langs))
	}

	expectedLangs := map[string]bool{"en": true, "th": true}
	for _, lang := range langs {
		if !expectedLangs[lang] {
			t.Errorf("Unexpected language: %s", lang)
		}
	}
}
