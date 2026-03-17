package ai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestLLMClient_Chat_ToolCallResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/chat/completions" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-key" {
			t.Fatalf("unexpected authorization header: %q", got)
		}
		if got := r.Header.Get("Content-Type"); !strings.Contains(got, "application/json") {
			t.Fatalf("unexpected content-type: %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "",
					"tool_calls": [{
						"id": "call_1",
						"type": "function",
						"function": {
							"name": "record_transaction",
							"arguments": "{\"type\":\"expense\",\"amount\":16000,\"category\":\"Makanan\",\"description\":\"ayam crispy\"}"
						}
					}]
				}
			}]
		}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "gpt-test")
	got, err := client.Chat(context.Background(), "system prompt", "beli ayam 16k")
	if err != nil {
		t.Fatalf("Chat() returned error: %v", err)
	}

	if got == nil {
		t.Fatal("expected non-nil response")
	}
	if got.Content != "" {
		t.Fatalf("expected empty content for tool-call response, got %q", got.Content)
	}
	if len(got.ToolCalls) != 1 {
		t.Fatalf("expected 1 tool call, got %d", len(got.ToolCalls))
	}
	if got.ToolCalls[0].Name != "record_transaction" {
		t.Fatalf("expected tool name record_transaction, got %q", got.ToolCalls[0].Name)
	}

	args := string(got.ToolCalls[0].Arguments)
	expectedParts := []string{`"type":"expense"`, `"amount":16000`, `"category":"Makanan"`, `"description":"ayam crispy"`}
	for _, part := range expectedParts {
		if !strings.Contains(args, part) {
			t.Fatalf("expected tool args to contain %q, got %s", part, args)
		}
	}
}

func TestLLMClient_Chat_TextOnlyResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"choices": [{
				"message": {
					"content": "Halo! Siap bantu catat keuangan kamu."
				}
			}]
		}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "gpt-test")
	got, err := client.Chat(context.Background(), "system", "halo")
	if err != nil {
		t.Fatalf("Chat() returned error: %v", err)
	}

	if got == nil {
		t.Fatal("expected non-nil response")
	}
	if got.Content != "Halo! Siap bantu catat keuangan kamu." {
		t.Fatalf("unexpected content: %q", got.Content)
	}
	if len(got.ToolCalls) != 0 {
		t.Fatalf("expected no tool calls, got %d", len(got.ToolCalls))
	}
}

func TestLLMClient_Chat_Timeout(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(150 * time.Millisecond)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[{"message":{"content":"late"}}]}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "gpt-test")

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()

	_, err := client.Chat(ctx, "system", "hello")
	if err == nil {
		t.Fatal("expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "llm request failed") {
		t.Fatalf("expected timeout/network error wrapper, got: %v", err)
	}
}

func TestLLMClient_Chat_MalformedJSONResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{this-is-not-valid-json`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "gpt-test")
	_, err := client.Chat(context.Background(), "system", "hello")
	if err == nil {
		t.Fatal("expected malformed JSON error, got nil")
	}
	if !strings.Contains(err.Error(), "failed to decode llm response json") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLLMClient_Chat_EmptyResponse(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"choices":[]}`))
	}))
	defer server.Close()

	client := NewLLMClient(server.URL, "test-key", "gpt-test")
	_, err := client.Chat(context.Background(), "system", "hello")
	if err == nil {
		t.Fatal("expected empty response error, got nil")
	}
	if !strings.Contains(err.Error(), "empty llm response: no choices") {
		t.Fatalf("unexpected error: %v", err)
	}
}
