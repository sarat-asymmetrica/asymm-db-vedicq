# Phase 1 Rollout Runbook (Postgres Security Runtime Cutover)

Date: 2026-02-21

## 1. Objective

Safely switch runtime security dependencies from memory/file persistence to canonical Postgres-backed components:
1. Token/session revocation (`security.revoked_tokens`, `security.revoked_sessions`).
2. Key resolution (`security.signing_keys`, `security.signing_key_versions`).
3. Nonce persistence (`security.nonce_watermarks`).

## 2. Preconditions

1. Migrations `0001` through `0009` applied successfully.
2. `go run ./cmd/dbctl migrate validate --dir db/migrations` passes.
3. `./scripts/validate_migrations.ps1` passes.
4. Application has connectivity to target Postgres with required privileges.
5. Current signing key rows are seeded in `security.signing_keys` and `security.signing_key_versions`.

## 3. Rollout Steps

1. Deploy code that includes `pkg/security` Postgres adapters and runtime wiring.
2. Enable Postgres security runtime configuration in staging.
3. Run smoke checks:
- issue token
- verify token
- revoke token/session
- reserve/load nonce watermark
4. Validate decision and event persistence paths in staging.
5. Promote configuration to production with canary traffic.
6. Monitor error rate and auth latency for 30+ minutes.
7. Complete full production cutover.

## 4. Verification Queries

Check key resolver seed:

```sql
SELECT key_id, is_current, valid_from, valid_until
FROM security.signing_key_versions
ORDER BY valid_from DESC;
```

Check revocation writes:

```sql
SELECT token_id, revoked_at, expires_at
FROM security.revoked_tokens
ORDER BY revoked_at DESC
LIMIT 20;
```

Check nonce watermark progression:

```sql
SELECT node_name, nonce_scope, reserved_high, window_size, updated_at
FROM security.nonce_watermarks
ORDER BY updated_at DESC
LIMIT 20;
```

Check lineage writes:

```sql
SELECT created_at, subject, action, allow, reason_code
FROM authz.policy_decisions
ORDER BY created_at DESC
LIMIT 20;
```

## 5. Rollback Plan

Trigger rollback if:
1. Auth failures exceed threshold.
2. Token verification failures correlate with key lookup failures.
3. Nonce reservation errors appear in runtime logs.

Rollback sequence:

1. Switch runtime config back to memory/file adapters.
2. Keep writes to Postgres disabled (or best-effort) during rollback window.
3. Restart affected services to clear stale resolver state.
4. Confirm auth flows recover.
5. Preserve DB records for postmortem.

Post-rollback:
1. Export error logs and failed request IDs.
2. Validate integrity of `security.*` tables.
3. Rehearse staging fix and rerun smoke checks before reattempt.

## 6. Incident Fallback Toggles

Recommended runtime toggles:
1. `SECURITY_RUNTIME_MODE=memory|postgres`
2. `SECURITY_KEY_RESOLVER_MODE=memory|postgres`
3. `SECURITY_REVOCATION_MODE=memory|postgres`
4. `SECURITY_NONCE_STORE_MODE=file|postgres`

## 7. Completion Criteria

1. Postgres-backed security runtime enabled in production.
2. No auth regression against baseline SLA.
3. Revocation and nonce persistence verified through queries.
4. Rollback path validated in staging and documented.
