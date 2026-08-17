package claude

import (
	"strings"
	"testing"

	"gemini-web-to-api/internal/modules/claude/dto"
)

func TestParseToolBridgeOutput(t *testing.T) {
	svc := &ClaudeService{}
	req := dto.MessageRequest{}

	t.Run("tool_use payload parsed", func(t *testing.T) {
		text := `{"status":"tool_use","tool_calls":[{"id":"toolu_1","name":"web_search","input":{"query":"go"}}]}`
		uses, content := svc.parseToolBridgeOutput(req, text)
		if len(uses) != 1 {
			t.Fatalf("uses len = %d, want 1", len(uses))
		}
		if uses[0].Type != "tool_use" || uses[0].ID != "toolu_1" || uses[0].Name != "web_search" {
			t.Errorf("use = %+v, want tool_use toolu_1/web_search", uses[0])
		}
		if uses[0].Input["query"] != "go" {
			t.Errorf("input = %v, want query=go", uses[0].Input)
		}
		if content != "" {
			t.Errorf("content = %q, want empty for tool_use", content)
		}
	})

	t.Run("missing id generated", func(t *testing.T) {
		text := `{"status":"tool_use","tool_calls":[{"name":"web_search","input":{}}]}`
		uses, _ := svc.parseToolBridgeOutput(req, text)
		if len(uses) != 1 || !strings.HasPrefix(uses[0].ID, "toolu_") {
			t.Errorf("uses = %+v, want generated toolu_ id", uses)
		}
	})

	t.Run("text payload returns content", func(t *testing.T) {
		_, content := svc.parseToolBridgeOutput(req, `{"status":"text","content":"plain answer"}`)
		if content != "plain answer" {
			t.Errorf("content = %q, want %q", content, "plain answer")
		}
	})

	t.Run("non-JSON output returned raw", func(t *testing.T) {
		_, content := svc.parseToolBridgeOutput(req, "not json at all")
		if content != "not json at all" {
			t.Errorf("content = %q, want raw text", content)
		}
	})
}
