package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// fetchModelsTimeout is the HTTP timeout for model list requests.
const fetchModelsTimeout = 15 * time.Second

// FetchProviderModels queries the provider's /models API and returns available model IDs.
// Supported: openai, openrouter, groq, ollama. Returns nil for unsupported providers.
func FetchProviderModels(provider, apiKey string) ([]string, error) {
	switch ProviderKind(provider) {
	case ProviderOpenAI:
		return fetchOpenAIModels(apiKey)
	case ProviderOpenRouter:
		return fetchOpenRouterModels(apiKey)
	case ProviderGroq:
		return fetchGroqModels(apiKey)
	case ProviderOllama:
		return fetchOllamaModels()
	default:
		return nil, nil
	}
}

// --- OpenAI ---

type openAIModel struct {
	ID string `json:"id"`
}

type openAIResponse struct {
	Data []openAIModel `json:"data"`
}

func fetchOpenAIModels(apiKey string) ([]string, error) {
	return fetchModelsJSON("https://api.openai.com/v1/models", apiKey, "", func(body []byte) ([]string, error) {
		var resp openAIResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(resp.Data))
		for _, m := range resp.Data {
			ids = append(ids, m.ID)
		}
		return ids, nil
	})
}

// --- OpenRouter ---

type openRouterModel struct {
	ID string `json:"id"`
}

type openRouterResponse struct {
	Data []openRouterModel `json:"data"`
}

func fetchOpenRouterModels(apiKey string) ([]string, error) {
	return fetchModelsJSON("https://openrouter.ai/api/v1/models", apiKey, "", func(body []byte) ([]string, error) {
		var resp openRouterResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(resp.Data))
		for _, m := range resp.Data {
			ids = append(ids, m.ID)
		}
		return ids, nil
	})
}

// --- Groq ---

type groqModel struct {
	ID string `json:"id"`
}

type groqResponse struct {
	Data []groqModel `json:"data"`
}

func fetchGroqModels(apiKey string) ([]string, error) {
	return fetchModelsJSON("https://api.groq.com/openai/v1/models", apiKey, "", func(body []byte) ([]string, error) {
		var resp groqResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(resp.Data))
		for _, m := range resp.Data {
			ids = append(ids, m.ID)
		}
		return ids, nil
	})
}

// --- Ollama ---

type ollamaModel struct {
	Name string `json:"name"`
}

type ollamaResponse struct {
	Models []ollamaModel `json:"models"`
}

func fetchOllamaModels() ([]string, error) {
	return fetchModelsJSON("http://localhost:11434/api/tags", "", "", func(body []byte) ([]string, error) {
		var resp ollamaResponse
		if err := json.Unmarshal(body, &resp); err != nil {
			return nil, err
		}
		ids := make([]string, 0, len(resp.Models))
		for _, m := range resp.Models {
			ids = append(ids, m.Name)
		}
		return ids, nil
	})
}

// --- generic HTTP fetch + parse ---

func fetchModelsJSON(url, apiKey, bearerPrefix string, parse func([]byte) ([]string, error)) ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), fetchModelsTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	if apiKey != "" {
		prefix := bearerPrefix
		if prefix == "" {
			prefix = "Bearer "
		}
		req.Header.Set("Authorization", prefix+apiKey)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("http request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API returned status %d: %s", resp.StatusCode, string(body))
	}

	return parse(body)
}
