-- Vedic x Betanet geometry and sequence persistence (v1)
-- Target: PostgreSQL 14+

BEGIN;

-- -------------------------------------------------------------------
-- Quaternion signatures and geometric artifacts
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS vedic.quaternion_signatures (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID REFERENCES control_plane.tenants(id) ON DELETE SET NULL,
    workspace_id            UUID REFERENCES control_plane.workspaces(id) ON DELETE SET NULL,
    signature_ref           TEXT NOT NULL,
    w                       DOUBLE PRECISION NOT NULL,
    x                       DOUBLE PRECISION NOT NULL,
    y                       DOUBLE PRECISION NOT NULL,
    z                       DOUBLE PRECISION NOT NULL,
    frequency_hz            DOUBLE PRECISION NOT NULL DEFAULT 7.83,
    phase_radians           DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    magnitude               DOUBLE PRECISION NOT NULL,
    norm_error              DOUBLE PRECISION NOT NULL DEFAULT 0.0,
    metadata_json           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, signature_ref),
    CONSTRAINT quaternion_signatures_magnitude_ck CHECK (magnitude >= 0),
    CONSTRAINT quaternion_signatures_norm_error_ck CHECK (norm_error >= 0)
);

CREATE INDEX IF NOT EXISTS quaternion_signatures_tenant_created_idx
    ON vedic.quaternion_signatures(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS vedic.policy_geometry (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    policy_decision_id      UUID NOT NULL REFERENCES authz.policy_decisions(id) ON DELETE CASCADE,
    quaternion_signature_id UUID REFERENCES vedic.quaternion_signatures(id) ON DELETE SET NULL,
    geodesic_distance       DOUBLE PRECISION NOT NULL,
    interpolation_t         DOUBLE PRECISION,
    geometry_label          TEXT NOT NULL DEFAULT 'decision_path',
    geometry_json           JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (policy_decision_id, geometry_label),
    CONSTRAINT policy_geometry_geodesic_ck CHECK (geodesic_distance >= 0),
    CONSTRAINT policy_geometry_interpolation_ck CHECK (interpolation_t IS NULL OR (interpolation_t >= 0 AND interpolation_t <= 1))
);

CREATE INDEX IF NOT EXISTS policy_geometry_distance_idx
    ON vedic.policy_geometry(geodesic_distance);

-- -------------------------------------------------------------------
-- Collatz run persistence and optional trace points
-- -------------------------------------------------------------------

CREATE TABLE IF NOT EXISTS vedic.collatz_runs (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id               UUID REFERENCES control_plane.tenants(id) ON DELETE SET NULL,
    workspace_id            UUID REFERENCES control_plane.workspaces(id) ON DELETE SET NULL,
    run_ref                 TEXT NOT NULL,
    input_n                 BIGINT NOT NULL,
    classical_steps         INTEGER NOT NULL,
    s3_steps                INTEGER NOT NULL,
    speedup_ratio           DOUBLE PRECISION,
    max_value               BIGINT,
    converged               BOOLEAN NOT NULL DEFAULT false,
    final_distance          DOUBLE PRECISION,
    run_json                JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (tenant_id, run_ref),
    CONSTRAINT collatz_runs_input_ck CHECK (input_n > 0),
    CONSTRAINT collatz_runs_classical_steps_ck CHECK (classical_steps >= 0),
    CONSTRAINT collatz_runs_s3_steps_ck CHECK (s3_steps >= 0),
    CONSTRAINT collatz_runs_speedup_ck CHECK (speedup_ratio IS NULL OR speedup_ratio >= 0),
    CONSTRAINT collatz_runs_final_distance_ck CHECK (final_distance IS NULL OR final_distance >= 0)
);

CREATE INDEX IF NOT EXISTS collatz_runs_tenant_created_idx
    ON vedic.collatz_runs(tenant_id, created_at DESC);

CREATE TABLE IF NOT EXISTS vedic.collatz_trace_points (
    id                      UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    collatz_run_id          UUID NOT NULL REFERENCES vedic.collatz_runs(id) ON DELETE CASCADE,
    step_index              INTEGER NOT NULL,
    sequence_domain         TEXT NOT NULL DEFAULT 's3', -- s3 or classical
    sequence_value          BIGINT,
    q_w                     DOUBLE PRECISION,
    q_x                     DOUBLE PRECISION,
    q_y                     DOUBLE PRECISION,
    q_z                     DOUBLE PRECISION,
    geodesic_distance       DOUBLE PRECISION,
    created_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (collatz_run_id, sequence_domain, step_index),
    CONSTRAINT collatz_trace_points_step_ck CHECK (step_index >= 0),
    CONSTRAINT collatz_trace_points_domain_ck CHECK (sequence_domain IN ('s3', 'classical')),
    CONSTRAINT collatz_trace_points_distance_ck CHECK (geodesic_distance IS NULL OR geodesic_distance >= 0)
);

CREATE INDEX IF NOT EXISTS collatz_trace_points_run_step_idx
    ON vedic.collatz_trace_points(collatz_run_id, step_index);

COMMIT;
