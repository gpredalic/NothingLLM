package db

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetSetting returns the value for a settings key.
// Returns ErrNotFound if the key does not exist.
func (d *DB) GetSetting(ctx context.Context, key string) (string, error) {
	var value string
	err := d.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key=$1`, key).Scan(&value)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return value, nil
}

// SetSetting inserts or updates a settings key-value pair.
func (d *DB) SetSetting(ctx context.Context, key, value string) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO settings(key, value, updated_at) VALUES($1, $2, now())
		ON CONFLICT(key) DO UPDATE SET value=$2, updated_at=now()`,
		key, value)
	return err
}

// IsSetupComplete checks if the setup wizard has been completed.
func (d *DB) IsSetupComplete(ctx context.Context) (bool, error) {
	v, err := d.GetSetting(ctx, "setup_complete")
	if errors.Is(err, ErrNotFound) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return v == "true", nil
}

// MarkSetupComplete marks the installation as fully configured.
func (d *DB) MarkSetupComplete(ctx context.Context) error {
	return d.SetSetting(ctx, "setup_complete", "true")
}

// LoadConfigFromDB reads application settings from the database.
// Returns a map of key-value pairs for the known config keys.
func (d *DB) LoadConfigFromDB(ctx context.Context) (map[string]string, error) {
	keys := []string{"openai_api_key", "qdrant_url", "qdrant_api_key", "session_secret"}
	out := make(map[string]string, len(keys))
	for _, k := range keys {
		v, err := d.GetSetting(ctx, k)
		if err != nil && !errors.Is(err, ErrNotFound) {
			return nil, fmt.Errorf("db: load config %q: %w", k, err)
		}
		out[k] = v
	}
	return out, nil
}

// CountAdmins returns the number of admin users.
func (d *DB) CountAdmins(ctx context.Context) (int, error) {
	var count int
	err := d.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users WHERE role='admin'`).Scan(&count)
	return count, err
}