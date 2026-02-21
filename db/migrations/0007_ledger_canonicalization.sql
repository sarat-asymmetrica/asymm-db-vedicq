-- Vedic x Betanet ledger canonicalization (v1)
-- Target: PostgreSQL 14+

BEGIN;

-- -------------------------------------------------------------------
-- Alias maintainer sets and records
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS ledger.alias_maintainer_sets (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID REFERENCES control_plane.tenants(id) ON DELETE SET NULL,
    set_ref                 TEXT NOT NULL,
    sequence_no             BIGINT NOT NULL,
    min_required_signers    SMALLINT NOT NULL DEFAULT 2,
    maintainers_json        JSONB NOT NULL DEFAULT '[]'::jsonb,
    status                  TEXT NOT NULL DEFAULT 'active',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, set_ref, sequence_no),
    CONSTRAINT alias_maintainer_sets_sequence_ck CHECK (sequence_no > 0),
    CONSTRAINT alias_maintainer_sets_required_ck CHECK (min_required_signers > 0),
    CONSTRAINT alias_maintainer_sets_status_ck CHECK (status IN ('active', 'superseded', 'retired'))
);

CREATE INDEX IF NOT EXISTS alias_maintainer_sets_tenant_created_idx
    ON ledger.alias_maintainer_sets(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS ledger.alias_records (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID REFERENCES control_plane.tenants(id) ON DELETE SET NULL,
    alias_maintainer_set_id UUID REFERENCES ledger.alias_maintainer_sets(id) ON DELETE SET NULL,
    alias_name              TEXT NOT NULL,
    node_id                 BYTEA NOT NULL,
    sequence_no             BIGINT NOT NULL,
    record_timestamp_ms     BIGINT NOT NULL,
    digest_hash             TEXT NOT NULL,
    status                  TEXT NOT NULL DEFAULT 'active',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, alias_name, sequence_no),
    CONSTRAINT alias_records_sequence_ck CHECK (sequence_no > 0),
    CONSTRAINT alias_records_timestamp_ck CHECK (record_timestamp_ms > 0),
    CONSTRAINT alias_records_status_ck CHECK (status IN ('active', 'superseded', 'expired'))
);

CREATE INDEX IF NOT EXISTS alias_records_tenant_alias_idx
    ON ledger.alias_records(tenant_id, alias_name);

CREATE TABLE IF NOT EXISTS ledger.alias_endorsements (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    alias_record_id         UUID NOT NULL REFERENCES ledger.alias_records(id) ON DELETE CASCADE,
    signer_fingerprint      BYTEA NOT NULL,
    signature_algorithm     TEXT NOT NULL DEFAULT 'hybrid',
    signature_blob          BYTEA NOT NULL,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (alias_record_id, signer_fingerprint)
);

CREATE INDEX IF NOT EXISTS alias_endorsements_alias_record_idx
    ON ledger.alias_endorsements(alias_record_id);

-- -------------------------------------------------------------------
-- Payment registry canonical records
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS ledger.payment_registry_records (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID REFERENCES control_plane.tenants(id) ON DELETE SET NULL,
    registry_type           TEXT NOT NULL DEFAULT 'PaymentRegistry',
    keyset_id               BYTEA NOT NULL,
    curve_id                TEXT NOT NULL,
    public_key_blob         BYTEA NOT NULL,
    schema_version          INTEGER NOT NULL,
    sequence_no             BIGINT NOT NULL,
    record_timestamp_ms     BIGINT NOT NULL,
    digest_hash             TEXT NOT NULL,
    status                  TEXT NOT NULL DEFAULT 'active',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, keyset_id, sequence_no),
    CONSTRAINT payment_registry_records_schema_ck CHECK (schema_version > 0),
    CONSTRAINT payment_registry_records_sequence_ck CHECK (sequence_no > 0),
    CONSTRAINT payment_registry_records_timestamp_ck CHECK (record_timestamp_ms > 0),
    CONSTRAINT payment_registry_records_status_ck CHECK (status IN ('active', 'superseded', 'revoked'))
);

CREATE INDEX IF NOT EXISTS payment_registry_records_tenant_created_idx
    ON ledger.payment_registry_records(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS ledger.payment_registry_endorsements (
    id                          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_registry_record_id  UUID NOT NULL REFERENCES ledger.payment_registry_records(id) ON DELETE CASCADE,
    signer_fingerprint          BYTEA NOT NULL,
    signature_algorithm         TEXT NOT NULL DEFAULT 'hybrid',
    signature_blob              BYTEA NOT NULL,
    created_at                  TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (payment_registry_record_id, signer_fingerprint)
);

CREATE INDEX IF NOT EXISTS payment_registry_endorsements_record_idx
    ON ledger.payment_registry_endorsements(payment_registry_record_id);

-- -------------------------------------------------------------------
-- Voucher keysets and forwarding accounting
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS ledger.voucher_keysets (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID REFERENCES control_plane.tenants(id) ON DELETE SET NULL,
    keyset_ref              TEXT NOT NULL,
    keyset_id               BYTEA NOT NULL,
    curve_id                TEXT NOT NULL,
    public_key_blob         BYTEA NOT NULL,
    description             TEXT,
    schema_version          INTEGER NOT NULL,
    status                  TEXT NOT NULL DEFAULT 'active',
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, keyset_ref),
    UNIQUE (tenant_id, keyset_id),
    CONSTRAINT voucher_keysets_schema_ck CHECK (schema_version > 0),
    CONSTRAINT voucher_keysets_status_ck CHECK (status IN ('active', 'deprecated', 'retired'))
);

CREATE TABLE IF NOT EXISTS ledger.forwarding_accounting (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID REFERENCES control_plane.tenants(id) ON DELETE SET NULL,
    relay_node_id           BYTEA NOT NULL,
    stream_id               BIGINT NOT NULL,
    bytes_forwarded         BIGINT NOT NULL,
    accounted_at_ms         BIGINT NOT NULL,
    signature_blob          BYTEA,
    metadata_json           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT forwarding_accounting_stream_ck CHECK (stream_id >= 0),
    CONSTRAINT forwarding_accounting_bytes_ck CHECK (bytes_forwarded >= 0),
    CONSTRAINT forwarding_accounting_time_ck CHECK (accounted_at_ms > 0)
);

CREATE INDEX IF NOT EXISTS forwarding_accounting_tenant_created_idx
    ON ledger.forwarding_accounting(tenant_id, created_at DESC);

CREATE INDEX IF NOT EXISTS forwarding_accounting_stream_idx
    ON ledger.forwarding_accounting(stream_id);

COMMIT;
