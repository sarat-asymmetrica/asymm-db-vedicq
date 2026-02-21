package authz

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// DecisionRecord represents a policy decision row for authz.policy_decisions.
type DecisionRecord struct {
	TenantID      *string
	WorkspaceID   *string
	Subject       string
	SessionID     *string
	PolicySetID   *string
	PolicySetKey  *string
	Tier          *string
	Action        string
	ResourceRef   string
	Allow         bool
	ReasonCode    string
	MatchedRuleID *string
	TraceHash     *string
	ContextJSON   []byte
}

// TraceStep represents an ordered trace step row for authz.policy_decision_trace_steps.
type TraceStep struct {
	StepOrder int
	RuleID    string
	Matched   bool
	Outcome   string
	Reason    string
}

// Repository persists authz decisions and trace steps.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("authz: nil db handle")
	}
	return &Repository{db: db}, nil
}

// PersistDecisionWithTrace stores a decision and its trace steps in one transaction.
func (r *Repository) PersistDecisionWithTrace(ctx context.Context, rec DecisionRecord, steps []TraceStep) (string, error) {
	if err := validateDecisionRecord(rec); err != nil {
		return "", err
	}
	if err := validateTraceSteps(steps); err != nil {
		return "", err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	decisionID, err := insertDecision(ctx, tx, rec)
	if err != nil {
		return "", err
	}
	for _, s := range steps {
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO authz.policy_decision_trace_steps
			 (policy_decision_id, step_order, rule_id, matched, outcome, reason)
			 VALUES ($1, $2, $3, $4, $5, $6)`,
			decisionID,
			s.StepOrder,
			s.RuleID,
			s.Matched,
			s.Outcome,
			s.Reason,
		); err != nil {
			return "", err
		}
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return decisionID, nil
}

func insertDecision(ctx context.Context, tx *sql.Tx, rec DecisionRecord) (string, error) {
	ctxJSON := rec.ContextJSON
	if len(ctxJSON) == 0 {
		ctxJSON = []byte("{}")
	}

	var decisionID string
	err := tx.QueryRowContext(
		ctx,
		`INSERT INTO authz.policy_decisions
		 (tenant_id, workspace_id, subject, session_id, policy_set_id, policy_set_key, tier,
		  action, resource_ref, allow, reason_code, matched_rule_id, trace_hash, context_json)
		 VALUES ($1, $2, $3, $4, $5, $6, $7,
		         $8, $9, $10, $11, $12, $13, $14)
		 RETURNING id::text`,
		rec.TenantID,
		rec.WorkspaceID,
		rec.Subject,
		rec.SessionID,
		rec.PolicySetID,
		rec.PolicySetKey,
		rec.Tier,
		rec.Action,
		rec.ResourceRef,
		rec.Allow,
		rec.ReasonCode,
		rec.MatchedRuleID,
		rec.TraceHash,
		ctxJSON,
	).Scan(&decisionID)
	if err != nil {
		return "", err
	}
	return decisionID, nil
}

func validateDecisionRecord(rec DecisionRecord) error {
	if strings.TrimSpace(rec.Subject) == "" {
		return fmt.Errorf("authz: subject is required")
	}
	if strings.TrimSpace(rec.Action) == "" {
		return fmt.Errorf("authz: action is required")
	}
	if strings.TrimSpace(rec.ResourceRef) == "" {
		return fmt.Errorf("authz: resource ref is required")
	}
	if strings.TrimSpace(rec.ReasonCode) == "" {
		return fmt.Errorf("authz: reason code is required")
	}
	if rec.Tier != nil {
		tier := strings.TrimSpace(*rec.Tier)
		if tier != "" && tier != "T1" && tier != "T2" && tier != "T3" && tier != "T4" {
			return fmt.Errorf("authz: invalid tier %q", tier)
		}
	}
	return nil
}

func validateTraceSteps(steps []TraceStep) error {
	for _, s := range steps {
		if s.StepOrder < 0 {
			return fmt.Errorf("authz: step order must be >= 0")
		}
		if strings.TrimSpace(s.RuleID) == "" {
			return fmt.Errorf("authz: trace rule id is required")
		}
		if strings.TrimSpace(s.Reason) == "" {
			return fmt.Errorf("authz: trace reason is required")
		}
		switch strings.TrimSpace(s.Outcome) {
		case "no_match", "allow", "deny":
		default:
			return fmt.Errorf("authz: invalid trace outcome %q", s.Outcome)
		}
	}
	return nil
}
