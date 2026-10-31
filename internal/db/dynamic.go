package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Model struct {
	ID            string     `json:"id"`
	Name          string     `json:"name"`
	ContextWindow int        `json:"context_window"`
	CreatedAt     *time.Time `json:"created_at,omitempty"`
	Enabled       bool       `json:"enabled"`
	SyncedAt      time.Time  `json:"synced_at"`
}

func (d *DB) UpsertModel(ctx context.Context, id, name string, contextWindow int, createdAt *time.Time) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO models(id, name, context_window, created_at)
		VALUES($1,$2,$3,$4)
		ON CONFLICT(id) DO UPDATE SET
			name = EXCLUDED.name,
			context_window = EXCLUDED.context_window,
			synced_at = now()`,
		id, name, contextWindow, createdAt)
	return err
}

func (d *DB) ListModels(ctx context.Context) ([]Model, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, name, context_window, created_at, enabled, synced_at
		FROM models ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var models []Model
	for rows.Next() {
		var m Model
		if err := rows.Scan(&m.ID, &m.Name, &m.ContextWindow, &m.CreatedAt, &m.Enabled, &m.SyncedAt); err != nil {
			return nil, err
		}
		models = append(models, m)
	}
	return models, rows.Err()
}

func (d *DB) SetModelEnabled(ctx context.Context, id string, enabled bool) error {
	_, err := d.pool.Exec(ctx, `UPDATE models SET enabled=$1 WHERE id=$2`, enabled, id)
	return err
}

type SearchProfile struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	TopK      int       `json:"top_k"`
	MinScore  float64   `json:"min_score"`
	TimeoutMs int       `json:"timeout_ms"`
	CreatedBy *string   `json:"created_by,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

func (d *DB) CreateSearchProfile(ctx context.Context, name string, topK int, minScore float64, timeoutMs int, createdBy *string) (*SearchProfile, error) {
	s := &SearchProfile{}
	err := d.pool.QueryRow(ctx, `
		INSERT INTO search_profiles(name, top_k, min_score, timeout_ms, created_by)
		VALUES($1,$2,$3,$4,$5)
		RETURNING id, name, top_k, min_score, timeout_ms, created_by, created_at`,
		name, topK, minScore, timeoutMs, createdBy,
	).Scan(&s.ID, &s.Name, &s.TopK, &s.MinScore, &s.TimeoutMs, &s.CreatedBy, &s.CreatedAt)
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (d *DB) ListSearchProfiles(ctx context.Context) ([]SearchProfile, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, name, top_k, min_score, timeout_ms, created_by, created_at
		FROM search_profiles ORDER BY created_at`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var profiles []SearchProfile
	for rows.Next() {
		var s SearchProfile
		if err := rows.Scan(&s.ID, &s.Name, &s.TopK, &s.MinScore, &s.TimeoutMs, &s.CreatedBy, &s.CreatedAt); err != nil {
			return nil, err
		}
		profiles = append(profiles, s)
	}
	return profiles, rows.Err()
}

func (d *DB) DeleteSearchProfile(ctx context.Context, id string) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM search_profiles WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

type CollectionGroup struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   *string   `json:"created_by,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

func (d *DB) CreateCollectionGroup(ctx context.Context, name, description string, createdBy *string) (*CollectionGroup, error) {
	g := &CollectionGroup{}
	err := d.pool.QueryRow(ctx, `
		INSERT INTO collection_groups(name, description, created_by) VALUES($1,$2,$3)
		RETURNING id, name, description, created_by, created_at`,
		name, description, createdBy,
	).Scan(&g.ID, &g.Name, &g.Description, &g.CreatedBy, &g.CreatedAt)
	if err != nil {
		return nil, err
	}
	return g, nil
}

func (d *DB) ListCollectionGroups(ctx context.Context) ([]CollectionGroup, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, name, description, created_by, created_at
		FROM collection_groups ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var groups []CollectionGroup
	for rows.Next() {
		var g CollectionGroup
		if err := rows.Scan(&g.ID, &g.Name, &g.Description, &g.CreatedBy, &g.CreatedAt); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	return groups, rows.Err()
}

func (d *DB) DeleteCollectionGroup(ctx context.Context, id string) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM collection_groups WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) AddGroupCollection(ctx context.Context, groupID, collectionName string) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO group_collections(group_id, collection_name) VALUES($1,$2)`,
		groupID, collectionName)
	return err
}

func (d *DB) RemoveGroupCollection(ctx context.Context, groupID, collectionName string) error {
	_, err := d.pool.Exec(ctx, `
		DELETE FROM group_collections WHERE group_id=$1 AND collection_name=$2`,
		groupID, collectionName)
	return err
}

func (d *DB) ListGroupCollections(ctx context.Context, groupID string) ([]string, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT collection_name FROM group_collections WHERE group_id=$1 ORDER BY collection_name`, groupID)
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

type WorkspaceSettings struct {
	WorkspaceID     string  `json:"workspace_id"`
	ModelID         string  `json:"model_id"`
	Temperature     float64 `json:"temperature"`
	MaxResults      int     `json:"max_results"`
	MaxContext      int     `json:"max_context"`
	SearchStrategy  string  `json:"search_strategy"`
	SearchProfileID *string `json:"search_profile_id,omitempty"`
	DefaultGroupID  *string `json:"default_group_id,omitempty"`
}

func (d *DB) GetWorkspaceSettings(ctx context.Context, workspaceID string) (*WorkspaceSettings, error) {
	s := &WorkspaceSettings{}
	err := d.pool.QueryRow(ctx, `
		SELECT workspace_id, model_id, temperature, max_results, max_context,
		       search_strategy, search_profile_id, default_group_id
		FROM workspace_settings WHERE workspace_id=$1`, workspaceID,
	).Scan(&s.WorkspaceID, &s.ModelID, &s.Temperature, &s.MaxResults, &s.MaxContext,
		&s.SearchStrategy, &s.SearchProfileID, &s.DefaultGroupID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return s, nil
}

func (d *DB) UpsertWorkspaceSettings(ctx context.Context, s *WorkspaceSettings) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO workspace_settings(workspace_id, model_id, temperature, max_results, max_context,
			search_strategy, search_profile_id, default_group_id)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8)
		ON CONFLICT(workspace_id) DO UPDATE SET
			model_id = EXCLUDED.model_id,
			temperature = EXCLUDED.temperature,
			max_results = EXCLUDED.max_results,
			max_context = EXCLUDED.max_context,
			search_strategy = EXCLUDED.search_strategy,
			search_profile_id = EXCLUDED.search_profile_id,
			default_group_id = EXCLUDED.default_group_id,
			updated_at = now()`,
		s.WorkspaceID, s.ModelID, s.Temperature, s.MaxResults, s.MaxContext,
		s.SearchStrategy, s.SearchProfileID, s.DefaultGroupID)
	return err
}

type CollectionMetadata struct {
	CollectionName string   `json:"collection_name"`
	Key            string   `json:"key"`
	DataType       string   `json:"data_type"`
	SampleValues   []string `json:"sample_values"`
}

func (d *DB) UpsertCollectionMetadata(ctx context.Context, collectionName string, keys []CollectionMetadata) error {
	for _, k := range keys {
		_, err := d.pool.Exec(ctx, `
			INSERT INTO collection_metadata(collection_name, key, data_type, sample_values)
			VALUES($1,$2,$3,$4)
			ON CONFLICT(collection_name, key) DO UPDATE SET
				data_type = EXCLUDED.data_type,
				sample_values = EXCLUDED.sample_values`,
			collectionName, k.Key, k.DataType, k.SampleValues)
		if err != nil {
			return err
		}
	}
	return nil
}

func (d *DB) ListCollectionMetadata(ctx context.Context, collectionName string) ([]CollectionMetadata, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT collection_name, key, data_type, sample_values
		FROM collection_metadata WHERE collection_name=$1
		ORDER BY key`, collectionName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var metas []CollectionMetadata
	for rows.Next() {
		var m CollectionMetadata
		if err := rows.Scan(&m.CollectionName, &m.Key, &m.DataType, &m.SampleValues); err != nil {
			return nil, err
		}
		metas = append(metas, m)
	}
	return metas, rows.Err()
}

type AuditEntry struct {
	UserID     *string `json:"user_id,omitempty"`
	Action     string  `json:"action"`
	EntityType string  `json:"entity_type"`
	EntityID   *string `json:"entity_id,omitempty"`
	Metadata   []byte  `json:"metadata"`
}

func (d *DB) WriteAudit(ctx context.Context, e *AuditEntry) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO audit_log(user_id, action, entity_type, entity_id, metadata)
		VALUES($1,$2,$3,$4,$5)`,
		e.UserID, e.Action, e.EntityType, e.EntityID, e.Metadata)
	return err
}