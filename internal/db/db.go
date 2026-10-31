package db

import (
	"context"
	"embed"
	"fmt"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// DB wraps the connection pool.
type DB struct {
	pool *pgxpool.Pool
}

// Open creates a connection pool and runs migrations.
func Open(ctx context.Context, dsn string) (*DB, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, fmt.Errorf("db: connect: %w", err)
	}
	db := &DB{pool: pool}
	if err := db.migrate(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("db: migrate: %w", err)
	}
	return db, nil
}

// Pool exposes the underlying pool for direct queries.
func (d *DB) Pool() *pgxpool.Pool {
	return d.pool
}

// Close closes the connection pool.
func (d *DB) Close() {
	d.pool.Close()
}

// migrate applies pending SQL migrations in order.
func (d *DB) migrate(ctx context.Context) error {
	_, err := d.pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS _migrations (
			filename TEXT PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT now()
		)`)
	if err != nil {
		return err
	}

	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		return err
	}
	var files []string
	for _, e := range entries {
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, name := range files {
		var exists bool
		err := d.pool.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM _migrations WHERE filename=$1)`, name).Scan(&exists)
		if err != nil {
			return err
		}
		if exists {
			continue
		}

		content, err := migrationsFS.ReadFile("migrations/" + name)
		if err != nil {
			return err
		}
		_, err = d.pool.Exec(ctx, string(content))
		if err != nil {
			return fmt.Errorf("apply %s: %w", name, err)
		}
		_, err = d.pool.Exec(ctx, `INSERT INTO _migrations(filename) VALUES($1)`, name)
		if err != nil {
			return err
		}
	}
	return nil
}