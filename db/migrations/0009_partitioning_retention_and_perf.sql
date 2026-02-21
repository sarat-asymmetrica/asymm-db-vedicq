-- Vedic x Betanet partitioning, retention, and performance support (v1)
-- Target: PostgreSQL 14+

BEGIN;

-- -------------------------------------------------------------------
-- Partition-ready event table (future ingestion target)
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS telemetry.security_events_partitioned (
    id                      UUID NOT NULL DEFAULT gen_random_uuid(),
    tenant_id               UUID REFERENCES control_plane.tenants(id) ON DELETE SET NULL,
    workspace_id            UUID REFERENCES control_plane.workspaces(id) ON DELETE SET NULL,
    actor_type              TEXT NOT NULL,
    actor_id                UUID,
    event_type              TEXT NOT NULL,
    severity                TEXT NOT NULL DEFAULT 'info',
    message                 TEXT NOT NULL,
    trace_hash              TEXT,
    event_json              JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (id, created_at),
    CONSTRAINT security_events_partitioned_actor_type_ck CHECK (actor_type IN ('user', 'service', 'system')),
    CONSTRAINT security_events_partitioned_severity_ck CHECK (severity IN ('debug', 'info', 'warn', 'error'))
) PARTITION BY RANGE (created_at);

CREATE TABLE IF NOT EXISTS telemetry.security_events_partitioned_default
    PARTITION OF telemetry.security_events_partitioned DEFAULT;

CREATE INDEX IF NOT EXISTS security_events_partitioned_tenant_created_idx
    ON telemetry.security_events_partitioned(tenant_id, created_at DESC);

-- -------------------------------------------------------------------
-- Retention and archival policy metadata
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS ops.retention_policies (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    table_schema            TEXT NOT NULL,
    table_name              TEXT NOT NULL,
    retention_days          INTEGER NOT NULL,
    archive_schema          TEXT,
    archive_table           TEXT,
    active                  BOOLEAN NOT NULL DEFAULT true,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (table_schema, table_name),
    CONSTRAINT retention_policies_days_ck CHECK (retention_days > 0)
);

CREATE TABLE IF NOT EXISTS ops.retention_runs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_id               UUID NOT NULL REFERENCES ops.retention_policies(id) ON DELETE CASCADE,
    started_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at             TIMESTAMPTZ,
    deleted_rows            BIGINT NOT NULL DEFAULT 0,
    archived_rows           BIGINT NOT NULL DEFAULT 0,
    status                  TEXT NOT NULL DEFAULT 'running',
    details_json            JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT retention_runs_status_ck CHECK (status IN ('running', 'completed', 'failed'))
);

CREATE INDEX IF NOT EXISTS retention_runs_policy_started_idx
    ON ops.retention_runs(policy_id, started_at DESC);

-- -------------------------------------------------------------------
-- Performance indexes and read models
-- -------------------------------------------------------------------

CREATE INDEX IF NOT EXISTS policy_decisions_action_created_idx
    ON authz.policy_decisions(action, created_at DESC);

CREATE INDEX IF NOT EXISTS policy_decisions_allow_created_idx
    ON authz.policy_decisions(allow, created_at DESC);

CREATE INDEX IF NOT EXISTS transactions_source_event_idx
    ON vedic.transactions(source_system, event_time DESC);

CREATE INDEX IF NOT EXISTS fraud_scores_scored_at_idx
    ON vedic.fraud_scores(scored_at DESC);

CREATE INDEX IF NOT EXISTS fraud_scores_model_level_idx
    ON vedic.fraud_scores(model_version_id, risk_level);

CREATE MATERIALIZED VIEW IF NOT EXISTS vedic.mv_transaction_risk_daily AS
SELECT
    date_trunc('day', fs.scored_at) AS day_bucket,
    t.tenant_id,
    fs.risk_level,
    count(*) AS score_count,
    avg(fs.risk_score) AS avg_risk_score
FROM vedic.fraud_scores fs
JOIN vedic.transactions t ON t.id = fs.transaction_id
GROUP BY 1, 2, 3;

CREATE INDEX IF NOT EXISTS mv_transaction_risk_daily_day_tenant_idx
    ON vedic.mv_transaction_risk_daily(day_bucket, tenant_id);

CREATE MATERIALIZED VIEW IF NOT EXISTS authz.mv_policy_decision_daily AS
SELECT
    date_trunc('day', created_at) AS day_bucket,
    tenant_id,
    allow,
    count(*) AS decision_count
FROM authz.policy_decisions
GROUP BY 1, 2, 3;

CREATE INDEX IF NOT EXISTS mv_policy_decision_daily_day_tenant_idx
    ON authz.mv_policy_decision_daily(day_bucket, tenant_id);

COMMIT;
