-- Vedic x Betanet operational outbox/replay scaffolding (v1)
-- Target: PostgreSQL 14+

BEGIN;

-- -------------------------------------------------------------------
-- Transactional outbox
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS ops.outbox (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    aggregate_type          TEXT NOT NULL,
    aggregate_id            UUID,
    event_type              TEXT NOT NULL,
    payload_json            JSONB NOT NULL,
    headers_json            JSONB NOT NULL DEFAULT '{}'::jsonb,
    dedupe_key              TEXT,
    status                  TEXT NOT NULL DEFAULT 'pending',
    available_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at            TIMESTAMPTZ,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT outbox_status_ck CHECK (status IN ('pending', 'processing', 'processed', 'failed'))
);

CREATE UNIQUE INDEX IF NOT EXISTS outbox_dedupe_key_uq
    ON ops.outbox(dedupe_key)
    WHERE dedupe_key IS NOT NULL;

CREATE INDEX IF NOT EXISTS outbox_status_available_idx
    ON ops.outbox(status, available_at);

CREATE TABLE IF NOT EXISTS ops.outbox_attempts (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    outbox_id               UUID NOT NULL REFERENCES ops.outbox(id) ON DELETE CASCADE,
    attempt_no              INTEGER NOT NULL,
    processor_name          TEXT NOT NULL,
    success                 BOOLEAN NOT NULL,
    error_code              TEXT,
    error_message           TEXT,
    attempted_at            TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (outbox_id, attempt_no),
    CONSTRAINT outbox_attempts_attempt_no_ck CHECK (attempt_no > 0)
);

CREATE INDEX IF NOT EXISTS outbox_attempts_outbox_attempted_idx
    ON ops.outbox_attempts(outbox_id, attempted_at DESC);

-- -------------------------------------------------------------------
-- Inbox + idempotency for inbound processing
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS ops.inbox (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    source_system           TEXT NOT NULL,
    source_message_id       TEXT NOT NULL,
    message_type            TEXT NOT NULL,
    payload_json            JSONB NOT NULL,
    received_at             TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at            TIMESTAMPTZ,
    status                  TEXT NOT NULL DEFAULT 'received',
    UNIQUE (source_system, source_message_id),
    CONSTRAINT inbox_status_ck CHECK (status IN ('received', 'processing', 'processed', 'failed', 'ignored'))
);

CREATE INDEX IF NOT EXISTS inbox_status_received_idx
    ON ops.inbox(status, received_at);

CREATE TABLE IF NOT EXISTS ops.idempotency_keys (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    idempotency_key         TEXT NOT NULL UNIQUE,
    scope                   TEXT NOT NULL,
    request_hash            TEXT NOT NULL,
    response_code           INTEGER,
    response_json           JSONB,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at              TIMESTAMPTZ NOT NULL,
    CONSTRAINT idempotency_keys_scope_ck CHECK (length(trim(scope)) > 0),
    CONSTRAINT idempotency_keys_expiry_ck CHECK (expires_at > created_at)
);

CREATE INDEX IF NOT EXISTS idempotency_keys_expires_idx
    ON ops.idempotency_keys(expires_at);

-- -------------------------------------------------------------------
-- Replay and migration control
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS ops.replay_checkpoints (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    replay_name             TEXT NOT NULL,
    partition_key           TEXT NOT NULL DEFAULT 'default',
    last_cursor             TEXT,
    last_event_time         TIMESTAMPTZ,
    status                  TEXT NOT NULL DEFAULT 'active',
    metadata_json           JSONB NOT NULL DEFAULT '{}'::jsonb,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (replay_name, partition_key),
    CONSTRAINT replay_checkpoints_status_ck CHECK (status IN ('active', 'paused', 'completed', 'failed'))
);

CREATE TABLE IF NOT EXISTS ops.migration_runs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    migration_key           TEXT NOT NULL,
    started_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at             TIMESTAMPTZ,
    status                  TEXT NOT NULL DEFAULT 'running',
    details_json            JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT migration_runs_status_ck CHECK (status IN ('running', 'completed', 'failed', 'rolled_back'))
);

CREATE INDEX IF NOT EXISTS migration_runs_key_started_idx
    ON ops.migration_runs(migration_key, started_at DESC);

COMMIT;
