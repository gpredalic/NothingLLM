package api

import (
	"encoding/json"
	"net/http"

	"nothingllm/internal/audit"
	"nothingllm/internal/auth"
	"nothingllm/internal/db"
	"nothingllm/internal/rag"
)

type Server struct {
	auth  *auth.Service
	db    *db.DB
	rag   *rag.Pipeline
	audit *audit.Logger
	mux   *http.ServeMux
}

func NewServer(authSvc *auth.Service, database *db.DB, pipeline *rag.Pipeline, auditLog *audit.Logger) *Server {
	s := &Server{
		auth:  authSvc,
		db:    database,
		rag:   pipeline,
		audit: auditLog,
		mux:   http.NewServeMux(),
	}
	s.routes()
	return s
}

func (s *Server) Handler() http.Handler {
	return s.mux
}

func (s *Server) routes() {
	s.mux.HandleFunc("POST /api/auth/register", s.handleRegister)
	s.mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	s.mux.HandleFunc("POST /api/auth/logout", s.handleLogout)

	s.mux.HandleFunc("GET /api/users", auth.RequireAuth(s.auth, s.handleListUsers))
	s.mux.HandleFunc("PATCH /api/users/{id}", auth.RequireAuth(s.auth, auth.RequireAdmin(s.handleUpdateUser)))
	s.mux.HandleFunc("DELETE /api/users/{id}", auth.RequireAuth(s.auth, auth.RequireAdmin(s.handleDeleteUser)))

	// Workspaces
	s.mux.HandleFunc("POST /api/workspaces", auth.RequireAuth(s.auth, s.handleCreateWorkspace))
	s.mux.HandleFunc("GET /api/workspaces", auth.RequireAuth(s.auth, s.handleListWorkspaces))
	s.mux.HandleFunc("GET /api/workspaces/{id}", auth.RequireAuth(s.auth, s.handleGetWorkspace))
	s.mux.HandleFunc("DELETE /api/workspaces/{id}", auth.RequireAuth(s.auth, s.handleDeleteWorkspace))
	s.mux.HandleFunc("POST /api/workspaces/{workspace_id}/members", auth.RequireAuth(s.auth, s.handleAddMember))
	s.mux.HandleFunc("DELETE /api/workspaces/{workspace_id}/members/{user_id}", auth.RequireAuth(s.auth, s.handleRemoveMember))
	s.mux.HandleFunc("GET /api/workspaces/{workspace_id}/members", auth.RequireAuth(s.auth, s.handleListMembers))

	// Collections
	s.mux.HandleFunc("GET /api/collections", auth.RequireAuth(s.auth, s.handleListCollections))
	s.mux.HandleFunc("GET /api/collections/{name}", auth.RequireAuth(s.auth, s.handleGetCollection))
	s.mux.HandleFunc("GET /api/collections/{name}/metadata", auth.RequireAuth(s.auth, s.handleGetMetadata))
	s.mux.HandleFunc("POST /api/workspaces/{workspace_id}/collections/{name}", auth.RequireAuth(s.auth, s.handleAssignCollection))
	s.mux.HandleFunc("DELETE /api/workspaces/{workspace_id}/collections/{name}", auth.RequireAuth(s.auth, s.handleUnassignCollection))
	s.mux.HandleFunc("GET /api/workspaces/{workspace_id}/collections", auth.RequireAuth(s.auth, s.handleListWorkspaceCollections))

	// RAG
	s.mux.HandleFunc("POST /api/workspaces/{workspace_id}/search", auth.RequireAuth(s.auth, s.handleSearch))
	s.mux.HandleFunc("POST /api/workspaces/{workspace_id}/chat", auth.RequireAuth(s.auth, s.handleChat))

	// Admin
	s.mux.HandleFunc("GET /api/models", auth.RequireAuth(s.auth, auth.RequireAdmin(s.handleListModels)))
	s.mux.HandleFunc("PATCH /api/models/{id}", auth.RequireAuth(s.auth, auth.RequireAdmin(s.handleUpdateModel)))
	s.mux.HandleFunc("PUT /api/models/default", auth.RequireAuth(s.auth, auth.RequireAdmin(s.handleSetDefaultModel)))
	s.mux.HandleFunc("GET /api/admin/settings", auth.RequireAuth(s.auth, auth.RequireAdmin(s.handleGetSettings)))
	s.mux.HandleFunc("PUT /api/admin/settings", auth.RequireAuth(s.auth, auth.RequireAdmin(s.handleUpdateSettings)))
	s.mux.HandleFunc("POST /api/admin/test-connection", auth.RequireAuth(s.auth, auth.RequireAdmin(s.handleTestConnection)))
	s.mux.HandleFunc("GET /api/admin/stats", auth.RequireAuth(s.auth, s.handleGetStats))
	s.mux.HandleFunc("GET /api/health", s.handleHealth)
}

func (s *Server) handleRegister(w http.ResponseWriter, r *http.Request) {
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
	user, err := s.auth.Register(r.Context(), body.Email, body.Password)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "registration failed")
		return
	}
	s.audit.Log(audit.Event{
		Action:     "create_user",
		EntityType: "user",
		EntityID:   user.ID,
		Metadata:   map[string]any{"email": user.Email, "role": user.Role},
	})
	writeJSON(w, http.StatusCreated, user)
}

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
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
	sessionID, err := s.auth.Login(r.Context(), body.Email, body.Password)
	if err != nil {
		if err == auth.ErrInvalidCredentials {
			writeErr(w, http.StatusUnauthorized, "invalid credentials")
			return
		}
		writeErr(w, http.StatusInternalServerError, "login failed")
		return
	}
	s.audit.Log(audit.Event{
		Action:     "login",
		EntityType: "user",
		EntityID:   body.Email,
	})
	http.SetCookie(w, &http.Cookie{
		Name:     "session",
		Value:    sessionID,
		Path:     "/",
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteStrictMode,
		MaxAge:   86400,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	c, err := r.Cookie("session")
	if err == nil {
		_ = s.auth.Logout(r.Context(), c.Value)
		s.audit.Log(audit.Event{
			Action:     "logout",
			EntityType: "session",
			EntityID:   c.Value,
		})
	}
	http.SetCookie(w, &http.Cookie{
		Name:   "session",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}