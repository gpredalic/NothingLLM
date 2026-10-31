package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Event represents an auditable action.
type Event struct {
	UserID     string    `json:"user_id"`
	Action     string    `json:"action"`
	EntityType string    `json:"entity_type"`
	EntityID   string    `json:"entity_id"`
	Metadata   map[string]any `json:"metadata"`
}

// Logger writes audit events asynchronously to the database.
// It never blocks the calling goroutine; events are sent to a
// bounded channel and drained by a background worker.
type Logger struct {
	ch   chan Event
	pool *pgxpool.Pool
}

// New creates an audit Logger with a bounded channel.
// The worker goroutine must be started with Run.
func New(pool *pgxpool.Pool) *Logger {
	return &Logger{
		ch:   make(chan Event, 256),
		pool: pool,
	}
}

// Log enqueues an event. Non-blocking: drops if the channel is full.
func (l *Logger) Log(e Event) {
	select {
	case l.ch <- e:
	default:
		slog.Warn("audit: channel full, dropping event", "action", e.Action)
	}
}

// Run starts the background worker. Call with go l.Run(ctx).
func (l *Logger) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case e := <-l.ch:
			l.write(ctx, e)
		}
	}
}

func (l *Logger) write(ctx context.Context, e Event) {
	// Use a short timeout so a slow DB doesn't back up the channel
	writeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	meta := "{}"
	if e.Metadata != nil {
		if b, err := json.Marshal(e.Metadata); err == nil {
			meta = string(b)
		}
	}

	var userID any
	if e.UserID != "" {
		userID = e.UserID
	}
	var entityID any
	if e.EntityID != "" {
		entityID = e.EntityID
	}

	_, err := l.pool.Exec(writeCtx, `
		INSERT INTO audit_log (user_id, action, entity_type, entity_id, metadata)
		VALUES ($1, $2, $3, $4, $5)`,
		userID, e.Action, e.EntityType, entityID, meta)
	if err != nil {
		slog.Error("audit: write failed", "action", e.Action, "err", err)
	}
}