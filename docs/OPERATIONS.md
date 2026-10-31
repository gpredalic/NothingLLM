# NothingLLM Operations Runbook

## Startup

### Fresh Install

```bash
# 1. Copy environment template
cp .env.example .env

# 2. Set required values
#    - OPENAI_API_KEY: your OpenAI key
#    - SESSION_SECRET: 64+ char random string (e.g. openssl rand -hex 32)
#    - POSTGRES_PASSWORD: strong password

# 3. Start the stack
docker compose up -d

# 4. Verify (wait ~30s for health gates)
curl -s http://localhost:8080/health    # {"status":"ok"}
curl -s http://localhost:8080/ready     # {"status":"ready"}
curl -s http://localhost:8080/version   # {"version":"...","git_commit":"...","build_date":"..."}
curl -s http://localhost:8080/metrics   # {"users":0,"workspaces":0,...}
curl -s http://localhost:3000           # HTML (frontend)

# 5. Register first user (becomes admin)
curl -X POST http://localhost:8080/api/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"admin@example.com","password":"..."}'
```

### Service Startup Order

Enforced by `depends_on: condition: service_healthy`:

```
postgres → qdrant → api → web
```

The API will not start until both Postgres and Qdrant report healthy.
The frontend will not start until the API reports healthy.

## Shutdown

```bash
# Graceful shutdown (10s drain)
docker compose down

# With data destruction (DESTRUCTIVE)
docker compose down -v
```

The API handles SIGTERM gracefully:
1. Stops accepting new connections
2. Waits up to 10s for in-flight requests
3. Closes DB pool
4. Exits

## Backup

```bash
# Manual backup
./backup.sh /var/backups/nothingllm

# Automated (cron)
0 2 * * * cd /opt/nothingllm && ./backup.sh /var/backups/nothingllm >> /var/log/backup.log 2>&1
```

Produces:
- `postgres.dump` — PostgreSQL custom-format (compressed)
- `qdrant_snapshot.tar` — Qdrant snapshot (if reachable)
- `manifest.json` — metadata

Retention: last 7 backups kept.

## Restore

```bash
# From a backup directory
./restore.sh /var/backups/nothingllm/nothingllm_20261031_020000
```

**WARNING:** Drops and recreates the database. Confirm before proceeding.

### Recovery Procedure

1. `docker compose stop api`
2. `./restore.sh <backup_dir>`
3. `docker compose up -d`
4. `curl http://localhost:8080/metrics` — verify data

## Upgrade

### Minor Version (same major)

```bash
# 1. Backup first
./backup.sh /var/backups/nothingllm

# 2. Pull new images
docker compose pull

# 3. Recreate services (volumes persist)
docker compose up -d

# 4. Migrations run automatically on API startup
# 5. Verify
curl -s http://localhost:8080/version
curl -s http://localhost:8080/ready
```

### Major Version

1. Perform a full backup
2. Review migration SQL files in `internal/db/migrations/`
3. Test on staging first
4. Follow the minor upgrade procedure
5. Verify all endpoints

## Rollback

If an upgrade causes issues:

```bash
# 1. Stop the new version
docker compose down

# 2. Restore from backup
./restore.sh /var/backups/nothingllm/<pre-upgrade-backup>

# 3. Start the old version (tag the image)
docker compose up -d

# 4. Verify
curl -s http://localhost:8080/version
```

### Image-Based Rollback

```bash
# If you tagged images by version:
docker tag nothingllm:1.0.0 nothingllm:current
docker compose up -d
```

## Observability

| Endpoint | Purpose |
|----------|---------|
| `GET /health` | Liveness (process up) |
| `GET /ready` | Readiness (DB + Qdrant connected) |
| `GET /metrics` | JSON: users, workspaces, collections, sessions, sync status |
| `GET /version` | Build info: version, git commit, build date |

### Audit Log

All significant actions are logged to the `audit_log` table:
- `login` / `logout`
- `create_user` / `delete_user` / `update_user`
- `create_workspace` / `delete_workspace`
- `assign_collection` / `remove_collection`

Query: `SELECT * FROM audit_log ORDER BY created_at DESC LIMIT 50;`

## Troubleshooting

### API not starting

```bash
docker compose logs api
# Check:
# - POSTGRES_URL is reachable
# - QDRANT_URL is reachable
# - Config generated correctly (entrypoint.sh)
```

### Qdrant unreachable

- `/ready` returns 503 with `reason: "qdrant"`
- Search/chat return 503
- System auto-recovers on next sync cycle
- Check: `docker compose logs qdrant`

### OpenAI errors

- Invalid key: `openai: 401: ...`
- Rate limited: `openai: 429: ...`
- No retries (single attempt per request)

### High memory usage

- Check `/metrics` for session growth
- Expired sessions are cleaned on validation
- Manual cleanup: `DELETE FROM sessions WHERE expires_at < now();`