package dto

import (
	"encoding/json"
	"testing"
)

func TestChatCompletionMessage_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name            string
		jsonData        string
		wantContent     string
		wantAttachments int
		wantErr         bool
	}{
		{
			name:        "plain string content",
			jsonData:    `{"role": "user", "content": "hello"}`,
			wantContent: "hello",
		},
		{
			name:        "null content",
			jsonData:    `{"role": "user", "content": null}`,
			wantContent: "",
		},
		{
			name: "text parts joined with newline",
			jsonData: `{"role": "user", "content": [
				{"type": "text", "text": "first"},
				{"type": "text", "text": "second"}
			]}`,
			wantContent: "first\nsecond",
		},
		{
			name: "data URL image becomes attachment",
			jsonData: `{"role": "user", "content": [
				{"type": "image_url", "image_url": {"url": "data:image/png;base64,aGVsbG8="}}
			]}`,
			wantAttachments: 1,
		},
		{
			name: "input_image alias becomes attachment",
			jsonData: `{"role": "user", "content": [
				{"type": "input_image", "image_url": {"url": "data:image/jpeg;base64,aGVsbG8="}}
			]}`,
			wantAttachments: 1,
		},
		{
			name: "remote image URL skipped",
			jsonData: `{"role": "user", "content": [
				{"type": "image_url", "image_url": {"url": "https://example.com/cat.png"}}
			]}`,
			wantAttachments: 0,
		},
		{
			name:     "numeric content rejected",
			jsonData: `{"role": "user", "content": 42}`,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg ChatCompletionMessage
			err := json.Unmarshal([]byte(tt.jsonData), &msg)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected unmarshal error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unmarshal failed: %v", err)
			}
			if msg.Content != tt.wantContent {
				t.Errorf("Content = %q, want %q", msg.Content, tt.wantContent)
			}
			if len(msg.Attachments) != tt.wantAttachments {
				t.Errorf("Attachments len = %d, want %d", len(msg.Attachments), tt.wantAttachments)
			}
		})
	}
}

func TestAttachmentFromDataURL(t *testing.T) {
	tests := []struct {
		name     string
		value    string
		index    int
		wantOK   bool
		wantName string
		wantMime string
	}{
		{name: "png data URL", value: "data:image/png;base64,aGVsbG8=", index: 1, wantOK: true, wantName: "image_1.png", wantMime: "image/png"},
		{name: "jpeg maps to jpg extension", value: "data:image/jpeg;base64,/9j/4A==", index: 2, wantOK: true, wantName: "image_2.jpg", wantMime: "image/jpeg"},
		{name: "empty mime falls back to octet-stream", value: "data:;base64,aA==", index: 1, wantOK: true, wantName: "image_1.bin", wantMime: "application/octet-stream"},
		{name: "non-data URL rejected", value: "https://example.com/a.png", index: 1, wantOK: false},
		{name: "missing base64 marker rejected", value: "data:image/png,aGVsbG8=", index: 1, wantOK: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			att, ok := attachmentFromDataURL(tt.value, tt.index)
			if ok != tt.wantOK {
				t.Fatalf("attachmentFromDataURL(%q) ok = %v, want %v", tt.value, ok, tt.wantOK)
			}
			if !ok {
				return
			}
			if att.Name != tt.wantName || att.MimeType != tt.wantMime {
				t.Errorf("attachment = (%q, %q), want (%q, %q)", att.Name, att.MimeType, tt.wantName, tt.wantMime)
			}
		})
	}
}

func TestToolChoiceMode(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "absent defaults to auto", raw: "", want: "auto"},
		{name: "string none", raw: `"none"`, want: "none"},
		{name: "string auto", raw: `"auto"`, want: "auto"},
		{name: "string required", raw: `"required"`, want: "required"},
		{name: "unknown string falls back to auto", raw: `"whatever"`, want: "auto"},
		{name: "function object", raw: `{"type": "function", "function": {"name": "get_weather"}}`, want: "function"},
		{name: "function object without name falls back to auto", raw: `{"type": "function", "function": {"name": ""}}`, want: "auto"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ChatCompletionRequest{ToolChoiceRaw: json.RawMessage(tt.raw)}
			if got := req.ToolChoiceMode(); got != tt.want {
				t.Errorf("ToolChoiceMode(%s) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestForcedToolName(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "absent returns empty", raw: "", want: ""},
		{name: "string mode returns empty", raw: `"required"`, want: ""},
		{name: "function object returns name", raw: `{"type": "function", "function": {"name": "get_weather"}}`, want: "get_weather"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := ChatCompletionRequest{ToolChoiceRaw: json.RawMessage(tt.raw)}
			if got := req.ForcedToolName(); got != tt.want {
				t.Errorf("ForcedToolName(%s) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}

func TestHasToolsEnabled(t *testing.T) {
	tool := ToolDefinition{Type: "function", Function: ToolFunctionDefinition{Name: "f"}}

	tests := []struct {
		name string
		req  ChatCompletionRequest
		want bool
	}{
		{name: "no tools disabled", req: ChatCompletionRequest{}, want: false},
		{name: "tools with default choice enabled", req: ChatCompletionRequest{Tools: []ToolDefinition{tool}}, want: true},
		{
			name: "tools with choice none disabled",
			req: ChatCompletionRequest{
				Tools:         []ToolDefinition{tool},
				ToolChoiceRaw: json.RawMessage(`"none"`),
			},
			want: false,
		},
		{
			name: "tools with required choice enabled",
			req: ChatCompletionRequest{
				Tools:         []ToolDefinition{tool},
				ToolChoiceRaw: json.RawMessage(`"required"`),
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.req.HasToolsEnabled(); got != tt.want {
				t.Errorf("HasToolsEnabled() = %v, want %v", got, tt.want)
			}
		})
	}
}
