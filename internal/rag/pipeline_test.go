package rag

import (
	"testing"
)

func TestSortByScore(t *testing.T) {
	results := []SearchResult{
		{Score: 0.5},
		{Score: 0.9},
		{Score: 0.7},
	}
	sortByScore(results)
	if results[0].Score != 0.9 {
		t.Errorf("first = %f, want 0.9", results[0].Score)
	}
	if results[1].Score != 0.7 {
		t.Errorf("second = %f, want 0.7", results[1].Score)
	}
	if results[2].Score != 0.5 {
		t.Errorf("third = %f, want 0.5", results[2].Score)
	}
}

func TestExtractText(t *testing.T) {
	// Test common field names
	payload := map[string]any{"text": "hello world"}
	if got := extractText(payload); got != "hello world" {
		t.Errorf("text field: %q", got)
	}

	payload = map[string]any{"content": "some content"}
	if got := extractText(payload); got != "some content" {
		t.Errorf("content field: %q", got)
	}

	// Fallback: concatenate string values
	payload = map[string]any{"field1": "some long string value", "num": 42}
	if got := extractText(payload); got != "some long string value" {
		t.Errorf("fallback: %q", got)
	}

	// Empty
	payload = map[string]any{"num": 42}
	if got := extractText(payload); got != "" {
		t.Errorf("empty: %q", got)
	}
}

func TestBuildContext(t *testing.T) {
	p := &Pipeline{}
	results := []SearchResult{
		{Score: 0.9, Payload: map[string]any{"text": "first result"}},
		{Score: 0.8, Payload: map[string]any{"text": "second result"}},
	}
	ctx := p.buildContext(results)
	if ctx == "" {
		t.Error("context should not be empty")
	}
	if !contains(ctx, "first result") {
		t.Error("context should contain first result")
	}
	if !contains(ctx, "second result") {
		t.Error("context should contain second result")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && indexOf(s, substr) >= 0)
}

func indexOf(s, substr string) int {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return i
		}
	}
	return -1
}