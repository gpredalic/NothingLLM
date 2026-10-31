package rag

import (
	"context"
	"fmt"
	"strings"
	"time"

	"nothingllm/internal/db"
	"nothingllm/internal/openai"
	"nothingllm/internal/qdrant"
)

type Pipeline struct {
	qdrant   *qdrant.Client
	openai   *openai.Client
	database *db.DB
}

func NewPipeline(qc *qdrant.Client, oc *openai.Client, database *db.DB) *Pipeline {
	return &Pipeline{qdrant: qc, openai: oc, database: database}
}

// SearchRequest is a workspace-scoped search request.
type SearchRequest struct {
	WorkspaceID string
	Query       string
	TopK        int
	MinScore    float64
	Timeout     time.Duration
}

// SearchResult is a single search hit.
type SearchResult struct {
	ID      string         `json:"id"`
	Score   float64        `json:"score"`
	Payload map[string]any `json:"payload"`
}

// Search performs a workspace-scoped vector search across all assigned collections.
func (p *Pipeline) Search(ctx context.Context, req *SearchRequest) ([]SearchResult, error) {
	if req.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, req.Timeout)
		defer cancel()
	}
	if req.TopK <= 0 {
		req.TopK = 5
	}

	collections, err := p.database.ListWorkspaceCollections(ctx, req.WorkspaceID)
	if err != nil {
		return nil, fmt.Errorf("rag: get workspace collections: %w", err)
	}
	if len(collections) == 0 {
		return nil, nil
	}

	queryVec, err := p.openai.Embed(ctx, "text-embedding-3-small", []string{req.Query})
	if err != nil {
		return nil, fmt.Errorf("rag: embed query: %w", err)
	}
	if len(queryVec) == 0 {
		return nil, fmt.Errorf("rag: no embedding returned")
	}

	var results []SearchResult
	for _, col := range collections {
		hits, err := p.qdrant.Search(ctx, col, queryVec[0], req.TopK, nil)
		if err != nil {
			// Skip collections that fail
			continue
		}
		for _, h := range hits {
			if req.MinScore > 0 && h.Score < req.MinScore {
				continue
			}
			results = append(results, SearchResult{
				ID:      h.ID,
				Score:   h.Score,
				Payload: h.Payload,
			})
		}
	}

	// Sort by score descending
	sortByScore(results)
	if len(results) > req.TopK {
		results = results[:req.TopK]
	}
	return results, nil
}

// ChatRequest is a RAG-augmented chat request.
type ChatRequest struct {
	WorkspaceID string
	Messages    []openai.ChatMessage
	ModelID     string
	Temperature float64
	MaxResults  int
}

// Chat performs a RAG-augmented chat completion.
func (p *Pipeline) Chat(ctx context.Context, req *ChatRequest) (*openai.ChatResponse, error) {
	if req.ModelID == "" {
		req.ModelID = "gpt-4o"
	}
	if req.Temperature == 0 {
		req.Temperature = 0.7
	}
	if req.MaxResults <= 0 {
		req.MaxResults = 5
	}

	// Extract the last user message as the search query
	var query string
	for i := len(req.Messages) - 1; i >= 0; i-- {
		if req.Messages[i].Role == "user" {
			query = req.Messages[i].Content
			break
		}
	}
	if query == "" {
		// No user message; just do a plain chat
		return p.openai.Chat(ctx, req.ModelID, req.Messages, req.Temperature, 0)
	}

	// Search for context
	searchResults, err := p.Search(ctx, &SearchRequest{
		WorkspaceID: req.WorkspaceID,
		Query:       query,
		TopK:        req.MaxResults,
		Timeout:     5 * time.Second,
	})
	if err != nil {
		// If search fails, fall back to plain chat
		return p.openai.Chat(ctx, req.ModelID, req.Messages, req.Temperature, 0)
	}

	// Build augmented messages
	messages := make([]openai.ChatMessage, 0, len(req.Messages)+1)
	if len(searchResults) > 0 {
		context := p.buildContext(searchResults)
		messages = append(messages, openai.ChatMessage{
			Role:    "system",
			Content: "Use the following context to answer the user's question. If the context is not relevant, answer from your own knowledge.\n\n" + context,
		})
	}
	messages = append(messages, req.Messages...)

	return p.openai.Chat(ctx, req.ModelID, messages, req.Temperature, 0)
}

func (p *Pipeline) buildContext(results []SearchResult) string {
	var sb strings.Builder
	for i, r := range results {
		sb.WriteString(fmt.Sprintf("[%d] (score: %.3f) ", i+1, r.Score))
		// Extract text from payload - look for common text fields
		text := extractText(r.Payload)
		sb.WriteString(text)
		sb.WriteString("\n\n")
	}
	return sb.String()
}

func extractText(payload map[string]any) string {
	// Try common text field names
	for _, key := range []string{"text", "content", "chunk", "page_content", "body"} {
		if v, ok := payload[key]; ok {
			if s, ok := v.(string); ok {
				return s
			}
		}
	}
	// Fallback: concatenate all string values
	var parts []string
	for _, v := range payload {
		if s, ok := v.(string); ok && len(s) > 10 {
			parts = append(parts, s)
		}
	}
	return strings.Join(parts, " ")
}

func sortByScore(results []SearchResult) {
	// Simple insertion sort (results are small)
	for i := 1; i < len(results); i++ {
		j := i
		for j > 0 && results[j].Score > results[j-1].Score {
			results[j], results[j-1] = results[j-1], results[j]
			j--
		}
	}
}