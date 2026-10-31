# NothingLLM — Architecture

> Self-hosted AI workspace. One Go binary. One Next.js frontend. One Postgres. One Qdrant. One provider.

## Philosophy

- Do one thing well. Understandable in hours, not weeks.
- Minimal dependencies, abstractions, configuration, infrastructure.
- **Dynamic-first**: discover and synchronize resources at runtime. No restart for new collections, models, or metadata.
- Business rules belong to data, not source code.
- Every feature must answer: "Can this be removed?" If yes, remove it.
- Boring technology over trendy technology.
- Simplicity over flexibility. Maintainability over abstraction.

## Stack

| Layer | Technology |
|-------|-----------|
| Backend | Go (single binary) |
| Frontend | Next.js, TypeScript, Tailwind, shadcn/ui |
| Database | PostgreSQL 16 |
| Vector DB | Qdrant |
| AI Provider | OpenAI only |
| Deployment | Docker Compose |

**No Kubernetes. No microservices. No Redis. No Elasticsearch. No event bus.**

---

## 1. System Architecture

```
┌─────────────────────────────────────────────────────────────────────┐
│                        Docker Compose                               │
│                                                                     │
│  ┌─────────────────────────────────────────────────────────────┐   │
│  │  app (Go binary)                                            │   │
│  │                                                             │   │
│  │  ┌──────────┐  ┌──────────┐  ┌──────────┐  ┌───────────┐  │   │
│  │  │  HTTP    │  │   RAG    │  │  Auth    │  │  Sync     │  │   │
│  │  │  Server  │  │  Engine  │  │  Mgmt    │  │  Engine   │  │   │
│  │  │  :8080   │  │          │  │          │  │  (bg)     │  │   │
│  │  └────┬─────┘  └────┬─────┘  └──────────┘  └─────┬─────┘  │   │
│  │       │             │                             │         │   │
│  └───────┼─────────────┼─────────────────────────────┼─────────┘   │
│          │             │                             │              │
│  ┌───────┼─────────────┼─────────────────────────────┼─────────┐   │
│  │       │             │                             │         │   │
│  │  ┌────▼─────┐  ┌────▼─────┐  ┌─────────────┐  ┌───▼────┐  │   │
│  │  │PostgreSQL│  │  OpenAI  │  │   Qdrant    │  │Next.js │  │   │
│  │  │  :5432   │  │  (REST)  │  │   :6333     │  │  :3000 │  │   │
│  │  └──────────┘  └──────────┘  └─────────────┘  └────────┘  │   │
│  └────────────────────────────────────────────────────────────┘   │
└─────────────────────────────────────────────────────────────────────┘
```

**Process model:**
- Go binary: `net/http` server + one background goroutine (Qdrant sync)
- Next.js: SSR + static export, serves the UI
- PostgreSQL: users, sessions, workspaces, conversations, audit log
- Qdrant: vectors, collections, payloads (owned externally)

**RAG pipeline (explicit, no framework):**

```
User question
  → OpenAI embedding (dense vector)
  → Resolve workspace → assigned collections
  → Filter to online collections (in-memory cache)
  → Parallel Qdrant search (max 10 concurrent, 3s timeout each)
  → Merge results, rank top-K
  → Build prompt (system + context + message)
  → OpenAI chat completion (streaming SSE)
  → Response
```

---

## 2. Database Schema

### Tables

| Table | Purpose |
|-------|---------|
| `users` | Local user accounts |
| `sessions` | Server-side session tokens |
| `password_resets` | Password reset tokens |
| `workspaces` | User workspaces |
| `workspace_members` | User ↔ Workspace membership |
| `collections` | Discovered Qdrant collection metadata |
| `workspace_collections` | Workspace ↔ Collection mapping |
| `documents` | Uploaded document metadata |
| `conversations` | Chat conversations |
| `messages` | Individual chat messages |
| `audit_log` | Append-only action log |

**11 tables. No JSONB. No soft deletes. No multi-tenancy columns.**

### ERD

```
┌─────────────┐       ┌──────────────────┐
│    users    │       │    sessions      │
├─────────────┤       ├──────────────────┤
│ id (PK)     │──┐    │ id (PK)          │
│ email (UQ)  │  │    │ user_id (FK)─────┼──┐
│ password    │  │    │ expires_at       │  │
│ role        │  │    │ created_at       │  │
│ created_at  │  │    └──────────────────┘  │
│ updated_at  │  │                          │
└─────────────┘  │                          │
                 │    ┌──────────────────┐  │
                 ├────│ password_resets  │  │
                 │    ├──────────────────┤  │
                 │    │ id (PK)          │  │
                 │    │ user_id (FK)─────┼──┘
                 │    │ token (UQ)       │
                 │    │ expires_at       │
                 │    │ used_at          │
                 │    └──────────────────┘
                 │
                 │    ┌──────────────────┐
                 ├────│   workspaces     │
                 │    ├──────────────────┤
                 │    │ id (PK)          │
                 │    │ owner_id (FK)────┼──→ users
                 │    │ name             │
                 │    │ description      │
                 │    │ created_at       │
                 │    └──────────────────┘
                 │           │
                 │           │  ┌──────────────────────┐
                 │           └──│ workspace_members    │
                 │              ├──────────────────────┤
                 │              │ workspace_id (FK)    │
                 │              │ user_id (FK)─────────┼──→ users
                 │              │ role                 │
                 │              │ created_at           │
                 │              └──────────────────────┘
                 │
                 │    ┌──────────────────┐
                 │    │  collections     │  (metadata mirror of Qdrant)
                 │    ├──────────────────┤
                 │    │ name (PK)        │
                 │    │ vector_count     │
                 │    │ dimensions       │
                 │    │ distance_metric  │
                 │    │ status           │
                 │    │ last_sync        │
                 │    │ first_seen       │
                 │    │ last_seen        │
                 │    └──────────────────┘
                 │           │
                 │           │  ┌────────────────────────┐
                 │           └──│ workspace_collections  │
                 │              ├────────────────────────┤
                 │              │ workspace_id (FK)      │
                 │              │ collection_name (FK)───┼──→ collections
                 │              │ created_at             │
                 │              │ PK (workspace_id,     │
                 │              │     collection_name)  │
                 │              └────────────────────────┘
                 │
                 │    ┌──────────────────┐
                 ├────│  documents       │
                 │    ├──────────────────┤
                 │    │ id (PK)          │
                 │    │ user_id (FK)─────┼──→ users
                 │    │ filename         │
                 │    │ content_type     │
                 │    │ size_bytes       │
                 │    │ collection_name  │──→ Qdrant collection
                 │    │ created_at       │
                 │    └──────────────────┘
                 │
                 │    ┌──────────────────┐
                 ├────│ conversations    │
                 │    ├──────────────────┤
                 │    │ id (PK)          │
                 │    │ user_id (FK)─────┼──→ users
                 │    │ workspace_id (FK)┼──→ workspaces (nullable)
                 │    │ title            │
                 │    │ created_at       │
                 │    │ updated_at       │
                 │    └──────────────────┘
                 │           │
                 │           │  ┌──────────────────┐
                 │           └──│    messages      │
                 │              ├──────────────────┤
                 │              │ id (PK)          │
                 │              │ conversation_id  │──→ conversations
                 │              │ role             │
                 │              │ content          │
                 │              │ created_at       │
                 │              └──────────────────┘
                 │
                 │    ┌──────────────────┐
                 └────│   audit_log      │
                      ├──────────────────┤
                      │ id (PK, serial)  │
                      │ user_id (FK)     │──→ users (nullable)
                      │ action           │
                      │ entity_type      │
                      │ entity_id        │
                      │ metadata (JSON)  │
                      │ created_at       │
                      └──────────────────┘
```

---

## 3. PostgreSQL Schema (DDL)

```sql
-- 001_init.sql
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE users (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email       TEXT NOT NULL UNIQUE,
    password    TEXT NOT NULL,
    role        TEXT NOT NULL DEFAULT 'user' CHECK (role IN ('admin','user')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE sessions (
    id          TEXT PRIMARY KEY,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    expires_at  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_sessions_user ON sessions(user_id);
CREATE INDEX idx_sessions_expires ON sessions(expires_at);

CREATE TABLE password_resets (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token       TEXT NOT NULL UNIQUE,
    expires_at  TIMESTAMPTZ NOT NULL,
    used_at     TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
```

```sql
-- 002_workspaces.sql
CREATE TABLE workspaces (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        TEXT NOT NULL,
    description TEXT DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_workspaces_owner ON workspaces(owner_id);

CREATE TABLE workspace_members (
    workspace_id UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role         TEXT NOT NULL DEFAULT 'member' CHECK (role IN ('owner','member')),
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, user_id)
);
CREATE INDEX idx_wm_user ON workspace_members(user_id);

CREATE TABLE collections (
    name            TEXT PRIMARY KEY,
    vector_count    INTEGER NOT NULL DEFAULT 0,
    dimensions      INTEGER NOT NULL DEFAULT 0,
    distance_metric TEXT NOT NULL DEFAULT 'Cosine',
    status          TEXT NOT NULL DEFAULT 'unknown'
                    CHECK (status IN ('online','offline','syncing','error','missing')),
    last_sync       TIMESTAMPTZ,
    first_seen      TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_seen       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE workspace_collections (
    workspace_id    UUID NOT NULL REFERENCES workspaces(id) ON DELETE CASCADE,
    collection_name TEXT NOT NULL REFERENCES collections(name) ON DELETE CASCADE,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (workspace_id, collection_name)
);
CREATE INDEX idx_wc_collection ON workspace_collections(collection_name);
```

```sql
-- 003_content.sql
CREATE TABLE documents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    filename        TEXT NOT NULL,
    content_type    TEXT NOT NULL,
    size_bytes      INTEGER NOT NULL,
    collection_name TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_documents_user ON documents(user_id);

CREATE TABLE conversations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    workspace_id UUID REFERENCES workspaces(id) ON DELETE SET NULL,
    title       TEXT NOT NULL DEFAULT 'New Chat',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_conversations_user ON conversations(user_id);

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role            TEXT NOT NULL CHECK (role IN ('system','user','assistant')),
    content         TEXT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_messages_conv ON messages(conversation_id);

CREATE TABLE audit_log (
    id          BIGSERIAL PRIMARY KEY,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    action      TEXT NOT NULL,
    entity_type TEXT NOT NULL,
    entity_id   TEXT,
    metadata    JSONB DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_audit_user ON audit_log(user_id);
CREATE INDEX idx_audit_action ON audit_log(action);
CREATE INDEX idx_audit_created ON audit_log(created_at);
```

```sql
-- 004_dynamic.sql

-- OpenAI model registry (synced from API)
CREATE TABLE models (
    id              TEXT PRIMARY KEY,        -- e.g., "gpt-4o"
    name            TEXT NOT NULL,           -- display name
    context_window  INTEGER NOT NULL DEFAULT 128000,
    created_at      TIMESTAMPTZ,             -- from OpenAI API
    enabled         BOOLEAN NOT NULL DEFAULT true,
    synced_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Search profiles (user-configurable)
CREATE TABLE search_profiles (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL,               -- e.g., "Quick Search", "Deep Search"
    top_k       INTEGER NOT NULL DEFAULT 5,
    min_score   REAL DEFAULT 0.0,
    timeout_ms  INTEGER NOT NULL DEFAULT 5000,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Collection groups (logical grouping, owned by NothingLLM)
CREATE TABLE collection_groups (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        TEXT NOT NULL UNIQUE,        -- e.g., "Engineering", "Finance"
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

-- Workspace settings (per-workspace configuration, no code changes)
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

-- Discovered metadata keys per collection (from Qdrant payload samples)
CREATE TABLE collection_metadata (
    collection_name TEXT NOT NULL REFERENCES collections(name) ON DELETE CASCADE,
    key             TEXT NOT NULL,           -- e.g., "department", "source"
    data_type       TEXT NOT NULL DEFAULT 'keyword',
    sample_values   TEXT[] DEFAULT '{}',     -- first N distinct values seen
    PRIMARY KEY (collection_name, key)
);
```

**Invariants:**
- Deleting a workspace cascades mappings. Qdrant collections are untouched.
- Deleting a `workspace_collections` row never calls Qdrant.
- The app never deletes rows from `collections`. Sync marks `status='missing'`.
- `audit_log` is append-only. No UPDATE, no DELETE.

---

## 4. API Design

Base URL: `http://localhost:8080`

### Auth

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/auth/register` | — | Create account `{email, password}` |
| POST | `/api/auth/login` | — | Login, sets session cookie |
| POST | `/api/auth/logout` | session | Destroy session |
| POST | `/api/auth/reset` | — | Request password reset `{email}` |
| POST | `/api/auth/reset/confirm` | — | Confirm `{token, new_password}` |

### Users (admin)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/users` | admin | List users |
| POST | `/api/users` | admin | Create user |
| PUT | `/api/users/{id}` | admin | Update (role, password) |
| DELETE | `/api/users/{id}` | admin | Delete user |

### Workspaces

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/workspaces` | session | List user's workspaces |
| POST | `/api/workspaces` | session | Create `{name, description}` |
| GET | `/api/workspaces/{id}` | session | Detail + members + collections |
| PUT | `/api/workspaces/{id}` | owner/admin | Update name/description |
| DELETE | `/api/workspaces/{id}` | owner/admin | Delete (Qdrant untouched) |
| POST | `/api/workspaces/{id}/members` | owner/admin | Add member `{user_id}` |
| DELETE | `/api/workspaces/{id}/members/{uid}` | owner/admin | Remove member |

### Collections (admin)

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/collections` | admin | List all + status |
| GET | `/api/collections/{name}` | admin | Detail |
| POST | `/api/collections/sync` | admin | Trigger immediate sync |
| POST | `/api/collections/{name}/assign` | admin | `{workspace_id}` |
| DELETE | `/api/collections/{name}/unassign` | admin | `{workspace_id}` |
| DELETE | `/api/collections/{name}` | admin | Mark missing (`?confirm=yes`) |

### Documents

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/documents` | session | List user's documents |
| POST | `/api/documents` | session | Upload (multipart: `file`, `collection`) |
| DELETE | `/api/documents/{id}` | session | Delete metadata + Qdrant points |

**Upload flow:**
```
POST /api/documents
  → validate type (pdf, txt, md), size (< 10MB)
  → parse to text
  → chunk (fixed window, overlap)
  → embed (OpenAI)
  → Qdrant upsert to target collection
  → INSERT documents row
  → INSERT audit_log
```

### Chat

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/api/conversations` | session | List |
| POST | `/api/conversations` | session | Create `{workspace_id?}` |
| GET | `/api/conversations/{id}` | session | Detail + messages |
| DELETE | `/api/conversations/{id}` | session | Delete |
| POST | `/api/chat` | session | Send message, SSE stream |

**`POST /api/chat`:**
```json
{
    "conversation_id": "uuid",
    "message": "What's the deployment procedure?",
    "collections": ["support_docs"],
    "filters": {
        "department": "engineering",
        "tag": "runbook"
    }
}
```

- `collections` optional — defaults to all workspace-assigned collections.
- `filters` optional — passed to Qdrant as metadata filter.

**Response:** `text/event-stream`
```
data: {"delta": "The deployment"}
data: {"delta": " procedure..."}
data: [DONE]
```

### Search

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| POST | `/api/search` | session | Semantic search |

```json
{
    "query": "onboarding steps",
    "collections": ["hr_docs"],
    "filters": { "department": "hr" },
    "limit": 5
}
```

### System

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET | `/health` | — | Liveness: `{"status":"ok"}` |
| GET | `/ready` | — | Readiness: checks Postgres, OpenAI, Qdrant |
| GET | `/api/audit` | admin | Query audit log (paginated) |

**`GET /ready` response:**
```json
{
    "status": "ready",
    "checks": {
        "postgres": "ok",
        "openai": "ok",
        "qdrant": "ok"
    }
}
```

If any check fails: `{"status": "not_ready", "checks": {"qdrant": "unreachable"}}`

### Conventions

- JSON request/response
- `401` no/invalid session, `403` insufficient role, `429` rate limited
- Session cookie: `nlm_session`, `HttpOnly; Secure; SameSite=Strict; Path=/`
- Rate limit: 100 req/min per IP, 5 req/min for login

---

## 5. Frontend Design

### Stack

- **Next.js 14** (App Router, server components where possible)
- **TypeScript** (strict mode)
- **Tailwind CSS** (utility-first, no custom CSS framework)
- **shadcn/ui** (headless components, copy-paste, no runtime dependency)

### Design Principles

Inspired by: ChatGPT, Linear, Vercel, Notion.

- Clean, professional, content-first.
- No gradients. No animations. No dashboard clutter.
- Minimal icons (Lucide, one per action max). Text labels preferred.
- Flat colors, subtle borders, generous whitespace.
- No Material Design. No enterprise bloat.

### Theme

Three modes: `light`, `dark`, `system`.

- `next-themes` library (1KB, no CSS-in-JS).
- CSS variables via Tailwind config.
- Persisted in `localStorage` as `nlm_theme`.

```ts
// tailwind.config.ts
theme: {
  extend: {
    colors: {
      background: 'hsl(var(--background))',
      foreground: 'hsl(var(--foreground))',
      muted: 'hsl(var(--muted))',
      accent: 'hsl(var(--accent))',
      border: 'hsl(var(--border))',
    }
  }
}
```

### Layout

```
┌─────────────────────────────────────────────────────────────┐
│  Sidebar (256px)    │            Main Content               │
│                     │                                       │
│  NothingLLM         │  ┌─────────────────────────────────┐  │
│  ─────────────      │  │                                 │  │
│  Chats              │  │         Page Content            │  │
│  Workspaces         │  │                                 │  │
│  Collections        │  │                                 │  │
│  Documents          │  │                                 │  │
│  Users (admin)      │  │                                 │  │
│  Settings           │  │                                 │  │
│                     │  │                                 │  │
│  ─────────────      │  │                                 │  │
│  [🌙] [user@x.com]  │  │                                 │  │
└─────────────────────────────────────────────────────────────┘
```

- Desktop: fixed sidebar, 256px.
- Mobile (< 768px): sidebar collapses to hamburger, overlay drawer.
- Main content: `max-w-3xl` for chat, `max-w-5xl` for tables.

### Pages

| Route | Description |
|-------|-------------|
| `/login` | Login + register (single page, tab switch) |
| `/` | Redirect to `/chats` |
| `/chats` | Conversation list |
| `/chats/[id]` | Chat view (streaming, markdown, code blocks) |
| `/workspaces` | Workspace list |
| `/workspaces/[id]` | Workspace detail (collections, members, chats) |
| `/collections` | Collection table (admin) |
| `/documents` | Document list |
| `/users` | User management (admin) |
| `/settings` | Profile, theme, password |

### Chat UI

- Streaming via `EventSource` (SSE).
- Markdown: `react-markdown` + `remark-gfm`.
- Code highlighting: `shiki` (server-side, zero client JS for highlighting).
- Copy button on code blocks and messages.
- Regenerate: re-sends last user message.
- Stop button during streaming.
- Auto-scroll pinned to bottom during stream.
- Keyboard: `Enter` send, `Shift+Enter` newline, `Ctrl+K` search.

### Collection UI (admin)

Table with columns: Name, Vectors, Dimension, Distance, Status, Last Sync.

Status: colored dot (green=online, gray=offline, yellow=missing, red=error).

Actions: Assign workspace (dropdown), Remove, Refresh metadata.
"Mark as missing" hidden under `⋯` menu, requires typing collection name.

### Accessibility

- WCAG 2.1 AA. Full keyboard navigation.
- `aria-label` on icon buttons. `role="log"` + `aria-live="polite"` for chat.
- Focus visible: ring outline on `:focus-visible`.
- Contrast 4.5:1 minimum.
- Semantic HTML: `<nav>`, `<main>`, `<table>`, `<button>`.

### Performance

| Metric | Target |
|--------|--------|
| LCP | < 1.5s |
| TBT | < 200ms |
| CLS | < 0.1 |
| Lighthouse | 90+ |

- Server components for static content (sidebar, tables).
- Client components only where interactivity is needed (chat input, SSE).
- `next/image` not needed (no images).
- Zero client-side data fetching libraries. Native `fetch` + `EventSource`.

---

## 6. Qdrant Sync Engine

### Scale Targets

| Metric | Target |
|--------|--------|
| Collections | 100+ |
| Total vectors | 10M+ |
| Search latency (p95) | < 200ms per collection |
| Workspace search (5 collections) | < 500ms total |

### Ownership

| Qdrant owns | NothingLLM owns |
|-------------|-----------------|
| Collections | Users |
| Vectors | Workspaces |
| Payloads | Permissions |
| Indexes | Collection assignments |
| Shards | Conversation history |

NothingLLM **never** creates, modifies, or deletes Qdrant data unless an admin explicitly triggers a point deletion (e.g., removing a document the user uploaded).

### Sync Loop

```go
// internal/sync/engine.go
type Engine struct {
    qdrant   *qdrant.Client
    db       *db.DB
    cache    *qdrant.Cache
    interval time.Duration
    log      *slog.Logger
}

func (e *Engine) Run(ctx context.Context) {
    e.sync() // immediate on start
    ticker := time.NewTicker(e.interval)
    for {
        select {
        case <-ctx.Done(): return
        case <-ticker.C: e.sync()
        }
    }
}

func (e *Engine) sync() {
    cols, err := e.qdrant.ListCollections(context.Background())
    if err != nil {
        e.db.MarkAllOffline(context.Background())
        e.cache.Invalidate()
        e.log.Error("qdrant sync failed", "err", err)
        return
    }
    seen := make(map[string]bool)
    for _, c := range cols {
        seen[c.Name] = true
        detail, err := e.qdrant.GetCollection(context.Background(), c.Name)
        if err != nil {
            e.db.SetStatus(context.Background(), c.Name, "error")
            continue
        }
        e.db.UpsertCollection(context.Background(), detail)
        e.cache.Update(detail)
    }
    e.db.MarkMissing(context.Background(), seen)
}
```

**Behavior:**
- New collection → insert row, status `online`, update cache
- Metadata changed → update `vector_count`, `dimensions`, `distance_metric`
- Collection gone → `status = 'missing'`, row + mappings preserved
- Qdrant down → all `offline`, cache invalidated, app keeps serving
- **Never** deletes rows from `collections` or `workspace_collections`

### In-Memory Cache

```go
// internal/qdrant/cache.go
type Cache struct {
    mu          sync.RWMutex
    collections map[string]CollectionMeta
}

func (c *Cache) Get(name string) (CollectionMeta, bool)
func (c *Cache) Update(meta CollectionMeta)
func (c *Cache) Online(names []string) []string  // filter to online
func (c *Cache) Invalidate()
```

- Populated by sync. Serves `/api/collections` without hitting Qdrant.
- Chat requests use cache to resolve which collections are online.
- Full replacement on each sync cycle.

### Parallel Search

```go
// internal/rag/retrieve.go
func (r *Retriever) SearchWorkspace(ctx context.Context, cols []string, q []float32, limit int, filter *qdrant.Filter) ([]Result, error) {
    var mu sync.Mutex
    var all []Result
    sem := make(chan struct{}, 10) // max 10 concurrent
    var wg sync.WaitGroup

    for _, name := range cols {
        wg.Add(1)
        go func(col string) {
            defer wg.Done()
            sem <- struct{}{}
            defer func() { <-sem }()

            reqCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
            defer cancel()

            results, err := r.qdrant.Search(reqCtx, col, qdrant.SearchRequest{
                Vector: q, Limit: limit, Filter: filter,
            })
            if err != nil {
                r.log.Warn("search failed", "col", col, "err", err)
                return
            }
            mu.Lock()
            all = append(all, results...)
            mu.Unlock()
        }(name)
    }
    wg.Wait()

    sort.Slice(all, func(i, j int) bool { return all[i].Score > all[j].Score })
    if len(all) > limit { all = all[:limit] }
    return all, nil
}
```

**Constraints:**
- Max 10 concurrent Qdrant requests
- 3s timeout per collection
- 5s total budget (enforced by caller context)
- One collection failing does NOT fail the request
- Only searches workspace-assigned, online collections

### Metadata Filtering

Filters pass directly to Qdrant. Supported keys:

```go
type Filter struct {
    Must    []Condition // AND
    Should  []Condition // OR
    MustNot []Condition // NOT
}

type Condition struct {
    Key   string // "department", "project", "source", "document", "version", "tag"
    Match any    // exact value, list, or range
}
```

Example request:
```json
{ "filters": { "department": "engineering", "tag": ["runbook", "postmortem"] } }
```

### Hybrid Search (Ready)

- `SearchRequest.SparseVector` field exists in client.
- Qdrant 1.10+ `/points/hybrid` endpoint.
- Config: `qdrant.hybrid: false` (default).
- App queries hybrid indexes but never creates them.

### Search Flow (per chat message)

```
1. Resolve workspace → assigned collection names (Postgres)
2. Filter to online (in-memory cache)
3. Embed query → dense vector (OpenAI)
4. Parallel Qdrant search (bounded, with optional filter)
5. Merge, sort by score, take top_K
6. Build prompt with context
7. Stream OpenAI response
```

**Total Qdrant calls per message:** N (assigned online collections). Never more.

---

## 7. Dynamic Configuration

**Principle: No static business logic. Configuration lives in the database, managed from the UI.**

### OpenAI Model Discovery

Models are synced from the OpenAI API. No hardcoded model names in code.

```go
// internal/sync/models.go
func (e *Engine) syncModels(ctx context.Context) {
    // GET https://api.openai.com/v1/models
    models, err := e.openai.ListModels(ctx)
    if err != nil {
        e.log.Warn("model sync failed", "err", err)
        return
    }
    for _, m := range models {
        e.db.UpsertModel(ctx, m.ID, m.Name, m.ContextWindow, m.CreatedAt)
    }
    // New models appear automatically. Admin enables/disables from UI.
}
```

**`models` table:**
| Field | Source |
|-------|--------|
| `id` | OpenAI API (`gpt-4o`, `gpt-4o-mini`, etc.) |
| `name` | Display name from API |
| `context_window` | From API |
| `enabled` | Admin toggle in UI (default: true) |
| `synced_at` | Last sync timestamp |

Workspace settings reference `model_id` (a TEXT FK to `models.id`). No hardcoded model strings in code.

**Sync frequency:** Every 6 hours (configurable). New models appear automatically.

### Dynamic Metadata Discovery

Qdrant payload schemas are unknown and arbitrary. NothingLLM discovers them by sampling points.

```go
// internal/sync/metadata.go
func (e *Engine) discoverMetadata(ctx context.Context, collectionName string) {
    // Sample 100 random points from the collection
    points, err := e.qdrant.SamplePoints(ctx, collectionName, 100)
    if err != nil {
        return
    }
    // Extract all payload keys + types + sample values
    keys := extractPayloadKeys(points)
    e.db.UpsertCollectionMetadata(ctx, collectionName, keys)
}
```

**`collection_metadata` table** stores discovered keys:
- `collection_name` — which Qdrant collection
- `key` — payload field name (e.g., `department`, `source`, `project`)
- `data_type` — `keyword`, `integer`, `float`, `datetime`, `text`
- `sample_values` — first 10 distinct values (for UI dropdown generation)

**The filtering UI is generated dynamically** from this table. No hardcoded filter fields.

### Workspace Settings (per-workspace, in database)

```go
// Stored in workspace_settings, loaded per request
type WorkspaceSettings struct {
    ModelID         string  // FK → models.id
    Temperature     float64
    MaxResults      int
    MaxContext      int
    SearchStrategy  string  // "workspace" | "selected" | "group"
    SearchProfileID *string // FK → search_profiles.id
    DefaultGroupID  *string // FK → collection_groups.id
}
```

**No workspace logic in code.** Admin modifies settings from UI at runtime. No restart.

### Search Profiles (dynamic, user-created)

Stored in `search_profiles`. Users can create custom profiles.

| Profile | top_k | min_score | timeout_ms |
|---------|-------|-----------|------------|
| Quick Search | 3 | 0.8 | 2000 |
| Deep Search | 10 | 0.5 | 8000 |
| Research | 20 | 0.3 | 15000 |

These are **examples seeded on first run**. Users create their own. No hardcoded profiles in code.

### Collection Groups

Logical grouping of collections. Owned by NothingLLM, not Qdrant.

```
Group: Engineering
  Collections: engineering-docs, engineering-wiki, incidents

Group: Compliance
  Collections: policies, audit-logs, regulations
```

- Created/edited from admin UI.
- A workspace can use a group as its default search scope.
- `search_strategy = 'group'` → search all collections in the default group.
- Collections remain owned by Qdrant. Groups are just references.

### Live Synchronization

The system reacts to external changes without restart:

| External Change | Detection | Action |
|----------------|-----------|--------|
| New Qdrant collection | Sync loop (60s) | Insert row, discover metadata, update cache |
| Collection removed | Sync loop | Mark `missing`, preserve mappings |
| Vector count changes | Sync loop | Update `vector_count` |
| New OpenAI model | Model sync (6h) | Insert into `models`, admin enables |
| Metadata schema changes | Metadata discovery | Update `collection_metadata` |
| Collection payload changes | Next metadata sample | Refresh sample values |

**No restart. No config file edit. No code change.**

### What Stays in config.yaml

Only connection strings and infrastructure settings:

```yaml
server:
  addr: ":8080"

database:
  dsn: "postgres://nlm:nlm@localhost:5432/nothingllm?sslmode=disable"

qdrant:
  url: "http://localhost:6333"
  api_key: ""
  sync_interval: "60s"

openai:
  api_key: "sk-..."

auth:
  session_ttl: "24h"
  bcrypt_cost: 12

rate_limit:
  requests_per_minute: 100
  login_per_minute: 5
```

**No model names. No chunk sizes. No top_k. No collection names. No business rules.**
All business configuration is in the database, editable from the UI.

### Admin UI Management Surface

| Entity | UI Actions |
|--------|-----------|
| Users | Create, edit role, delete, reset password |
| Workspaces | Create, edit settings (model, temperature, strategy) |
| Collections | View status, assign/unassign, refresh metadata |
| Collection Groups | Create, rename, add/remove collections |
| Search Profiles | Create, edit parameters, delete |
| OpenAI Models | Enable/disable, view context window |
| Metadata | View discovered keys per collection (read-only) |


## 8. Authentication Model

- **bcrypt** (cost 12) for password storage
- **Session tokens**: 256-bit random hex, stored server-side in Postgres
- **Cookie**: `nlm_session`, `HttpOnly; Secure; SameSite=Strict; Path=/`
- **CSRF**: `SameSite=Strict` + `Origin` header check on state-changing requests
- **Rate limiting**: `golang.org/x/time/rate`, per-IP token bucket
- **No JWT. No OAuth. No LDAP. No SSO.**

### Flow

```
Register:  POST /api/auth/register → bcrypt → INSERT users → audit_log
Login:     POST /api/auth/login → verify bcrypt → generate token
           → INSERT sessions → Set-Cookie → audit_log
Request:   read cookie → JOIN sessions+users → check expiry → context
Logout:    POST /api/auth/logout → DELETE session → clear cookie → audit_log
Reset:     POST /api/auth/reset → INSERT password_resets (1h TTL)
           POST /api/auth/reset/confirm → UPDATE users → DELETE sessions
```

### Audit Log Actions

| Action | entity_type | Trigger |
|--------|-------------|---------|
| `login` | `user` | Successful login |
| `logout` | `user` | Logout |
| `document_upload` | `document` | File uploaded |
| `collection_assign` | `workspace_collection` | Collection assigned |
| `collection_unassign` | `workspace_collection` | Collection removed |
| `user_create` | `user` | Admin creates user |
| `user_delete` | `user` | Admin deletes user |

---

## 9. Workspace Model

A workspace is a grouping owned by a user. It references collections and contains members.

```
Workspace: Engineering
  Owner: alice@example.com
  Members: alice, bob, carol
  Collections: engineering-docs, engineering-wiki, incidents
  Chats: 42 conversations
```

**Rules:**
- Workspace does NOT own vectors. It references Qdrant collections.
- One collection can belong to many workspaces.
- Deleting a workspace: cascades `workspace_members` and `workspace_collections`. Qdrant untouched.
- Owner can add/remove members, assign/unassign collections.
- Members can chat, search, upload documents within the workspace.

---

## 10. Collection Mapping Design

Many-to-many: `workspace_collections (workspace_id, collection_name)`

```
Workspaces              Qdrant Collections
┌──────────┐           ┌──────────────────┐
│Engineering│──────────►│ engineering-docs │
│Engineering│──────────►│ engineering-wiki │
│Engineering│──────────►│ incidents        │
│  Support  │──────────►│ incidents        │  ← shared
│  Support  │──────────►│ faq              │
└──────────┘           └──────────────────┘
```

**Invariants:**
- `DELETE FROM workspace_collections WHERE ...` → mapping gone, Qdrant collection intact
- `DELETE FROM workspaces WHERE id = ...` → mappings cascade, Qdrant intact
- App never calls Qdrant delete on collection or points (except document cleanup)
- Collection "deletion" in admin UI only sets `status='missing'` locally

---

## 11. Docker Compose

```yaml
services:
  db:
    image: postgres:16-alpine
    environment:
      POSTGRES_USER: nlm
      POSTGRES_PASSWORD: nlm
      POSTGRES_DB: nothingllm
    volumes:
      - pgdata:/var/lib/postgresql/data
      - ./migrations:/docker-entrypoint-initdb.d
    ports:
      - "5432:5432"
    healthcheck:
      test: ["CMD", "pg_isready", "-U", "nlm"]
      interval: 5s
      timeout: 3s
      retries: 5

  qdrant:
    image: qdrant/qdrant:latest
    volumes:
      - qdrant_data:/qdrant/storage
    ports:
      - "6333:6333"

  api:
    build:
      context: .
      dockerfile: Dockerfile.api
    ports:
      - "8080:8080"
    volumes:
      - ./config.yaml:/app/config.yaml
    depends_on:
      db:
        condition: service_healthy
      qdrant:
        condition: service_started
    healthcheck:
      test: ["CMD", "wget", "-qO-", "http://localhost:8080/health"]
      interval: 10s
      timeout: 3s
      retries: 3

  web:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:3000"
    environment:
      - NEXT_PUBLIC_API_URL=http://localhost:8080
    depends_on:
      - api

volumes:
  pgdata:
  qdrant_data:
```

### Dockerfile.api

```dockerfile
FROM golang:1.24-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o /app/nothingllm ./cmd/nothingllm

FROM alpine:3.20
RUN apk add --no-cache ca-certificates tzdata
COPY --from=build /app/nothingllm /app/nothingllm
WORKDIR /app
EXPOSE 8080
ENTRYPOINT ["/app/nothingllm"]
```

### Dockerfile (frontend)

```dockerfile
FROM node:22-alpine AS deps
WORKDIR /app
COPY package.json package-lock.json ./
RUN npm ci

FROM node:22-alpine AS build
WORKDIR /app
COPY --from=deps /app/node_modules ./node_modules
COPY . .
RUN npm run build

FROM node:22-alpine AS run
WORKDIR /app
COPY --from=build /app/.next ./.next
COPY --from=build /app/node_modules ./node_modules
COPY --from=build /app/package.json ./
EXPOSE 3000
CMD ["npm", "start"]
```

**4 services. 2 volumes. No orchestrator.**

---

## 12. Configuration

`config.yaml` contains **only** connection strings and infrastructure settings. All business configuration lives in the database, editable from the admin UI.

```yaml
# config.yaml — the ONLY static file

server:
  addr: ":8080"

database:
  dsn: "postgres://nlm:nlm@localhost:5432/nothingllm?sslmode=disable"

qdrant:
  url: "http://localhost:6333"
  api_key: ""
  sync_interval: "60s"

openai:
  api_key: "sk-..."
  embedding_model: "text-embedding-3-small"  # used for uploads only

auth:
  session_ttl: "24h"
  bcrypt_cost: 12

rate_limit:
  requests_per_minute: 100
  login_per_minute: 5
```

**17 lines. No model names. No top_k. No chunk sizes. No collection names.**
Everything else is in the database:
- Which model each workspace uses → `workspace_settings.model_id`
- Temperature, max results, context → `workspace_settings`
- Search profiles → `search_profiles` table
- Collection groups → `collection_groups` table
- Which collections are active → `collections.status`
- Which metadata fields exist → `collection_metadata` table

---

## 13. Implementation Plan

### Phase 1: Backend Skeleton (Day 1–2)

- [ ] `go mod init nothingllm`
- [ ] `internal/config/config.go` — load + validate YAML
- [ ] `internal/db/db.go` — pgx pool, migration runner
- [ ] `migrations/001_init.sql`, `002_workspaces.sql`, `003_content.sql`
- [ ] `internal/api/server.go` — `http.ServeMux`, routes
- [ ] `cmd/nothingllm/main.go` — wire config → db → server, graceful shutdown
- [ ] `GET /health`, `GET /ready`
- [ ] **Verify**: `docker compose up`, `curl /health` → 200

### Phase 2: Auth (Day 3–4)

- [ ] `internal/auth/auth.go` — register, login, logout, reset
- [ ] `internal/auth/middleware.go` — session, role, rate limit
- [ ] `internal/db/users.go`, `internal/db/sessions.go`
- [ ] `internal/api/handlers_auth.go`
- [ ] Audit log on login/logout
- [ ] **Verify**: register → login → 401 → logout → 401

### Phase 3: Qdrant Client + Sync (Day 5–7)

- [ ] `internal/qdrant/client.go` — REST client
- [ ] `internal/qdrant/cache.go` — in-memory cache
- [ ] `internal/sync/engine.go` — background loop
- [ ] `internal/db/collections.go` — upsert, mark missing/offline
- [ ] **Verify**: create collection in Qdrant → appears in DB within 60s

### Phase 4: Workspaces + Collections (Day 8–10)

- [ ] `internal/db/workspaces.go` — CRUD + members + mappings
- [ ] `internal/api/handlers_workspaces.go`
- [ ] `internal/api/handlers_collections.go`
- [ ] Audit log on assign/unassign
- [ ] **Verify**: assign collection to workspace, Qdrant untouched

### Phase 5: RAG + Chat (Day 11–14)

- [ ] `internal/openai/client.go`, `chat.go` (streaming), `embed.go`
- [ ] `internal/rag/chunk.go`, `retrieve.go`
- [ ] `internal/db/conversations.go`
- [ ] `internal/api/handlers_chat.go` — SSE
- [ ] `internal/api/handlers_search.go`
- [ ] **Verify**: chat with grounded streaming answer

### Phase 6: Documents (Day 15–16)

- [ ] `internal/parse/pdf.go`, `text.go`, `markdown.go`
- [ ] `internal/qdrant/points.go` — upsert
- [ ] `internal/api/handlers_documents.go`
- [ ] **Verify**: upload PDF → searchable in chat

### Phase 7: Frontend (Day 17–22)

- [ ] `npm create next-app` (TypeScript, Tailwind, App Router)
- [ ] shadcn/ui: button, input, card, table, dropdown, dialog, badge
- [ ] Login page, sidebar layout, theme provider
- [ ] Chats (list + detail with SSE streaming)
- [ ] Workspaces (list + detail)
- [ ] Collections (admin table with status)
- [ ] Documents, Users, Settings pages
- [ ] **Verify**: full browser flow

### Phase 8: Hardening (Day 23–25)

- [ ] Rate limiting, input validation
- [ ] Structured logging (`log/slog` JSON)
- [ ] Graceful shutdown
- [ ] Qdrant-down resilience test
- [ ] Lighthouse 90+
- [ ] README, `docker compose up` from zero
- [ ] **Verify**: cold start to first chat < 30s

---

## 14. Project Structure

```
nothingllm/
├── cmd/nothingllm/main.go
├── internal/
│   ├── config/config.go
│   ├── db/
│   │   ├── db.go
│   │   ├── users.go
│   │   ├── sessions.go
│   │   ├── workspaces.go
│   │   ├── collections.go
│   │   └── conversations.go
│   ├── auth/
│   │   ├── auth.go
│   │   └── middleware.go
│   ├── api/
│   │   ├── server.go
│   │   ├── handlers_auth.go
│   │   ├── handlers_users.go
│   │   ├── handlers_workspaces.go
│   │   ├── handlers_collections.go
│   │   ├── handlers_chat.go
│   │   ├── handlers_search.go
│   │   └── handlers_documents.go
│   ├── qdrant/
│   │   ├── client.go
│   │   ├── cache.go
│   │   └── points.go
│   ├── sync/engine.go
│   ├── rag/
│   │   ├── chunk.go
│   │   ├── embed.go
│   │   └── retrieve.go
│   ├── openai/
│   │   ├── client.go
│   │   ├── chat.go
│   │   ├── embed.go
│   │   └── tts.go
│   └── parse/
│       ├── pdf.go
│       ├── text.go
│       └── markdown.go
├── frontend/
│   ├── app/
│   │   ├── (auth)/login/page.tsx
│   │   ├── (main)/
│   │   │   ├── layout.tsx
│   │   │   ├── chats/page.tsx
│   │   │   ├── chats/[id]/page.tsx
│   │   │   ├── workspaces/page.tsx
│   │   │   ├── workspaces/[id]/page.tsx
│   │   │   ├── collections/page.tsx
│   │   │   ├── documents/page.tsx
│   │   │   ├── users/page.tsx
│   │   │   └── settings/page.tsx
│   │   └── layout.tsx
│   ├── components/
│   │   ├── sidebar.tsx
│   │   ├── chat-message.tsx
│   │   ├── collection-table.tsx
│   │   └── ui/
│   ├── lib/
│   │   ├── api.ts
│   │   └── utils.ts
│   ├── package.json
│   ├── tailwind.config.ts
│   └── Dockerfile
├── migrations/
│   ├── 001_init.sql
│   ├── 002_workspaces.sql
│   └── 003_content.sql
├── config.yaml
├── Dockerfile.api
├── docker-compose.yml
├── go.mod
└── README.md
```

---

## 15. Dependencies

### Go (5 modules)

| Module | Purpose |
|--------|---------|
| `github.com/jackc/pgx/v5` | PostgreSQL |
| `golang.org/x/crypto` | bcrypt |
| `golang.org/x/time` | Rate limiter |
| `gopkg.in/yaml.v3` | Config |
| `github.com/ledongthuc/pdf` | PDF extraction |

Qdrant and OpenAI via `net/http`. No SDKs.

### Frontend

Next.js + Tailwind + shadcn/ui. No state management library. No data-fetching library. Native `fetch` + `EventSource`.

---

## 16. What This Is NOT

| No | Reason |
|----|--------|
| No static model names in code | Model registry in DB, synced from API |
| No hardcoded metadata fields | Discovered from Qdrant payload samples |
| No hardcoded search profiles | Dynamic, user-created in DB |
| No hardcoded collection lists | Qdrant is authoritative, synced live |
| No `if workspace == "x"` logic | All rules in `workspace_settings` |
| No config file edits after startup | Admin UI manages everything |
| No Kubernetes | 4 containers don't need it |
| No microservices | One Go binary + one Next.js app |
| No CQRS / Event Sourcing | Overkill |
| No Redis / Elasticsearch / RabbitMQ / NATS | Not needed |
| No pgvector | Qdrant only |
| No Weaviate / Milvus / Pinecone / Chroma | Qdrant only |
| No provider abstraction | OpenAI only |
| No plugin system | YAGNI |
| No GraphQL | REST is sufficient |
| No DI framework | Constructor injection in `main.go` |
| No generic repositories | Concrete functions |
| No abstraction "for future" | Build when needed |
| No agent marketplace | Not a feature |
| No workflow builder | Not a feature |

---

## 17. Success Metrics

| Metric | Target |
|--------|--------|
| Go lines | < 5000 |
| RAM (idle) | < 100 MB |
| Startup time | < 2 seconds |
| Go dependencies | 5 modules |
| Config | 1 file, 17 lines (connections only) |
| Infrastructure | Postgres + Qdrant + API + Web |
| Deployment | `docker compose up` |
| Time to understand | < 1 hour |
| Lighthouse | 90+ |
