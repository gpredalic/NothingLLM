# Commit Convention

NothingLLM uses [Conventional Commits](https://www.conventionalcommits.org/).

## Format

```
<type>: <description>

[optional body]

[optional footer(s)]
```

## Types

| Type | Usage |
|------|-------|
| `feat` | New feature |
| `fix` | Bug fix |
| `refactor` | Code change that neither fixes a bug nor adds a feature |
| `docs` | Documentation only |
| `test` | Adding or updating tests |
| `build` | Build system, dependencies, Docker images |
| `ci` | CI/CD configuration |
| `chore` | Maintenance (cleanup, config, non-code) |
| `release` | Version tag preparation |
| `perf` | Performance improvement |
| `style` | Formatting, whitespace (no logic change) |

## Examples

```
feat: add workspace management
fix: handle qdrant timeout in search
refactor: simplify rag pipeline context building
docs: update deployment guide
test: add auth integration tests
build: update docker image to go 1.24
ci: add frontend build step to pipeline
chore: remove unused imports
release: prepare v0.1.0-rc1
perf: batch qdrant collection queries
style: align struct field comments
```

## Breaking Changes

Append `!` after the type and explain in the footer:

```
feat!: change session cookie to SameSite=Strict

BREAKING CHANGE: existing sessions are invalidated.
Users must re-login after upgrade.
```

## Rules

- Subject line: imperative mood, no period, ≤ 72 chars
- Body: explain *what* and *why* (optional)
- Footer: `BREAKING CHANGE:`, `Refs:`, `Closes:`
- One logical change per commit
- No merge commits in PRs (use squash)

## Branch Naming

| Branch | Pattern |
|--------|---------|
| Main | `main` |
| Development | `develop` |
| Feature | `feature/<name>` |
| Release | `release/vX.Y.Z` |
| Hotfix | `hotfix/<name>` |