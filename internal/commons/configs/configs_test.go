package configs

import (
	"os"
	"strings"
	"testing"
)

// clearEnv unsets config-related environment variables for the duration of a
// test, restoring original values afterwards.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, key := range []string{"PORT", "GEMINI_1PSID", "GEMINI_1PSIDTS"} {
		key := key
		if value, ok := os.LookupEnv(key); ok {
			os.Unsetenv(key)
			t.Cleanup(func() { os.Setenv(key, value) })
		}
	}
}

func TestNewAppliesDefaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("GEMINI_1PSID", "psid-value")
	t.Setenv("GEMINI_1PSIDTS", "psidts-value")

	cfg, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}

	if cfg.Server.Port != defaultServerPort {
		t.Errorf("Port = %q, want %q", cfg.Server.Port, defaultServerPort)
	}
	if cfg.Gemini.RefreshInterval != defaultGeminiRefreshInterval {
		t.Errorf("RefreshInterval = %d, want %d", cfg.Gemini.RefreshInterval, defaultGeminiRefreshInterval)
	}
	if cfg.Gemini.MaxRetries != defaultGeminiMaxRetries {
		t.Errorf("MaxRetries = %d, want %d", cfg.Gemini.MaxRetries, defaultGeminiMaxRetries)
	}
	if cfg.Gemini.Secure1PSID != "psid-value" || cfg.Gemini.Secure1PSIDTS != "psidts-value" {
		t.Errorf("cookies = (%q, %q), want provided values", cfg.Gemini.Secure1PSID, cfg.Gemini.Secure1PSIDTS)
	}
}

func TestNewValidatesRequiredCookies(t *testing.T) {
	tests := []struct {
		name        string
		psid        string
		psidts      string
		wantMissing string
	}{
		{name: "missing PSID fails", psid: "", psidts: "ts", wantMissing: "GEMINI_1PSID"},
		{name: "PSID without PSIDTS fails", psid: "psid", psidts: "", wantMissing: "GEMINI_1PSIDTS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			clearEnv(t)
			t.Setenv("GEMINI_1PSID", tt.psid)
			t.Setenv("GEMINI_1PSIDTS", tt.psidts)

			_, err := New()
			if err == nil || !strings.Contains(err.Error(), tt.wantMissing) {
				t.Errorf("New() error = %v, want mention of %s", err, tt.wantMissing)
			}
		})
	}
}

func TestNewRejectsNonNumericPort(t *testing.T) {
	clearEnv(t)
	t.Setenv("GEMINI_1PSID", "psid")
	t.Setenv("GEMINI_1PSIDTS", "psidts")
	t.Setenv("PORT", "http")

	if _, err := New(); err == nil || !strings.Contains(err.Error(), "invalid PORT") {
		t.Errorf("New() error = %v, want invalid PORT error", err)
	}
}
