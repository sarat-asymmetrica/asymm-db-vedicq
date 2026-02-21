-- Vedic x Betanet decision lineage scaffolding (v1)
-- Target: PostgreSQL 14+

BEGIN;

-- -------------------------------------------------------------------
-- Authorization decision persistence
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS authz.policy_decisions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID REFERENCES control_plane.tenants(id) ON DELETE SET NULL,
    workspace_id            UUID REFERENCES control_plane.workspaces(id) ON DELETE SET NULL,
    subject                 TEXT NOT NULL,
    session_id              TEXT,
    policy_set_id           UUID REFERENCES authz.policy_sets(id) ON DELETE SET NULL,
    policy_set_key          TEXT,
    tier                    TEXT,
    action                  TEXT NOT NULL,
    resource_ref            TEXT NOT NULL,
    allow                   BOOLEAN NOT NULL,
    reason_code             TEXT NOT NULL,
    matched_rule_id         TEXT,
    trace_hash              TEXT,
    context_json            JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT policy_decisions_tier_ck CHECK (tier IS NULL OR tier IN ('T1', 'T2', 'T3', 'T4'))
);

CREATE INDEX IF NOT EXISTS policy_decisions_tenant_created_idx
    ON authz.policy_decisions(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS policy_decisions_subject_created_idx
    ON authz.policy_decisions(subject, created_at DESC);

CREATE INDEX IF NOT EXISTS policy_decisions_trace_hash_idx
    ON authz.policy_decisions(trace_hash)
    WHERE trace_hash IS NOT NULL;

CREATE TABLE IF NOT EXISTS authz.policy_decision_trace_steps (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_decision_id      UUID NOT NULL REFERENCES authz.policy_decisions(id) ON DELETE CASCADE,
    step_order              INTEGER NOT NULL,
    rule_id                 TEXT NOT NULL,
    matched                 BOOLEAN NOT NULL,
    outcome                 TEXT NOT NULL,
    reason                  TEXT NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (policy_decision_id, step_order),
    CONSTRAINT policy_decision_trace_steps_order_ck CHECK (step_order >= 0),
    CONSTRAINT policy_decision_trace_steps_outcome_ck CHECK (outcome IN ('no_match', 'allow', 'deny'))
);

CREATE INDEX IF NOT EXISTS policy_decision_trace_steps_rule_idx
    ON authz.policy_decision_trace_steps(rule_id);

-- -------------------------------------------------------------------
-- Normalized security event stream
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS telemetry.security_events (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
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
    CONSTRAINT security_events_actor_type_ck CHECK (actor_type IN ('user', 'service', 'system')),
    CONSTRAINT security_events_severity_ck CHECK (severity IN ('debug', 'info', 'warn', 'error'))
);

CREATE INDEX IF NOT EXISTS security_events_tenant_created_idx
    ON telemetry.security_events(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS security_events_type_created_idx
    ON telemetry.security_events(event_type, created_at DESC);

-- -------------------------------------------------------------------
-- Generic lineage links across decisions, events, and resources
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS telemetry.event_links (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id                UUID NOT NULL REFERENCES telemetry.security_events(id) ON DELETE CASCADE,
    link_kind               TEXT NOT NULL, -- policy_decision, audit_event, resource, fraud_score, etc.
    linked_id               UUID NOT NULL,
    metadata_json           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (event_id, link_kind, linked_id)
);

CREATE INDEX IF NOT EXISTS event_links_kind_linked_idx
    ON telemetry.event_links(link_kind, linked_id);

COMMIT;
