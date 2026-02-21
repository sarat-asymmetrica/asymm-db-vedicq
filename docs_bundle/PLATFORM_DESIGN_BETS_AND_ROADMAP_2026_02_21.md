# Platform Design Bets and Roadmap (AI-Native, Robust, User-Easy)

Date: 2026-02-21  
Scope: Build a first-party platform akin to Supabase, but optimized for user ease, robustness, mathematical novelty, and AI-native operations.

## 1. Product Thesis

We will build a unified data platform with:
1. Strong developer primitives (Postgres, auth, storage, APIs, events).
2. Clear non-expert UX (intent-first flows, safe defaults).
3. Deterministic, explainable decision systems powered by our mathematical substrate.
4. AI-augmented operations and DB engineering from day one.

Platform promise:
1. Easier than assembling infra manually.
2. More explainable and verifiable than generic backend platforms.
3. Safer by default for multi-tenant production workloads.

## 2. Where We Deliberately Diverge

## 2.1 User Ease Bets (Circumvent Common Friction)

1. Intent-first onboarding over tool-first onboarding.
2. Policy templates and simulators over raw RLS-first setup.
3. Guided migration pipeline with automatic preflight checks.
4. One unified project graph (auth, data, events, functions linked by lineage).
5. Built-in incident and cost visibility instead of optional add-ons.

## 2.2 Robustness Bets (Non-Negotiable)

1. Deny-by-default access model.
2. Migration safety gates (shadow apply, lock-time checks, rollback plans).
3. Immutable audit and decision traces.
4. Idempotent event processing via outbox/inbox primitives.
5. SLOs and failure-mode runbooks as product artifacts, not internal docs.

## 2.3 Novelty Bets (Math and Compute Advantage)

1. Deterministic feature lineage and replayable scoring.
2. Geometry-aware policy analytics for drift and anomaly detection.
3. Fast prefiltering from Vedic primitives to reduce compute load.
4. Decision reproducibility guarantees as a customer-visible capability.
5. Trace-linked causal chain: request -> policy -> score -> action.

## 3. AI-Native Platform Decisions

AI is not only an app feature; it is a platform primitive.

## 3.1 AI-Assisted Build and Operations

1. Migration copilot workflows for generation, linting, and risk checks.
2. Query copilot with bounded rewrite suggestions and EXPLAIN-based validation.
3. Automated schema diff review with policy/security checks.
4. Incident copilot that reasons from logs, traces, and SQL stats.
5. Replay copilot for postmortem and what-if rollout simulations.

## 3.2 AI Guardrails

1. AI-generated SQL never auto-applies in production without policy checks.
2. Require deterministic migration IDs and rollback scripts.
3. Human approval for schema-breaking changes.
4. Model outputs logged as typed events for auditability.
5. Reproducibility checks for all scoring and policy decisions.

## 3.3 AI as Product Capability

1. “Explain this denial” endpoint backed by decision traces.
2. “Propose least-privilege policy” endpoint with simulation preview.
3. “Investigate anomaly” workflow joining risk, policy, and telemetry.
4. “Cost/perf advisor” over tenant workloads and index plans.

## 4. Target Architecture

## 4.1 Control Plane

Responsibilities:
1. Org/project lifecycle.
2. Billing/metering.
3. Access control administration.
4. Region and compliance controls.
5. Upgrade and migration orchestration.

## 4.2 Data Plane

Responsibilities:
1. Tenant-facing database and APIs.
2. Auth token verification and policy enforcement.
3. Realtime/event processing.
4. Storage/object metadata integration.
5. Risk and decision compute services.

## 4.3 Canonical Schemas

Already established/underway:
1. `control_plane`
2. `authz`
3. `security`
4. `ledger`
5. `telemetry`
6. `vedic`
7. `ops`

External-facing naming policy:
1. Internal schema names may remain technical.
2. API and UI labels must stay generalized and domain-neutral.

## 5. Strategic Decision Matrix (Decide Early)

1. Tenant isolation default:
- Shared + strict RLS for baseline.
- Dedicated project/DB for higher tiers.

2. Auth strategy:
- Native auth baseline.
- Enterprise federation (OIDC/SAML) as tiered add-on.

3. API contract:
- Stable facade APIs with semantic versioning.
- SQL/RPC for advanced users behind explicit capability flags.

4. Region strategy:
- Region pinning for paid tiers.
- Data residency policy from control plane.

5. Compute model:
- Managed functions for extensibility.
- Asynchronous workers for heavy analytics and replay tasks.

## 6. Roadmap (Phased)

## Phase 0: Foundations (Done/In Progress)

Status:
1. Canonical migration chain established through `0009`.
2. Sprint cadence and execution logging in place.

Exit criteria:
1. Migration chain validates in CI.
2. Documentation maps schema to service responsibilities.

## Phase 1: Core Platform Runtime

Deliverables:
1. Postgres-backed security stores (revocation, key lifecycle, nonce state).
2. Repository layer and transactional patterns in `pkg/db`.
3. Decision persistence wiring from authz engine.

Exit criteria:
1. No in-memory/file security state in production paths.
2. Full integration tests for auth/token/revocation flow.

## Phase 2: User-Ease Layer

Deliverables:
1. Intent-based setup flows.
2. Policy templates and policy simulation UI/API.
3. “Why denied” and “why risky” explainability endpoints.

Exit criteria:
1. New tenant can deploy with minimal SQL knowledge.
2. Policy debugging works without manual trace reconstruction.

## Phase 3: AI-Native Operations

Deliverables:
1. Migration copilot + SQL safety linting.
2. Query advisor with benchmark loop.
3. Incident copilot and replay workflow.

Exit criteria:
1. AI suggestions are gated by deterministic checks.
2. Ops teams can resolve incidents from generated runbooks + traces.

## Phase 4: Novel Computation Productization

Deliverables:
1. Deterministic risk/decision replay service.
2. Geometry-aware policy drift detection.
3. Causal decision graph explorer.

Exit criteria:
1. Replay consistency above threshold for certified versions.
2. Drift alerts and anomaly investigations demonstrate operational value.

## Phase 5: Service Packaging and Monetization

Deliverables:
1. Tiered plans with metered limits and SLO differences.
2. Dedicated tenant isolation option.
3. Admin control plane for support and compliance workflows.

Exit criteria:
1. Billing, metering, and support operations are end-to-end functional.
2. Launch-ready documentation and runbooks complete.

## 7. 15-Minute Sprint Operating Model

Each sprint must produce:
1. A clear scoped change.
2. A validation result.
3. Timestamped log entries.
4. Next sprint candidate.

Sprint quality gates:
1. Migration or code compiles/tests.
2. Security impact noted.
3. Rollback or fallback path identified.
4. User-facing naming reviewed for generalization.

## 8. Risks and Mitigations

1. Risk: Over-complex first release.
Mitigation: Strict v1 scope, progressive exposure of advanced features.

2. Risk: AI-generated change regressions.
Mitigation: Mandatory lint/test/approval gates and shadow checks.

3. Risk: Multi-tenant leakage via policy gaps.
Mitigation: RLS template library, static checks, default deny policies.

4. Risk: Operational burden at scale.
Mitigation: Early investment in metering, observability, and replay automation.

5. Risk: Novel math seen as opaque.
Mitigation: Explainability and deterministic replay as first-class UX.

## 9. Success Metrics

User ease:
1. Time-to-first-deploy.
2. Percent of customers shipping without manual SQL.
3. Mean time to resolve auth/policy errors.

Robustness:
1. Migration failure rate.
2. Incident rate and MTTR.
3. RPO/RTO and restore drill pass rate.

Novelty impact:
1. Decision replay consistency.
2. Drift/anomaly detection precision and recall.
3. Investigation time reduction from causal linkage.

AI leverage:
1. Percent of changes authored with AI assistance and safely merged.
2. Copilot suggestion acceptance rate with no post-deploy regressions.
3. Reduction in manual DBA toil hours.

## 10. Immediate Execution Queue

1. Implement repository and service wiring for migrations `0003` to `0009`.
2. Add migration CI job to validate sequence, transaction guards, and dry-run checks.
3. Add first policy simulation endpoint with trace persistence.
4. Add first generalized user-facing risk endpoint backed by `vedic.fraud_scores`.
5. Add AI-assisted migration review pipeline with hard safety gates.

