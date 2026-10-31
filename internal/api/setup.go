package api

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"nothingllm/internal/db"
	"nothingllm/internal/openai"
	"nothingllm/internal/qdrant"
)

// SetupServer handles the first-run setup wizard.
type SetupServer struct {
	database *db.DB
}

func NewSetupServer(database *db.DB) *SetupServer {
	return &SetupServer{database: database}
}

// Handler returns the setup mux.
func (s *SetupServer) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/setup/status", s.handleStatus)
	mux.HandleFunc("POST /api/setup/admin", s.handleCreateAdmin)
	mux.HandleFunc("POST /api/setup/config", s.handleSaveConfig)
	mux.HandleFunc("POST /api/setup/test", s.handleTestConnectivity)
	mux.HandleFunc("POST /api/setup/complete", s.handleComplete)
	return mux
}

func (s *SetupServer) handleStatus(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	complete, _ := s.database.IsSetupComplete(ctx)
	adminCount, _ := s.database.CountAdmins(ctx)

	cfg := map[string]bool{
		"openai_configured": false,
		"qdrant_configured": false,
	}
	if v, err := s.database.GetSetting(ctx, "openai_api_key"); err == nil && v != "" {
		cfg["openai_configured"] = true
	}
	if v, err := s.database.GetSetting(ctx, "qdrant_url"); err == nil && v != "" {
		cfg["qdrant_configured"] = true
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"complete":  complete,
		"has_admin": adminCount > 0,
		"config":    cfg,
	})
}

func (s *SetupServer) handleCreateAdmin(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	complete, _ := s.database.IsSetupComplete(ctx)
	if complete {
		writeErr(w, http.StatusForbidden, "setup already complete")
		return
	}
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Email == "" || body.Password == "" {
		writeErr(w, http.StatusBadRequest, "email and password are required")
		return
	}
	if len(body.Email) > 254 || len(body.Password) > 128 {
		writeErr(w, http.StatusBadRequest, "input too long")
		return
	}
	user, err := s.database.CreateUser(ctx, body.Email, body.Password, "admin")
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to create admin")
		return
	}
	writeJSON(w, http.StatusCreated, map[string]string{"id": user.ID, "email": user.Email})
}

func (s *SetupServer) handleSaveConfig(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	complete, _ := s.database.IsSetupComplete(ctx)
	if complete {
		writeErr(w, http.StatusForbidden, "setup already complete")
		return
	}
	var body struct {
		OpenAIAPIKey string `json:"openai_api_key"`
		QdrantURL    string `json:"qdrant_url"`
		QdrantAPIKey string `json:"qdrant_api_key"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.OpenAIAPIKey != "" {
		if err := s.database.SetSetting(ctx, "openai_api_key", body.OpenAIAPIKey); err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to save openai config")
			return
		}
	}
	if body.QdrantURL != "" {
		if err := s.database.SetSetting(ctx, "qdrant_url", body.QdrantURL); err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to save qdrant config")
			return
		}
	}
	if body.QdrantAPIKey != "" {
		if err := s.database.SetSetting(ctx, "qdrant_api_key", body.QdrantAPIKey); err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to save qdrant config")
			return
		}
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *SetupServer) handleTestConnectivity(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	openaiKey, _ := s.database.GetSetting(ctx, "openai_api_key")
	qdrantURL, _ := s.database.GetSetting(ctx, "qdrant_url")
	qdrantKey, _ := s.database.GetSetting(ctx, "qdrant_api_key")

	results := map[string]any{}

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

func (s *SetupServer) handleComplete(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	complete, _ := s.database.IsSetupComplete(ctx)
	if complete {
		writeErr(w, http.StatusForbidden, "setup already complete")
		return
	}
	adminCount, _ := s.database.CountAdmins(ctx)
	if adminCount == 0 {
		writeErr(w, http.StatusBadRequest, "admin user required")
		return
	}
	if _, err := s.database.GetSetting(ctx, "openai_api_key"); err != nil {
		writeErr(w, http.StatusBadRequest, "openai_api_key required")
		return
	}
	if _, err := s.database.GetSetting(ctx, "qdrant_url"); err != nil {
		writeErr(w, http.StatusBadRequest, "qdrant_url required")
		return
	}

	secret, err := generateSecret()
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to generate secret")
		return
	}
	if err := s.database.SetSetting(ctx, "session_secret", secret); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to save secret")
		return
	}
	if err := s.database.MarkSetupComplete(ctx); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to mark complete")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok", "message": "setup complete"})
}

func generateSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}