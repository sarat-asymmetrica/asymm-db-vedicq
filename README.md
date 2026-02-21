# Vedic Platform Experiments

This repository is a focused extraction from `asymm_all_math/vedic_qiskit` for deployment experiments (for example Sliplane).

## Included Scope

- `cmd/dbctl`: migration validation/status/up.
- `cmd/platform_runtime`: runtime selfcheck entrypoint.
- `pkg/db`, `pkg/security`, `pkg/authz`, `pkg/telemetry`, `pkg/analytics`, `pkg/platform`.
- `db/migrations` (`0001` to `0009`) and migration scripts.
- `integration/` Phase 1 schema matrix tests (env-gated).
- `docs_bundle/` strategy/runbook/backlog docs.

## Quick Start

1. Validate migrations:
```bash
go run ./cmd/dbctl migrate validate --dir db/migrations
```
2. Run tests:
```bash
go test ./pkg/... ./integration -v
```
3. Runtime selfcheck (requires `DATABASE_URL`):
```bash
go run ./cmd/platform_runtime selfcheck
```
4. Run runtime service:
```bash
go run ./cmd/platform_runtime serve --host 0.0.0.0 --port 8080
```

## Docker

Build:
```bash
docker build -t vedic-platform-experiments:local .
```

Default container command:
```bash
platform_runtime serve
```

Run migrations via command override:
```bash
docker run --rm -e DATABASE_URL="$DATABASE_URL" vedic-platform-experiments:local \
  dbctl migrate up --dir /app/db/migrations
```

## Sliplane Deployment Notes

1. Deploy PostgreSQL as a private service first.
2. Deploy this app service with `DATABASE_URL` using internal hostname.
3. First run migration command override (`dbctl migrate up --dir /app/db/migrations`).
4. Optionally run one-shot selfcheck command override (`platform_runtime selfcheck`).
5. Keep default service command as long-running runtime (`platform_runtime serve`).

## Next Step

If you choose a final GitHub repo URL, update `go.mod` module path accordingly.
