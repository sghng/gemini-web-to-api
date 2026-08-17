package models

import (
	"encoding/json"
	"testing"
)

func TestMessage_UnmarshalJSON(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		wantRole    string
		wantContent string
	}{
		{
			name:        "Simple text message",
			jsonData:    `{"role": "user", "content": "Hello"}`,
			wantRole:    "user",
			wantContent: "Hello",
		},
		{
			name: "Claude tool_use message",
			jsonData: `{
				"role": "assistant",
				"content": [
					{"type": "text", "text": "Let me run this"},
					{"type": "tool_use", "id": "toolu_123", "name": "run_command", "input": {"cmd": "ls"}}
				]
			}`,
			wantRole:    "assistant",
			wantContent: "Let me run this\n[Call Tool: run_command with ID toolu_123 and Input: {\"cmd\": \"ls\"}]",
		},
		{
			name: "Claude tool_result message with string content",
			jsonData: `{
				"role": "user",
				"content": [
					{"type": "tool_result", "tool_use_id": "toolu_123", "content": "file1.txt\nfile2.txt"}
				]
			}`,
			wantRole:    "user",
			wantContent: "[Tool Result for ID toolu_123]: file1.txt\nfile2.txt",
		},
		{
			name: "Claude tool_result message with structured content blocks",
			jsonData: `{
				"role": "user",
				"content": [
					{
						"type": "tool_result",
						"tool_use_id": "toolu_123",
						"content": [{"type": "text", "text": "result content"}]
					}
				]
			}`,
			wantRole:    "user",
			wantContent: "[Tool Result for ID toolu_123]: result content",
		},
		{
			name: "OpenAI tool result message",
			jsonData: `{
				"role": "tool",
				"tool_call_id": "call_123",
				"content": "command output"
			}`,
			wantRole:    "tool",
			wantContent: "[Tool Result for ID call_123]: command output",
		},
		{
			name: "OpenAI tool_calls message",
			jsonData: `{
				"role": "assistant",
				"content": "Preparing to execute",
				"tool_calls": [
					{
						"id": "call_123",
						"type": "function",
						"function": {
							"name": "read_file",
							"arguments": "{\"path\":\"main.go\"}"
						}
					}
				]
			}`,
			wantRole:    "assistant",
			wantContent: "Preparing to execute\n[Call Tool: read_file with ID call_123 and Input: {\"path\":\"main.go\"}]",
		},
		{
			name:        "null content yields empty message",
			jsonData:    `{"role": "assistant", "content": null}`,
			wantRole:    "assistant",
			wantContent: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var msg Message
			if err := json.Unmarshal([]byte(tt.jsonData), &msg); err != nil {
				t.Fatalf("Failed to unmarshal JSON: %v", err)
			}
			if msg.Role != tt.wantRole {
				t.Errorf("Role got = %q, want %q", msg.Role, tt.wantRole)
			}
			if msg.Content != tt.wantContent {
				t.Errorf("Content got = %q, want %q", msg.Content, tt.wantContent)
			}
		})
	}
}

func TestMessage_UnmarshalJSON_ImageParts(t *testing.T) {
	t.Run("base64 image becomes attachment", func(t *testing.T) {
		data := `{
			"role": "user",
			"content": [
				{"type": "text", "text": "describe this"},
				{"type": "image", "source": {"type": "base64", "media_type": "image/png", "data": "aGVsbG8="}}
			]
		}`
		var msg Message
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if msg.Content != "describe this" {
			t.Errorf("Content = %q, want %q", msg.Content, "describe this")
		}
		if len(msg.Attachments) != 1 {
			t.Fatalf("Attachments len = %d, want 1", len(msg.Attachments))
		}
		att := msg.Attachments[0]
		if att.Name != "image_2.png" || att.MimeType != "image/png" || att.Data != "aGVsbG8=" {
			t.Errorf("Attachment = %+v, want image_2.png image/png aGVsbG8=", att)
		}
	})

	t.Run("non-base64 image is skipped", func(t *testing.T) {
		data := `{
			"role": "user",
			"content": [
				{"type": "image", "source": {"type": "url", "media_type": "image/png", "data": ""}}
			]
		}`
		var msg Message
		if err := json.Unmarshal([]byte(data), &msg); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if len(msg.Attachments) != 0 {
			t.Errorf("Attachments len = %d, want 0 for non-base64 source", len(msg.Attachments))
		}
	})
}

func TestMessage_UnmarshalJSON_RejectsUnsupportedContent(t *testing.T) {
	var msg Message
	if err := json.Unmarshal([]byte(`{"role": "user", "content": 42}`), &msg); err == nil {
		t.Fatal("expected error for numeric content")
	}
}
