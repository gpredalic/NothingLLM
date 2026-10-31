# RC1 Release Checklist

## Testing

- [ ] `go test ./...` passes (all packages)
- [ ] `scripts/test.sh` exits 0
- [ ] `docker compose build` succeeds
- [ ] `docker compose up -d` starts all 4 services
- [ ] All services report `healthy` within 60s

## Endpoints

- [ ] `GET /health` returns `{"status":"ok"}`
- [ ] `GET /ready` returns `{"status":"ready"}`
- [ ] `GET /version` returns version, git_commit, build_date
- [ ] `GET /metrics` returns users, workspaces, collections, active_sessions

## Functional

- [ ] Register creates first user as admin
- [ ] Login sets HttpOnly+Secure+SameSite=Strict cookie
- [ ] Workspace CRUD works (create, list, get, delete)
- [ ] Collection sync discovers Qdrant collections
- [ ] Collection assign/unassign to workspace works
- [ ] Search returns results from Qdrant
- [ ] Chat returns OpenAI response with context
- [ ] Logout invalidates session

## Resilience

- [ ] Qdrant down: API stays up, `/ready` returns 503
- [ ] OpenAI down: chat returns 502 with meaningful error
- [ ] No panics on any failure path
- [ ] Rate limiter returns 429 on excess

## Operations

- [ ] `backup.sh` produces valid postgres.dump
- [ ] `restore.sh` restores from backup
- [ ] `release.sh 0.1.0-rc1` completes all 7 steps
- [ ] Git tag `v0.1.0-rc1` created
- [ ] `VERSION` file contains `0.1.0-rc1`
- [ ] `CHANGELOG.md` has entry for `0.1.0-rc1`

## Security

- [ ] No `.env` files in git
- [ ] No secrets in committed code
- [ ] Cookies: HttpOnly, Secure, SameSite=Strict
- [ ] Input validation on all endpoints
- [ ] SQL uses parameterized queries (no string concat)

## Documentation

- [ ] `README.md` — quick start accurate
- [ ] `docs/PRODUCTION.md` — deployment guide
- [ ] `docs/OPERATIONS.md` — ops runbook
- [ ] `COMMIT_CONVENTION.md` — commit format
- [ ] `VERSIONING.md` — SemVer policy
- [ ] `.github/workflows/ci.yml` — CI pipeline

## Source Control

- [ ] `.gitignore` covers all artifacts
- [ ] `main` branch protected
- [ ] `develop` branch is integration target
- [ ] Conventional commits used
- [ ] No binaries in repository