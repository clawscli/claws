package ai

import (
	"context"
	"encoding/json"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/document"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime/types"

	appaws "github.com/clawscli/claws/internal/aws"
	appconfig "github.com/clawscli/claws/internal/config"
	apperrors "github.com/clawscli/claws/internal/errors"
	"github.com/clawscli/claws/internal/log"
)

// Role represents the role of a message participant.
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
)

// StopReason indicates why the model stopped generating.
type StopReason string

const (
	StopReasonEndTurn   StopReason = "end_turn"
	StopReasonToolUse   StopReason = "tool_use"
	StopReasonMaxTokens StopReason = "max_tokens"
)

// Message represents a single message in a conversation.
// Each message contains one or more ContentBlocks.
type Message struct {
	Role    Role           `json:"role"`
	Content []ContentBlock `json:"content"`
}

// ContentBlock represents a content element within a message.
// Only one field should be set at a time.
type ContentBlock struct {
	// Text content
	Text string `json:"text,omitempty"`

	// Tool use request from LLM
	ToolUse *ToolUseContent `json:"toolUse,omitempty"`

	// Tool result from application
	ToolResult *ToolResultContent `json:"toolResult,omitempty"`

	// Extended Thinking (Reasoning content from Bedrock API)
	Reasoning          string `json:"reasoning,omitempty"`
	ReasoningSignature string `json:"reasoningSignature,omitempty"`
}

// ToolUseContent represents a tool invocation request from the LLM.
type ToolUseContent struct {
	ID         string         `json:"toolUseId"`
	Name       string         `json:"name"`
	Input      map[string]any `json:"input"`
	InputError string         `json:"-"`
}

// ToolResultContent represents the result of a tool execution.
type ToolResultContent struct {
	ToolUseID string `json:"toolUseId"`
	Content   string `json:"content"`
	IsError   bool   `json:"isError,omitempty"`
}

// StreamEvent represents an event from streaming response.
type StreamEvent struct {
	Type       string
	Text       string
	Thinking   *ThinkingContent
	ToolUse    *ToolUseContent
	StopReason StopReason
	Error      error
}

// ThinkingContent represents thinking/reasoning content.
type ThinkingContent struct {
	Text      string
	Signature string
}

// Tool represents a tool definition for the LLM.
type Tool struct {
	Name        string
	Description string
	InputSchema map[string]any
}

// Client wraps the Bedrock runtime client.
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
	// Use AI-specific profile/region if configured
	fileCfg := appconfig.File()
	if profile := fileCfg.GetAIProfile(); profile != "" {
		ctx = appaws.WithSelectionOverride(ctx, appconfig.ProfileSelectionFromID(profile))
	}
	if region := fileCfg.GetAIRegion(); region != "" {
		ctx = appaws.WithRegionOverride(ctx, region)
	}

	awsCfg, err := appaws.NewConfig(ctx)
	if err != nil {
		return nil, apperrors.Wrap(err, "load aws config")
	}

	c := &Client{
		client: bedrockruntime.NewFromConfig(awsCfg),
	}

	for _, opt := range opts {
		opt(c)
	}

	return c, nil
}

// ConverseStream sends a streaming request and returns a channel of events.
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

func (c *Client) buildConverseStreamInput(messages []Message, systemPrompt string) *bedrockruntime.ConverseStreamInput {
	log.Debug("buildConverseStreamInput", "modelID", c.modelID, "maxTokens", c.maxTokens, "thinkingBudget", c.thinkingBudget)
	input := &bedrockruntime.ConverseStreamInput{
		ModelId:  aws.String(c.modelID),
		Messages: convertMessages(messages),
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
	}

	return input
}

// convertMessages converts our Message type to Bedrock API types.
func convertMessages(messages []Message) []types.Message {
	result := make([]types.Message, len(messages))
	for i, msg := range messages {
		result[i] = types.Message{
			Role:    types.ConversationRole(msg.Role),
			Content: convertContentBlocks(msg.Content),
		}
	}
	return result
}

// convertContentBlocks converts our ContentBlock to Bedrock API types.
// Based on dt's implementation.
func convertContentBlocks(blocks []ContentBlock) []types.ContentBlock {
	result := make([]types.ContentBlock, 0, len(blocks))
	for _, block := range blocks {
		if block.Text != "" {
			result = append(result, &types.ContentBlockMemberText{Value: block.Text})
		}
		if block.ToolUse != nil {
			result = append(result, &types.ContentBlockMemberToolUse{
				Value: types.ToolUseBlock{
					ToolUseId: aws.String(block.ToolUse.ID),
					Name:      aws.String(block.ToolUse.Name),
					Input:     document.NewLazyDocument(block.ToolUse.Input),
				},
			})
		}
		if block.ToolResult != nil {
			status := types.ToolResultStatusSuccess
			if block.ToolResult.IsError {
				status = types.ToolResultStatusError
			}
			result = append(result, &types.ContentBlockMemberToolResult{
				Value: types.ToolResultBlock{
					ToolUseId: aws.String(block.ToolResult.ToolUseID),
					Status:    status,
					Content: []types.ToolResultContentBlock{
						&types.ToolResultContentBlockMemberText{Value: block.ToolResult.Content},
					},
				},
			})
		}
		if block.Reasoning != "" {
			reasoningBlock := types.ReasoningTextBlock{
				Text: aws.String(block.Reasoning),
			}
			if block.ReasoningSignature != "" {
				reasoningBlock.Signature = aws.String(block.ReasoningSignature)
			}
			result = append(result, &types.ContentBlockMemberReasoningContent{
				Value: &types.ReasoningContentBlockMemberReasoningText{
					Value: reasoningBlock,
				},
			})
		}
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

// processStream processes the streaming response from Bedrock.
// Based on dt's implementation.
func (c *Client) processStream(ctx context.Context, output *bedrockruntime.ConverseStreamOutput, events chan<- StreamEvent) {
	defer close(events)

	stream := output.GetStream()
	defer func() {
		if err := stream.Close(); err != nil {
			log.Debug("stream close error", "error", err)
		}
	}()

	// Track current content block state
	var currentToolUse *ToolUseContent
	var toolInputBuffer string

	var thinkingText string
	var thinkingSignature string
	var isThinkingBlock bool

	for event := range stream.Events() {
		select {
		case <-ctx.Done():
			events <- StreamEvent{Type: "error", Error: ctx.Err()}
			return
		default:
		}

		switch e := event.(type) {
		case *types.ConverseStreamOutputMemberContentBlockStart:
			// Start of a new content block
			start := e.Value.Start
			switch s := start.(type) {
			case *types.ContentBlockStartMemberToolUse:
				// Initialize tool use tracking
				currentToolUse = &ToolUseContent{
					ID:   aws.ToString(s.Value.ToolUseId),
					Name: aws.ToString(s.Value.Name),
				}
				toolInputBuffer = ""
			}

		case *types.ConverseStreamOutputMemberContentBlockDelta:
			switch delta := e.Value.Delta.(type) {
			case *types.ContentBlockDeltaMemberText:
				events <- StreamEvent{Type: "text", Text: delta.Value}
			case *types.ContentBlockDeltaMemberReasoningContent:
				// Mark that we're processing a thinking block
				isThinkingBlock = true

				// Process reasoning delta types
				switch reasoningDelta := delta.Value.(type) {
				case *types.ReasoningContentBlockDeltaMemberText:
					thinkingText += reasoningDelta.Value
					// Stream text chunks as they arrive
					events <- StreamEvent{
						Type:     "thinking",
						Thinking: &ThinkingContent{Text: reasoningDelta.Value},
					}
				case *types.ReasoningContentBlockDeltaMemberSignature:
					thinkingSignature = reasoningDelta.Value
				case *types.ReasoningContentBlockDeltaMemberRedactedContent:
					// Redacted content - ignore
				}
			case *types.ContentBlockDeltaMemberToolUse:
				// Accumulate tool use input
				if currentToolUse != nil {
					toolInputBuffer += aws.ToString(delta.Value.Input)
				}
			}

		case *types.ConverseStreamOutputMemberContentBlockStop:
			if currentToolUse != nil {
				var input map[string]any
				if err := json.Unmarshal([]byte(toolInputBuffer), &input); err != nil {
					log.Debug("failed to parse tool input JSON", "error", err)
					input = make(map[string]any)
					currentToolUse.InputError = err.Error()
				}
				currentToolUse.Input = input

				events <- StreamEvent{
					Type:    "tool_use",
					ToolUse: currentToolUse,
				}

				currentToolUse = nil
				toolInputBuffer = ""
			}

			// If we were processing a thinking block, send the complete version with signature
			if isThinkingBlock {
				// Send complete thinking event with both text and signature
				events <- StreamEvent{
					Type: "thinking_complete",
					Thinking: &ThinkingContent{
						Text:      thinkingText,
						Signature: thinkingSignature,
					},
				}

				// Reset thinking state
				thinkingText = ""
				thinkingSignature = ""
				isThinkingBlock = false
			}

		case *types.ConverseStreamOutputMemberMessageStop:
			events <- StreamEvent{
				Type:       "done",
				StopReason: convertStopReason(e.Value.StopReason),
			}
			return
		}
	}

	if err := stream.Err(); err != nil {
		events <- StreamEvent{Type: "error", Error: err}
	}
}

func convertStopReason(reason types.StopReason) StopReason {
	switch reason {
	case types.StopReasonEndTurn:
		return StopReasonEndTurn
	case types.StopReasonToolUse:
		return StopReasonToolUse
	case types.StopReasonMaxTokens:
		return StopReasonMaxTokens
	default:
		return StopReasonEndTurn
	}
}

// Helper functions for building messages

// NewUserMessage creates a user message with text content.
func NewUserMessage(text string) Message {
	return Message{
		Role:    RoleUser,
		Content: []ContentBlock{{Text: text}},
	}
}

// NewAssistantMessage creates an assistant message with content blocks.
func NewAssistantMessage(blocks ...ContentBlock) Message {
	return Message{
		Role:    RoleAssistant,
		Content: blocks,
	}
}

// NewToolResultMessage creates a user message with tool results.
func NewToolResultMessage(results ...ToolResultContent) Message {
	blocks := make([]ContentBlock, len(results))
	for i, r := range results {
		blocks[i] = ContentBlock{ToolResult: &r}
	}
	return Message{
		Role:    RoleUser,
		Content: blocks,
	}
}
