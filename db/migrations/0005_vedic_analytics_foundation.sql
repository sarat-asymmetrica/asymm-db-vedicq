-- Vedic x Betanet analytics foundation (v1)
-- Target: PostgreSQL 14+

BEGIN;

-- -------------------------------------------------------------------
-- Model/version registry for reproducible scoring
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS vedic.model_versions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    model_key               TEXT NOT NULL,
    model_version           TEXT NOT NULL,
    model_family            TEXT NOT NULL DEFAULT 'risk_analytics',
    config_checksum         TEXT NOT NULL,
    config_json             JSONB NOT NULL DEFAULT '{}'::jsonb,
    status                  TEXT NOT NULL DEFAULT 'active',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (model_key, model_version),
    CONSTRAINT model_versions_status_ck CHECK (status IN ('active', 'deprecated', 'retired'))
);

CREATE INDEX IF NOT EXISTS model_versions_family_status_idx
    ON vedic.model_versions(model_family, status);

-- -------------------------------------------------------------------
-- Canonical transaction facts
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS vedic.transactions (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID REFERENCES control_plane.tenants(id) ON DELETE SET NULL,
    workspace_id            UUID REFERENCES control_plane.workspaces(id) ON DELETE SET NULL,
    resource_id             UUID REFERENCES control_plane.resources(id) ON DELETE SET NULL,
    external_txn_id         TEXT NOT NULL,
    source_system           TEXT NOT NULL DEFAULT 'unknown',
    event_time              TIMESTAMPTZ NOT NULL,
    txn_type                TEXT NOT NULL,
    amount                  NUMERIC(20, 4) NOT NULL,
    currency                TEXT NOT NULL DEFAULT 'USD',
    old_balance             NUMERIC(20, 4),
    new_balance             NUMERIC(20, 4),
    metadata_json           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, source_system, external_txn_id),
    CONSTRAINT transactions_amount_ck CHECK (amount >= 0)
);

CREATE INDEX IF NOT EXISTS transactions_tenant_event_time_idx
    ON vedic.transactions(tenant_id, event_time DESC);

CREATE INDEX IF NOT EXISTS transactions_type_event_time_idx
    ON vedic.transactions(txn_type, event_time DESC);

-- -------------------------------------------------------------------
-- Derived features (deterministic extraction)
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS vedic.transaction_features (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id          UUID NOT NULL REFERENCES vedic.transactions(id) ON DELETE CASCADE,
    model_version_id        UUID REFERENCES vedic.model_versions(id) ON DELETE SET NULL,
    amount_cents            BIGINT NOT NULL,
    digital_root            SMALLINT NOT NULL,
    is_round_amount         BOOLEAN NOT NULL,
    drain_ratio             NUMERIC(10, 6),
    amount_bucket           TEXT NOT NULL,
    feature_json            JSONB NOT NULL DEFAULT '{}'::jsonb,
    computed_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (transaction_id, model_version_id),
    CONSTRAINT transaction_features_digital_root_ck CHECK (digital_root BETWEEN 0 AND 9),
    CONSTRAINT transaction_features_drain_ratio_ck CHECK (drain_ratio IS NULL OR (drain_ratio >= 0 AND drain_ratio <= 1))
);

CREATE INDEX IF NOT EXISTS transaction_features_digital_root_idx
    ON vedic.transaction_features(digital_root);

CREATE INDEX IF NOT EXISTS transaction_features_amount_bucket_idx
    ON vedic.transaction_features(amount_bucket);

-- -------------------------------------------------------------------
-- Risk scoring outputs (user-facing naming remains generic)
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS vedic.fraud_scores (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    transaction_id          UUID NOT NULL REFERENCES vedic.transactions(id) ON DELETE CASCADE,
    model_version_id        UUID NOT NULL REFERENCES vedic.model_versions(id) ON DELETE RESTRICT,
    score_name              TEXT NOT NULL DEFAULT 'transaction_risk',
    risk_score              NUMERIC(6, 5) NOT NULL,
    risk_level              TEXT NOT NULL,
    reason_codes_json       JSONB NOT NULL DEFAULT '[]'::jsonb,
    explanation             TEXT,
    scored_at               TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (transaction_id, model_version_id, score_name),
    CONSTRAINT fraud_scores_risk_score_ck CHECK (risk_score >= 0 AND risk_score <= 1),
    CONSTRAINT fraud_scores_risk_level_ck CHECK (risk_level IN ('LOW', 'MEDIUM', 'HIGH', 'CRITICAL'))
);

CREATE INDEX IF NOT EXISTS fraud_scores_risk_level_scored_at_idx
    ON vedic.fraud_scores(risk_level, scored_at DESC);

CREATE INDEX IF NOT EXISTS fraud_scores_transaction_idx
    ON vedic.fraud_scores(transaction_id);

COMMIT;
