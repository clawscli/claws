package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	appaws "github.com/clawscli/claws/internal/aws"
	apperrors "github.com/clawscli/claws/internal/errors"
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
	Thinking  string
	ToolCalls []ToolCall
	Error     error
}

type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
}

type Client struct {
	client         *bedrockruntime.Client
	modelID        string
	tools          []Tool
	maxTokens      int32
	thinkingBudget int
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

func WithMaxTokens(maxTokens int) ClientOption {
	return func(c *Client) {
		c.maxTokens = int32(maxTokens)
	}
}

func WithThinkingBudget(budget int) ClientOption {
	return func(c *Client) {
		c.thinkingBudget = budget
	}
}

func NewClient(ctx context.Context, opts ...ClientOption) (*Client, error) {
	cfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "load aws config")
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
		return nil, apperrors.Wrap(err, "converse")
	}

	return c.parseConverseOutput(output)
}

func (c *Client) ConverseStream(ctx context.Context, messages []Message, systemPrompt string) (<-chan StreamEvent, error) {
	input := c.buildConverseStreamInput(messages, systemPrompt)

	output, err := c.client.ConverseStream(ctx, input)
	if err != nil {
		return nil, apperrors.Wrap(err, "converse stream")
	}

	events := make(chan StreamEvent, 10)
	go c.processStream(ctx, output, events)

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

	if c.maxTokens > 0 {
		input.InferenceConfig = &types.InferenceConfiguration{
			MaxTokens: aws.Int32(c.maxTokens),
		}
	}

	if c.thinkingBudget > 0 && strings.Contains(c.modelID, "anthropic.claude") {
		thinkingConfig := map[string]any{
			"thinking": map[string]any{
				"type":          "enabled",
				"budget_tokens": c.thinkingBudget,
			},
			"anthropic_beta": []string{"interleaved-thinking-2025-05-14"},
		}
		input.AdditionalModelRequestFields = document.NewLazyDocument(thinkingConfig)
		if input.InferenceConfig == nil {
			input.InferenceConfig = &types.InferenceConfiguration{}
		}
		input.InferenceConfig.Temperature = aws.Float32(1.0)
	}

	return input
}

func (c *Client) buildConverseStreamInput(messages []Message, systemPrompt string) *bedrockruntime.ConverseStreamInput {
	log.Debug("buildConverseStreamInput", "modelID", c.modelID, "maxTokens", c.maxTokens, "thinkingBudget", c.thinkingBudget)
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

	if c.maxTokens > 0 {
		input.InferenceConfig = &types.InferenceConfiguration{
			MaxTokens: aws.Int32(c.maxTokens),
		}
	}

	if c.thinkingBudget > 0 && strings.Contains(c.modelID, "anthropic.claude") {
		log.Debug("applying thinking config", "budget", c.thinkingBudget)
		thinkingConfig := map[string]any{
			"thinking": map[string]any{
				"type":          "enabled",
				"budget_tokens": c.thinkingBudget,
			},
			"anthropic_beta": []string{"interleaved-thinking-2025-05-14"},
		}
		input.AdditionalModelRequestFields = document.NewLazyDocument(thinkingConfig)
		if input.InferenceConfig == nil {
			input.InferenceConfig = &types.InferenceConfiguration{}
		}
		input.InferenceConfig.Temperature = aws.Float32(1.0)
	} else {
		log.Debug("thinking not applied", "budget", c.thinkingBudget, "modelMatch", strings.Contains(c.modelID, "anthropic.claude"))
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
					inputBytes, err := json.Marshal(b.Value.Input)
					if err != nil {
						log.Debug("json.Marshal failed", "error", err)
					} else {
						if err := json.Unmarshal(inputBytes, &inputMap); err != nil {
							log.Debug("json.Unmarshal failed", "error", err)
						}
					}
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

func (c *Client) processStream(ctx context.Context, output *bedrockruntime.ConverseStreamOutput, events chan<- StreamEvent) {
	defer close(events)

	stream := output.GetStream()
	defer func() {
		if err := stream.Close(); err != nil {
			log.Debug("stream close error", "error", err)
		}
	}()

	var currentToolCall *ToolCall
	var toolInputJSON string

	for event := range stream.Events() {
		select {
		case <-ctx.Done():
			events <- StreamEvent{Type: "error", Error: ctx.Err()}
			return
		default:
		}

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
			log.Debug("content block delta", "deltaType", fmt.Sprintf("%T", e.Value.Delta))
			switch delta := e.Value.Delta.(type) {
			case *types.ContentBlockDeltaMemberText:
				events <- StreamEvent{Type: "text", Text: delta.Value}
			case *types.ContentBlockDeltaMemberToolUse:
				if delta.Value.Input != nil {
					toolInputJSON += *delta.Value.Input
				}
			case *types.ContentBlockDeltaMemberReasoningContent:
				log.Debug("reasoning content", "valueType", fmt.Sprintf("%T", delta.Value))
				switch reasoningDelta := delta.Value.(type) {
				case *types.ReasoningContentBlockDeltaMemberText:
					log.Debug("thinking text", "len", len(reasoningDelta.Value))
					events <- StreamEvent{Type: "thinking", Thinking: reasoningDelta.Value}
				case *types.ReasoningContentBlockDeltaMemberSignature:
					log.Debug("thinking signature received")
				case *types.ReasoningContentBlockDeltaMemberRedactedContent:
					log.Debug("thinking redacted")
				}
			}

		case *types.ConverseStreamOutputMemberContentBlockStop:
			if currentToolCall != nil {
				var inputMap map[string]any
				if err := json.Unmarshal([]byte(toolInputJSON), &inputMap); err != nil {
					log.Debug("failed to parse tool input JSON", "error", err)
				}
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
