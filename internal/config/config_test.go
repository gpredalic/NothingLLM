package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadValid(t *testing.T) {
	yaml := `
server:
  addr: ":9090"
database:
  dsn: "postgres://test:test@localhost:5432/test"
qdrant:
  url: "http://localhost:6333"
openai:
  api_key: "sk-test"
  embedding_model: "text-embedding-3-small"
auth:
  session_ttl: "24h"
  bcrypt_cost: 12
rate_limit:
  requests_per_minute: 100
  login_per_minute: 5
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Addr != ":9090" {
		t.Errorf("addr = %q, want :9090", cfg.Server.Addr)
	}
	if cfg.Database.DSN == "" {
		t.Error("dsn should not be empty")
	}
	if cfg.Auth.BcryptCost != 12 {
		t.Errorf("bcrypt_cost = %d, want 12", cfg.Auth.BcryptCost)
	}
}

func TestLoadMissingRequired(t *testing.T) {
	yaml := `
database:
  dsn: ""
qdrant:
  url: "http://localhost:6333"
openai:
  api_key: "sk-test"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := Load(path)
	if err == nil {
		t.Error("expected error for missing DSN")
	}
}

func TestDefaults(t *testing.T) {
	yaml := `
database:
  dsn: "postgres://test"
qdrant:
  url: "http://localhost:6333"
openai:
  api_key: "sk-test"
`
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Addr != ":8080" {
		t.Errorf("default addr = %q, want :8080", cfg.Server.Addr)
	}
	if cfg.Qdrant.SyncInterval.Seconds() != 60 {
		t.Errorf("default sync_interval = %v, want 60s", cfg.Qdrant.SyncInterval)
	}
}