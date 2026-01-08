package ai

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	appaws "github.com/clawscli/claws/internal/aws"
	"github.com/clawscli/claws/internal/log"
)

const DefaultModel = "global.anthropic.claude-haiku-4-5-20251001-v1:0"

type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

type ToolCall struct {
	ID    string         `json:"id"`
	Name  string         `json:"name"`
	Input map[string]any `json:"input"`
}

type ToolResult struct {
	ID      string `json:"id"`
	Content string `json:"content"`
	IsError bool   `json:"is_error,omitempty"`
}

type StreamEvent struct {
	Type      string
	Text      string
	ToolCalls []ToolCall
	Error     error
}

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
}

type Client struct {
	client  *bedrockruntime.Client
	modelID string
	tools   []Tool
}

type ClientOption func(*Client)

func WithModel(modelID string) ClientOption {
	return func(c *Client) {
		c.modelID = modelID
	}
}

func WithTools(tools []Tool) ClientOption {
	return func(c *Client) {
		c.tools = tools
	}
}

func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, fmt.Errorf("load aws config: %w", err)
	}

	c := &Client{
		client:  bedrockruntime.NewFromConfig(cfg),
		modelID: DefaultModel,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

func (c *Client) Converse(ctx context.Context, messages []Message, systemPrompt string) (*ConversationResult, error) {
	input := c.buildConverseInput(messages, systemPrompt)

	output, err := c.client.Converse(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("converse: %w", err)
	}

	return c.parseConverseOutput(output)
}

func (c *Client) ConverseStream(ctx context.Context, messages []Message, systemPrompt string) (<-chan StreamEvent, error) {
	input := c.buildConverseStreamInput(messages, systemPrompt)

	output, err := c.client.ConverseStream(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("converse stream: %w", err)
	}

	events := make(chan StreamEvent, 10)
	go c.processStream(output, events)

	return events, nil
}

type ConversationResult struct {
	Message    string
	ToolCalls  []ToolCall
	StopReason string
}

func (c *Client) buildConverseInput(messages []Message, systemPrompt string) *bedrockruntime.ConverseInput {
	input := &bedrockruntime.ConverseInput{
		ModelId:  aws.String(c.modelID),
		Messages: c.convertMessages(messages),
	}

	if systemPrompt != "" {
		input.System = []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{Value: systemPrompt},
		}
	}

	if len(c.tools) > 0 {
		input.ToolConfig = c.buildToolConfig()
	}

	return input
}

func (c *Client) buildConverseStreamInput(messages []Message, systemPrompt string) *bedrockruntime.ConverseStreamInput {
	input := &bedrockruntime.ConverseStreamInput{
		ModelId:  aws.String(c.modelID),
		Messages: c.convertMessages(messages),
	}

	if systemPrompt != "" {
		input.System = []types.SystemContentBlock{
			&types.SystemContentBlockMemberText{Value: systemPrompt},
		}
	}

	if len(c.tools) > 0 {
		input.ToolConfig = c.buildToolConfig()
	}

	return input
}

func (c *Client) convertMessages(messages []Message) []types.Message {
	result := make([]types.Message, 0, len(messages))

	for _, msg := range messages {
		result = append(result, types.Message{
			Role: types.ConversationRole(msg.Role),
			Content: []types.ContentBlock{
				&types.ContentBlockMemberText{Value: msg.Content},
			},
		})
	}

	return result
}

func (c *Client) buildToolConfig() *types.ToolConfiguration {
	toolDefs := make([]types.Tool, 0, len(c.tools))

	for _, t := range c.tools {
		toolDefs = append(toolDefs, &types.ToolMemberToolSpec{
			Value: types.ToolSpecification{
				Name:        aws.String(t.Name),
				Description: aws.String(t.Description),
				InputSchema: &types.ToolInputSchemaMemberJson{
					Value: document.NewLazyDocument(t.InputSchema),
				},
			},
		})
	}

	return &types.ToolConfiguration{
		Tools: toolDefs,
	}
}

func (c *Client) parseConverseOutput(output *bedrockruntime.ConverseOutput) (*ConversationResult, error) {
	result := &ConversationResult{}

	if output.StopReason != "" {
		result.StopReason = string(output.StopReason)
	}

	if output.Output == nil {
		return result, nil
	}

	msg, ok := output.Output.(*types.ConverseOutputMemberMessage)
	if !ok {
		return result, nil
	}

	for _, block := range msg.Value.Content {
		switch b := block.(type) {
		case *types.ContentBlockMemberText:
			result.Message += b.Value
		case *types.ContentBlockMemberToolUse:
			var inputMap map[string]any
			if b.Value.Input != nil {
				if err := b.Value.Input.UnmarshalSmithyDocument(&inputMap); err != nil {
					log.Debug("UnmarshalSmithyDocument failed, trying json.Marshal", "error", err)
					inputBytes, _ := json.Marshal(b.Value.Input)
					log.Debug("json.Marshal result", "bytes", string(inputBytes))
					_ = json.Unmarshal(inputBytes, &inputMap)
				}
			}
			log.Debug("tool called", "name", aws.ToString(b.Value.Name), "input", inputMap)

			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:    aws.ToString(b.Value.ToolUseId),
				Name:  aws.ToString(b.Value.Name),
				Input: inputMap,
			})
		}
	}

	return result, nil
}

func (c *Client) processStream(output *bedrockruntime.ConverseStreamOutput, events chan<- StreamEvent) {
	defer close(events)

	stream := output.GetStream()
	defer func() { _ = stream.Close() }()

	var currentToolCall *ToolCall
	var toolInputJSON string

	for event := range stream.Events() {
		switch e := event.(type) {
		case *types.ConverseStreamOutputMemberContentBlockStart:
			if toolUse, ok := e.Value.Start.(*types.ContentBlockStartMemberToolUse); ok {
				currentToolCall = &ToolCall{
					ID:   aws.ToString(toolUse.Value.ToolUseId),
					Name: aws.ToString(toolUse.Value.Name),
				}
				toolInputJSON = ""
			}

		case *types.ConverseStreamOutputMemberContentBlockDelta:
			switch delta := e.Value.Delta.(type) {
			case *types.ContentBlockDeltaMemberText:
				events <- StreamEvent{Type: "text", Text: delta.Value}
			case *types.ContentBlockDeltaMemberToolUse:
				if delta.Value.Input != nil {
					toolInputJSON += *delta.Value.Input
				}
			}

		case *types.ConverseStreamOutputMemberContentBlockStop:
			if currentToolCall != nil {
				var inputMap map[string]any
				_ = json.Unmarshal([]byte(toolInputJSON), &inputMap)
				currentToolCall.Input = inputMap

				events <- StreamEvent{
					Type:      "tool_use",
					ToolCalls: []ToolCall{*currentToolCall},
				}
				currentToolCall = nil
			}

		case *types.ConverseStreamOutputMemberMessageStop:
			events <- StreamEvent{Type: "done"}
			return
		}
	}

	if err := stream.Err(); err != nil {
		events <- StreamEvent{Type: "error", Error: err}
	}
}

func (c *Client) AddToolResultMessages(messages []Message, toolCalls []ToolCall, results []ToolResult) []Message {
	assistantContent := ""
	for _, tc := range toolCalls {
		inputJSON, _ := json.Marshal(tc.Input)
		assistantContent += fmt.Sprintf("[Tool: %s]\n%s\n", tc.Name, string(inputJSON))
	}

	messages = append(messages, Message{
		Role:    RoleAssistant,
		Content: assistantContent,
	})

	var resultContent string
	for _, r := range results {
		if r.IsError {
			resultContent += fmt.Sprintf("[Tool Result - Error]\n%s\n", r.Content)
		} else {
			resultContent += fmt.Sprintf("[Tool Result]\n%s\n", r.Content)
		}
	}

	messages = append(messages, Message{
		Role:    RoleUser,
		Content: resultContent,
	})

	return messages
}
