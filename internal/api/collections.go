package api

import (
	"net/http"

	"nothingllm/internal/audit"
	"nothingllm/internal/auth"
	"nothingllm/internal/db"
)

func (s *Server) handleListCollections(w http.ResponseWriter, r *http.Request) {
	cols, err := s.db.ListCollections(r.Context())
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list collections")
		return
	}
	writeJSON(w, http.StatusOK, cols)
}

func (s *Server) handleGetCollection(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	c, err := s.db.GetCollection(r.Context(), name)
	if err != nil {
		writeErr(w, http.StatusNotFound, "collection not found")
		return
	}
	writeJSON(w, http.StatusOK, c)
}

func (s *Server) handleAssignCollection(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	wsID := r.PathValue("workspace_id")
	name := r.PathValue("name")
	if err := s.db.AssignCollection(r.Context(), wsID, name); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to assign collection")
		return
	}
	s.audit.Log(audit.Event{
		UserID:     user.ID,
		Action:     "assign_collection",
		EntityType: "workspace",
		EntityID:   wsID,
		Metadata:   map[string]any{"collection": name},
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleUnassignCollection(w http.ResponseWriter, r *http.Request) {
	user := auth.GetUser(r.Context())
	wsID := r.PathValue("workspace_id")
	name := r.PathValue("name")
	if err := s.db.UnassignCollection(r.Context(), wsID, name); err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to unassign collection")
		return
	}
	s.audit.Log(audit.Event{
		UserID:     user.ID,
		Action:     "remove_collection",
		EntityType: "workspace",
		EntityID:   wsID,
		Metadata:   map[string]any{"collection": name},
	})
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleListWorkspaceCollections(w http.ResponseWriter, r *http.Request) {
	wsID := r.PathValue("workspace_id")
	names, err := s.db.ListWorkspaceCollections(r.Context(), wsID)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to list workspace collections")
		return
	}
	writeJSON(w, http.StatusOK, names)
}

func (s *Server) handleGetMetadata(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("name")
	metas, err := s.db.ListCollectionMetadata(r.Context(), name)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "failed to get metadata")
		return
	}
	writeJSON(w, http.StatusOK, metas)
}

var _ = auth.GetUser
var _ = db.ErrNotFound