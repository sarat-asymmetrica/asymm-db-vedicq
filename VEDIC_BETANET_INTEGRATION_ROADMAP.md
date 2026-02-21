# Vedic Qiskit x Betanet Integration Roadmap (Canonical PostgreSQL Edition)

Last Updated: 2026-02-21
Target Workspace: `vedic_qiskit/`
Scope: Merge Betanet security/control-plane strengths with Vedic-Qiskit math engines into one production-grade Postgres-first system.

## 1. Executive Direction

The previous roadmap over-weighted security hardening. This revision makes PostgreSQL the canonical system of record for:
- Identity, tenancy, auth, policy, consent, and audit lineage.
- Runtime security state (revocation, key lifecycle, nonce watermarks).
- Vedic analytics state (digital-root features, fraud scores, quaternion signatures, collatz trajectories).
- Cross-domain resource governance and traceability.

Design principle:
- Betanet provides control-plane rigor and security semantics.
- Vedic-Qiskit provides deterministic, high-throughput mathematical feature extraction.
- Postgres unifies both as the canonical data model and decision ledger.

## 2. Current-State Engine Audit

### 2.1 Vedic-Qiskit (today)

Core engines available and production-ready at package level:
- `pkg/vedic`: digital root and Williams batching primitives.
- `pkg/fraud`: deterministic fraud scoring engine.
- `pkg/quaternion`: S3 quaternion math, SLERP, geodesic distance.
- `pkg/collatz`: sequence modeling and S3 navigation abstractions.
- `pkg/lattice`: W-state and fractal structures (currently compute-side, no canonical persistence).

Observed gap:
- No canonical DB schema for identity/auth/policy/audit or for Vedic feature lineage.
- Most outputs are in-memory/CLI/demo oriented.

### 2.2 Betanet (today)

Strong security/control-plane primitives already exist:
- Auth claims/token/policy engines with deterministic traces.
- Keyring + revocation interfaces (current implementations are in-memory).
- Node hardening and nonce persistence.
- Alias ledger and payment registry data models.
- Mesh/relay/defense telemetry and accounting primitives.
- Postgres baseline migrations already define: identity/tenancy/resource/access/audit + policy seed tables.

Observed gap:
- Runtime stores still memory/file based in places (revocation, keyring, nonce state, some ledgers).
- Vedic math analytics are not represented in canonical schema.

## 3. Target Canonical Data Platform

## 3.1 Database Role

Postgres is authoritative for:
- Control plane and security policy.
- Security and authorization decision logs.
- Vedic-derived analytical features and model outputs.
- Integration state (ingest checkpoints, outbox, replay, backfill).

## 3.2 Logical Schemas

Create these namespaces:
- `control_plane`: users, tenants, workspaces, memberships, apps, installations, resources, grants, consents, API credentials, automations.
- `authz`: policy sets/rules, policy versions, policy decisions, decision traces.
- `security`: key metadata, key rotations, token revocations, session revocations, nonce reservations, threat markers, anomaly incidents.
- `ledger`: alias records, alias endorsements, payment registries, registry endorsements, voucher metadata, accounting records.
- `telemetry`: normalized security events, relay/mesh metrics, defense metrics, ingestion logs.
- `vedic`: digital-root features, fraud features, fraud scores, quaternion signatures, collatz traces, lattice snapshots.
- `ops`: migration metadata, outbox/inbox, replay checkpoints, retention jobs, audit controls.

## 3.3 Canonical Modeling Rules

- UUID primary keys for global entities, deterministic hashes for immutable trace payloads.
- `created_at` and `updated_at` on mutable tables.
- Partition high-volume event/score tables by time (`created_at` monthly).
- Use `jsonb` only for extension fields, not core query keys.
- Enforce data contracts via check constraints and enum-like lookup tables.
- All writes that trigger downstream behavior go through transactional outbox rows.

## 4. Canonical Tables to Add (beyond Betanet baseline)

## 4.1 Security Runtime Persistence

- `security.signing_keys`
- `security.signing_key_versions`
- `security.revoked_tokens`
- `security.revoked_sessions`
- `security.nonce_watermarks`
- `security.threat_level_history`
- `security.anomaly_reports`

Purpose:
- Replace in-memory revocation/keyring and file nonce state with transactional, queryable, auditable state.

## 4.2 Ledger and Payment Canonicalization

- `ledger.alias_records`
- `ledger.alias_endorsements`
- `ledger.alias_maintainer_sets`
- `ledger.payment_registry_records`
- `ledger.payment_registry_endorsements`
- `ledger.voucher_keysets`
- `ledger.forwarding_accounting`

Purpose:
- Promote Betanet alias/payment/accounting semantics into durable, queryable canonical storage.

## 4.3 Vedic Analytics Schema

- `vedic.transactions` (normalized input facts)
- `vedic.transaction_features` (digital root, roundness, drain ratio, bucket)
- `vedic.fraud_scores` (risk score, risk level, versioned scorer config)
- `vedic.quaternion_signatures` (w,x,y,z, freq, phase, norm_error)
- `vedic.policy_geometry` (decision quaternion embeddings and distances)
- `vedic.collatz_runs` (input n, classical_steps, s3_steps, speedup)
- `vedic.collatz_trace_points` (optional compressed path points)
- `vedic.lattice_snapshots` (optional, sparse snapshot for diagnostics)
- `vedic.model_versions` (immutable versioned config + checksum)

Purpose:
- Make Vedic outputs reproducible, explainable, and joinable with control-plane/audit events.

## 4.4 Decision and Audit Lineage

- `authz.policy_decisions`
- `authz.policy_decision_trace_steps`
- `telemetry.security_events`
- `telemetry.event_links` (event-to-decision/event-to-resource/event-to-score)

Purpose:
- Close the loop from request -> policy evaluation -> Vedic scoring -> final action.

## 5. Integration Architecture in `vedic_qiskit`

Add DB-centric modules:
- `pkg/db/`: connection, tx helpers, migration runner hooks, repository contracts.
- `pkg/controlplane/`: Betanet-compatible entity services backed by Postgres.
- `pkg/security/`: keyring/revocation/nonce services backed by Postgres.
- `pkg/authz/`: policy evaluation + decision persistence + trace hashing.
- `pkg/vedicdb/`: feature extraction persistence and analytical read models.
- `pkg/outbox/`: transactional outbox dispatcher.
- `cmd/vedic_control_plane/`: API service for auth/policy/audit operations.
- `cmd/vedic_analytics/`: streaming or batch scorer service writing `vedic.*` tables.
- `db/migrations/`: canonical migration chain owned by `vedic_qiskit`.

## 6. Migration and Cutover Plan

## Phase A: Baseline Canonical Schema

- Import/adapt Betanet `0001` and `0002` into `vedic_qiskit/db/migrations` as baseline.
- Add schema namespaces and naming conventions.
- Create compatibility views for legacy table names if needed.

Exit criteria:
- Baseline control-plane and policy seed tables migrate cleanly in empty DB.

## Phase B: Runtime Security Persistence

- Implement Postgres-backed keyring/revocation/nonce stores.
- Keep in-memory adapters behind interfaces for tests.
- Add rotation and revocation cleanup jobs.

Exit criteria:
- Auth flows pass with DB-backed revocation and key lookup only.

## Phase C: Vedic Analytics Canonicalization

- Persist transaction inputs and derived Vedic features.
- Version scorer configs and write immutable score records.
- Add materialized views for common fraud/risk dashboards.

Exit criteria:
- Deterministic replay yields identical feature and score rows for same model version.

## Phase D: Decision Lineage and Joins

- Persist policy decisions + trace steps + trace hashes.
- Link decisions to scores/resources/actors.
- Add investigator queries and indexed paths.

Exit criteria:
- Single query can reconstruct why an action was allowed/denied.

## Phase E: Dual-Write + Backfill + Cutover

- Optional dual-write from legacy stores during transition.
- Run backfill jobs with checksums and row-count reconciliations.
- Flip reads to canonical Postgres.

Exit criteria:
- Parity checks pass; legacy persistence paths disabled.

## 7. Full DB Concerns (Required)

## 7.1 Reliability and HA

- Primary + replica deployment with monitored replication lag.
- PITR backups and quarterly restore drills.
- Migration safety: preflight checks, lock-time budgets, rollback scripts.

## 7.2 Security

- TLS in transit, encryption at rest.
- Separate app roles: migrator, writer, reader, analyst, auditor.
- RLS for tenant-scoped tables where multi-tenant exposure exists.
- Secret/key material stored encrypted or referenced through secure KMS integration.

## 7.3 Performance

- Index strategy driven by actual join/filter paths.
- Partitioning for `telemetry.*`, `authz.policy_decisions`, `vedic.fraud_scores`.
- Retention and archive policies by table class.
- Use `EXPLAIN (ANALYZE, BUFFERS)` gates for critical queries.

## 7.4 Operability

- SLOs: write latency, read latency, replication lag, deadlock rate.
- DB observability: query stats, lock contention, bloat, vacuum health.
- Runbooks for failover, replay, corruption checks, and emergency key revocation.

## 8. Validation Matrix

Functional:
- Auth issue/verify/refresh/revoke with DB-backed state.
- Policy evaluation determinism against seeded baselines.
- Vedic scoring determinism and reproducibility by model version.
- Full lineage join from request to decision to score to audit event.

Non-functional:
- Race-safe behavior under concurrent token issuance/revocation.
- Benchmarks on high-volume inserts for telemetry and scores.
- Recovery tests for crash at migration boundaries.
- Security regression for injection, privilege escalation, and replay.

## 9. Delivery Work Breakdown and Estimates

Planning formula used:
`Estimated Time = (Simple x 1.5) + (Medium x 6) + (Complex x 8) + (Docs x 2.5) minutes`

Estimated feature mix for this roadmap delivery:
- Simple: 6 (config, wiring, migration scaffolding)
- Medium: 10 (repositories, services, read models)
- Complex: 11 (schema redesign, backfill, cutover, HA/security controls)
- Docs: 6 (runbooks, data contracts, migration guides)

Estimated total:
- `(6 x 1.5) + (10 x 6) + (11 x 8) + (6 x 2.5) = 172 minutes` (~2.9 hours active implementation time, excluding long-running backfills/soak tests).

Recommended execution cadence:
- Batch related features in groups of 3-5 changes.
- Commit at each phase boundary (A-E).
- Keep migration and rollback artifacts in the same PR as schema changes.

## 10. Immediate Next Implementation Steps

1. Create `vedic_qiskit/db/migrations/0001` and `0002` from Betanet baseline with namespace normalization.
2. Add `pkg/security/postgres_revocation_store.go`, `pkg/security/postgres_keyring.go`, `pkg/security/postgres_nonce_store.go`.
3. Add `vedic` schema migrations and typed repositories for feature/score persistence.
4. Implement policy decision persistence and trace linkage tables.
5. Add backfill + reconciliation scripts and an operator cutover checklist.
6. Add race/perf/integration tests and restore-drill scripts before production cutover.

## 11. `asymm-db-vedicq-runtime` Creation Path (Current Next Steps)

Status:
- `platform_runtime selfcheck` is successful as a one-shot command.
- `platform_runtime serve` is now the long-running default container command.

Next milestones:
1. Milestone R1: Service baseline stabilization.
   - Add `/livez`, `/readyz`, and structured startup logs with config echo (non-secret).
   - Add health probe settings to deployment docs (initial delay, timeout, period).
2. Milestone R2: Runtime API v1 scaffold.
   - Add HTTP endpoints for decision write path and telemetry ingest write path.
   - Enforce request IDs and idempotency keys for mutating calls.
3. Milestone R3: Operator jobs split.
   - Keep `dbctl migrate up` and `platform_runtime selfcheck` as one-shot job commands.
   - Keep `platform_runtime serve` as service command only.
4. Milestone R4: Postgres production posture.
   - Add least-privilege DB roles (`migrator`, `runtime_writer`, `runtime_reader`).
   - Add connection pool limits and statement timeout defaults.
5. Milestone R5: Cutover readiness.
   - Run integration matrix against live Postgres.
   - Capture rollback drill and data parity evidence before broader traffic.

## 11. Definition of Done

The integration is complete when:
- Postgres is canonical for control-plane + security runtime + Vedic analytics.
- In-memory/file security state is removed from production paths.
- Decision, audit, and Vedic score lineage is queryable end-to-end.
- Backfill reconciliation and restore drills pass.
- Performance/security SLO gates are met and documented.
