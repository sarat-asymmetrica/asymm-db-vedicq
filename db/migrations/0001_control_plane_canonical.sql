-- Vedic x Betanet canonical control-plane schema (v1)
-- Target: PostgreSQL 14+

BEGIN;

CREATE EXTENSION IF NOT EXISTS pgcrypto;

CREATE SCHEMA IF NOT EXISTS control_plane;
CREATE SCHEMA IF NOT EXISTS authz;
CREATE SCHEMA IF NOT EXISTS security;
CREATE SCHEMA IF NOT EXISTS ledger;
CREATE SCHEMA IF NOT EXISTS telemetry;
CREATE SCHEMA IF NOT EXISTS vedic;
CREATE SCHEMA IF NOT EXISTS ops;

-- -------------------------------------------------------------------
-- Identity and tenancy
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS control_plane.users (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    display_name      TEXT NOT NULL,
    primary_phone     TEXT,
    primary_email     TEXT,
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT users_status_ck CHECK (status IN ('active', 'suspended', 'deleted'))
);

CREATE UNIQUE INDEX IF NOT EXISTS users_primary_phone_uq
    ON control_plane.users(primary_phone) WHERE primary_phone IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS users_primary_email_uq
    ON control_plane.users(primary_email) WHERE primary_email IS NOT NULL;

CREATE TABLE IF NOT EXISTS control_plane.identities (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id           UUID NOT NULL REFERENCES control_plane.users(id) ON DELETE CASCADE,
    provider          TEXT NOT NULL,
    provider_subject  TEXT NOT NULL,
    verified_at       TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (provider, provider_subject)
);

CREATE TABLE IF NOT EXISTS control_plane.tenants (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug              TEXT NOT NULL UNIQUE,
    name              TEXT NOT NULL,
    plan              TEXT NOT NULL DEFAULT 'free',
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT tenants_status_ck CHECK (status IN ('active', 'suspended', 'deleted'))
);

CREATE TABLE IF NOT EXISTS control_plane.workspaces (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES control_plane.tenants(id) ON DELETE CASCADE,
    slug              TEXT NOT NULL,
    name              TEXT NOT NULL,
    workspace_type    TEXT NOT NULL DEFAULT 'default',
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, slug),
    CONSTRAINT workspaces_status_ck CHECK (status IN ('active', 'archived'))
);

CREATE TABLE IF NOT EXISTS control_plane.memberships (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES control_plane.tenants(id) ON DELETE CASCADE,
    workspace_id      UUID REFERENCES control_plane.workspaces(id) ON DELETE CASCADE,
    user_id           UUID NOT NULL REFERENCES control_plane.users(id) ON DELETE CASCADE,
    role              TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, workspace_id, user_id, role),
    CONSTRAINT memberships_status_ck CHECK (status IN ('active', 'invited', 'revoked'))
);

CREATE INDEX IF NOT EXISTS memberships_user_idx ON control_plane.memberships(user_id);
CREATE INDEX IF NOT EXISTS memberships_tenant_idx ON control_plane.memberships(tenant_id);

-- -------------------------------------------------------------------
-- Product/app installation layer
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS control_plane.apps (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    app_key           TEXT NOT NULL UNIQUE,
    display_name      TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT apps_status_ck CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE IF NOT EXISTS control_plane.app_installations (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES control_plane.tenants(id) ON DELETE CASCADE,
    workspace_id      UUID REFERENCES control_plane.workspaces(id) ON DELETE CASCADE,
    app_id            UUID NOT NULL REFERENCES control_plane.apps(id) ON DELETE RESTRICT,
    config_json       JSONB NOT NULL DEFAULT '{}'::jsonb,
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, workspace_id, app_id),
    CONSTRAINT app_installations_status_ck CHECK (status IN ('active', 'disabled'))
);

-- -------------------------------------------------------------------
-- Shared resource registry and governance
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS control_plane.resources (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES control_plane.tenants(id) ON DELETE CASCADE,
    workspace_id      UUID REFERENCES control_plane.workspaces(id) ON DELETE CASCADE,
    owner_user_id     UUID REFERENCES control_plane.users(id) ON DELETE SET NULL,
    source_app_id     UUID REFERENCES control_plane.apps(id) ON DELETE SET NULL,
    resource_type     TEXT NOT NULL,
    resource_ref      TEXT NOT NULL,
    classification    TEXT NOT NULL DEFAULT 'internal',
    metadata_json     JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, resource_type, resource_ref),
    CONSTRAINT resources_classification_ck CHECK (classification IN ('public', 'internal', 'restricted'))
);

CREATE INDEX IF NOT EXISTS resources_tenant_idx ON control_plane.resources(tenant_id);
CREATE INDEX IF NOT EXISTS resources_workspace_idx ON control_plane.resources(workspace_id);
CREATE INDEX IF NOT EXISTS resources_type_idx ON control_plane.resources(resource_type);

CREATE TABLE IF NOT EXISTS control_plane.resource_links (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES control_plane.tenants(id) ON DELETE CASCADE,
    src_resource_id   UUID NOT NULL REFERENCES control_plane.resources(id) ON DELETE CASCADE,
    dst_resource_id   UUID NOT NULL REFERENCES control_plane.resources(id) ON DELETE CASCADE,
    link_type         TEXT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (src_resource_id, dst_resource_id, link_type),
    CONSTRAINT resource_links_not_self_ck CHECK (src_resource_id <> dst_resource_id)
);

-- -------------------------------------------------------------------
-- Authorization and consent
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS control_plane.role_bindings (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES control_plane.tenants(id) ON DELETE CASCADE,
    workspace_id      UUID REFERENCES control_plane.workspaces(id) ON DELETE CASCADE,
    principal_type    TEXT NOT NULL,
    principal_id      UUID NOT NULL,
    role_name         TEXT NOT NULL,
    granted_by_user   UUID REFERENCES control_plane.users(id) ON DELETE SET NULL,
    expires_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, workspace_id, principal_type, principal_id, role_name),
    CONSTRAINT role_bindings_principal_type_ck CHECK (principal_type IN ('user', 'service', 'app_installation'))
);

CREATE TABLE IF NOT EXISTS control_plane.grants (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES control_plane.tenants(id) ON DELETE CASCADE,
    workspace_id      UUID REFERENCES control_plane.workspaces(id) ON DELETE CASCADE,
    principal_type    TEXT NOT NULL,
    principal_id      UUID NOT NULL,
    resource_id       UUID NOT NULL REFERENCES control_plane.resources(id) ON DELETE CASCADE,
    action            TEXT NOT NULL,
    effect            TEXT NOT NULL DEFAULT 'allow',
    condition_json    JSONB NOT NULL DEFAULT '{}'::jsonb,
    granted_by_user   UUID REFERENCES control_plane.users(id) ON DELETE SET NULL,
    expires_at        TIMESTAMPTZ,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (principal_type, principal_id, resource_id, action),
    CONSTRAINT grants_principal_type_ck CHECK (principal_type IN ('user', 'service', 'app_installation')),
    CONSTRAINT grants_effect_ck CHECK (effect IN ('allow', 'deny'))
);

CREATE INDEX IF NOT EXISTS grants_resource_idx ON control_plane.grants(resource_id);
CREATE INDEX IF NOT EXISTS grants_tenant_idx ON control_plane.grants(tenant_id);

CREATE TABLE IF NOT EXISTS control_plane.consents (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES control_plane.tenants(id) ON DELETE CASCADE,
    user_id             UUID NOT NULL REFERENCES control_plane.users(id) ON DELETE CASCADE,
    app_installation_id UUID NOT NULL REFERENCES control_plane.app_installations(id) ON DELETE CASCADE,
    scope               TEXT NOT NULL,
    status              TEXT NOT NULL DEFAULT 'granted',
    granted_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at          TIMESTAMPTZ,
    UNIQUE (user_id, app_installation_id, scope),
    CONSTRAINT consents_status_ck CHECK (status IN ('granted', 'revoked'))
);

-- -------------------------------------------------------------------
-- Integration/API access and automation
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS control_plane.api_credentials (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id           UUID NOT NULL REFERENCES control_plane.tenants(id) ON DELETE CASCADE,
    workspace_id        UUID REFERENCES control_plane.workspaces(id) ON DELETE CASCADE,
    app_installation_id UUID REFERENCES control_plane.app_installations(id) ON DELETE SET NULL,
    key_hash            TEXT NOT NULL UNIQUE,
    scopes_json         JSONB NOT NULL DEFAULT '[]'::jsonb,
    status              TEXT NOT NULL DEFAULT 'active',
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_used_at        TIMESTAMPTZ,
    CONSTRAINT api_credentials_status_ck CHECK (status IN ('active', 'revoked'))
);

CREATE TABLE IF NOT EXISTS control_plane.automation_flows (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID NOT NULL REFERENCES control_plane.tenants(id) ON DELETE CASCADE,
    workspace_id      UUID REFERENCES control_plane.workspaces(id) ON DELETE CASCADE,
    owner_user_id     UUID REFERENCES control_plane.users(id) ON DELETE SET NULL,
    flow_name         TEXT NOT NULL,
    definition_json   JSONB NOT NULL,
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT automation_flows_status_ck CHECK (status IN ('active', 'disabled'))
);

-- -------------------------------------------------------------------
-- Audit and decision trace lineage
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS telemetry.audit_events (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id         UUID REFERENCES control_plane.tenants(id) ON DELETE SET NULL,
    workspace_id      UUID REFERENCES control_plane.workspaces(id) ON DELETE SET NULL,
    actor_type        TEXT NOT NULL,
    actor_id          UUID,
    action            TEXT NOT NULL,
    target_type       TEXT NOT NULL,
    target_id         UUID,
    decision          TEXT,
    reason_code       TEXT,
    trace_hash        TEXT,
    event_json        JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT audit_actor_type_ck CHECK (actor_type IN ('user', 'service', 'system')),
    CONSTRAINT audit_decision_ck CHECK (decision IS NULL OR decision IN ('allow', 'deny', 'error'))
);

CREATE INDEX IF NOT EXISTS audit_events_tenant_created_idx ON telemetry.audit_events(tenant_id, created_at DESC);
CREATE INDEX IF NOT EXISTS audit_events_trace_hash_idx ON telemetry.audit_events(trace_hash);

COMMIT;
