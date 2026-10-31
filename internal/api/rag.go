package api

import (
	"encoding/json"
	"net/http"

	"nothingllm/internal/auth"
	"nothingllm/internal/openai"
	"nothingllm/internal/rag"
)

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	wsID := r.PathValue("workspace_id")
	var body struct {
		Query    string  `json:"query"`
		TopK     int     `json:"top_k"`
		MinScore float64 `json:"min_score"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Query == "" {
		writeErr(w, http.StatusBadRequest, "query is required")
		return
	}

	results, err := s.rag.Search(r.Context(), &rag.SearchRequest{
		WorkspaceID: wsID,
		Query:       body.Query,
		TopK:        body.TopK,
		MinScore:    body.MinScore,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "search failed")
		return
	}
	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleChat(w http.ResponseWriter, r *http.Request) {
	wsID := r.PathValue("workspace_id")
	var body struct {
		Messages []openai.ChatMessage `json:"messages"`
		ModelID  string               `json:"model_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if len(body.Messages) == 0 {
		writeErr(w, http.StatusBadRequest, "messages are required")
		return
	}

	resp, err := s.rag.Chat(r.Context(), &rag.ChatRequest{
		WorkspaceID: wsID,
		Messages:    body.Messages,
		ModelID:     body.ModelID,
	})
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "chat failed")
		return
	}
	writeJSON(w, http.StatusOK, resp)
}

var _ = auth.GetUser