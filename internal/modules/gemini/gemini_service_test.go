package gemini

import (
	"testing"

	"gemini-web-to-api/internal/modules/gemini/dto"
)

func TestParseToolBridgeOutput(t *testing.T) {
	svc := &GeminiService{}
	req := dto.GeminiGenerateRequest{}

	t.Run("call payload parsed into function calls", func(t *testing.T) {
		text := `{"status":"call","tool_calls":[{"name":"translate","arguments":{"text":"hola"}}]}`
		calls, content := svc.parseToolBridgeOutput(req, text)
		if len(calls) != 1 {
			t.Fatalf("calls len = %d, want 1", len(calls))
		}
		if calls[0].Name != "translate" {
			t.Errorf("Name = %q, want translate", calls[0].Name)
		}
		if string(calls[0].Args) != `{"text":"hola"}` {
			t.Errorf("Args = %s, want raw arguments JSON", calls[0].Args)
		}
		if content != "" {
			t.Errorf("content = %q, want empty for call status", content)
		}
	})

	t.Run("text payload returns content", func(t *testing.T) {
		_, content := svc.parseToolBridgeOutput(req, `{"status":"text","content":"answer"}`)
		if content != "answer" {
			t.Errorf("content = %q, want %q", content, "answer")
		}
	})

	t.Run("non-JSON output returned raw", func(t *testing.T) {
		_, content := svc.parseToolBridgeOutput(req, "not json")
		if content != "not json" {
			t.Errorf("content = %q, want raw text", content)
		}
	})
}
