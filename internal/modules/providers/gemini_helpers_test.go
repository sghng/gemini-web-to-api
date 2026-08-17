package providers

import (
	"strings"
	"testing"
)

func TestResolveAvailableModelCases(t *testing.T) {
	models := []ModelInfo{
		{ID: "gemini-3.6-flash"},
		{ID: "gemini-3.7-flash-09-2025"},
		{ID: "gemini-3.7-flash-10-2025"},
	}

	tests := []struct {
		name      string
		requested string
		wantID    string
		wantOK    bool
	}{
		{name: "exact match", requested: "gemini-3.6-flash", wantID: "gemini-3.6-flash", wantOK: true},
		{name: "empty request rejected", requested: "", wantOK: false},
		{name: "ambiguous prefix rejected", requested: "gemini-3.7-flash", wantID: "gemini-3.7-flash", wantOK: false},
		{name: "unknown model rejected", requested: "gpt-4", wantID: "gpt-4", wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotOK := resolveAvailableModel(tt.requested, models)
			if gotOK != tt.wantOK || gotID != tt.wantID {
				t.Errorf("resolveAvailableModel(%q) = (%q, %v), want (%q, %v)",
					tt.requested, gotID, gotOK, tt.wantID, tt.wantOK)
			}
		})
	}
}

// TestBuildGenerateInner guards the StreamGenerate wire payload shape: Google
// addresses fields positionally, so a misplaced slot breaks generation.
func TestBuildGenerateInner(t *testing.T) {
	t.Run("prompt only message content", func(t *testing.T) {
		inner := buildGenerateInner("hello", nil, "gemini-flash", "en", "req-1", false)

		content, ok := inner[0].([]interface{})
		if !ok || len(content) != 1 || content[0] != "hello" {
			t.Errorf("inner[0] = %v, want [hello]", inner[0])
		}
		if inner[3] != "gemini-flash" {
			t.Errorf("inner[3] = %v, want model id", inner[3])
		}
		if inner[59] != "req-1" {
			t.Errorf("inner[59] = %v, want request ID", inner[59])
		}
		if inner[45] != nil || inner[67] != nil {
			t.Errorf("temporary flags set for persistent chat: inner[45]=%v inner[67]=%v", inner[45], inner[67])
		}
	})

	t.Run("files embedded in message content", func(t *testing.T) {
		files := []uploadedFile{{ID: "file-id-1", Name: "report.pdf"}}
		inner := buildGenerateInner("summarize", files, "gemini-flash", "en", "req-2", false)

		content, ok := inner[0].([]interface{})
		if !ok || len(content) != 7 {
			t.Fatalf("inner[0] = %v, want 7-slot file message", inner[0])
		}
		if content[0] != "summarize" {
			t.Errorf("prompt slot = %v, want summarize", content[0])
		}
		fileData, ok := content[3].([]interface{})
		if !ok || len(fileData) != 1 {
			t.Fatalf("file refs slot = %v, want one entry", content[3])
		}
		entry, ok := fileData[0].([]interface{})
		if !ok || len(entry) != 2 || entry[1] != "report.pdf" {
			t.Errorf("file entry = %v, want [id, report.pdf]", fileData[0])
		}
	})

	t.Run("temporary chat sets persistence flags", func(t *testing.T) {
		inner := buildGenerateInner("hi", nil, "gemini-flash", "en", "req-3", true)
		if inner[45] != 1 || inner[67] != 0 {
			t.Errorf("temporary flags = (%v, %v), want (1, 0)", inner[45], inner[67])
		}
	})
}

func TestExtractBardError(t *testing.T) {
	t.Run("no error block returns empty", func(t *testing.T) {
		item := []interface{}{"some text", []interface{}{float64(1), "fine"}}
		if got := extractBardError(item); got != "" {
			t.Errorf("extractBardError() = %q, want empty", got)
		}
	})

	t.Run("error block with codes reported", func(t *testing.T) {
		item := []interface{}{
			[]interface{}{
				"type.googleapis.com/assistant.boq.bard.application.BardErrorInfo",
				[]interface{}{float64(13), float64(1152)},
			},
		}
		got := extractBardError(item)
		if !strings.Contains(got, "BardErrorInfo") || !strings.Contains(got, "[13 1152]") {
			t.Errorf("extractBardError() = %q, want codes mentioned", got)
		}
	})
}

func TestExtractGeneratedImagesRejectsMalformed(t *testing.T) {
	t.Run("short candidate rejected", func(t *testing.T) {
		if got := extractGeneratedImages([]interface{}{"a", "b"}); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})

	t.Run("malformed media slot rejected", func(t *testing.T) {
		candidate := make([]interface{}, 13)
		candidate[12] = "not-a-list"
		if got := extractGeneratedImages(candidate); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
}

func TestNormalizeImageURLCases(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "https URL passthrough", input: "https://example.com/cat.png", want: "https://example.com/cat.png"},
		{name: "protocol-relative upgraded", input: "//example.com/cat.png", want: "https://example.com/cat.png"},
		{name: "bare googleusercontent upgraded", input: "lh3.googleusercontent.com/abc", want: "https://lh3.googleusercontent.com/abc"},
		{name: "trailing punctuation stripped", input: "https://example.com/cat.png.", want: "https://example.com/cat.png"},
		{name: "svg rejected", input: "https://example.com/icon.svg", want: ""},
		{name: "non-image extension rejected", input: "https://example.com/doc.txt", want: ""},
		{name: "placeholder path rejected", input: "https://googleusercontent.com/image_generation_content/1", want: ""},
		{name: "nonsense rejected", input: "not a url at all", want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeImageURL(tt.input); got != tt.want {
				t.Errorf("normalizeImageURL(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
