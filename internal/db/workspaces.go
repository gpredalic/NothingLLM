package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Workspace struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	CreatedBy   string    `json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

func (d *DB) CreateWorkspace(ctx context.Context, name, description, createdBy string) (*Workspace, error) {
	w := &Workspace{}
	err := d.pool.QueryRow(ctx, `
		INSERT INTO workspaces(name, description, created_by) VALUES($1,$2,$3)
		RETURNING id, name, description, created_by, created_at`,
		name, description, createdBy,
	).Scan(&w.ID, &w.Name, &w.Description, &w.CreatedBy, &w.CreatedAt)
	if err != nil {
		return nil, err
	}
	// Creator is automatically the owner.
	_, err = d.pool.Exec(ctx, `
		INSERT INTO workspace_members(workspace_id, user_id, role) VALUES($1,$2,'owner')`,
		w.ID, createdBy)
	return w, err
}

func (d *DB) GetWorkspace(ctx context.Context, id string) (*Workspace, error) {
	w := &Workspace{}
	err := d.pool.QueryRow(ctx, `
		SELECT id, name, description, created_by, created_at FROM workspaces WHERE id=$1`, id,
	).Scan(&w.ID, &w.Name, &w.Description, &w.CreatedBy, &w.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (d *DB) ListWorkspaces(ctx context.Context, userID string) ([]Workspace, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT w.id, w.name, w.description, w.created_by, w.created_at
		FROM workspaces w
		JOIN workspace_members wm ON wm.workspace_id = w.id
		WHERE wm.user_id = $1
		ORDER BY w.created_at`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ws []Workspace
	for rows.Next() {
		var w Workspace
		if err := rows.Scan(&w.ID, &w.Name, &w.Description, &w.CreatedBy, &w.CreatedAt); err != nil {
			return nil, err
		}
		ws = append(ws, w)
	}
	return ws, rows.Err()
}

func (d *DB) DeleteWorkspace(ctx context.Context, id string) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM workspaces WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) AddWorkspaceMember(ctx context.Context, workspaceID, userID string, role string) error {
	_, err := d.pool.Exec(ctx, `
		INSERT INTO workspace_members(workspace_id, user_id, role) VALUES($1,$2,$3)`,
		workspaceID, userID, role)
	return err
}

func (d *DB) RemoveWorkspaceMember(ctx context.Context, workspaceID, userID string) error {
	_, err := d.pool.Exec(ctx, `
		DELETE FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`,
		workspaceID, userID)
	return err
}

// GetWorkspaceMemberRole returns the role of a user in a workspace.
// Returns ErrNotFound if the user is not a member.
func (d *DB) GetWorkspaceMemberRole(ctx context.Context, workspaceID, userID string) (string, error) {
	var role string
	err := d.pool.QueryRow(ctx, `
		SELECT role FROM workspace_members WHERE workspace_id=$1 AND user_id=$2`,
		workspaceID, userID,
	).Scan(&role)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", err
	}
	return role, nil
}

func (d *DB) ListWorkspaceMembers(ctx context.Context, workspaceID string) ([]map[string]string, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT u.id, u.email, wm.role FROM workspace_members wm
		JOIN users u ON u.id = wm.user_id
		WHERE wm.workspace_id = $1
		ORDER BY wm.created_at`, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var members []map[string]string
	for rows.Next() {
		var id, email, role string
		if err := rows.Scan(&id, &email, &role); err != nil {
			return nil, err
		}
		members = append(members, map[string]string{"id": id, "email": email, "role": role})
	}
	return members, rows.Err()
}