package openai

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestEmbed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"index":0,"embedding":[0.1,0.2,0.3]}]}`))
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL)
	vectors, err := client.Embed(context.Background(), "text-embedding-3-small", []string{"hello"})
	if err != nil {
		t.Fatal(err)
	}
	if len(vectors) != 1 {
		t.Fatalf("got %d vectors, want 1", len(vectors))
	}
	if len(vectors[0]) != 3 {
		t.Errorf("vector dim = %d, want 3", len(vectors[0]))
	}
}

func TestChat(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"choices":[{"message":{"role":"assistant","content":"Hi!"}}],"model":"gpt-4o"}`))
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL)
	resp, err := client.Chat(context.Background(), "gpt-4o", []ChatMessage{{Role: "user", Content: "Hello"}}, 0.7, 0)
	if err != nil {
		t.Fatal(err)
	}
	if resp.Content != "Hi!" {
		t.Errorf("content = %q, want Hi!", resp.Content)
	}
}

func TestListModels(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"data":[{"id":"gpt-4o","object":"model","created":1700000000,"context_window":128000}]}`))
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL)
	models, err := client.ListModels(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(models) != 1 {
		t.Fatalf("got %d models, want 1", len(models))
	}
	if models[0].ID != "gpt-4o" {
		t.Errorf("model id = %q", models[0].ID)
	}
}

func TestPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test-key", server.URL)
	if err := client.Ping(context.Background()); err != nil {
		t.Fatal("ping should succeed:", err)
	}
}

func TestPingFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient("bad-key", server.URL)
	if err := client.Ping(context.Background()); err == nil {
		t.Error("ping should fail")
	}
}