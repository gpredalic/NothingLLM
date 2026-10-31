# Project
NothingLLM

# Stack
- Go (backend API server)
- Next.js (frontend)
- PostgreSQL (primary data store)
- Qdrant (vector database)
- OpenAI (LLM + embeddings)

# Architecture Principles
- Dynamic-first
- No hardcoded collections
- No hardcoded models
- Database-driven configuration
- Qdrant is source of truth
- Simplicity over abstractions

# Build Commands
```
go test ./...
go test -race ./...
npm run build
docker compose build
docker compose up -d
```

# Validation Checklist
- [ ] Backend tests pass
- [ ] Frontend builds
- [ ] Docker healthy
- [ ] Health endpoint works
- [ ] Ready endpoint works
- [ ] Workspace CRUD works
- [ ] User CRUD works
- [ ] Collection assignment works
- [ ] Login works
- [ ] Setup wizard works

# Known Issues
- (none currently tracked)

# Lessons Learned
- 2025-10-31: Response format mismatch between Go backend (bare JSON arrays) and TypeScript frontend (expected wrapper objects) caused all list views to fail. Fixed by aligning frontend types to bare arrays.
- 2025-10-31: Frontend types must strictly mirror backend DB structs. Field name mismatches (e.g. `last_synced_at` vs `updated_at`) cause silent UI failures.
- 2025-10-31: Compilation (tsc, go build) is necessary but NOT sufficient. Always run tests and validate endpoints.
