# NothingLLM

AI workspace manager with RAG pipeline.

## Architecture

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   Next.js   │────▶│  Go API     │────▶│  PostgreSQL │
│  (web:3000) │     │  (api:8080) │     │ (pg:5432)   │
└─────────────┘     └──────┬──────┘     └─────────────┘
                           │
                           ▼
                    ┌─────────────┐     ┌─────────────┐
                    │   Qdrant    │     │   OpenAI    │
                    │(qdrant:6333)│     │  (external) │
                    └─────────────┘     └─────────────┘
```

## Quick Start

```bash
cp .env.example .env
# Edit .env: set OPENAI_API_KEY and SESSION_SECRET

docker compose up -d
```

Access:

| Service  | URL                  |
|----------|----------------------|
| Frontend | http://localhost:3000 |
| API      | http://localhost:8080 |
| Qdrant   | http://localhost:6333 |
| Postgres | localhost:5432        |

### Verify

```bash
# API health
curl http://localhost:8080/health

# API readiness (checks DB + Qdrant)
curl http://localhost:8080/ready

# Qdrant
curl http://localhost:6333/healthz

# Frontend
curl -I http://localhost:3000
```

## Development

```bash
docker compose -f docker-compose.yml -f docker-compose.dev.yml up -d
```

Dev mode mounts source code into containers for hot reload.

## Configuration

All configuration is via environment variables (see `.env.example`):

| Variable           | Description                          | Default                    |
|--------------------|--------------------------------------|----------------------------|
| `POSTGRES_PASSWORD`| PostgreSQL password                  | `nlm`                      |
| `QDRANT_API_KEY`   | Qdrant API key (optional)            | *(empty)*                  |
| `OPENAI_API_KEY`   | OpenAI API key                       | *(required)*               |
| `SESSION_SECRET`   | Session signing secret               | *(required in production)* |

No secrets are committed to the repository.

## Services

| Service    | Image               | Port | Volume          |
|------------|---------------------|------|-----------------|
| postgres   | postgres:16-alpine  | 5432 | postgres_data   |
| qdrant     | qdrant/qdrant:latest| 6333 | qdrant_data     |
| api        | built from Dockerfile | 8080 | —             |
| web        | built from frontend/Dockerfile | 3000 | —         |

## Startup Order

```
postgres → qdrant → api → web
```

Enforced via `depends_on` with `condition: service_healthy`.

## Health Checks

| Service  | Method                        |
|----------|-------------------------------|
| postgres | `pg_isready -U nlm -d nothingllm` |
| qdrant   | `GET /healthz`                |
| api      | `GET /ready`                  |
| web      | `GET /`                       |

## Troubleshooting

### API not starting

```bash
docker compose logs api
```

Check that `POSTGRES_URL` and `QDRANT_URL` are reachable. The API requires both to be healthy before starting.

### Frontend shows connection errors

The frontend proxies `/api/*` to the Go backend. Verify the API is healthy:

```bash
docker compose exec web wget -qO- http://api:8080/ready
```

### Qdrant collections not appearing

The sync engine runs every 60s. Check Qdrant is accessible from the API container:

```bash
docker compose exec api wget -qO- http://qdrant:6333/collections
```

### Database migrations failed

```bash
docker compose logs api | grep -i "migrate\|error"
```

Migrations run automatically on API startup. If a migration fails, fix the issue and restart:

```bash
docker compose restart api
```

### Reset everything

```bash
docker compose down -v
docker compose up -d
```

This destroys all data (Postgres + Qdrant volumes).

## Production Notes

- Set `SESSION_SECRET` to a random 64-character string
- Set a strong `POSTGRES_PASSWORD`
- Restrict port exposure if deploying behind a reverse proxy
- Enable TLS termination at your load balancer
- Back up `postgres_data` and `qdrant_data` volumes regularly