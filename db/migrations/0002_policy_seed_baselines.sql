-- Vedic x Betanet canonical policy seed model (v1)
-- Target: PostgreSQL 14+

BEGIN;

CREATE TABLE IF NOT EXISTS authz.policy_sets (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_key        TEXT NOT NULL UNIQUE,
    tier              TEXT NOT NULL,
    version           INTEGER NOT NULL DEFAULT 1,
    display_name      TEXT NOT NULL,
    status            TEXT NOT NULL DEFAULT 'active',
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT policy_sets_tier_ck CHECK (tier IN ('T1', 'T2', 'T3', 'T4')),
    CONSTRAINT policy_sets_status_ck CHECK (status IN ('active', 'disabled'))
);

CREATE TABLE IF NOT EXISTS authz.policy_rules (
    id                UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_set_id     UUID NOT NULL REFERENCES authz.policy_sets(id) ON DELETE CASCADE,
    rule_id           TEXT NOT NULL,
    priority          INTEGER NOT NULL,
    effect            TEXT NOT NULL,
    action_patterns   JSONB NOT NULL DEFAULT '[]'::jsonb,
    resource_pattern  TEXT NOT NULL DEFAULT '*',
    required_scopes   JSONB NOT NULL DEFAULT '[]'::jsonb,
    allowed_methods   JSONB NOT NULL DEFAULT '[]'::jsonb,
    required_context  JSONB NOT NULL DEFAULT '{}'::jsonb,
    reason_code       TEXT NOT NULL,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (policy_set_id, rule_id),
    CONSTRAINT policy_rules_effect_ck CHECK (effect IN ('allow', 'deny'))
);

CREATE INDEX IF NOT EXISTS policy_rules_set_priority_idx
    ON authz.policy_rules(policy_set_id, priority, rule_id);

INSERT INTO authz.policy_sets (policy_key, tier, version, display_name, status)
VALUES
    ('baseline_t1_v1', 'T1', 1, 'Baseline Tier T1 v1', 'active'),
    ('baseline_t2_v1', 'T2', 1, 'Baseline Tier T2 v1', 'active'),
    ('baseline_t3_v1', 'T3', 1, 'Baseline Tier T3 v1', 'active'),
    ('baseline_t4_v1', 'T4', 1, 'Baseline Tier T4 v1', 'active')
ON CONFLICT (policy_key) DO NOTHING;

WITH sets AS (
    SELECT policy_key, id FROM authz.policy_sets
),
seed_rules AS (
    SELECT
        s.id AS policy_set_id,
        r.rule_id,
        r.priority,
        r.effect,
        r.action_patterns::jsonb AS action_patterns,
        r.resource_pattern,
        r.required_scopes::jsonb AS required_scopes,
        r.allowed_methods::jsonb AS allowed_methods,
        r.required_context::jsonb AS required_context,
        r.reason_code
    FROM sets s
    JOIN (
        VALUES
            ('baseline_t1_v1', 'allow.read', 100, 'allow', '["read:*"]', '*', '["read:*"]', '[]', '{}', 'policy.allow.read'),
            ('baseline_t1_v1', 'allow.write', 110, 'allow', '["write:*"]', '*', '["write:*"]', '[]', '{}', 'policy.allow.write'),
            ('baseline_t1_v1', 'allow.admin.t1', 200, 'allow', '["admin:*"]', '*', '["admin:*"]', '["passkey","mfa","password"]', '{}', 'policy.allow.admin.t1'),

            ('baseline_t2_v1', 'allow.read', 100, 'allow', '["read:*"]', '*', '["read:*"]', '[]', '{}', 'policy.allow.read'),
            ('baseline_t2_v1', 'allow.write', 110, 'allow', '["write:*"]', '*', '["write:*"]', '[]', '{}', 'policy.allow.write'),
            ('baseline_t2_v1', 'allow.admin.t2', 200, 'allow', '["admin:*"]', '*', '["admin:*"]', '["passkey","mfa"]', '{}', 'policy.allow.admin.t2'),

            ('baseline_t3_v1', 'allow.read', 100, 'allow', '["read:*"]', '*', '["read:*"]', '[]', '{}', 'policy.allow.read'),
            ('baseline_t3_v1', 'allow.write', 110, 'allow', '["write:*"]', '*', '["write:*"]', '[]', '{}', 'policy.allow.write'),
            ('baseline_t3_v1', 'allow.admin.t3.stepup', 200, 'allow', '["admin:*"]', '*', '["admin:*"]', '["passkey","mfa"]', '{"step_up":"true"}', 'policy.allow.admin.t3.stepup'),
            ('baseline_t3_v1', 'allow.txn.t3.stepup', 210, 'allow', '["txn:*"]', '*', '["txn:*"]', '["passkey","mfa"]', '{"step_up":"true"}', 'policy.allow.txn.t3.stepup'),

            ('baseline_t4_v1', 'allow.read', 100, 'allow', '["read:*"]', '*', '["read:*"]', '[]', '{}', 'policy.allow.read'),
            ('baseline_t4_v1', 'allow.write', 110, 'allow', '["write:*"]', '*', '["write:*"]', '[]', '{}', 'policy.allow.write'),
            ('baseline_t4_v1', 'allow.admin.t4.stepup', 200, 'allow', '["admin:*"]', '*', '["admin:*"]', '["passkey"]', '{"step_up":"true"}', 'policy.allow.admin.t4.stepup'),
            ('baseline_t4_v1', 'allow.txn.t4.stepup', 210, 'allow', '["txn:*"]', '*', '["txn:*"]', '["passkey"]', '{"step_up":"true"}', 'policy.allow.txn.t4.stepup')
    ) AS r(policy_key, rule_id, priority, effect, action_patterns, resource_pattern, required_scopes, allowed_methods, required_context, reason_code)
        ON r.policy_key = s.policy_key
)
INSERT INTO authz.policy_rules (
    policy_set_id,
    rule_id,
    priority,
    effect,
    action_patterns,
    resource_pattern,
    required_scopes,
    allowed_methods,
    required_context,
    reason_code
)
SELECT
    policy_set_id,
    rule_id,
    priority,
    effect,
    action_patterns,
    resource_pattern,
    required_scopes,
    allowed_methods,
    required_context,
    reason_code
FROM seed_rules
ON CONFLICT (policy_set_id, rule_id) DO NOTHING;

COMMIT;
