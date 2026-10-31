package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"user_id"`
	ExpiresAt time.Time `json:"expires_at"`
}

func (d *DB) CreateSession(ctx context.Context, id, userID string, ttl time.Duration) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO sessions(id, user_id, expires_at) VALUES($1,$2,now()+$3)`,
		id, userID, ttl)
	return err
}

func (d *DB) GetSession(ctx context.Context, id string) (*Session, error) {
	s := &Session{}
	err := d.pool.QueryRow(ctx, `
		SELECT id, user_id, expires_at FROM sessions WHERE id=$1`, id,
	).Scan(&s.ID, &s.UserID, &s.ExpiresAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (d *DB) DeleteSession(ctx context.Context, id string) error {
	_, err := d.pool.Exec(ctx, `DELETE FROM sessions WHERE id=$1`, id)
	return err
}

func (d *DB) DeleteExpiredSessions(ctx context.Context) (int64, error) {
	tag, err := d.pool.Exec(ctx, `DELETE FROM sessions WHERE expires_at < now()`)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}