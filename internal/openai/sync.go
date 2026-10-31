package openai

import (
	"context"
	"log/slog"
	"time"

	"nothingllm/internal/db"
)

// ModelSyncer periodically syncs the model registry from OpenAI.
type ModelSyncer struct {
	client   *Client
	database *db.DB
	logger   *slog.Logger
}

func NewModelSyncer(client *Client, database *db.DB) *ModelSyncer {
	return &ModelSyncer{
		client:   client,
		database: database,
		logger:   slog.Default(),
	}
}

// Sync fetches models from OpenAI and upserts them into the database.
func (m *ModelSyncer) Sync(ctx context.Context) error {
	models, err := m.client.ListModels(ctx)
	if err != nil {
		return err
	}
	for _, model := range models {
		var createdAt *time.Time
		if model.Created > 0 {
			t := time.Unix(model.Created, 0)
			createdAt = &t
		}
		cw := model.ContextWindow
		if cw == 0 {
			cw = 128000 // default
		}
		if err := m.database.UpsertModel(ctx, model.ID, model.ID, cw, createdAt); err != nil {
			m.logger.Error("model sync: upsert failed", "id", model.ID, "err", err)
		}
	}
	m.logger.Info("model sync complete", "count", len(models))
	return nil
}