package openai

import (
	"encoding/json"
	"strings"
	"testing"

	"gemini-web-to-api/internal/modules/openai/dto"
)

func weatherTool() dto.ToolDefinition {
	return dto.ToolDefinition{
		Type: "function",
		Function: dto.ToolFunctionDefinition{
			Name:        "get_weather",
			Description: "Get current weather",
			Parameters:  json.RawMessage(`{"type":"object","properties":{"city":{"type":"string"}}}`),
		},
	}
}

func TestBuildImageGenerationPrompt(t *testing.T) {
	t.Run("size omitted when empty", func(t *testing.T) {
		got := buildImageGenerationPrompt("a cat", "")
		if !strings.Contains(got, "Prompt: a cat") || strings.Contains(got, "Requested size") {
			t.Errorf("unexpected prompt: %q", got)
		}
	})

	t.Run("size appended when provided", func(t *testing.T) {
		got := buildImageGenerationPrompt("a cat", "1024x1024")
		if !strings.Contains(got, "Requested size/aspect: 1024x1024") {
			t.Errorf("size section missing: %q", got)
		}
	})
}

func TestParseToolBridgeOutput(t *testing.T) {
	svc := &OpenAIService{}
	req := dto.ChatCompletionRequest{Tools: []dto.ToolDefinition{weatherTool()}}

	t.Run("parses valid tool call payload", func(t *testing.T) {
		text := `{"tool_calls":[{"name":"get_weather","arguments":{"city":"Hanoi"}}]}`
		calls, _ := svc.parseToolBridgeOutput(req, text)
		if len(calls) != 1 {
			t.Fatalf("calls len = %d, want 1", len(calls))
		}
		if calls[0].Type != "function" || calls[0].Function.Name != "get_weather" {
			t.Errorf("call = %+v, want function/get_weather", calls[0])
		}
		if calls[0].Function.Arguments != `{"city":"Hanoi"}` {
			t.Errorf("arguments = %q, want compact JSON", calls[0].Function.Arguments)
		}
	})

	t.Run("parses content-only payload", func(t *testing.T) {
		_, content := svc.parseToolBridgeOutput(req, `{"content":"just text"}`)
		if content != "just text" {
			t.Errorf("content = %q, want %q", content, "just text")
		}
	})

	t.Run("non-JSON output returned as plain content", func(t *testing.T) {
		_, content := svc.parseToolBridgeOutput(req, "I cannot help with that")
		if content != "I cannot help with that" {
			t.Errorf("content = %q, want raw text", content)
		}
	})

	t.Run("unknown tool names filtered out", func(t *testing.T) {
		text := `{"tool_calls":[{"name":"hack_mainframe","arguments":{}}]}`
		calls, _ := svc.parseToolBridgeOutput(req, text)
		if len(calls) != 0 {
			t.Errorf("calls = %v, want unknown tool filtered", calls)
		}
	})

	t.Run("forced tool name filters other calls", func(t *testing.T) {
		forcedReq := dto.ChatCompletionRequest{
			Tools:         []dto.ToolDefinition{weatherTool()},
			ToolChoiceRaw: json.RawMessage(`{"type":"function","function":{"name":"get_weather"}}`),
		}
		text := `{"tool_calls":[{"name":"other","arguments":{}},{"name":"get_weather","arguments":{}}]}`
		calls, _ := svc.parseToolBridgeOutput(forcedReq, text)
		if len(calls) != 1 || calls[0].Function.Name != "get_weather" {
			t.Errorf("calls = %+v, want only forced get_weather", calls)
		}
	})

	t.Run("code fenced JSON still parsed", func(t *testing.T) {
		text := "```json\n{\"tool_calls\":[{\"name\":\"get_weather\",\"arguments\":{}}]}\n```"
		calls, _ := svc.parseToolBridgeOutput(req, text)
		if len(calls) != 1 {
			t.Errorf("calls len = %d, want 1 for fenced payload", len(calls))
		}
	})
}

func TestBuildFallbackToolCalls(t *testing.T) {
	svc := &OpenAIService{}

	t.Run("forced tool yields fallback call", func(t *testing.T) {
		req := dto.ChatCompletionRequest{
			Tools:         []dto.ToolDefinition{weatherTool()},
			ToolChoiceRaw: json.RawMessage(`{"type":"function","function":{"name":"get_weather"}}`),
		}
		calls := svc.buildFallbackToolCalls(req)
		if len(calls) != 1 || calls[0].Function.Name != "get_weather" || calls[0].Function.Arguments != "{}" {
			t.Errorf("calls = %+v, want forced get_weather with empty args", calls)
		}
	})

	t.Run("required mode falls back to first function tool", func(t *testing.T) {
		req := dto.ChatCompletionRequest{
			Tools:         []dto.ToolDefinition{weatherTool()},
			ToolChoiceRaw: json.RawMessage(`"required"`),
		}
		calls := svc.buildFallbackToolCalls(req)
		if len(calls) != 1 || calls[0].Function.Name != "get_weather" {
			t.Errorf("calls = %+v, want first function tool", calls)
		}
	})

	t.Run("auto mode yields no fallback", func(t *testing.T) {
		req := dto.ChatCompletionRequest{Tools: []dto.ToolDefinition{weatherTool()}}
		if calls := svc.buildFallbackToolCalls(req); calls != nil {
			t.Errorf("calls = %+v, want nil for auto mode", calls)
		}
	})
}

func TestExtractFirstJSONObject(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "nested object", input: `{"a":{"b":2}} trailing`, want: `{"a":{"b":2}}`},
		{name: "braces inside string ignored", input: `{"a":"}{"}`, want: `{"a":"}{"}`},
		{name: "no object", input: "plain text", want: ""},
		{name: "unclosed object", input: `{"a":1`, want: ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := extractFirstJSONObject(tt.input); got != tt.want {
				t.Errorf("extractFirstJSONObject(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestNormalizeArguments(t *testing.T) {
	tests := []struct {
		name string
		raw  string
		want string
	}{
		{name: "empty becomes empty object", raw: ``, want: `{}`},
		{name: "object compacted", raw: "{ \"a\" : 1 }", want: `{"a":1}`},
		{name: "string-encoded JSON unwrapped", raw: `"{\"a\":1}"`, want: `{"a":1}`},
		{name: "invalid JSON becomes empty object", raw: `{not json}`, want: `{}`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeArguments(json.RawMessage(tt.raw)); got != tt.want {
				t.Errorf("normalizeArguments(%q) = %q, want %q", tt.raw, got, tt.want)
			}
		})
	}
}
