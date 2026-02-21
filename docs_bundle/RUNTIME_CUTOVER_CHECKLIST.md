# Runtime Cutover Checklist

Last Updated: 2026-02-21

## Scope

This checklist gates production cutover for `asymm-db-vedicq-runtime`.

## R1 Service Stabilization

1. `platform_runtime serve` is default command in container image.
2. Liveness endpoint `/livez` returns `200`.
3. Readiness endpoint `/readyz` returns `200` only when DB health check passes.
4. Startup logs include non-secret config summary (bind addr, db host, pool sizing, statement timeout).
5. Platform probe configuration:
   - initial delay: `10s`
   - period: `10s`
   - timeout: `3s`
   - failure threshold: `3`

## R2 Runtime API v1

1. Decision write endpoint deployed: `POST /v1/decisions`.
2. Telemetry write endpoint deployed: `POST /v1/telemetry/events`.
3. `X-Request-ID` required on all mutating endpoints.
4. `Idempotency-Key` required on all mutating endpoints.
5. Duplicate idempotent requests with matching payload return cached response.
6. Mutating endpoints require `Authorization: Bearer <token>` or `X-API-Key`.
7. `RUNTIME_API_TOKENS` is set and rotated via platform secrets.

## R3 Jobs vs Service

1. Migration command run as one-shot job:
   - `dbctl migrate up --dir /app/db/migrations`
2. Self-check run as one-shot job:
   - `platform_runtime selfcheck`
3. Long-running service command:
   - `platform_runtime serve`

## R4 Postgres Posture

1. Runtime pool env configured:
   - `DB_MAX_OPEN_CONNS`
   - `DB_MAX_IDLE_CONNS`
   - `DB_STATEMENT_TIMEOUT_MS`
2. Least-privilege roles provisioned from `db/bootstrap/runtime_roles.sql`.
3. Runtime credentials use writer role; analytics/reporting uses reader role.
4. CORS allowlist configured via `RUNTIME_ALLOWED_ORIGINS` (no wildcard).
5. Runtime rate limit configured (`RUNTIME_RATE_LIMIT_PER_MINUTE`, `RUNTIME_RATE_LIMIT_BURST`).

## R5 Cutover Evidence

1. Execute:
   - `./scripts/run_live_cutover_matrix.ps1`
2. Store generated evidence log path under `evidence/`.
3. Confirm:
   - migrations applied
   - selfcheck passed
   - integration matrix passed
4. Perform rollback drill in staging and attach run notes.
