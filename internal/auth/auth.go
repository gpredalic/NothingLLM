package auth

import (
	"context"
	"errors"
	"fmt"
	"time"

	"nothingllm/internal/db"

	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrSessionExpired     = errors.New("session expired")
)

type Service struct {
	db         *db.DB
	sessionTTL time.Duration
	bcryptCost int
}

func NewService(database *db.DB, sessionTTL time.Duration, bcryptCost int) *Service {
	return &Service{db: database, sessionTTL: sessionTTL, bcryptCost: bcryptCost}
}

// Register creates a new user and returns their credentials.
func (s *Service) Register(ctx context.Context, email, password string) (*db.User, error) {
	if email == "" || password == "" {
		return nil, errors.New("email and password are required")
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), s.bcryptCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}
	role := "user"
	// First user becomes admin
	var count int
	if err := s.db.Pool().QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return nil, err
	}
	if count == 0 {
		role = "admin"
	}
	return s.db.CreateUser(ctx, email, string(hash), role)
}

// Login validates credentials and creates a session.
func (s *Service) Login(ctx context.Context, email, password string) (string, error) {
	user, err := s.db.GetUserByEmail(ctx, email)
	if errors.Is(err, db.ErrNotFound) {
		return "", ErrInvalidCredentials
	}
	if err != nil {
		return "", err
	}
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(password)); err != nil {
		return "", ErrInvalidCredentials
	}
	sessionID, err := generateSessionID()
	if err != nil {
		return "", err
	}
	if err := s.db.CreateSession(ctx, sessionID, user.ID, s.sessionTTL); err != nil {
		return "", err
	}
	return sessionID, nil
}

// Logout destroys the session.
func (s *Service) Logout(ctx context.Context, sessionID string) error {
	return s.db.DeleteSession(ctx, sessionID)
}

// ValidateSession checks if a session is valid and returns the user.
func (s *Service) ValidateSession(ctx context.Context, sessionID string) (*db.User, error) {
	if sessionID == "" {
		return nil, ErrSessionExpired
	}
	session, err := s.db.GetSession(ctx, sessionID)
	if errors.Is(err, db.ErrNotFound) {
		return nil, ErrSessionExpired
	}
	if err != nil {
		return nil, err
	}
	if time.Now().After(session.ExpiresAt) {
		_ = s.db.DeleteSession(ctx, sessionID)
		return nil, ErrSessionExpired
	}
	return s.db.GetUserByID(ctx, session.UserID)
}