package qdrant

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"nothingllm/internal/db"
)

type SyncEngine struct {
	client *Client
	database *db.DB
	interval time.Duration
	logger *slog.Logger
}

func NewSyncEngine(client *Client, database *db.DB, interval time.Duration) *SyncEngine {
	return &SyncEngine{
		client:   client,
		database: database,
		interval: interval,
		logger:   slog.Default(),
	}
}

// Run starts the sync loop. It blocks until ctx is cancelled.
func (s *SyncEngine) Run(ctx context.Context) {
	// Initial sync
	s.syncOnce(ctx)

	ticker := time.NewTicker(s.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			s.syncOnce(ctx)
		}
	}
}

// SyncOnce performs a single sync cycle.
func (s *SyncEngine) SyncOnce(ctx context.Context) {
	s.syncOnce(ctx)
}

func (s *SyncEngine) syncOnce(ctx context.Context) {
	collections, err := s.client.ListCollections(ctx)
	if err != nil {
		s.logger.Error("qdrant sync: list collections failed", "err", err)
		return
	}

	seen := make(map[string]bool, len(collections))
	for _, col := range collections {
		seen[col.Name] = true

		status := "online"
		if col.Status != "green" {
			status = "offline"
		}

		if err := s.database.UpsertCollection(ctx, col.Name, col.Vectors, col.Dimensions, status); err != nil {
			s.logger.Error("qdrant sync: upsert collection", "name", col.Name, "err", err)
			continue
		}

		// Sync metadata (payload schema)
		s.syncMetadata(ctx, col)
	}

	// Mark missing collections
	var missing []string
	for _, c := range s.allCollections(ctx) {
		if !seen[c] && c != "" {
			missing = append(missing, c)
		}
	}
	if len(missing) > 0 {
		if err := s.database.MarkMissing(ctx, missing); err != nil {
			s.logger.Error("qdrant sync: mark missing", "err", err)
		}
	}
}

func (s *SyncEngine) syncMetadata(ctx context.Context, col CollectionInfo) {
	if len(col.PayloadSchema) == 0 {
		return
	}
	var keys []db.CollectionMetadata
	for key, field := range col.PayloadSchema {
		var samples []string
		for _, sv := range field.SampleValues {
			samples = append(samples, toString(sv))
		}
		keys = append(keys, db.CollectionMetadata{
			CollectionName: col.Name,
			Key:            key,
			DataType:       field.DataType,
			SampleValues:   samples,
		})
	}
	if err := s.database.UpsertCollectionMetadata(ctx, col.Name, keys); err != nil {
		s.logger.Error("qdrant sync: upsert metadata", "collection", col.Name, "err", err)
	}
}

func (s *SyncEngine) allCollections(ctx context.Context) []string {
	cols, err := s.database.ListCollections(ctx)
	if err != nil {
		return nil
	}
	names := make([]string, len(cols))
	for i, c := range cols {
		names[i] = c.Name
	}
	return names
}

func toString(v any) string {
	switch val := v.(type) {
	case string:
		return val
	case float64:
		return fmt.Sprintf("%g", val)
	case bool:
		if val {
			return "true"
		}
		return "false"
	default:
		return ""
	}
}