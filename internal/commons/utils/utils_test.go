package utils

import (
	"strings"
	"testing"

	"gemini-web-to-api/internal/commons/models"
)

func TestExtractThinkingAndText(t *testing.T) {
	tests := []struct {
		name          string
		input         string
		wantThinking  string
		wantText      string
	}{
		{
			name:          "No thinking tokens",
			input:         "Hello world",
			wantThinking:  "",
			wantText:      "Hello world",
		},
		{
			name:          "Standard tags with thought",
			input:         "Hello! <ctrl94>thought\nThinking process...\n<ctrl95>Actual response here",
			wantThinking:  "Thinking process...",
			wantText:      "Hello! Actual response here",
		},
		{
			name:          "No end tag",
			input:         "Hello! <ctrl94>thought\nStill thinking...",
			wantThinking:  "Still thinking...",
			wantText:      "Hello!",
		},
		{
			name:          "Alternate start tag",
			input:         "<ctrl94>\nThinking...\n<ctrl95>\nDone",
			wantThinking:  "Thinking...",
			wantText:      "Done",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotThinking, gotText := ExtractThinkingAndText(tt.input)
			if gotThinking != tt.wantThinking {
				t.Errorf("ExtractThinkingAndText() gotThinking = %q, want %q", gotThinking, tt.wantThinking)
			}
			if gotText != tt.wantText {
				t.Errorf("ExtractThinkingAndText() gotText = %q, want %q", gotText, tt.wantText)
			}
		})
	}
}

func TestBuildPromptFromMessages(t *testing.T) {
	tests := []struct {
		name         string
		messages     []models.Message
		systemPrompt string
		want         string
	}{
		{
			name:         "system prompt precedes conversation",
			messages:     []models.Message{{Role: "user", Content: "question"}},
			systemPrompt: "Answer tersely",
			want:         "System: Answer tersely\n\nUser: question",
		},
		{
			name: "assistant and model roles map to Model",
			messages: []models.Message{
				{Role: "user", Content: "Hi"},
				{Role: "assistant", Content: "Hello"},
				{Role: "model", Content: "Hey"},
			},
			want: "User: Hi\nModel: Hello\nModel: Hey",
		},
		{
			name: "empty content falls back to attachment count",
			messages: []models.Message{
				{Role: "user", Content: " ", Attachments: []models.Attachment{{Name: "a.png"}, {Name: "b.png"}}},
			},
			want: "User: [2 file(s) attached]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := BuildPromptFromMessages(tt.messages, tt.systemPrompt); got != tt.want {
				t.Errorf("BuildPromptFromMessages() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateMessages(t *testing.T) {
	tests := []struct {
		name        string
		messages    []models.Message
		wantErrPart string
	}{
		{name: "empty slice rejected", messages: nil, wantErrPart: "cannot be empty"},
		{
			name:        "all whitespace rejected",
			messages:    []models.Message{{Role: "user", Content: "  "}},
			wantErrPart: "all messages have empty content",
		},
		{name: "valid message accepted", messages: []models.Message{{Role: "user", Content: "hi"}}},
		{
			name:     "attachment without text accepted",
			messages: []models.Message{{Role: "user", Attachments: []models.Attachment{{Name: "a.png"}}}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMessages(tt.messages)
			if tt.wantErrPart == "" {
				if err != nil {
					t.Errorf("ValidateMessages() = %v, want nil", err)
				}
				return
			}
			if err == nil || !strings.Contains(err.Error(), tt.wantErrPart) {
				t.Errorf("ValidateMessages() = %v, want error containing %q", err, tt.wantErrPart)
			}
		})
	}
}

func TestStripCodeFence(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "plain text unchanged", input: "hello", want: "hello"},
		{name: "json fence stripped", input: "```json\n{\"a\": 1}\n```", want: "{\"a\": 1}"},
		{name: "unlabeled fence stripped", input: "```\n{}\n```", want: "{}"},
		{name: "missing closing fence tolerated", input: "```json\n{\"a\": 1}", want: "{\"a\": 1}"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := StripCodeFence(tt.input); got != tt.want {
				t.Errorf("StripCodeFence(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}
