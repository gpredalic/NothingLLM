package db

import (
	"context"
	"time"
)

type Collection struct {
	Name          string    `json:"name"`
	VectorCount   int64     `json:"vector_count"`
	Dimensions    int       `json:"dimensions"`
	Status        string    `json:"status"`
	DiscoveredAt  time.Time `json:"discovered_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

func (d *DB) UpsertCollection(ctx context.Context, name string, vectorCount int64, dimensions int, status string) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO collections(name, vector_count, dimensions, status)
		VALUES($1,$2,$3,$4)
		ON CONFLICT(name) DO UPDATE SET
			vector_count = EXCLUDED.vector_count,
			dimensions = EXCLUDED.dimensions,
			status = EXCLUDED.status,
			updated_at = now()`,
		name, vectorCount, dimensions, status)
	return err
}

func (d *DB) GetCollection(ctx context.Context, name string) (*Collection, error) {
	c := &Collection{}
	err := d.pool.QueryRow(ctx, `
		SELECT name, vector_count, dimensions, status, discovered_at, updated_at
		FROM collections WHERE name=$1`, name,
	).Scan(&c.Name, &c.VectorCount, &c.Dimensions, &c.Status, &c.DiscoveredAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (d *DB) ListCollections(ctx context.Context) ([]Collection, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT name, vector_count, dimensions, status, discovered_at, updated_at
		FROM collections ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []Collection
	for rows.Next() {
		var c Collection
		if err := rows.Scan(&c.Name, &c.VectorCount, &c.Dimensions, &c.Status, &c.DiscoveredAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func (d *DB) MarkMissing(ctx context.Context, names []string) error {
	if len(names) == 0 {
		return nil
	}
	_, err := d.pool.Exec(ctx, `
		UPDATE collections SET status='missing', updated_at=now()
		WHERE name = ANY($1) AND status != 'missing'`, names)
	return err
}

func (d *DB) AssignCollection(ctx context.Context, workspaceID, collectionName string) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO workspace_collections(workspace_id, collection_name) VALUES($1,$2)`,
		workspaceID, collectionName)
	return err
}

func (d *DB) UnassignCollection(ctx context.Context, workspaceID, collectionName string) error {
	_, err := d.pool.Exec(ctx, `
		DELETE FROM workspace_collections WHERE workspace_id=$1 AND collection_name=$2`,
		workspaceID, collectionName)
	return err
}

func (d *DB) ListWorkspaceCollections(ctx context.Context, workspaceID string) ([]string, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT collection_name FROM workspace_collections
		WHERE workspace_id=$1
		ORDER BY collection_name`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var names []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		names = append(names, name)
	}
	return names, rows.Err()
}