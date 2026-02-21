package platform

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	analyticsrepo "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/analytics"
	authzrepo "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/authz"
	dbpkg "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/db"
	securityrepo "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/security"
	telemetryrepo "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/telemetry"
)

// Runtime wires all Phase 1 Postgres-backed repositories into one service context.
type Runtime struct {
	DB            *sql.DB
	Security      *securityrepo.RuntimeDeps
	AuthzRepo     *authzrepo.Repository
	TelemetryRepo *telemetryrepo.Repository
	AnalyticsRepo *analyticsrepo.Repository
}

// BuildPhase1Runtime opens a DB connection and wires repositories.
func BuildPhase1Runtime(ctx context.Context, dbCfg dbpkg.Config, secCfg securityrepo.RuntimeConfig) (*Runtime, error) {
	conn, err := dbpkg.Open(ctx, dbCfg)
	if err != nil {
		return nil, err
	}

	secDeps, err := securityrepo.BuildPostgresRuntime(conn, secCfg)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	authzRepo, err := authzrepo.NewRepository(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	tRepo, err := telemetryrepo.NewRepository(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}
	aRepo, err := analyticsrepo.NewRepository(conn)
	if err != nil {
		_ = conn.Close()
		return nil, err
	}

	return &Runtime{
		DB:            conn,
		Security:      secDeps,
		AuthzRepo:     authzRepo,
		TelemetryRepo: tRepo,
		AnalyticsRepo: aRepo,
	}, nil
}

// Close releases the database handle.
func (r *Runtime) Close() error {
	if r == nil || r.DB == nil {
		return nil
	}
	return r.DB.Close()
}

// HealthCheck verifies DB connectivity.
func (r *Runtime) HealthCheck(ctx context.Context, timeout time.Duration) error {
	if r == nil || r.DB == nil {
		return fmt.Errorf("platform: runtime db not initialized")
	}
	return dbpkg.HealthCheck(ctx, r.DB, timeout)
}

// RecordDecisionAndEvent writes an auth decision plus linked security event.
func (r *Runtime) RecordDecisionAndEvent(
	ctx context.Context,
	decision authzrepo.DecisionRecord,
	trace []authzrepo.TraceStep,
	event telemetryrepo.SecurityEventRecord,
) (string, string, error) {
	if r == nil || r.AuthzRepo == nil || r.TelemetryRepo == nil {
		return "", "", fmt.Errorf("platform: runtime repositories not initialized")
	}
	decisionID, err := r.AuthzRepo.PersistDecisionWithTrace(ctx, decision, trace)
	if err != nil {
		return "", "", err
	}
	eventID, err := r.TelemetryRepo.PersistSecurityEventWithLinks(
		ctx,
		event,
		[]telemetryrepo.EventLink{
			{
				LinkKind: "policy_decision",
				LinkedID: decisionID,
			},
		},
	)
	if err != nil {
		return decisionID, "", err
	}
	return decisionID, eventID, nil
}
