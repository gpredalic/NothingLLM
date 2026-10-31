package db

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
)

type Conversation struct {
	ID          string    `json:"id"`
	WorkspaceID string    `json:"workspace_id"`
	UserID      string    `json:"user_id"`
	Title       string    `json:"title"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type Message struct {
	ID           string    `json:"id"`
	ConversationID string   `json:"conversation_id"`
	Role         string    `json:"role"`
	Content      string    `json:"content"`
	CreatedAt    time.Time `json:"created_at"`
}

func (d *DB) CreateConversation(ctx context.Context, workspaceID, userID, title string) (*Conversation, error) {
	c := &Conversation{}
	err := d.pool.QueryRow(ctx, `
		INSERT INTO conversations(workspace_id, user_id, title) VALUES($1,$2,$3)
		RETURNING id, workspace_id, user_id, title, created_at, updated_at`,
		workspaceID, userID, title,
	).Scan(&c.ID, &c.WorkspaceID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (d *DB) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	c := &Conversation{}
	err := d.pool.QueryRow(ctx, `
		SELECT id, workspace_id, user_id, title, created_at, updated_at
		FROM conversations WHERE id=$1`, id,
	).Scan(&c.ID, &c.WorkspaceID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return c, nil
}

func (d *DB) ListConversations(ctx context.Context, userID string) ([]Conversation, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, workspace_id, user_id, title, created_at, updated_at
		FROM conversations WHERE user_id=$1
		ORDER BY updated_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var convs []Conversation
	for rows.Next() {
		var c Conversation
		if err := rows.Scan(&c.ID, &c.WorkspaceID, &c.UserID, &c.Title, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		convs = append(convs, c)
	}
	return convs, rows.Err()
}

func (d *DB) DeleteConversation(ctx context.Context, id string) error {
	tag, err := d.pool.Exec(ctx, `DELETE FROM conversations WHERE id=$1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (d *DB) AddMessage(ctx context.Context, conversationID, role, content string) (*Message, error) {
	m := &Message{}
	err := d.pool.QueryRow(ctx, `
		INSERT INTO messages(conversation_id, role, content) VALUES($1,$2,$3)
		RETURNING id, conversation_id, role, content, created_at`,
		conversationID, role, content,
	).Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt)
	if err != nil {
		return nil, err
	}
	_, _ = d.pool.Exec(ctx, `UPDATE conversations SET updated_at=now() WHERE id=$1`, conversationID)
	return m, nil
}

func (d *DB) ListMessages(ctx context.Context, conversationID string) ([]Message, error) {
	rows, err := d.pool.Query(ctx, `
		SELECT id, conversation_id, role, content, created_at
		FROM messages WHERE conversation_id=$1
		ORDER BY created_at ASC`, conversationID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs []Message
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.ConversationID, &m.Role, &m.Content, &m.CreatedAt); err != nil {
			return nil, err
		}
		msgs = append(msgs, m)
	}
	return msgs, rows.Err()
}