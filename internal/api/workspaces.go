package api

import (
	"encoding/json"
	"errors"
	"net/http"

	"nothingllm/internal/audit"
	"nothingllm/internal/auth"
	"nothingllm/internal/db"
)

// accessLevel defines the minimum access required for a workspace operation.
type accessLevel int

const (
	accessView accessLevel = iota // member or above
	accessManage                   // owner or admin
)

// checkWorkspaceAccess verifies the user has the required access level for a workspace.
// Returns (true, nil) if authorized, (false, httpStatus) if not.
func (s *Server) checkWorkspaceAccess(r *http.Request, workspaceID string, required accessLevel) (bool, int) {
	user := auth.GetUser(r.Context())
	if user == nil {
		return false, http.StatusUnauthorized
	}
	// Admins can access all workspaces.
	if user.Role == "admin" {
		return true, 0
	}
	role, err := s.db.GetWorkspaceMemberRole(r.Context(), workspaceID, user.ID)
	if err != nil {
		if errors.Is(err, db.ErrNotFound) {
			return false, http.StatusForbidden
		}
		return false, http.StatusInternalServerError
	}
	switch required {
	case accessView:
		// Any member (owner, member) can view.
		return true, 0
	case accessManage:
		// Only owner can manage.
		if role == "owner" {
			return true, 0
		}
		return false, http.StatusForbidden
	}
	return false, http.StatusInternalServerError
}

func (s *Server) handleCreateWorkspace(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Name == "" {
		writeErr(w, http.StatusBadRequest, "name is required")
		return
	}
	ws, err := s.db.CreateWorkspace(r.Context(), body.Name, body.Description, user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to create workspace")
		return
	}
	s.audit.Log(audit.Event{
		UserID:     user.ID,
		Action:     "create_workspace",
		EntityType: "workspace",
		EntityID:   ws.ID,
		Metadata:   map[string]any{"name": ws.Name},
	})
	writeJSON(w, http.StatusCreated, ws)
}

func (s *Server) handleListWorkspaces(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	ws, err := s.db.ListWorkspaces(r.Context(), user.ID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list workspaces")
		return
	}
	writeJSON(w, http.StatusOK, ws)
}

func (s *Server) handleGetWorkspace(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	ok, status := s.checkWorkspaceAccess(r, id, accessView)
	if !ok {
		writeErr(w, status, "access denied")
		return
	}
	ws, err := s.db.GetWorkspace(r.Context(), id)
	if errors.Is(err, db.ErrNotFound) {
		writeErr(w, http.StatusNotFound, "workspace not found")
		return
	}
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to get workspace")
		return
	}
	writeJSON(w, http.StatusOK, ws)
}

func (s *Server) handleDeleteWorkspace(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	id := r.PathValue("id")
	ok, status := s.checkWorkspaceAccess(r, id, accessManage)
	if !ok {
		writeErr(w, status, "access denied")
		return
	}
	if err := s.db.DeleteWorkspace(r.Context(), id); err != nil {
		if errors.Is(err, db.ErrNotFound) {
			writeErr(w, http.StatusNotFound, "workspace not found")
			return
		}
		writeErr(w, http.StatusInternalServerError, "failed to delete workspace")
		return
	}
	s.audit.Log(audit.Event{
		UserID:     user.ID,
		Action:     "delete_workspace",
		EntityType: "workspace",
		EntityID:   id,
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleAddMember(w http.ResponseWriter, r *http.Request) {
	wsID := r.PathValue("workspace_id")
	ok, status := s.checkWorkspaceAccess(r, wsID, accessManage)
	if !ok {
		writeErr(w, status, "access denied")
		return
	}
	var body struct {
		UserID string `json:"user_id"`
		Role   string `json:"role"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request body")
		return
	}
	if body.Role == "" {
		body.Role = "member"
	}
	if err := s.db.AddWorkspaceMember(r.Context(), wsID, body.UserID, body.Role); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to add member")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleRemoveMember(w http.ResponseWriter, r *http.Request) {
	wsID := r.PathValue("workspace_id")
	ok, status := s.checkWorkspaceAccess(r, wsID, accessManage)
	if !ok {
		writeErr(w, status, "access denied")
		return
	}
	userID := r.PathValue("user_id")
	if err := s.db.RemoveWorkspaceMember(r.Context(), wsID, userID); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to remove member")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListMembers(w http.ResponseWriter, r *http.Request) {
	wsID := r.PathValue("workspace_id")
	members, err := s.db.ListWorkspaceMembers(r.Context(), wsID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list members")
		return
	}
	writeJSON(w, http.StatusOK, members)
}