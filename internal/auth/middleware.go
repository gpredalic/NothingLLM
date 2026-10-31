package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"net/http"

	"nothingllm/internal/db"

	"golang.org/x/time/rate"
)

type contextKey string

const UserKey contextKey = "user"

// GetUser retrieves the authenticated user from request context.
func GetUser(ctx context.Context) *db.User {
	u, _ := ctx.Value(UserKey).(*db.User)
	return u
}

// RequireAuth wraps a handler, rejecting unauthenticated requests.
func RequireAuth(svc *Service, next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		sessionID := getSessionCookie(r)
		user, err := svc.ValidateSession(r.Context(), sessionID)
		if err != nil {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			return
		}
		ctx := context.WithValue(r.Context(), UserKey, user)
		next(w, r.WithContext(ctx))
	}
}

// RequireAdmin wraps a handler, requiring admin role.
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user := GetUser(r.Context())
		if user == nil || user.Role != "admin" {
			writeJSON(w, http.StatusForbidden, map[string]string{"error": "admin required"})
			return
		}
		next(w, r)
	}
}

// RateLimiter provides simple rate limiting.
type RateLimiter struct {
	limiter *rate.Limiter
}

func NewRateLimiter(rps float64, burst int) *RateLimiter {
	return &RateLimiter{limiter: rate.NewLimiter(rate.Limit(rps), burst)}
}

func (rl *RateLimiter) Middleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !rl.limiter.Allow() {
			writeJSON(w, http.StatusTooManyRequests, map[string]string{"error": "rate limit exceeded"})
			return
		}
		next(w, r)
	}
}

func getSessionCookie(r *http.Request) string {
	c, err := r.Cookie("session")
	if err != nil {
		return ""
	}
	return c.Value
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func generateSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}