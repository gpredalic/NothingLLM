-- 004_dynamic.sql

CREATE TABLE models (
    id              TEXT PRIMARY KEY,
    name            TEXT NOT NULL,
    context_window  INTEGER NOT NULL DEFAULT 128000,
    created_at      TIMESTAMPTZ,
    enabled         BOOLEAN NOT NULL DEFAULT true,
    synced_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE search_profiles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,
    top_k       INTEGER NOT NULL DEFAULT 5,
    min_score   REAL DEFAULT 0.0,
    timeout_ms  INTEGER NOT NULL DEFAULT 5000,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE collection_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,
    description TEXT DEFAULT '',
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE group_collections (
    group_id        UUID NOT NULL REFERENCES collection_groups(id) ON DELETE CASCADE,
    collection_name TEXT NOT NULL REFERENCES collections(name) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (group_id, collection_name)
);

CREATE TABLE workspace_settings (
    workspace_id    UUID PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    model_id        TEXT NOT NULL REFERENCES models(id),
    temperature     REAL NOT NULL DEFAULT 0.7,
    max_results     INTEGER NOT NULL DEFAULT 5,
    max_context     INTEGER NOT NULL DEFAULT 4096,
    search_strategy TEXT NOT NULL DEFAULT 'workspace'
                    CHECK (search_strategy IN ('workspace','selected','group')),
    search_profile_id UUID REFERENCES search_profiles(id) ON DELETE SET NULL,
    default_group_id  UUID REFERENCES collection_groups(id) ON DELETE SET NULL,
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE collection_metadata (
    collection_name TEXT NOT NULL REFERENCES collections(name) ON DELETE CASCADE,
    key             TEXT NOT NULL,
    data_type       TEXT NOT NULL DEFAULT 'keyword',
    sample_values   TEXT[] DEFAULT '{}',
    PRIMARY KEY (collection_name, key)
);