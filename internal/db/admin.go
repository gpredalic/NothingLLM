package db

import (
	"context"
	"fmt"
)

// Ping checks database connectivity.
func (d *DB) Ping(ctx context.Context) error {
	return d.pool.Ping(ctx)
}

// AdminStats holds aggregate counts for the admin dashboard.
type AdminStats struct {
	UserCount       int `json:"user_count"`
	WorkspaceCount  int `json:"workspace_count"`
	CollectionCount int `json:"collection_count"`
	VectorCount     int `json:"vector_count"`
	ActiveSessions  int `json:"active_sessions"`
}

// GetAdminStats returns aggregate counts for the admin dashboard.
func (d *DB) GetAdminStats(ctx context.Context) (*AdminStats, error) {
	stats := &AdminStats{}

	if err := d.pool.QueryRow(ctx, `SELECT COUNT(*) FROM users`).Scan(&stats.UserCount); err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	if err := d.pool.QueryRow(ctx, `SELECT COUNT(*) FROM workspaces`).Scan(&stats.WorkspaceCount); err != nil {
		return nil, fmt.Errorf("count workspaces: %w", err)
	}
	if err := d.pool.QueryRow(ctx, `SELECT COUNT(*) FROM collections`).Scan(&stats.CollectionCount); err != nil {
		return nil, fmt.Errorf("count collections: %w", err)
	}
	if err := d.pool.QueryRow(ctx, `SELECT COALESCE(SUM(vector_count), 0) FROM collections`).Scan(&stats.VectorCount); err != nil {
		return nil, fmt.Errorf("count vectors: %w", err)
	}
	if err := d.pool.QueryRow(ctx, `SELECT COUNT(*) FROM sessions`).Scan(&stats.ActiveSessions); err != nil {
		return nil, fmt.Errorf("count sessions: %w", err)
	}
	return stats, nil
}
