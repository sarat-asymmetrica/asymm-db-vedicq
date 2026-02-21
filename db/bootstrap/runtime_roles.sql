-- Runtime least-privilege role bootstrap.
-- Execute with a privileged account (not via app migrations).

BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'asymm_migrator') THEN
        CREATE ROLE asymm_migrator NOLOGIN;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'asymm_runtime_writer') THEN
        CREATE ROLE asymm_runtime_writer NOLOGIN;
    END IF;
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'asymm_runtime_reader') THEN
        CREATE ROLE asymm_runtime_reader NOLOGIN;
    END IF;
END $$;

-- Replace `YOUR_DB_NAME` before execution.
-- GRANT CONNECT ON DATABASE YOUR_DB_NAME TO asymm_migrator, asymm_runtime_writer, asymm_runtime_reader;

GRANT USAGE, CREATE ON SCHEMA control_plane, authz, security, ledger, telemetry, vedic, ops TO asymm_migrator;
GRANT USAGE ON SCHEMA control_plane, authz, security, ledger, telemetry, vedic, ops TO asymm_runtime_writer, asymm_runtime_reader;

GRANT SELECT, INSERT, UPDATE, DELETE ON ALL TABLES IN SCHEMA control_plane, authz, security, ledger, telemetry, vedic, ops TO asymm_runtime_writer;
GRANT SELECT ON ALL TABLES IN SCHEMA control_plane, authz, security, ledger, telemetry, vedic, ops TO asymm_runtime_reader;

GRANT USAGE, SELECT, UPDATE ON ALL SEQUENCES IN SCHEMA control_plane, authz, security, ledger, telemetry, vedic, ops TO asymm_runtime_writer;
GRANT USAGE, SELECT ON ALL SEQUENCES IN SCHEMA control_plane, authz, security, ledger, telemetry, vedic, ops TO asymm_runtime_reader;

ALTER DEFAULT PRIVILEGES IN SCHEMA control_plane, authz, security, ledger, telemetry, vedic, ops
GRANT SELECT, INSERT, UPDATE, DELETE ON TABLES TO asymm_runtime_writer;
ALTER DEFAULT PRIVILEGES IN SCHEMA control_plane, authz, security, ledger, telemetry, vedic, ops
GRANT SELECT ON TABLES TO asymm_runtime_reader;

COMMIT;
