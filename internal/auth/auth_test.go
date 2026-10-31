package auth

import (
	"testing"
)

func TestGenerateSessionID(t *testing.T) {
	id, err := generateSessionID()
	if err != nil {
		t.Fatal(err)
	}
	if len(id) != 64 {
		t.Errorf("session ID length = %d, want 64", len(id))
	}
	// Two IDs should be different
	id2, _ := generateSessionID()
	if id == id2 {
		t.Error("two session IDs should be different")
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, 2)
	// Should allow 2 requests (burst)
	if !rl.limiter.Allow() {
		t.Error("first request should be allowed")
	}
	if !rl.limiter.Allow() {
		t.Error("second request should be allowed")
	}
	// Third should be denied (no time to refill)
	if rl.limiter.Allow() {
		t.Error("third request should be denied")
	}
}