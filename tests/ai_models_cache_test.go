package integration

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/zaidejjo/zgit/pkg/core/ai"
)

// openTestStore is defined in db_store_test.go in the same package.

func TestStoreCacheEntryTTL(t *testing.T) {
	s := openTestStore(t)

	scope := "ai_models:openai"
	key := "models"
	ttl := int64(86400) // 24h

	// Miss on empty store
	data, ok := s.GetCacheEntry(scope, key, ttl)
	if ok {
		t.Errorf("expected cache miss on empty store, got ok=true")
	}
	if data != nil {
		t.Errorf("expected nil data on miss, got %v", data)
	}

	// Set and hit
	models := []string{"gpt-4o-mini", "gpt-4o", "claude-3-haiku"}
	data, err := json.Marshal(models)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.SetCacheEntry(scope, key, data); err != nil {
		t.Fatal(err)
	}

	data, ok = s.GetCacheEntry(scope, key, ttl)
	if !ok {
		t.Fatal("expected cache hit after set")
	}
	var got []string
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatal(err)
	}
	if len(got) != len(models) {
		t.Errorf("got %d models, want %d", len(got), len(models))
	}
	for i := range models {
		if got[i] != models[i] {
			t.Errorf("got[%d] = %q, want %q", i, got[i], models[i])
		}
	}

	// Expired entry: TTL of 0 means immediately expired? No, ttlSeconds > 0 check.
	// Set with very short TTL, sleep, then check expiry.
	shortTTL := int64(1) // 1 second
	if err := s.SetCacheEntry(scope, key+"-short", data); err != nil {
		t.Fatal(err)
	}
	// Fresh hit
	if _, ok := s.GetCacheEntry(scope, key+"-short", shortTTL); !ok {
		t.Error("expected fresh cache hit for short TTL")
	}
	// Sleep past TTL — fetched_at uses Unix seconds, so sleep enough to
	// cross a second boundary.
	time.Sleep(2500 * time.Millisecond)
	if _, ok := s.GetCacheEntry(scope, key+"-short", shortTTL); ok {
		t.Error("expected cache miss after TTL expiry")
	}
}

func TestFetchProviderModelsHTTP(t *testing.T) {
	// Mock OpenAI /models endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/models" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		resp := map[string]interface{}{
			"data": []map[string]interface{}{
				{"id": "gpt-4o-mini"},
				{"id": "gpt-4o"},
				{"id": "gpt-3.5-turbo"},
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	// Unsupported provider returns nil, nil (by design — no error)
	models, err := ai.FetchProviderModels("nonexistent-provider", "dummy-key")
	if err != nil {
		t.Errorf("expected nil error for unsupported provider, got %v", err)
	}
	if models != nil {
		t.Errorf("expected nil models for unsupported provider, got %v", models)
	}

	// Test Ollama (no API key needed) — will fail since no server, but
	// should return an error, not a panic.
	_, err = ai.FetchProviderModels("ollama", "")
	if err == nil {
		t.Error("expected error for Ollama with no server")
	}
}
