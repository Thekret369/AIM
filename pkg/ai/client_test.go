package ai

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"
	"time"
)

func TestOpenAICompatibleClientChatStream(t *testing.T) {
	var requestBody map[string]interface{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-key" {
			t.Fatalf("unexpected auth header %q", auth)
		}
		if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
			t.Fatalf("decode request body: %v", err)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n"))
		_, _ = w.Write([]byte("data: {\"choices\":[{\"delta\":{\"content\":\" world\"}}],\"usage\":{\"prompt_tokens\":3,\"completion_tokens\":2,\"total_tokens\":5}}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(time.Second)
	var chunks []string
	resp, err := client.ChatStream(context.Background(), ChatRequest{
		BaseURL: server.URL,
		APIKey:  "test-key",
		Model:   "stream-model",
		Messages: []ChatMessage{
			{Role: "user", Content: "hi"},
		},
	}, func(chunk ChatStreamChunk) error {
		chunks = append(chunks, chunk.Delta)
		return nil
	})
	if err != nil {
		t.Fatalf("chat stream: %v", err)
	}
	if resp.Content != "hello world" {
		t.Fatalf("unexpected content %q", resp.Content)
	}
	if resp.Usage.TotalTokens != 5 {
		t.Fatalf("unexpected usage %+v", resp.Usage)
	}
	if !reflect.DeepEqual(chunks, []string{"hello", " world"}) {
		t.Fatalf("unexpected chunks %+v", chunks)
	}
	if stream, ok := requestBody["stream"].(bool); !ok || !stream {
		t.Fatalf("expected stream=true, got %+v", requestBody["stream"])
	}
}

func TestOpenAICompatibleClientChatStreamProviderError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"error\":{\"message\":\"provider failed\"}}\n\n"))
	}))
	defer server.Close()

	client := NewOpenAICompatibleClient(time.Second)
	_, err := client.ChatStream(context.Background(), ChatRequest{
		BaseURL: server.URL,
		Model:   "stream-model",
		Messages: []ChatMessage{
			{Role: "user", Content: "hi"},
		},
	}, nil)
	if err == nil || !strings.Contains(err.Error(), "provider failed") {
		t.Fatalf("expected provider error, got %v", err)
	}
}
