package ai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// ToolCallingProvider abstracts LLM chat with tool/function calling.
type ToolCallingProvider interface {
	// Chat sends messages with optional tools and returns the assistant response.
	Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (Message, error)
}

// --- OpenAI / compatible (DeepSeek, OpenRouter, custom) ---

type openAIToolRequest struct {
	Model       string              `json:"model"`
	Messages    []openAIToolMessage `json:"messages"`
	MaxTokens   int                 `json:"max_tokens"`
	Temperature float64             `json:"temperature"`
	Tools       []openAIToolDef     `json:"tools,omitempty"`
}

type openAIToolMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAIToolDef struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type openAIToolCall struct {
	ID       string         `json:"id"`
	Type     string         `json:"type"`
	Function openAIFuncCall `json:"function"`
}

type openAIFuncCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openAIChatResponse struct {
	Choices []openAIChoice `json:"choices"`
	Error   *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

type openAIChoice struct {
	Message openAIResponseMessage `json:"message"`
}

type openAIResponseMessage struct {
	Role      string           `json:"role"`
	Content   string           `json:"content"`
	ToolCalls []openAIToolCall `json:"tool_calls,omitempty"`
}

func (o *OpenAI) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (Message, error) {
	endpoint := o.cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoints[ProviderOpenAI]
	}
	model := o.cfg.Model
	if model == "" {
		model = DefaultModels[ProviderOpenAI]
	}

	reqMsg := make([]openAIToolMessage, len(messages))
	for i, m := range messages {
		om := openAIToolMessage{Role: m.Role, Content: m.Content, ToolCallID: m.ToolCallID}
		if len(m.ToolCalls) > 0 {
			om.ToolCalls = make([]openAIToolCall, len(m.ToolCalls))
			for j, tc := range m.ToolCalls {
				om.ToolCalls[j] = openAIToolCall{
					ID:   tc.ID,
					Type: "function",
					Function: openAIFuncCall{
						Name:      tc.Name,
						Arguments: tc.Arguments,
					},
				}
			}
		}
		reqMsg[i] = om
	}

	toolDefs := make([]openAIToolDef, len(tools))
	for i, t := range tools {
		toolDefs[i] = openAIToolDef{
			Type: "function",
			Function: openAIFunction{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.Parameters,
			},
		}
	}

	reqBody := openAIToolRequest{
		Model:       model,
		Messages:    reqMsg,
		MaxTokens:   4096,
		Temperature: 0.3,
		Tools:       toolDefs,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return Message{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return Message{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+o.cfg.APIKey)

	if o.cfg.Provider == ProviderOpenRouter {
		httpReq.Header.Set("HTTP-Referer", "https://zgit.app")
		httpReq.Header.Set("X-Title", "zgit")
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return Message{}, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return Message{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return Message{}, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respData))
	}

	var chatResp openAIChatResponse
	if err := json.Unmarshal(respData, &chatResp); err != nil {
		return Message{}, fmt.Errorf("parse response: %w", err)
	}
	if chatResp.Error != nil {
		return Message{}, fmt.Errorf("API error: %s", chatResp.Error.Message)
	}
	if len(chatResp.Choices) == 0 {
		return Message{}, fmt.Errorf("API returned no choices")
	}

	choice := chatResp.Choices[0].Message
	msg := Message{Role: choice.Role, Content: choice.Content}
	if len(choice.ToolCalls) > 0 {
		msg.ToolCalls = make([]ToolCall, len(choice.ToolCalls))
		for i, tc := range choice.ToolCalls {
			msg.ToolCalls[i] = ToolCall{
				ID:        tc.ID,
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			}
		}
	}
	return msg, nil
}

// --- Anthropic Claude ---

type anthropicToolRequest struct {
	Model       string             `json:"model"`
	MaxTokens   int                `json:"max_tokens"`
	System      string             `json:"system,omitempty"`
	Messages    []anthropicToolMsg `json:"messages"`
	Temperature float64            `json:"temperature"`
	Tools       []anthropicToolDef `json:"tools,omitempty"`
}

type anthropicToolMsg struct {
	Role    string           `json:"role"`
	Content []anthropicBlock `json:"content"`
}

type anthropicBlock struct {
	Type      string          `json:"type"` // "text", "tool_use", "tool_result"
	Text      string          `json:"text,omitempty"`
	ID        string          `json:"id,omitempty"`          // tool_use id
	Name      string          `json:"name,omitempty"`        // tool_use name
	Input     json.RawMessage `json:"input,omitempty"`       // tool_use input
	ToolUseID string          `json:"tool_use_id,omitempty"` // tool_result
	Content   string          `json:"content,omitempty"`     // tool_result
	IsError   bool            `json:"is_error,omitempty"`    // tool_result
}

type anthropicToolDef struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicChatResponse struct {
	Content    []anthropicBlock `json:"content"`
	StopReason string           `json:"stop_reason"`
	Error      *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (a *Anthropic) Chat(ctx context.Context, messages []Message, tools []ToolDefinition) (Message, error) {
	endpoint := a.cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoints[ProviderAnthropic]
	}
	model := a.cfg.Model
	if model == "" {
		model = DefaultModels[ProviderAnthropic]
	}

	// Extract system message (Anthropic uses top-level system param)
	var systemText string
	var userMessages []Message
	for _, m := range messages {
		if m.Role == "system" {
			systemText = m.Content
		} else {
			userMessages = append(userMessages, m)
		}
	}

	reqMsgs := make([]anthropicToolMsg, len(userMessages))
	for i, m := range userMessages {
		blocks := []anthropicBlock{}
		if m.Content != "" {
			blocks = append(blocks, anthropicBlock{Type: "text", Text: m.Content})
		}
		if len(m.ToolCalls) > 0 {
			for _, tc := range m.ToolCalls {
				var input json.RawMessage
				if tc.Arguments != "" {
					input = json.RawMessage(tc.Arguments)
				}
				blocks = append(blocks, anthropicBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Name,
					Input: input,
				})
			}
		}
		if m.Role == "tool" {
			blocks = []anthropicBlock{{
				Type:      "tool_result",
				ToolUseID: m.ToolCallID,
				Content:   m.Content,
			}}
		}
		if len(blocks) == 0 {
			blocks = []anthropicBlock{{Type: "text", Text: ""}}
		}
		reqMsgs[i] = anthropicToolMsg{Role: m.Role, Content: blocks}
	}

	// Convert tool definitions to Anthropic format
	anthropicTools := make([]anthropicToolDef, len(tools))
	for i, t := range tools {
		anthropicTools[i] = anthropicToolDef{
			Name:        t.Name,
			Description: t.Description,
			InputSchema: t.Parameters,
		}
	}

	reqBody := anthropicToolRequest{
		Model:       model,
		MaxTokens:   4096,
		System:      systemText,
		Messages:    reqMsgs,
		Temperature: 0.3,
		Tools:       anthropicTools,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return Message{}, fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return Message{}, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", a.cfg.APIKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return Message{}, fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return Message{}, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return Message{}, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respData))
	}

	var anthropicResp anthropicChatResponse
	if err := json.Unmarshal(respData, &anthropicResp); err != nil {
		return Message{}, fmt.Errorf("parse response: %w", err)
	}
	if anthropicResp.Error != nil {
		return Message{}, fmt.Errorf("API error: %s", anthropicResp.Error.Message)
	}

	result := Message{Role: "assistant"}
	for _, block := range anthropicResp.Content {
		if block.Type == "text" {
			result.Content += block.Text
		} else if block.Type == "tool_use" {
			args := ""
			if block.Input != nil {
				args = string(block.Input)
			}
			if result.ToolCalls == nil {
				result.ToolCalls = []ToolCall{}
			}
			result.ToolCalls = append(result.ToolCalls, ToolCall{
				ID:        block.ID,
				Name:      block.Name,
				Arguments: args,
			})
		}
	}
	result.Content = strings.TrimSpace(result.Content)
	return result, nil
}

// Ensure both providers implement ToolCallingProvider.
var _ ToolCallingProvider = (*OpenAI)(nil)
var _ ToolCallingProvider = (*Anthropic)(nil)

// NewToolCallingProvider returns the correct provider for the given config.
func NewToolCallingProvider(cfg Config) (ToolCallingProvider, error) {
	switch cfg.Provider {
	case ProviderOpenAI, ProviderDeepSeek, ProviderOpenRouter, ProviderCustom:
		return NewOpenAI(cfg), nil
	case ProviderAnthropic:
		return NewAnthropic(cfg), nil
	default:
		return nil, fmt.Errorf("unsupported AI provider for tool calling: %s", cfg.Provider)
	}
}
