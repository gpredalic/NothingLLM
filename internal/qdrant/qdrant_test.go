package qdrant

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestListCollections(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/collections":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"collections":[{"name":"test_col"}]}}`))
		case "/collections/test_col":
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{"result":{"points_count":100,"status":"green","config":{"params":{"vectors":{"size":1536}}}}}`))
		default:
			w.WriteHeader(404)
		}
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	cols, err := client.ListCollections(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(cols) != 1 {
		t.Fatalf("got %d collections, want 1", len(cols))
	}
	if cols[0].Name != "test_col" {
		t.Errorf("name = %q", cols[0].Name)
	}
	if cols[0].Vectors != 100 {
		t.Errorf("vectors = %d, want 100", cols[0].Vectors)
	}
	if cols[0].Dimensions != 1536 {
		t.Errorf("dimensions = %d, want 1536", cols[0].Dimensions)
	}
}

func TestGetCollectionInfo(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"result":{"points_count":42,"status":"green","config":{"params":{"vectors":{"size":768}}}}}`))
	}))
	defer server.Close()

	client := NewClient(server.URL, "test-key")
	info, err := client.GetCollectionInfo(context.Background(), "mycol")
	if err != nil {
		t.Fatal(err)
	}
	if info.Vectors != 42 {
		t.Errorf("vectors = %d, want 42", info.Vectors)
	}
	if info.Dimensions != 768 {
		t.Errorf("dimensions = %d, want 768", info.Dimensions)
	}
}

func TestPing(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	if err := client.Ping(context.Background()); err != nil {
		t.Fatal("ping should succeed:", err)
	}
}

func TestPingFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	client := NewClient(server.URL, "")
	if err := client.Ping(context.Background()); err == nil {
		t.Error("ping should fail")
	}
}

func TestToString(t *testing.T) {
	if s := toString("hello"); s != "hello" {
		t.Errorf("string: %q", s)
	}
	if s := toString(42.0); s != "42" {
		t.Errorf("float: %q", s)
	}
	if s := toString(true); s != "true" {
		t.Errorf("bool: %q", s)
	}
	if s := toString(nil); s != "" {
		t.Errorf("nil: %q", s)
	}
}