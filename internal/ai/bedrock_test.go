package ai

import (
	"testing"
)

func TestNewUserMessage(t *testing.T) {
	msg := NewUserMessage("hello world")

	if msg.Role != RoleUser {
		t.Errorf("expected role %q, got %q", RoleUser, msg.Role)
	}
	if len(msg.Content) != 1 {
		t.Fatalf("expected 1 content block, got %d", len(msg.Content))
	}
	if msg.Content[0].Text != "hello world" {
		t.Errorf("expected text %q, got %q", "hello world", msg.Content[0].Text)
	}
}

func TestNewAssistantMessage(t *testing.T) {
	blocks := []ContentBlock{
		{Text: "response text"},
		{ToolUse: &ToolUseContent{ID: "123", Name: "test_tool", Input: map[string]any{"key": "value"}}},
	}
	msg := NewAssistantMessage(blocks...)

	if msg.Role != RoleAssistant {
		t.Errorf("expected role %q, got %q", RoleAssistant, msg.Role)
	}
	if len(msg.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(msg.Content))
	}
	if msg.Content[0].Text != "response text" {
		t.Errorf("expected text %q, got %q", "response text", msg.Content[0].Text)
	}
	if msg.Content[1].ToolUse == nil {
		t.Fatal("expected tool use block")
	}
	if msg.Content[1].ToolUse.Name != "test_tool" {
		t.Errorf("expected tool name %q, got %q", "test_tool", msg.Content[1].ToolUse.Name)
	}
}

func TestNewToolResultMessage(t *testing.T) {
	results := []ToolResultContent{
		{ToolUseID: "123", Content: "success result", IsError: false},
		{ToolUseID: "456", Content: "error message", IsError: true},
	}
	msg := NewToolResultMessage(results...)

	if msg.Role != RoleUser {
		t.Errorf("expected role %q, got %q", RoleUser, msg.Role)
	}
	if len(msg.Content) != 2 {
		t.Fatalf("expected 2 content blocks, got %d", len(msg.Content))
	}
	if msg.Content[0].ToolResult == nil {
		t.Fatal("expected tool result block")
	}
	if msg.Content[0].ToolResult.ToolUseID != "123" {
		t.Errorf("expected tool use ID %q, got %q", "123", msg.Content[0].ToolResult.ToolUseID)
	}
	if msg.Content[0].ToolResult.IsError {
		t.Error("expected IsError to be false for first result")
	}
	if !msg.Content[1].ToolResult.IsError {
		t.Error("expected IsError to be true for second result")
	}
}

func TestConvertStopReason(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected StopReason
	}{
		{"end_turn", "end_turn", StopReasonEndTurn},
		{"tool_use", "tool_use", StopReasonToolUse},
		{"max_tokens", "max_tokens", StopReasonMaxTokens},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.expected) != tt.input {
				t.Errorf("StopReason constant mismatch: expected %q, got %q", tt.input, tt.expected)
			}
		})
	}
}

func TestContentBlockTypes(t *testing.T) {
	t.Run("text block", func(t *testing.T) {
		block := ContentBlock{Text: "hello"}
		if block.Text != "hello" {
			t.Errorf("expected text %q, got %q", "hello", block.Text)
		}
		if block.ToolUse != nil || block.ToolResult != nil {
			t.Error("other fields should be nil for text block")
		}
	})

	t.Run("tool use block", func(t *testing.T) {
		block := ContentBlock{
			ToolUse: &ToolUseContent{
				ID:    "tool-123",
				Name:  "query_resources",
				Input: map[string]any{"service": "ec2", "region": "us-east-1"},
			},
		}
		if block.ToolUse == nil {
			t.Fatal("expected tool use")
		}
		if block.ToolUse.ID != "tool-123" {
			t.Errorf("expected ID %q, got %q", "tool-123", block.ToolUse.ID)
		}
		if block.ToolUse.Input["service"] != "ec2" {
			t.Errorf("expected service %q, got %v", "ec2", block.ToolUse.Input["service"])
		}
	})

	t.Run("tool result block", func(t *testing.T) {
		block := ContentBlock{
			ToolResult: &ToolResultContent{
				ToolUseID: "tool-123",
				Content:   "Found 5 instances",
				IsError:   false,
			},
		}
		if block.ToolResult == nil {
			t.Fatal("expected tool result")
		}
		if block.ToolResult.Content != "Found 5 instances" {
			t.Errorf("expected content %q, got %q", "Found 5 instances", block.ToolResult.Content)
		}
	})

	t.Run("reasoning block", func(t *testing.T) {
		block := ContentBlock{
			Reasoning:          "Let me think about this...",
			ReasoningSignature: "sig123",
		}
		if block.Reasoning != "Let me think about this..." {
			t.Errorf("expected reasoning %q, got %q", "Let me think about this...", block.Reasoning)
		}
		if block.ReasoningSignature != "sig123" {
			t.Errorf("expected signature %q, got %q", "sig123", block.ReasoningSignature)
		}
	})
}

func TestToolDefinition(t *testing.T) {
	tool := Tool{
		Name:        "test_tool",
		Description: "A test tool",
		InputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"param1": map[string]any{
					"type":        "string",
					"description": "First parameter",
				},
			},
			"required": []string{"param1"},
		},
	}

	if tool.Name != "test_tool" {
		t.Errorf("expected name %q, got %q", "test_tool", tool.Name)
	}
	if tool.Description != "A test tool" {
		t.Errorf("expected description %q, got %q", "A test tool", tool.Description)
	}

	props, ok := tool.InputSchema["properties"].(map[string]any)
	if !ok {
		t.Fatal("expected properties to be map")
	}
	if _, ok := props["param1"]; !ok {
		t.Error("expected param1 in properties")
	}
}

func TestClientOptions(t *testing.T) {
	t.Run("WithModel", func(t *testing.T) {
		c := &Client{}
		opt := WithModel("test-model")
		opt(c)
		if c.modelID != "test-model" {
			t.Errorf("expected model %q, got %q", "test-model", c.modelID)
		}
	})

	t.Run("WithMaxTokens", func(t *testing.T) {
		c := &Client{}
		opt := WithMaxTokens(1000)
		opt(c)
		if c.maxTokens != 1000 {
			t.Errorf("expected maxTokens %d, got %d", 1000, c.maxTokens)
		}
	})

	t.Run("WithThinkingBudget", func(t *testing.T) {
		c := &Client{}
		opt := WithThinkingBudget(5000)
		opt(c)
		if c.thinkingBudget != 5000 {
			t.Errorf("expected thinkingBudget %d, got %d", 5000, c.thinkingBudget)
		}
	})

	t.Run("WithTools", func(t *testing.T) {
		c := &Client{}
		tools := []Tool{
			{Name: "tool1"},
			{Name: "tool2"},
		}
		opt := WithTools(tools)
		opt(c)
		if len(c.tools) != 2 {
			t.Errorf("expected 2 tools, got %d", len(c.tools))
		}
	})
}

func TestStreamEvent(t *testing.T) {
	t.Run("text event", func(t *testing.T) {
		event := StreamEvent{Type: "text", Text: "hello"}
		if event.Type != "text" {
			t.Errorf("expected type %q, got %q", "text", event.Type)
		}
		if event.Text != "hello" {
			t.Errorf("expected text %q, got %q", "hello", event.Text)
		}
	})

	t.Run("thinking event", func(t *testing.T) {
		event := StreamEvent{
			Type:     "thinking",
			Thinking: &ThinkingContent{Text: "reasoning..."},
		}
		if event.Thinking == nil {
			t.Fatal("expected thinking content")
		}
		if event.Thinking.Text != "reasoning..." {
			t.Errorf("expected thinking text %q, got %q", "reasoning...", event.Thinking.Text)
		}
	})

	t.Run("tool_use event", func(t *testing.T) {
		event := StreamEvent{
			Type:    "tool_use",
			ToolUse: &ToolUseContent{ID: "123", Name: "test"},
		}
		if event.ToolUse == nil {
			t.Fatal("expected tool use")
		}
		if event.ToolUse.Name != "test" {
			t.Errorf("expected tool name %q, got %q", "test", event.ToolUse.Name)
		}
	})

	t.Run("done event", func(t *testing.T) {
		event := StreamEvent{Type: "done", StopReason: StopReasonEndTurn}
		if event.StopReason != StopReasonEndTurn {
			t.Errorf("expected stop reason %q, got %q", StopReasonEndTurn, event.StopReason)
		}
	})

	t.Run("error event", func(t *testing.T) {
		event := StreamEvent{Type: "error", Error: &testError{"test error"}}
		if event.Error == nil {
			t.Fatal("expected error")
		}
		if event.Error.Error() != "test error" {
			t.Errorf("expected error %q, got %q", "test error", event.Error.Error())
		}
	})
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
