package dto

import (
	"encoding/json"
	"testing"
)

func TestAnthropicSystemContent_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		want     string
	}{
		{
			name:     "plain string form",
			jsonData: `"You are helpful"`,
			want:     "You are helpful",
		},
		{
			name:     "empty string",
			jsonData: `""`,
			want:     "",
		},
		{
			name:     "single text block",
			jsonData: `[{"type": "text", "text": "Be terse"}]`,
			want:     "Be terse",
		},
		{
			name:     "multiple text blocks joined with newline",
			jsonData: `[{"type": "text", "text": "First"}, {"type": "text", "text": "Second"}]`,
			want:     "First\nSecond",
		},
		{
			name:     "non-text blocks skipped",
			jsonData: `[{"type": "image", "text": "ignored"}, {"type": "text", "text": "kept"}]`,
			want:     "kept",
		},
		{
			name:     "array with no text blocks yields empty",
			jsonData: `[{"type": "image"}]`,
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var s AnthropicSystemContent
			if err := json.Unmarshal([]byte(tt.jsonData), &s); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if s.Text != tt.want {
				t.Errorf("Text = %q, want %q", s.Text, tt.want)
			}
		})
	}
}
