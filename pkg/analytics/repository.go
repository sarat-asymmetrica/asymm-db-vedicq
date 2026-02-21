package analytics

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// Repository handles persistence for vedic analytics foundation tables.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("analytics: nil db handle")
	}
	return &Repository{db: db}, nil
}

type ModelVersionRecord struct {
	ModelKey       string
	ModelVersion   string
	ModelFamily    string
	ConfigChecksum string
	ConfigJSON     []byte
	Status         string
}

type TransactionRecord struct {
	TenantID      *string
	WorkspaceID   *string
	ResourceID    *string
	ExternalTxnID string
	SourceSystem  string
	EventTime     string // RFC3339 formatted timestamp for portability in this layer
	TxnType       string
	Amount        string
	Currency      string
	OldBalance    *string
	NewBalance    *string
	MetadataJSON  []byte
}

type FeatureRecord struct {
	TransactionID  string
	ModelVersionID *string
	AmountCents    int64
	DigitalRoot    int
	IsRoundAmount  bool
	DrainRatio     *string
	AmountBucket   string
	FeatureJSON    []byte
}

type ScoreRecord struct {
	TransactionID   string
	ModelVersionID  string
	ScoreName       string
	RiskScore       string
	RiskLevel       string
	ReasonCodesJSON []byte
	Explanation     *string
}

func (r *Repository) UpsertModelVersion(ctx context.Context, rec ModelVersionRecord) (string, error) {
	if err := validateModelVersionRecord(rec); err != nil {
		return "", err
	}
	cfgJSON := rec.ConfigJSON
	if len(cfgJSON) == 0 {
		cfgJSON = []byte("{}")
	}
	status := strings.TrimSpace(rec.Status)
	if status == "" {
		status = "active"
	}
	modelFamily := strings.TrimSpace(rec.ModelFamily)
	if modelFamily == "" {
		modelFamily = "risk_analytics"
	}

	var id string
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO vedic.model_versions
		 (model_key, model_version, model_family, config_checksum, config_json, status)
		 VALUES ($1, $2, $3, $4, $5, $6)
		 ON CONFLICT (model_key, model_version) DO UPDATE
		 SET model_family = EXCLUDED.model_family,
		     config_checksum = EXCLUDED.config_checksum,
		     config_json = EXCLUDED.config_json,
		     status = EXCLUDED.status
		 RETURNING id::text`,
		rec.ModelKey,
		rec.ModelVersion,
		modelFamily,
		rec.ConfigChecksum,
		cfgJSON,
		status,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

// UpsertTransaction enforces idempotent ingestion by tenant+source+external id.
func (r *Repository) UpsertTransaction(ctx context.Context, rec TransactionRecord) (string, error) {
	if err := validateTransactionRecord(rec); err != nil {
		return "", err
	}
	meta := rec.MetadataJSON
	if len(meta) == 0 {
		meta = []byte("{}")
	}

	var id string
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO vedic.transactions
		 (tenant_id, workspace_id, resource_id, external_txn_id, source_system, event_time,
		  txn_type, amount, currency, old_balance, new_balance, metadata_json)
		 VALUES ($1, $2, $3, $4, $5, $6::timestamptz,
		         $7, $8::numeric, $9, $10::numeric, $11::numeric, $12)
		 ON CONFLICT (tenant_id, source_system, external_txn_id) DO UPDATE
		 SET workspace_id = EXCLUDED.workspace_id,
		     resource_id = EXCLUDED.resource_id,
		     event_time = EXCLUDED.event_time,
		     txn_type = EXCLUDED.txn_type,
		     amount = EXCLUDED.amount,
		     currency = EXCLUDED.currency,
		     old_balance = EXCLUDED.old_balance,
		     new_balance = EXCLUDED.new_balance,
		     metadata_json = EXCLUDED.metadata_json
		 RETURNING id::text`,
		rec.TenantID,
		rec.WorkspaceID,
		rec.ResourceID,
		rec.ExternalTxnID,
		rec.SourceSystem,
		rec.EventTime,
		rec.TxnType,
		rec.Amount,
		rec.Currency,
		rec.OldBalance,
		rec.NewBalance,
		meta,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *Repository) UpsertTransactionFeatures(ctx context.Context, rec FeatureRecord) (string, error) {
	if err := validateFeatureRecord(rec); err != nil {
		return "", err
	}
	featureJSON := rec.FeatureJSON
	if len(featureJSON) == 0 {
		featureJSON = []byte("{}")
	}

	var id string
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO vedic.transaction_features
		 (transaction_id, model_version_id, amount_cents, digital_root, is_round_amount,
		  drain_ratio, amount_bucket, feature_json)
		 VALUES ($1::uuid, $2::uuid, $3, $4, $5, $6::numeric, $7, $8)
		 ON CONFLICT (transaction_id, model_version_id) DO UPDATE
		 SET amount_cents = EXCLUDED.amount_cents,
		     digital_root = EXCLUDED.digital_root,
		     is_round_amount = EXCLUDED.is_round_amount,
		     drain_ratio = EXCLUDED.drain_ratio,
		     amount_bucket = EXCLUDED.amount_bucket,
		     feature_json = EXCLUDED.feature_json
		 RETURNING id::text`,
		rec.TransactionID,
		rec.ModelVersionID,
		rec.AmountCents,
		rec.DigitalRoot,
		rec.IsRoundAmount,
		rec.DrainRatio,
		rec.AmountBucket,
		featureJSON,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func (r *Repository) UpsertScore(ctx context.Context, rec ScoreRecord) (string, error) {
	if err := validateScoreRecord(rec); err != nil {
		return "", err
	}
	reasons := rec.ReasonCodesJSON
	if len(reasons) == 0 {
		reasons = []byte("[]")
	}
	scoreName := strings.TrimSpace(rec.ScoreName)
	if scoreName == "" {
		scoreName = "transaction_risk"
	}

	var id string
	err := r.db.QueryRowContext(
		ctx,
		`INSERT INTO vedic.fraud_scores
		 (transaction_id, model_version_id, score_name, risk_score, risk_level, reason_codes_json, explanation)
		 VALUES ($1::uuid, $2::uuid, $3, $4::numeric, $5, $6, $7)
		 ON CONFLICT (transaction_id, model_version_id, score_name) DO UPDATE
		 SET risk_score = EXCLUDED.risk_score,
		     risk_level = EXCLUDED.risk_level,
		     reason_codes_json = EXCLUDED.reason_codes_json,
		     explanation = EXCLUDED.explanation
		 RETURNING id::text`,
		rec.TransactionID,
		rec.ModelVersionID,
		scoreName,
		rec.RiskScore,
		rec.RiskLevel,
		reasons,
		rec.Explanation,
	).Scan(&id)
	if err != nil {
		return "", err
	}
	return id, nil
}

func validateModelVersionRecord(rec ModelVersionRecord) error {
	if strings.TrimSpace(rec.ModelKey) == "" {
		return fmt.Errorf("analytics: model key is required")
	}
	if strings.TrimSpace(rec.ModelVersion) == "" {
		return fmt.Errorf("analytics: model version is required")
	}
	if strings.TrimSpace(rec.ConfigChecksum) == "" {
		return fmt.Errorf("analytics: config checksum is required")
	}
	status := strings.TrimSpace(rec.Status)
	if status != "" && status != "active" && status != "deprecated" && status != "retired" {
		return fmt.Errorf("analytics: invalid model status %q", rec.Status)
	}
	return nil
}

func validateTransactionRecord(rec TransactionRecord) error {
	if strings.TrimSpace(rec.ExternalTxnID) == "" {
		return fmt.Errorf("analytics: external transaction id is required")
	}
	if strings.TrimSpace(rec.SourceSystem) == "" {
		return fmt.Errorf("analytics: source system is required")
	}
	if strings.TrimSpace(rec.EventTime) == "" {
		return fmt.Errorf("analytics: event time is required")
	}
	if strings.TrimSpace(rec.TxnType) == "" {
		return fmt.Errorf("analytics: txn type is required")
	}
	if strings.TrimSpace(rec.Amount) == "" {
		return fmt.Errorf("analytics: amount is required")
	}
	if strings.TrimSpace(rec.Currency) == "" {
		return fmt.Errorf("analytics: currency is required")
	}
	return nil
}

func validateFeatureRecord(rec FeatureRecord) error {
	if strings.TrimSpace(rec.TransactionID) == "" {
		return fmt.Errorf("analytics: transaction id is required")
	}
	if rec.DigitalRoot < 0 || rec.DigitalRoot > 9 {
		return fmt.Errorf("analytics: digital root must be between 0 and 9")
	}
	if strings.TrimSpace(rec.AmountBucket) == "" {
		return fmt.Errorf("analytics: amount bucket is required")
	}
	return nil
}

func validateScoreRecord(rec ScoreRecord) error {
	if strings.TrimSpace(rec.TransactionID) == "" {
		return fmt.Errorf("analytics: transaction id is required")
	}
	if strings.TrimSpace(rec.ModelVersionID) == "" {
		return fmt.Errorf("analytics: model version id is required")
	}
	if strings.TrimSpace(rec.RiskScore) == "" {
		return fmt.Errorf("analytics: risk score is required")
	}
	switch strings.TrimSpace(rec.RiskLevel) {
	case "LOW", "MEDIUM", "HIGH", "CRITICAL":
	default:
		return fmt.Errorf("analytics: invalid risk level %q", rec.RiskLevel)
	}
	return nil
}
