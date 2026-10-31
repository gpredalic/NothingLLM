package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"nothingllm/internal/auth"
	"nothingllm/internal/db"
)

// newTestServer creates a Server with a nil DB for testing authorization
// paths that return before touching the database (admin, nil user).
func newTestServer() *Server {
	return &Server{
		mux: http.NewServeMux(),
	}
}

func TestCheckWorkspaceAccess_NilUser(t *testing.T) {
	s := newTestServer()
	req := httptest.NewRequest("GET", "/api/workspaces/test-id", nil)

	// No user in context
	ok, status := s.checkWorkspaceAccess(req, "test-id", accessView)
	if ok {
		t.Error("expected access denied for nil user")
	}
	if status != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", status, http.StatusUnauthorized)
	}
}

func TestCheckWorkspaceAccess_Admin(t *testing.T) {
	s := newTestServer()
	admin := &db.User{ID: "admin-1", Email: "admin@test.com", Role: "admin"}

	req := httptest.NewRequest("GET", "/api/workspaces/test-id", nil)
	ctx := context.WithValue(req.Context(), auth.UserKey, admin)
	req = req.WithContext(ctx)

	// Admin should always pass, regardless of access level
	ok, status := s.checkWorkspaceAccess(req, "test-id", accessView)
	if !ok {
		t.Error("expected admin to have view access")
	}
	if status != 0 {
		t.Errorf("status = %d, want 0", status)
	}

	ok, status = s.checkWorkspaceAccess(req, "test-id", accessManage)
	if !ok {
		t.Error("expected admin to have manage access")
	}
	if status != 0 {
		t.Errorf("status = %d, want 0", status)
	}
}

// TestHandlerWorkspaceAuth_NilUser verifies that workspace handlers return 401
// when no authenticated user is in context.
func TestHandlerWorkspaceAuth_NilUser(t *testing.T) {
	s := newTestServer()
	s.routes()

	tests := []struct {
		name   string
		method string
		path   string
	}{
		{"get workspace", "GET", "/api/workspaces/ws-123"},
		{"delete workspace", "DELETE", "/api/workspaces/ws-123"},
		{"add member", "POST", "/api/workspaces/ws-123/members"},
		{"remove member", "DELETE", "/api/workspaces/ws-123/members/user-456"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, strings.NewReader("{}"))
			rec := httptest.NewRecorder()
			s.mux.ServeHTTP(rec, req)
			// Without auth middleware, the handler will get a nil user
			// and should return 401.
			if rec.Code != http.StatusUnauthorized {
				t.Errorf("%s %s: status = %d, want %d", tt.method, tt.path, rec.Code, http.StatusUnauthorized)
			}
		})
	}
}

// TestHandlerWorkspaceAuth_Admin verifies that admin users can access
// workspace endpoints (authorization passes, then fails at DB layer
// which returns 500 since db is nil — proving auth check passed).
func TestHandlerWorkspaceAuth_Admin_PassesAuthCheck(t *testing.T) {
	s := newTestServer()
	s.routes()

	admin := &db.User{ID: "admin-1", Email: "admin@test.com", Role: "admin"}

	// Simulate a request with admin in context by calling the handler directly
	req := httptest.NewRequest("GET", "/api/workspaces/ws-123", nil)
	ctx := context.WithValue(req.Context(), auth.UserKey, admin)
	req = req.WithContext(ctx)

	rec := httptest.NewRecorder()
	// Call handleGetWorkspace directly — admin passes auth check,
	// then fails at s.db.GetWorkspace (nil db → panic recovered or 500).
	// We verify the auth check itself passed by checking it doesn't return 401/403.
	defer func() {
		if r := recover(); r != nil {
			// Panic from nil db is expected — proves we passed the auth gate.
			t.Log("handler panicked at DB layer (expected with nil db), auth check passed")
		}
	}()
	s.handleGetWorkspace(rec, req)

	// If we get here without panic, status should NOT be 401 or 403
	if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusForbidden {
		t.Errorf("admin should pass auth check, got status %d", rec.Code)
	}
}

// TestAccessLevelConstants verifies the access level ordering.
func TestAccessLevelConstants(t *testing.T) {
	if accessView != 0 {
		t.Errorf("accessView = %d, want 0", accessView)
	}
	if accessManage != 1 {
		t.Errorf("accessManage = %d, want 1", accessManage)
	}
}