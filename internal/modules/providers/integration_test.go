//go:build integration

package providers_test

// Live smoke test for the three critical paths against the real Gemini web
// endpoint: authentication, model listing, text generation, and image
// generation. It never runs by default; invoke it explicitly with:
//
//	GEMINI_1PSID=... GEMINI_1PSIDTS=... go test ./internal/modules/providers/ -tags=integration -v -timeout=5m
//
// Cookies are read from the environment only — nothing is written to disk
// beyond the client's standard .cookies cache in a temp working directory.

import (
	"context"
	"os"
	"testing"
	"time"

	"gemini-web-to-api/internal/commons/configs"
	"gemini-web-to-api/internal/modules/providers"

	"go.uber.org/zap"
)

func newLiveClient(t *testing.T) *providers.Client {
	t.Helper()

	psid := os.Getenv("GEMINI_1PSID")
	psidts := os.Getenv("GEMINI_1PSIDTS")
	if psid == "" || psidts == "" {
		t.Skip("GEMINI_1PSID/GEMINI_1PSIDTS not set; skipping live smoke test")
	}

	t.Chdir(t.TempDir()) // keep the .cookies cache out of the repo

	cfg := &configs.Config{}
	cfg.Gemini.Secure1PSID = psid
	cfg.Gemini.Secure1PSIDTS = psidts
	cfg.Gemini.RefreshInterval = 30
	cfg.Gemini.MaxRetries = 3

	client := providers.NewClient(cfg, zap.NewNop())
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	if err := client.Init(ctx); err != nil {
		t.Fatalf("Init() failed (cookies may be expired): %v", err)
	}
	t.Cleanup(func() { client.Close() })
	return client
}

func TestLiveSmoke(t *testing.T) {
	client := newLiveClient(t)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	t.Run("access models", func(t *testing.T) {
		models := client.ListModels()
		if len(models) == 0 {
			t.Fatal("ListModels() returned no models")
		}
		t.Logf("available models: %v", client.ListModelsIDs())
	})

	t.Run("text request/reply", func(t *testing.T) {
		resp, err := client.GenerateContent(ctx, "Reply with exactly: OK")
		if err != nil {
			t.Fatalf("GenerateContent() failed: %v", err)
		}
		if resp.Text == "" {
			t.Fatal("GenerateContent() returned empty text")
		}
		t.Logf("reply: %.80q", resp.Text)
	})

	t.Run("image generation", func(t *testing.T) {
		resp, err := client.GenerateContent(ctx,
			"Generate an image of a plain solid red circle on a white background.",
			providers.WithGeneratedImageDownload(false),
		)
		if err != nil {
			t.Fatalf("GenerateContent() for image failed: %v", err)
		}
		generated := 0
		for _, img := range resp.Images {
			if img.Generated {
				generated++
			}
		}
		if generated == 0 {
			t.Fatalf("no generated images in response (account may lack image generation); got %+v", resp.Images)
		}
	})
}
