# NothingLLM Production Guide

## Deployment

### Prerequisites

- Docker 24+ with Compose v2
- 4GB+ RAM, 2GB+ disk
- OpenAI API key
- (Optional) Qdrant API key for auth-enabled Qdrant

### Quick Start

```bash
cp .env.example .env
# Edit .env: set OPENAI_API_KEY and SESSION_SECRET
docker compose up -d
```

Services:
| Service | Port | Health Check |
|---------|------|-------------|
| Frontend (Next.js) | 3000 | `GET /` |
| API (Go) | 8080 | `GET /ready` |
| PostgreSQL | 5432 | `pg_isready` |
| Qdrant | 6333 | `GET /healthz` |

### Configuration

All config via environment variables (see `.env.example`):

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `POSTGRES_PASSWORD` | Yes | - | PostgreSQL password |
| `OPENAI_API_KEY` | Yes | - | OpenAI API key |
| `SESSION_SECRET` | Yes | - | 64-char random string for session signing |
| `QDRANT_API_KEY` | No | empty | Qdrant API key (if auth enabled) |

### Upgrades

1. Pull latest images: `docker compose pull`
2. Recreate services (data persists in volumes): `docker compose up -d`
3. Migrations run automatically on API startup.
4. Verify: `curl http://localhost:8080/ready`

## Backup

### Procedure

```bash
# Daily backup (add to cron)
./backup.sh /var/backups/nothingllm
```

Creates a timestamped directory containing:
- `postgres.dump` — compressed PostgreSQL custom-format dump
- `qdrant_snapshot.tar` — Qdrant snapshot (if Qdrant is reachable)
- `manifest.json` — backup metadata

Retention: last 7 backups kept automatically.

### Scheduling

```cron
# Daily at 2am
0 2 * * * cd /opt/nothingllm && ./backup.sh /var/backups/nothingllm >> /var/log/nothingllm-backup.log 2>&1
```

## Restore

### Procedure

```bash
./restore.sh /var/backups/nothingllm/nothingllm_20261031_020000
```

**WARNING:** This drops and recreates the database. Qdrant snapshot restore is additive.

### Recovery Steps

1. Stop the API: `docker compose stop api`
2. Run restore: `./restore.sh <backup_dir>`
3. Restart: `docker compose up -d`
4. Verify: `curl http://localhost:8080/metrics`

## Observability

### Endpoints

| Endpoint | Description |
|----------|-------------|
| `GET /health` | Liveness probe (always 200 if process is up) |
| `GET /ready` | Readiness probe (checks DB + Qdrant connectivity) |
| `GET /metrics` | JSON metrics (users, workspaces, collections, sessions, sync status) |

### Metrics Example

```json
{
  "users": 5,
  "workspaces": 2,
  "collections": 3,
  "active_sessions": 1,
  "sync_status": "ok"
}
```

### Logging

Structured JSON logs to stdout. Collect with your log aggregator (e.g., `docker logs nothingllm-api`).

## Troubleshooting

### API won't start

```bash
docker compose logs api
# Check:
# - POSTGRES_URL reachable
# - QDRANT_URL reachable
# - Config file generated (entrypoint.sh)
```

### Qdrant unreachable (degraded mode)

- API stays up; `/ready` returns 503 with `reason: "qdrant"`
- Search/chat return 503 with meaningful error message
- Collection listing shows cached data from PostgreSQL
- System auto-recovers when Qdrant comes back (next sync cycle)

### OpenAI rate limited / outage

- Chat returns 502 with `openai: 429: ...` error
- No retry storm (single attempt per request)
- Embeddings for new documents will fail; existing vectors remain searchable

### Session issues

- Sessions expire after 24h
- Expired sessions are cleaned up on next validation
- Check `active_sessions` in `/metrics` for anomalies

### Database connection pool exhaustion

- Default pool: 25 connections
- Symptoms: requests time out, `connection pool exhausted` in logs
- Fix: increase `pool_max_conns` in DSN or add more replicas

## Security

- Passwords: bcrypt (cost 12)
- Sessions: 32-byte random IDs, HttpOnly + Secure + SameSite=Strict cookies
- Rate limiting: 100 req/min general, 5 login/min
- Input validation: email ≤ 254 chars, password ≤ 128 chars
- SQL injection: parameterized queries (pgx)
- XSS: Next.js escapes by default; API returns JSON only