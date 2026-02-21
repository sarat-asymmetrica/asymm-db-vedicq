package telemetry

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
)

// SecurityEventRecord represents telemetry.security_events input.
type SecurityEventRecord struct {
	TenantID    *string
	WorkspaceID *string
	ActorType   string
	ActorID     *string
	EventType   string
	Severity    string
	Message     string
	TraceHash   *string
	EventJSON   []byte
}

// EventLink represents telemetry.event_links input.
type EventLink struct {
	LinkKind     string
	LinkedID     string
	MetadataJSON []byte
}

// Repository persists normalized security events and lineage links.
type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) (*Repository, error) {
	if db == nil {
		return nil, fmt.Errorf("telemetry: nil db handle")
	}
	return &Repository{db: db}, nil
}

// PersistSecurityEventWithLinks writes an event and related links atomically.
func (r *Repository) PersistSecurityEventWithLinks(
	ctx context.Context,
	rec SecurityEventRecord,
	links []EventLink,
) (string, error) {
	if strings.TrimSpace(rec.Severity) == "" {
		rec.Severity = "info"
	}
	if err := validateSecurityEvent(rec); err != nil {
		return "", err
	}
	if err := validateEventLinks(links); err != nil {
		return "", err
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	eventID, err := insertSecurityEvent(ctx, tx, rec)
	if err != nil {
		return "", err
	}

	for _, link := range links {
		meta := link.MetadataJSON
		if len(meta) == 0 {
			meta = []byte("{}")
		}
		if _, err := tx.ExecContext(
			ctx,
			`INSERT INTO telemetry.event_links
			 (event_id, link_kind, linked_id, metadata_json)
			 VALUES ($1, $2, $3::uuid, $4)`,
			eventID,
			link.LinkKind,
			link.LinkedID,
			meta,
		); err != nil {
			return "", err
		}
	}

	if err := tx.Commit(); err != nil {
		return "", err
	}
	return eventID, nil
}

func insertSecurityEvent(ctx context.Context, tx *sql.Tx, rec SecurityEventRecord) (string, error) {
	evJSON := rec.EventJSON
	if len(evJSON) == 0 {
		evJSON = []byte("{}")
	}

	var eventID string
	err := tx.QueryRowContext(
		ctx,
		`INSERT INTO telemetry.security_events
		 (tenant_id, workspace_id, actor_type, actor_id, event_type, severity, message, trace_hash, event_json)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		 RETURNING id::text`,
		rec.TenantID,
		rec.WorkspaceID,
		rec.ActorType,
		rec.ActorID,
		rec.EventType,
		rec.Severity,
		rec.Message,
		rec.TraceHash,
		evJSON,
	).Scan(&eventID)
	if err != nil {
		return "", err
	}
	return eventID, nil
}

func validateSecurityEvent(rec SecurityEventRecord) error {
	if strings.TrimSpace(rec.ActorType) == "" {
		return fmt.Errorf("telemetry: actor type is required")
	}
	switch strings.TrimSpace(rec.ActorType) {
	case "user", "service", "system":
	default:
		return fmt.Errorf("telemetry: invalid actor type %q", rec.ActorType)
	}
	if strings.TrimSpace(rec.EventType) == "" {
		return fmt.Errorf("telemetry: event type is required")
	}
	if strings.TrimSpace(rec.Message) == "" {
		return fmt.Errorf("telemetry: message is required")
	}
	sev := strings.TrimSpace(rec.Severity)
	switch sev {
	case "debug", "info", "warn", "error":
		return nil
	default:
		return fmt.Errorf("telemetry: invalid severity %q", rec.Severity)
	}
}

func validateEventLinks(links []EventLink) error {
	for _, l := range links {
		if strings.TrimSpace(l.LinkKind) == "" {
			return fmt.Errorf("telemetry: link kind is required")
		}
		if strings.TrimSpace(l.LinkedID) == "" {
			return fmt.Errorf("telemetry: linked id is required")
		}
	}
	return nil
}
