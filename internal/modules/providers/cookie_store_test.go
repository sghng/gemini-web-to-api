package providers

import (
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"go.uber.org/zap"
)

func TestCleanCookie(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain value unchanged", input: "abc123", want: "abc123"},
		{name: "double quotes stripped", input: `"abc123"`, want: "abc123"},
		{name: "whitespace and semicolon removed", input: ` "abc123;" `, want: "abc123"},
		{name: "empty stays empty", input: "", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanCookie(tt.input); got != tt.want {
				t.Errorf("cleanCookie(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestToHTTPCookies(t *testing.T) {
	t.Run("empty store yields no cookies", func(t *testing.T) {
		store := &CookieStore{}
		if got := store.ToHTTPCookies(); len(got) != 0 {
			t.Errorf("ToHTTPCookies() len = %d, want 0", len(got))
		}
	})

	t.Run("both cookies carry security attributes", func(t *testing.T) {
		store := &CookieStore{Secure1PSID: `"quoted-psid"`, Secure1PSIDTS: "ts;"}
		got := store.ToHTTPCookies()
		if len(got) != 2 {
			t.Fatalf("ToHTTPCookies() len = %d, want 2", len(got))
		}

		byName := map[string]*http.Cookie{}
		for _, c := range got {
			byName[c.Name] = c
		}

		if psid := byName["__Secure-1PSID"]; psid == nil || psid.Value != "quoted-psid" {
			t.Errorf("PSID cookie = %+v, want cleaned value", psid)
		}
		if ts := byName["__Secure-1PSIDTS"]; ts == nil || ts.Value != "ts" {
			t.Errorf("PSIDTS cookie = %+v, want cleaned value", ts)
		}

		for _, c := range got {
			if c.Domain != ".google.com" || !c.Secure || !c.HttpOnly || c.SameSite != http.SameSiteNoneMode {
				t.Errorf("cookie %s missing security attributes: %+v", c.Name, c)
			}
		}
	})
}

// TestCookieCacheRoundTrip exercises the persistence path that keeps sessions
// alive across restarts. It runs inside a temporary working directory so the
// real .cookies dir is never touched.
func TestCookieCacheRoundTrip(t *testing.T) {
	t.Chdir(t.TempDir())
	client := &Client{
		cookies: &CookieStore{Secure1PSID: "my-psid", Secure1PSIDTS: "my-psidts"},
		log:     zap.NewNop(),
	}

	if err := client.SaveCachedCookies(); err != nil {
		t.Fatalf("SaveCachedCookies() error = %v", err)
	}

	matches, err := filepath.Glob(filepath.Join(".cookies", "*.txt"))
	if err != nil || len(matches) != 1 {
		t.Fatalf("expected one cache file, got %v (err %v)", matches, err)
	}
	info, err := os.Stat(matches[0])
	if err != nil {
		t.Fatalf("stat cache file: %v", err)
	}
	if perm := info.Mode().Perm(); perm != 0o600 {
		t.Errorf("cache file perm = %o, want 600", perm)
	}

	loaded, err := client.LoadCachedCookies()
	if err != nil {
		t.Fatalf("LoadCachedCookies() error = %v", err)
	}
	if loaded != "my-psidts" {
		t.Errorf("LoadCachedCookies() = %q, want %q", loaded, "my-psidts")
	}

	if err := client.ClearCookieCache(); err != nil {
		t.Fatalf("ClearCookieCache() error = %v", err)
	}
	if _, err := client.LoadCachedCookies(); err == nil {
		t.Error("LoadCachedCookies() after clear succeeded, want error")
	}
}

func TestCookieCacheEdgeCases(t *testing.T) {
	t.Run("load without PSID fails", func(t *testing.T) {
		t.Chdir(t.TempDir())
		client := &Client{cookies: &CookieStore{}, log: zap.NewNop()}
		if _, err := client.LoadCachedCookies(); err == nil {
			t.Error("LoadCachedCookies() succeeded, want error for missing PSID")
		}
	})

	t.Run("save without values is a no-op", func(t *testing.T) {
		t.Chdir(t.TempDir())
		client := &Client{cookies: &CookieStore{}, log: zap.NewNop()}
		if err := client.SaveCachedCookies(); err != nil {
			t.Errorf("SaveCachedCookies() error = %v, want nil no-op", err)
		}
		matches, _ := filepath.Glob(filepath.Join(".cookies", "*.txt"))
		if len(matches) != 0 {
			t.Errorf("cache files created for empty cookies: %v", matches)
		}
	})
}
