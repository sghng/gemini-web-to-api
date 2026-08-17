package dto

import (
	"encoding/json"
	"testing"
)

func TestPart_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name     string
		jsonData string
		check    func(t *testing.T, p Part)
	}{
		{
			name:     "text part",
			jsonData: `{"text": "hello"}`,
			check: func(t *testing.T, p Part) {
				if p.Text != "hello" {
					t.Errorf("Text = %q, want hello", p.Text)
				}
			},
		},
		{
			name:     "snake_case inline_data",
			jsonData: `{"inline_data": {"mime_type": "image/png", "data": "aA=="}}`,
			check: func(t *testing.T, p Part) {
				if p.InlineData == nil || p.InlineData.MimeType != "image/png" || p.InlineData.Data != "aA==" {
					t.Errorf("InlineData = %+v, want image/png aA==", p.InlineData)
				}
			},
		},
		{
			name:     "camelCase inlineData",
			jsonData: `{"inlineData": {"mimeType": "image/jpeg", "data": "bB=="}}`,
			check: func(t *testing.T, p Part) {
				if p.InlineData == nil || p.InlineData.MimeType != "image/jpeg" {
					t.Errorf("InlineData = %+v, want image/jpeg", p.InlineData)
				}
			},
		},
		{
			name:     "camelCase functionCall",
			jsonData: `{"functionCall": {"name": "search", "args": {"q": "go"}}}`,
			check: func(t *testing.T, p Part) {
				if p.FunctionCall == nil || p.FunctionCall.Name != "search" {
					t.Errorf("FunctionCall = %+v, want search", p.FunctionCall)
				}
			},
		},
		{
			name:     "snake_case function_response",
			jsonData: `{"function_response": {"name": "search", "response": {"ok": true}}}`,
			check: func(t *testing.T, p Part) {
				if p.FunctionResponse == nil || p.FunctionResponse.Name != "search" {
					t.Errorf("FunctionResponse = %+v, want search", p.FunctionResponse)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var p Part
			if err := json.Unmarshal([]byte(tt.jsonData), &p); err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			tt.check(t, p)
		})
	}
}

func TestInlineData_UnmarshalJSON(t *testing.T) {
	t.Run("snake_case mime_type fallback", func(t *testing.T) {
		var d InlineData
		if err := json.Unmarshal([]byte(`{"mime_type": "image/webp", "data": "bB=="}`), &d); err != nil {
			t.Fatal(err)
		}
		if d.MimeType != "image/webp" || d.Data != "bB==" {
			t.Errorf("InlineData = %+v, want image/webp bB==", d)
		}
	})

	t.Run("camelCase takes precedence", func(t *testing.T) {
		var d InlineData
		if err := json.Unmarshal([]byte(`{"mimeType": "image/png", "mime_type": "image/webp", "data": ""}`), &d); err != nil {
			t.Fatal(err)
		}
		if d.MimeType != "image/png" {
			t.Errorf("MimeType = %q, want camelCase image/png to win", d.MimeType)
		}
	})
}
