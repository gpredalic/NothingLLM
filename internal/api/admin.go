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
	defaultModel, _ := s.db.GetSetting(r.Context(), "default_model")
	type modelWithDefault struct {
		db.Model
		IsDefault bool `json:"is_default"`
	}
	out := make([]modelWithDefault, 0, len(models))
	for _, m := range models {
		out = append(out, modelWithDefault{Model: m, IsDefault: m.ID == defaultModel})
	}
	writeJSON(w, http.StatusOK, out)
}

func (s *Server) handleSetDefaultModel(w http.ResponseWriter, r *http.Request) {
	var body struct {
		ID string `json:"id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.ID == "" {
		// Clear default
		if err := s.db.SetSetting(r.Context(), "default_model", ""); err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to clear default model")
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
		return
	}
	// Verify model exists and is enabled
	models, err := s.db.ListModels(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list models")
		return
	}
	found := false
	for _, m := range models {
		if m.ID == body.ID {
			if !m.Enabled {
				writeErr(w, http.StatusBadRequest, "model is not enabled")
				return
			}
			found = true
			break
		}
	}
	if !found {
		writeErr(w, http.StatusNotFound, "model not found")
		return
	}
	if err := s.db.SetSetting(r.Context(), "default_model", body.ID); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to set default model")
		return
	}
	s.audit.Log(audit.Event{
		Action:     "set_default_model",
		EntityType: "model",
		EntityID:   body.ID,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
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
	openaiURL, _ := s.db.GetSetting(ctx, "openai_base_url")
	if openaiKey == "" {
		results["openai"] = map[string]string{"status": "not_configured"}
	} else {
		client := openai.NewClient(openaiKey, openaiURL)
		testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := client.Ping(testCtx); err != nil {
			results["openai"] = map[string]string{"status": "failed", "error": err.Error()}
		} else {
			results["openai"] = map[string]string{"status": "ok"}
		}
	}

	// BGE-M3 embeddings service
	bgeURL, _ := s.db.GetSetting(ctx, "bge_m3_url")
	if bgeURL == "" {
		results["bge_m3"] = map[string]string{"status": "not_configured"}
	} else {
		client := openai.NewClient("", bgeURL)
		testCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		if err := client.Ping(testCtx); err != nil {
			results["bge_m3"] = map[string]string{"status": "failed", "error": err.Error()}
		} else {
			results["bge_m3"] = map[string]string{"status": "ok"}
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
		testCtx, cancel := context.WithTimeout(ctx, 1*time.Second)
		defer cancel()
		if err := client.Ping(testCtx); err != nil {
			results["qdrant"] = map[string]string{"status": "down", "error": err.Error()}
		} else {
			results["qdrant"] = map[string]string{"status": "ok"}
		}
	}

	openaiKey, _ := s.db.GetSetting(ctx, "openai_api_key")
	openaiURL, _ := s.db.GetSetting(ctx, "openai_base_url")
	if openaiKey == "" {
		results["openai"] = map[string]string{"status": "not_configured"}
	} else {
		client := openai.NewClient(openaiKey, openaiURL)
		testCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := client.Ping(testCtx); err != nil {
			results["openai"] = map[string]string{"status": "down", "error": err.Error()}
		} else {
			results["openai"] = map[string]string{"status": "ok"}
		}
	}

	// BGE-M3 embeddings service
	bgeURL, _ := s.db.GetSetting(ctx, "bge_m3_url")
	if bgeURL == "" {
		results["bge_m3"] = map[string]string{"status": "not_configured"}
	} else {
		client := openai.NewClient("", bgeURL)
		testCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
		defer cancel()
		if err := client.Ping(testCtx); err != nil {
			results["bge_m3"] = map[string]string{"status": "down", "error": err.Error()}
		} else {
			results["bge_m3"] = map[string]string{"status": "ok"}
		}
	}

	writeJSON(w, http.StatusOK, results)
}

// handleSetupStatus reports setup state. Always available (both setup and full mode).
func (s *Server) handleSetupStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	complete, _ := s.db.IsSetupComplete(ctx)
	adminCount, _ := s.db.CountAdmins(ctx)

	writeJSON(w, http.StatusOK, map[string]any{
		"complete":  complete,
		"has_admin": adminCount > 0,
	})
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
