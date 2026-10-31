# Versioning

NothingLLM follows [Semantic Versioning 2.0.0](https://semver.org/).

## Format

```
vMAJOR.MINOR.PATCH[-PRERELEASE]
```

## Components

| Component | Meaning |
|-----------|---------|
| `MAJOR` | Incompatible API changes, schema migrations that break old data |
| `MINOR` | New features, backwards-compatible additions |
| `PATCH` | Bug fixes, backwards-compatible patches |
| `PRERELEASE` | `rc1`, `rc2`, `beta1`, `alpha1` — unstable pre-release |

## Examples

```
v0.1.0-rc1   # First release candidate
v0.1.0-rc2   # Second release candidate (fixes from rc1)
v0.1.0       # Stable release
v0.1.1       # Bug fix
v0.2.0       # New feature (e.g., audit logging)
v1.0.0       # First stable API
```

## Pre-Release (0.x)

During 0.x development, MINOR bumps may include breaking changes.
This is standard SemVer for pre-1.0 software.

## Tagging

- Tags are created by `release.sh`
- Format: `v0.1.0`, `v0.1.0-rc1`
- Tags are annotated: `git tag -a v0.1.0 -m "Release 0.1.0"`
- Tags are immutable — do not move them

## Release Branches

- Created from `develop`: `release/v0.1.0`
- Only bug fixes and release prep on release branches
- Merged back to `main` and `develop` on completion

## Hotfixes

- Branch from the tagged release: `hotfix/session-expiry`
- Bump PATCH: `v0.1.0` → `v0.1.1`
- Merged to `main`, then back to `develop`

## Build Metadata

Injected at build time via `-ldflags`:
- `main.Version` — from `VERSION` file or `--build-arg`
- `main.GitCommit` — short SHA
- `main.BuildDate` — ISO 8601 timestamp

Exposed at `GET /version`.