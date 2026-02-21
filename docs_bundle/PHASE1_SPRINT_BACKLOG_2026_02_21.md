# Phase 1 Sprint Backlog (Execution-Ready)

Date: 2026-02-21  
Source plan: `docs/PLATFORM_DESIGN_BETS_AND_ROADMAP_2026_02_21.md`  
Current migration baseline: `db/migrations/0001` through `0009`

## 1. Scope

Phase 1 objective:
1. Wire runtime services to canonical Postgres schemas.
2. Remove production dependence on in-memory/file security persistence.
3. Establish integration and validation pipelines for safe rollout.

Out of scope:
1. Full end-user UX flows.
2. Advanced AI copilot features (planned for later phases).

## 2. Estimation Method

Formula:
`Estimated Time = (Simple x 1.5) + (Medium x 6) + (Complex x 8) + (Docs x 2.5) minutes`

Per-ticket estimates below use:
1. `simple` for config/scaffolding/small wiring.
2. `medium` for feature modules and tests.
3. `complex` for integration behavior and migration-sensitive runtime changes.
4. `docs` for runbooks/spec updates.

## 3. Ticket Backlog

## P1-001: Postgres DB Core Package

- Priority: P0
- Owner: AI Agent
- Type: medium
- Estimate: 6 min
- Dependencies: none
- Deliverables:
1. `pkg/db/config.go` (connection config)
2. `pkg/db/connect.go` (pool open/health)
3. `pkg/db/tx.go` (transaction helpers)
- Acceptance criteria:
1. Package compiles.
2. Connection health check is unit-testable.
3. No hardcoded secrets/DSNs.

## P1-002: Migration Runner Integration

- Priority: P0
- Owner: AI Agent
- Type: medium
- Estimate: 6 min
- Dependencies: P1-001
- Deliverables:
1. `pkg/db/migrate.go` with deterministic migration discovery.
2. CLI hook in `cmd` for migrate up/status.
- Acceptance criteria:
1. Migrations `0001-0009` discoverable in order.
2. Dry-run mode supported.
3. Fails fast on sequence gaps.

## P1-003: Postgres Revocation Store

- Priority: P0
- Owner: AI Agent
- Type: complex
- Estimate: 8 min
- Dependencies: P1-001, migration `0003`
- Deliverables:
1. `pkg/security/postgres_revocation_store.go`
2. Interface compatibility with existing revocation store contract.
- Acceptance criteria:
1. Token/session revoke + query parity with memory store behavior.
2. Expired records handled correctly.
3. Concurrency-safe under race test.

## P1-004: Postgres Keyring Resolver

- Priority: P0
- Owner: AI Agent
- Type: complex
- Estimate: 8 min
- Dependencies: P1-001, migration `0003`
- Deliverables:
1. `pkg/security/postgres_keyring.go`
2. `Current()` and `Lookup()` mapped to `security.signing_key_versions`.
- Acceptance criteria:
1. Deterministic current key resolution.
2. Unknown key behavior matches existing contract.
3. Rotation-safe for overlapping validity windows.

## P1-005: Postgres Nonce Watermark Store

- Priority: P0
- Owner: AI Agent
- Type: medium
- Estimate: 6 min
- Dependencies: P1-001, migration `0003`
- Deliverables:
1. `pkg/security/postgres_nonce_store.go`
2. Adapter compatible with `harden.NonceCounter` callback pattern.
- Acceptance criteria:
1. Atomic reserve updates.
2. No nonce regression under parallel calls.
3. Startup reads persisted watermark.

## P1-006: Auth Service Wiring to Postgres Stores

- Priority: P0
- Owner: AI Agent
- Type: complex
- Estimate: 8 min
- Dependencies: P1-003, P1-004, P1-005
- Deliverables:
1. Service constructor wiring updates.
2. Config flags to toggle memory vs Postgres for non-prod.
- Acceptance criteria:
1. Issue/verify/refresh/revoke uses Postgres in primary path.
2. Existing auth tests pass with Postgres-backed fixtures.
3. No API contract break.

## P1-007: Policy Decision Persistence Layer

- Priority: P1
- Owner: AI Agent
- Type: complex
- Estimate: 8 min
- Dependencies: P1-001, migration `0004`
- Deliverables:
1. `pkg/authz/repository.go` for `authz.policy_decisions`.
2. Trace-step persistence to `authz.policy_decision_trace_steps`.
- Acceptance criteria:
1. Every evaluated decision can be persisted with ordered trace steps.
2. Trace hash is stored and queryable.
3. Write failures return clear errors.

## P1-008: Security Event Normalization Writer

- Priority: P1
- Owner: AI Agent
- Type: medium
- Estimate: 6 min
- Dependencies: P1-001, migration `0004`
- Deliverables:
1. `pkg/telemetry/repository.go` for `telemetry.security_events` + `event_links`.
2. Mapping from existing `telemetry.SecurityEvent` to canonical DB schema.
- Acceptance criteria:
1. Event inserts with severity/type validation.
2. Linking to decisions/resources supported.
3. Backpressure/retry behavior defined for writer failures.

## P1-009: Analytics Foundation Repository

- Priority: P1
- Owner: AI Agent
- Type: complex
- Estimate: 8 min
- Dependencies: P1-001, migration `0005`
- Deliverables:
1. `pkg/analytics/repository.go` for `transactions`, `transaction_features`, `fraud_scores`, `model_versions`.
2. Idempotent transaction ingestion API by `(tenant, source_system, external_txn_id)`.
- Acceptance criteria:
1. Duplicate ingestion is handled safely.
2. Score writes enforce model-version relation.
3. User-facing score label defaults to generalized values.

## P1-010: Migration CI and Validation Pipeline

- Priority: P0
- Owner: AI Agent
- Type: medium
- Estimate: 6 min
- Dependencies: P1-002, existing `scripts/validate_migrations.ps1`
- Deliverables:
1. CI job to run migration validator.
2. Optional ephemeral Postgres apply check in CI.
- Acceptance criteria:
1. PR fails on invalid migration sequence/guards.
2. CI logs show migration validation result clearly.
3. Pipeline can be run locally with same command set.

## P1-011: Integration Test Matrix (Phase 1)

- Priority: P0
- Owner: AI Agent
- Type: complex
- Estimate: 8 min
- Dependencies: P1-003 through P1-009
- Deliverables:
1. Integration tests for auth, revocation, nonce watermark, policy persistence, event writing.
2. Race test targets for concurrent token and nonce operations.
- Acceptance criteria:
1. Tests run green in CI.
2. Failure outputs identify schema/table context.
3. Baseline performance checks defined for hot paths.

## P1-012: Rollout Runbook + Rollback Plan

- Priority: P0
- Owner: AI Agent
- Type: docs
- Estimate: 2.5 min
- Dependencies: P1-003 through P1-011
- Deliverables:
1. `docs/PHASE1_ROLLOUT_RUNBOOK.md`
2. Rollback sequence for Postgres-backed security store switch.
- Acceptance criteria:
1. Step-by-step production switch and rollback documented.
2. Includes verification queries and health checks.
3. Includes incident fallback toggles.

## 4. Recommended Sprint Sequence

Sprint A (P0 Core Foundation):
1. P1-001
2. P1-002
3. P1-010

Sprint B (Security Runtime Cutover):
1. P1-003
2. P1-004
3. P1-005
4. P1-006

Sprint C (Decision + Event Lineage):
1. P1-007
2. P1-008

Sprint D (Analytics Wiring):
1. P1-009

Sprint E (Hardening and Launch Readiness):
1. P1-011
2. P1-012

## 5. Phase 1 Definition of Done

1. Auth runtime uses Postgres-backed key/revocation/nonce persistence by default.
2. Policy decisions and traces persist to canonical tables.
3. Security events and lineage links persist to canonical tables.
4. Analytics foundation tables are wired for deterministic ingest and scoring writes.
5. Migration validation and integration tests run in CI.
6. Rollout and rollback runbooks are complete and reviewed.

## 6. Immediate Next Ticket to Execute

Start with:
1. P1-001 `Postgres DB Core Package`
2. P1-002 `Migration Runner Integration`
3. P1-010 `Migration CI and Validation Pipeline`

