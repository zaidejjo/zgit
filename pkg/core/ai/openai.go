package ai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// OpenAI implements Generator for OpenAI-compatible APIs (OpenAI, DeepSeek, OpenRouter, custom).
type OpenAI struct {
	cfg Config
}

// NewOpenAI creates a new OpenAI-compatible provider.
func NewOpenAI(cfg Config) *OpenAI {
	return &OpenAI{cfg: cfg}
}

// chatRequest is the request body for OpenAI-compatible chat completions.
type chatRequest struct {
	Model       string        `json:"model"`
	Messages    []chatMessage `json:"messages"`
	MaxTokens   int           `json:"max_tokens"`
	Temperature float64       `json:"temperature"`
	Stream      bool          `json:"stream,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// chatResponse is the response body from OpenAI-compatible APIs.
type chatResponse struct {
	Choices []struct {
		Message struct {
			Content string `json:"content"`
		} `json:"message"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
	} `json:"error,omitempty"`
}

func (o *OpenAI) GenerateCommitMessage(ctx context.Context, diff string, cfg Config) (string, error) {
	return o.GenerateCommitMessageWithContext(ctx, diff, CommitContext{}, cfg)
}

func (o *OpenAI) GenerateCommitMessageWithContext(ctx context.Context, diff string, context CommitContext, cfg Config) (string, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoints[cfg.Provider]
	}

	model := cfg.Model
	if model == "" {
		model = DefaultModels[cfg.Provider]
	}

	// Truncate diff to avoid token limits (max ~6000 chars for most models)
	truncatedDiff := diff
	if len(truncatedDiff) > 6000 {
		truncatedDiff = truncatedDiff[:6000] + "\n# ... (diff truncated)"
	}

	// Ensure diff has content
	if strings.TrimSpace(truncatedDiff) == "" {
		// Check if this is an empty repo (no commits at all)
		return "", fmt.Errorf("staged diff is empty — stage some changes first (this repo has no commits yet)")
	}

	// Build context-aware system prompt
	systemPrompt := buildSystemPrompt()
	if context.Branch != "" || len(context.RecentMessages) > 0 {
		var contextLines []string
		if context.Branch != "" {
			contextLines = append(contextLines, fmt.Sprintf("Current branch: %s", context.Branch))
		}
		if context.StatusSummary != "" {
			contextLines = append(contextLines, fmt.Sprintf("Repo status: %s", context.StatusSummary))
		}
		if context.ChangedFiles > 0 {
			contextLines = append(contextLines, fmt.Sprintf("Changed files: %d", context.ChangedFiles))
		}
		if len(context.RecentMessages) > 0 {
			contextLines = append(contextLines, "Recent commit messages (match this style):")
			for _, msg := range context.RecentMessages {
				contextLines = append(contextLines, fmt.Sprintf("  - %s", msg))
			}
		}
		contextBlock := strings.Join(contextLines, "\n")
		systemPrompt = fmt.Sprintf("%s\n\nRepository Context:\n%s", systemPrompt, contextBlock)
	}

	reqBody := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Generate a commit message for this diff:\n\n```\n%s\n```", truncatedDiff)},
		},
		MaxTokens:   300,
		Temperature: 0.3,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	// OpenRouter specific headers
	if cfg.Provider == ProviderOpenRouter {
		httpReq.Header.Set("HTTP-Referer", "https://zgit.app")
		httpReq.Header.Set("X-Title", "zgit")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	respData, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respData))
	}

	var chatResp chatResponse
	if err := json.Unmarshal(respData, &chatResp); err != nil {
		return "", fmt.Errorf("parse response: %w", err)
	}

	if chatResp.Error != nil {
		return "", fmt.Errorf("API error: %s", chatResp.Error.Message)
	}

	if len(chatResp.Choices) == 0 {
		return "", fmt.Errorf("API returned no choices")
	}

	msg := strings.TrimSpace(chatResp.Choices[0].Message.Content)
	return msg, nil
}

// GenerateCommitMessageStream streams tokens via onToken callback for OpenAI-compatible APIs.
func (o *OpenAI) GenerateCommitMessageStream(ctx context.Context, diff string, context CommitContext, cfg Config, onToken func(string)) (string, error) {
	endpoint := cfg.Endpoint
	if endpoint == "" {
		endpoint = DefaultEndpoints[cfg.Provider]
	}

	model := cfg.Model
	if model == "" {
		model = DefaultModels[cfg.Provider]
	}

	// Truncate diff
	truncatedDiff := diff
	if len(truncatedDiff) > 6000 {
		truncatedDiff = truncatedDiff[:6000] + "\n# ... (diff truncated)"
	}

	if strings.TrimSpace(truncatedDiff) == "" {
		return "", fmt.Errorf("staged diff is empty — stage some changes first (this repo has no commits yet)")
	}

	// Build context-aware system prompt
	systemPrompt := buildSystemPrompt()
	if context.Branch != "" || len(context.RecentMessages) > 0 {
		var contextLines []string
		if context.Branch != "" {
			contextLines = append(contextLines, fmt.Sprintf("Current branch: %s", context.Branch))
		}
		if context.StatusSummary != "" {
			contextLines = append(contextLines, fmt.Sprintf("Repo status: %s", context.StatusSummary))
		}
		if context.ChangedFiles > 0 {
			contextLines = append(contextLines, fmt.Sprintf("Changed files: %d", context.ChangedFiles))
		}
		if len(context.RecentMessages) > 0 {
			contextLines = append(contextLines, "Recent commit messages (match this style):")
			for _, msg := range context.RecentMessages {
				contextLines = append(contextLines, fmt.Sprintf("  - %s", msg))
			}
		}
		contextBlock := strings.Join(contextLines, "\n")
		systemPrompt = fmt.Sprintf("%s\n\nRepository Context:\n%s", systemPrompt, contextBlock)
	}

	reqBody := chatRequest{
		Model: model,
		Messages: []chatMessage{
			{Role: "system", Content: systemPrompt},
			{Role: "user", Content: fmt.Sprintf("Generate a commit message for this diff:\n\n```\n%s\n```", truncatedDiff)},
		},
		MaxTokens:   300,
		Temperature: 0.3,
		Stream:      true,
	}

	body, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(body))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+cfg.APIKey)

	if cfg.Provider == ProviderOpenRouter {
		httpReq.Header.Set("HTTP-Referer", "https://zgit.app")
		httpReq.Header.Set("X-Title", "zgit")
	}

	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return "", fmt.Errorf("API request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		respData, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(respData))
	}

	var fullContent string
	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}
		var chunk openAIStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			continue
		}
		if len(chunk.Choices) == 0 {
			continue
		}
		content := chunk.Choices[0].Delta.Content
		if content != "" {
			fullContent += content
			if onToken != nil {
				onToken(content)
			}
		}
		if chunk.Choices[0].FinishReason != "" {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return "", fmt.Errorf("stream read error: %w", err)
	}

	return strings.TrimSpace(fullContent), nil
}
