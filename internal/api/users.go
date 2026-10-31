package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"nothingllm/internal/audit"
	"nothingllm/internal/auth"
	"nothingllm/internal/db"
)

func (s *Server) handleListUsers(w http.ResponseWriter, r *http.Request) {
	users, err := s.db.ListUsers(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list users")
		return
	}
	writeJSON(w, http.StatusOK, users)
}

func (s *Server) handleUpdateUser(w http.ResponseWriter, r *http.Request) {
	admin := auth.GetUser(r.Context())
	id := r.PathValue("id")
	var body struct {
		Role     *string `json:"role"`
		Password *string `json:"password"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Role != nil {
		if err := s.db.UpdateUserRole(r.Context(), id, *body.Role); err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to update role")
			return
		}
		s.audit.Log(audit.Event{
			UserID:     admin.ID,
			Action:     "update_user",
			EntityType: "user",
			EntityID:   id,
			Metadata:   map[string]any{"field": "role", "value": *body.Role},
		})
	}
	if body.Password != nil {
		if err := s.db.UpdateUserPassword(r.Context(), id, *body.Password); err != nil {
			writeErr(w, http.StatusInternalServerError, "failed to update password")
			return
		}
		s.audit.Log(audit.Event{
			UserID:     admin.ID,
			Action:     "update_user",
			EntityType: "user",
			EntityID:   id,
			Metadata:   map[string]any{"field": "password"},
		})
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleDeleteUser(w http.ResponseWriter, r *http.Request) {
	admin := auth.GetUser(r.Context())
	id := r.PathValue("id")
	if admin != nil && admin.ID == id {
		writeErr(w, http.StatusBadRequest, "cannot delete yourself")
		return
	}
	if err := s.db.DeleteUser(r.Context(), id); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "user not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "failed to delete user")
		return
	}
	s.audit.Log(audit.Event{
		UserID:     admin.ID,
		Action:     "delete_user",
		EntityType: "user",
		EntityID:   id,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}