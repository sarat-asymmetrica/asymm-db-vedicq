package integration

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/db"
)

// Phase 1 integration matrix:
// 1. migration tables and canonical schemas exist.
// 2. security runtime tables from 0003 are available.
// 3. decision lineage tables from 0004 are available.
// 4. analytics foundation tables from 0005 are available.
//
// This suite is environment-gated and skips unless INTEGRATION_DATABASE_URL is set.

func TestPhase1SchemaMatrix(t *testing.T) {
	conn := openIntegrationDB(t)
	defer conn.Close()

	required := []struct {
		schema string
		table  string
	}{
		{"security", "revoked_tokens"},
		{"security", "revoked_sessions"},
		{"security", "nonce_watermarks"},
		{"authz", "policy_decisions"},
		{"authz", "policy_decision_trace_steps"},
		{"telemetry", "security_events"},
		{"telemetry", "event_links"},
		{"vedic", "model_versions"},
		{"vedic", "transactions"},
		{"vedic", "transaction_features"},
		{"vedic", "fraud_scores"},
		{"ops", "schema_migrations"},
	}

	for _, tt := range required {
		t.Run(tt.schema+"."+tt.table, func(t *testing.T) {
			if !tableExists(t, conn, tt.schema, tt.table) {
				t.Fatalf("missing table %s.%s", tt.schema, tt.table)
			}
		})
	}
}

func openIntegrationDB(t *testing.T) *sql.DB {
	t.Helper()
	url := os.Getenv("INTEGRATION_DATABASE_URL")
	if url == "" {
		t.Skip("INTEGRATION_DATABASE_URL not set; skipping integration tests")
	}
	cfg := db.Config{
		DriverName:      firstNonEmpty(os.Getenv("INTEGRATION_DB_DRIVER"), "postgres"),
		DatabaseURL:     url,
		ConnectTimeout:  5 * time.Second,
		MaxIdleConns:    2,
		MaxOpenConns:    4,
		ConnMaxIdleTime: 2 * time.Minute,
		ConnMaxLifetime: 10 * time.Minute,
	}
	ctx := context.Background()
	conn, err := db.Open(ctx, cfg)
	if err != nil {
		t.Fatalf("open integration db: %v", err)
	}
	return conn
}

func tableExists(t *testing.T, conn *sql.DB, schema, table string) bool {
	t.Helper()
	var exists bool
	err := conn.QueryRow(
		`SELECT EXISTS (
		   SELECT 1
		     FROM information_schema.tables
		    WHERE table_schema = $1
		      AND table_name = $2
		)`,
		schema,
		table,
	).Scan(&exists)
	if err != nil {
		t.Fatalf("table existence query failed for %s.%s: %v", schema, table, err)
	}
	return exists
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}
