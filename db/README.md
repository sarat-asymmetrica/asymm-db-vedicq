# Vedic Qiskit Canonical DB

This folder contains canonical PostgreSQL migrations for the Vedic x Betanet integration.

## Layout

- `migrations/` ordered SQL migrations.

## Conventions

- PostgreSQL 14+ target.
- `0001` and `0002` establish baseline control-plane and policy seed.
- New migrations must be additive and ordered.
- Prefer schema-qualified references (for example `control_plane.users`).

## Apply (example)

```bash
psql "$DATABASE_URL" -f db/migrations/0001_control_plane_canonical.sql
psql "$DATABASE_URL" -f db/migrations/0002_policy_seed_baselines.sql
```

## Canonical Sources of Truth

- Control plane identity/governance metadata: `control_plane`.
- Policy sets and rules: `authz`.
- Decision and event lineage tables: `telemetry` and `authz` (subsequent migrations).
