package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"nothingllm/internal/audit"
	"nothingllm/internal/db"
	"nothingllm/internal/openai"
	"nothingllm/internal/qdrant"
)

func (s *Server) handleListModels(w http.ResponseWriter, r *http.Request) {
	models, err := s.db.ListModels(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list models")
		return
	}
	if models == nil {
		models = []db.Model{}
	}
	writeJSON(w, http.StatusOK, models)
}

func (s *Server) handleUpdateModel(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var body struct {
		Enabled *bool `json:"enabled"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Enabled == nil {
		writeErr(w, http.StatusBadRequest, "enabled field is required")
		return
	}
	if err := s.db.SetModelEnabled(r.Context(), id, *body.Enabled); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to update model")
		return
	}
	s.audit.Log(audit.Event{
		Action:     "update_model",
		EntityType: "model",
		EntityID:   id,
		Metadata:   map[string]any{"enabled": *body.Enabled},
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleGetSettings(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	cfg, err := s.db.LoadConfigFromDB(ctx)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to load settings")
		return
	}
	openaiKey := cfg["openai_api_key"]
	maskedKey := ""
	if openaiKey != "" {
		if len(openaiKey) > 8 {
			maskedKey = openaiKey[:4] + "..." + openaiKey[len(openaiKey)-4:]
		} else {
			maskedKey = "****"
		}
	}
	embModel, _ := s.db.GetSetting(ctx, "embedding_model")
	writeJSON(w, http.StatusOK, map[string]any{
		"openai_api_key":  maskedKey,
		"qdrant_url":      cfg["qdrant_url"],
		"qdrant_api_key":  maskValue(cfg["qdrant_api_key"]),
		"embedding_model": embModel,
	})
}

func (s *Server) handleUpdateSettings(w http.ResponseWriter, r *http.Request) {
	var body struct {
		OpenAIAPIKey   *string `json:"openai_api_key"`
		QdrantURL      *string `json:"qdrant_url"`
		QdrantAPIKey   *string `json:"qdrant_api_key"`
		EmbeddingModel *string `json:"embedding_model"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	ctx := r.Context()
	if body.OpenAIAPIKey != nil && *body.OpenAIAPIKey != "" {
		if err := s.db.SetSetting(ctx, "openai_api_key", *body.OpenAIAPIKey); err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to save openai key")
			return
		}
	}
	if body.QdrantURL != nil && *body.QdrantURL != "" {
		if err := s.db.SetSetting(ctx, "qdrant_url", *body.QdrantURL); err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to save qdrant url")
			return
		}
	}
	if body.QdrantAPIKey != nil && *body.QdrantAPIKey != "" {
		if err := s.db.SetSetting(ctx, "qdrant_api_key", *body.QdrantAPIKey); err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to save qdrant key")
			return
		}
	}
	if body.EmbeddingModel != nil && *body.EmbeddingModel != "" {
		if err := s.db.SetSetting(ctx, "embedding_model", *body.EmbeddingModel); err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to save embedding model")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}


func (s *Server) handleTestConnection(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	results := map[string]any{}

	openaiKey, _ := s.db.GetSetting(ctx, "openai_api_key")
	if openaiKey == "" {
		results["openai"] = map[string]string{"status": "not_configured"}
	} else {
		client := openai.NewClient(openaiKey, "")
		testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := client.Ping(testCtx); err != nil {
			results["openai"] = map[string]string{"status": "failed", "error": err.Error()}
		} else {
			results["openai"] = map[string]string{"status": "ok"}
		}
	}

	qdrantURL, _ := s.db.GetSetting(ctx, "qdrant_url")
	qdrantKey, _ := s.db.GetSetting(ctx, "qdrant_api_key")
	if qdrantURL == "" {
		results["qdrant"] = map[string]string{"status": "not_configured"}
	} else {
		client := qdrant.NewClient(qdrantURL, qdrantKey)
		testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := client.Ping(testCtx); err != nil {
			results["qdrant"] = map[string]string{"status": "failed", "error": err.Error()}
		} else {
			results["qdrant"] = map[string]string{"status": "ok"}
		}
	}

	writeJSON(w, http.StatusOK, results)
}

func (s *Server) handleGetStats(w http.ResponseWriter, r *http.Request) {
	stats, err := s.db.GetAdminStats(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to get stats")
		return
	}
	writeJSON(w, http.StatusOK, stats)
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	results := map[string]any{}

	if err := s.db.Ping(ctx); err != nil {
		results["database"] = map[string]string{"status": "down", "error": err.Error()}
	} else {
		results["database"] = map[string]string{"status": "ok"}
	}

	qdrantURL, _ := s.db.GetSetting(ctx, "qdrant_url")
	qdrantKey, _ := s.db.GetSetting(ctx, "qdrant_api_key")
	if qdrantURL == "" {
		results["qdrant"] = map[string]string{"status": "not_configured"}
	} else {
		client := qdrant.NewClient(qdrantURL, qdrantKey)
		testCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := client.Ping(testCtx); err != nil {
			results["qdrant"] = map[string]string{"status": "down", "error": err.Error()}
		} else {
			results["qdrant"] = map[string]string{"status": "ok"}
		}
	}

	openaiKey, _ := s.db.GetSetting(ctx, "openai_api_key")
	if openaiKey == "" {
		results["openai"] = map[string]string{"status": "not_configured"}
	} else {
		client := openai.NewClient(openaiKey, "")
		testCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := client.Ping(testCtx); err != nil {
			results["openai"] = map[string]string{"status": "down", "error": err.Error()}
		} else {
			results["openai"] = map[string]string{"status": "ok"}
		}
	}

	writeJSON(w, http.StatusOK, results)
}

func maskValue(v string) string {
	if v == "" {
		return ""
	}
	if len(v) > 4 {
		return "****" + v[len(v)-4:]
	}
	return "****"
}
