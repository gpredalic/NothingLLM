package db

import (
	"testing"
)

func TestMigrationsEmbedded(t *testing.T) {
	entries, err := migrationsFS.ReadDir("migrations")
	if err != nil {
		t.Fatal("failed to read embedded migrations:", err)
	}
	if len(entries) != 4 {
		t.Errorf("expected 4 migration files, got %d", len(entries))
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}
	expected := []string{"001_init.sql", "002_workspaces.sql", "003_content.sql", "004_dynamic.sql"}
	for i, name := range expected {
		if names[i] != name {
			t.Errorf("migration[%d] = %q, want %q", i, names[i], name)
		}
	}
}

func TestMigrationsContent(t *testing.T) {
	content, err := migrationsFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		t.Fatal(err)
	}
	if len(content) == 0 {
		t.Error("001_init.sql should not be empty")
	}
}