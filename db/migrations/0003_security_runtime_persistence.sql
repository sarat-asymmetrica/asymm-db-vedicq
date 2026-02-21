-- Vedic x Betanet runtime security persistence (v1)
-- Target: PostgreSQL 14+

BEGIN;

-- -------------------------------------------------------------------
-- Signing keys and rotation metadata
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS security.signing_keys (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    key_scope           TEXT NOT NULL, -- auth_access, auth_refresh, service, etc.
    algorithm           TEXT NOT NULL, -- HS256, Ed25519, hybrid refs
    status              TEXT NOT NULL DEFAULT 'active',
    source              TEXT NOT NULL DEFAULT 'internal',
    kms_key_ref         TEXT,
    metadata_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    deactivated_at      TIMESTAMPTZ,
    CONSTRAINT signing_keys_status_ck CHECK (status IN ('active', 'retired', 'revoked')),
    CONSTRAINT signing_keys_scope_ck CHECK (length(trim(key_scope)) > 0),
    CONSTRAINT signing_keys_algorithm_ck CHECK (length(trim(algorithm)) > 0)
);

CREATE INDEX IF NOT EXISTS signing_keys_scope_status_idx
    ON security.signing_keys(key_scope, status);

CREATE TABLE IF NOT EXISTS security.signing_key_versions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    signing_key_id      UUID NOT NULL REFERENCES security.signing_keys(id) ON DELETE CASCADE,
    key_id              TEXT NOT NULL, -- external kid presented in tokens
    version             INTEGER NOT NULL,
    key_hash            TEXT NOT NULL, -- hash/fingerprint, no plaintext key material
    is_current          BOOLEAN NOT NULL DEFAULT false,
    valid_from          TIMESTAMPTZ NOT NULL DEFAULT now(),
    valid_until         TIMESTAMPTZ,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (signing_key_id, version),
    UNIQUE (key_id),
    CONSTRAINT signing_key_versions_version_ck CHECK (version > 0)
);

CREATE UNIQUE INDEX IF NOT EXISTS signing_key_versions_current_uq
    ON security.signing_key_versions(signing_key_id)
    WHERE is_current = true;

CREATE INDEX IF NOT EXISTS signing_key_versions_validity_idx
    ON security.signing_key_versions(valid_from, valid_until);

-- -------------------------------------------------------------------
-- Token/session revocation durability
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS security.revoked_tokens (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    token_id            TEXT NOT NULL UNIQUE,
    session_id          TEXT,
    reason_code         TEXT NOT NULL DEFAULT 'revoked',
    revoked_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ NOT NULL,
    metadata_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT revoked_tokens_token_id_ck CHECK (length(trim(token_id)) > 0),
    CONSTRAINT revoked_tokens_expiry_ck CHECK (expires_at > revoked_at)
);

CREATE INDEX IF NOT EXISTS revoked_tokens_expiry_idx
    ON security.revoked_tokens(expires_at);

CREATE INDEX IF NOT EXISTS revoked_tokens_session_idx
    ON security.revoked_tokens(session_id)
    WHERE session_id IS NOT NULL;

CREATE TABLE IF NOT EXISTS security.revoked_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id          TEXT NOT NULL UNIQUE,
    reason_code         TEXT NOT NULL DEFAULT 'revoked',
    revoked_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    expires_at          TIMESTAMPTZ NOT NULL,
    metadata_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
    CONSTRAINT revoked_sessions_session_id_ck CHECK (length(trim(session_id)) > 0),
    CONSTRAINT revoked_sessions_expiry_ck CHECK (expires_at > revoked_at)
);

CREATE INDEX IF NOT EXISTS revoked_sessions_expiry_idx
    ON security.revoked_sessions(expires_at);

-- -------------------------------------------------------------------
-- Monotonic nonce reservation state
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS security.nonce_watermarks (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    node_name           TEXT NOT NULL,
    nonce_scope         TEXT NOT NULL DEFAULT 'default',
    reserved_high       BIGINT NOT NULL,
    window_size         BIGINT NOT NULL,
    updated_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    metadata_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
    UNIQUE (node_name, nonce_scope),
    CONSTRAINT nonce_watermarks_reserved_high_ck CHECK (reserved_high >= 0),
    CONSTRAINT nonce_watermarks_window_size_ck CHECK (window_size > 0)
);

CREATE INDEX IF NOT EXISTS nonce_watermarks_updated_idx
    ON security.nonce_watermarks(updated_at DESC);

COMMIT;
