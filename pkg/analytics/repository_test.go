package analytics

import "testing"

func TestValidateModelVersionRecord(t *testing.T) {
	rec := ModelVersionRecord{
		ModelKey:       "risk_core",
		ModelVersion:   "v1",
		ConfigChecksum: "abc123",
		Status:         "active",
	}
	if err := validateModelVersionRecord(rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateTransactionRecord(t *testing.T) {
	rec := TransactionRecord{
		ExternalTxnID: "tx-1",
		SourceSystem:  "gateway",
		EventTime:     "2026-02-21T00:00:00Z",
		TxnType:       "TRANSFER",
		Amount:        "10.00",
		Currency:      "USD",
	}
	if err := validateTransactionRecord(rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateFeatureRecord(t *testing.T) {
	rec := FeatureRecord{
		TransactionID: "00000000-0000-0000-0000-000000000000",
		DigitalRoot:   9,
		AmountBucket:  "HIGH",
	}
	if err := validateFeatureRecord(rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateScoreRecord(t *testing.T) {
	rec := ScoreRecord{
		TransactionID:  "00000000-0000-0000-0000-000000000000",
		ModelVersionID: "00000000-0000-0000-0000-000000000000",
		RiskScore:      "0.85",
		RiskLevel:      "HIGH",
	}
	if err := validateScoreRecord(rec); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateScoreRecordBadLevel(t *testing.T) {
	rec := ScoreRecord{
		TransactionID:  "00000000-0000-0000-0000-000000000000",
		ModelVersionID: "00000000-0000-0000-0000-000000000000",
		RiskScore:      "0.85",
		RiskLevel:      "SEVERE",
	}
	if err := validateScoreRecord(rec); err == nil {
		t.Fatalf("expected validation error")
	}
}
