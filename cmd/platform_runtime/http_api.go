package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	authzrepo "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/authz"
	dbpkg "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/db"
	platform "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/platform"
	telemetryrepo "github.com/sarat-asymmetrica/vedic-platform-experiments/pkg/telemetry"
)

type httpAPI struct {
	rt             *platform.Runtime
	healthTimeout  time.Duration
	writeTimeout   time.Duration
	idempotencyTTL time.Duration
}

func newHTTPAPI(rt *platform.Runtime, healthTimeout, writeTimeout, idempotencyTTL time.Duration) *httpAPI {
	return &httpAPI{
		rt:             rt,
		healthTimeout:  healthTimeout,
		writeTimeout:   writeTimeout,
		idempotencyTTL: idempotencyTTL,
	}
}

func (a *httpAPI) handleLiveness(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "alive"})
}

func (a *httpAPI) handleHealthz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), a.healthTimeout)
	defer cancel()
	if err := a.rt.HealthCheck(ctx, a.healthTimeout); err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "unhealthy")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (a *httpAPI) handleReadyz(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	ctx, cancel := context.WithTimeout(r.Context(), a.healthTimeout)
	defer cancel()
	if err := a.rt.HealthCheck(ctx, a.healthTimeout); err != nil {
		writeJSONError(w, http.StatusServiceUnavailable, "not-ready")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ready"})
}

func (a *httpAPI) handleDecisionWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
	if requestID == "" {
		writeJSONError(w, http.StatusBadRequest, "X-Request-ID header is required")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeJSONError(w, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	if len(body) == 0 {
		writeJSONError(w, http.StatusBadRequest, "request body is required")
		return
	}
	reqHash := dbpkg.SHA256Hex(append([]byte("decision:"), body...))

	ctx, cancel := context.WithTimeout(r.Context(), a.writeTimeout)
	defer cancel()
	reservedKey, cached, err := reserveIdempotencyKey(ctx, a.rt.DB, "v1/decisions", idempotencyKey, reqHash, a.idempotencyTTL)
	if err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}
	if !reservedKey {
		if cached != nil {
			writeRawJSON(w, cached.ResponseCode, cached.ResponseJSON)
			return
		}
		writeJSONError(w, http.StatusConflict, "request is already in progress")
		return
	}

	var req decisionWriteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}

	decisionCtx := map[string]interface{}{"request_id": requestID}
	for k, v := range req.DecisionContext {
		decisionCtx[k] = v
	}
	eventCtx := map[string]interface{}{"request_id": requestID}
	for k, v := range req.Event {
		eventCtx[k] = v
	}

	traceHash := dbpkg.SHA256Hex(body)
	decisionRecord := authzrepo.DecisionRecord{
		TenantID:      req.TenantID,
		WorkspaceID:   req.WorkspaceID,
		Subject:       req.Subject,
		SessionID:     req.SessionID,
		PolicySetID:   req.PolicySetID,
		PolicySetKey:  req.PolicySetKey,
		Tier:          req.Tier,
		Action:        req.Action,
		ResourceRef:   req.ResourceRef,
		Allow:         req.Allow,
		ReasonCode:    req.ReasonCode,
		MatchedRuleID: req.MatchedRuleID,
		TraceHash:     &traceHash,
		ContextJSON:   mustMarshalJSON(decisionCtx),
	}
	trace := make([]authzrepo.TraceStep, 0, len(req.Trace))
	for _, s := range req.Trace {
		trace = append(trace, authzrepo.TraceStep{
			StepOrder: s.StepOrder,
			RuleID:    s.RuleID,
			Matched:   s.Matched,
			Outcome:   s.Outcome,
			Reason:    s.Reason,
		})
	}
	eventRecord := telemetryrepo.SecurityEventRecord{
		TenantID:    req.TenantID,
		WorkspaceID: req.WorkspaceID,
		ActorType:   req.ActorType,
		ActorID:     req.ActorID,
		EventType:   req.EventType,
		Severity:    req.Severity,
		Message:     req.Message,
		TraceHash:   &traceHash,
		EventJSON:   mustMarshalJSON(eventCtx),
	}

	decisionID, eventID, err := a.rt.RecordDecisionAndEvent(ctx, decisionRecord, trace, eventRecord)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("failed to persist decision/event: %v", err))
		return
	}
	resp := map[string]string{
		"request_id":  requestID,
		"decision_id": decisionID,
		"event_id":    eventID,
		"status":      "accepted",
	}
	respBody := mustMarshalJSON(resp)
	if err := storeIdempotencyResponse(ctx, a.rt.DB, "v1/decisions", idempotencyKey, http.StatusAccepted, respBody); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to store idempotency response")
		return
	}
	writeRawJSON(w, http.StatusAccepted, respBody)
}

func (a *httpAPI) handleTelemetryWrite(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	requestID := strings.TrimSpace(r.Header.Get("X-Request-ID"))
	if requestID == "" {
		writeJSONError(w, http.StatusBadRequest, "X-Request-ID header is required")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" {
		writeJSONError(w, http.StatusBadRequest, "Idempotency-Key header is required")
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, "failed to read request body")
		return
	}
	if len(body) == 0 {
		writeJSONError(w, http.StatusBadRequest, "request body is required")
		return
	}
	reqHash := dbpkg.SHA256Hex(append([]byte("telemetry:"), body...))

	ctx, cancel := context.WithTimeout(r.Context(), a.writeTimeout)
	defer cancel()
	reservedKey, cached, err := reserveIdempotencyKey(ctx, a.rt.DB, "v1/telemetry/events", idempotencyKey, reqHash, a.idempotencyTTL)
	if err != nil {
		writeJSONError(w, http.StatusConflict, err.Error())
		return
	}
	if !reservedKey {
		if cached != nil {
			writeRawJSON(w, cached.ResponseCode, cached.ResponseJSON)
			return
		}
		writeJSONError(w, http.StatusConflict, "request is already in progress")
		return
	}

	var req telemetryWriteRequest
	if err := json.Unmarshal(body, &req); err != nil {
		writeJSONError(w, http.StatusBadRequest, "invalid JSON payload")
		return
	}
	eventCtx := map[string]interface{}{"request_id": requestID}
	for k, v := range req.Event {
		eventCtx[k] = v
	}
	record := telemetryrepo.SecurityEventRecord{
		TenantID:    req.TenantID,
		WorkspaceID: req.WorkspaceID,
		ActorType:   req.ActorType,
		ActorID:     req.ActorID,
		EventType:   req.EventType,
		Severity:    req.Severity,
		Message:     req.Message,
		TraceHash:   req.TraceHash,
		EventJSON:   mustMarshalJSON(eventCtx),
	}
	links := make([]telemetryrepo.EventLink, 0, len(req.Links))
	for _, link := range req.Links {
		links = append(links, telemetryrepo.EventLink{
			LinkKind:     link.LinkKind,
			LinkedID:     link.LinkedID,
			MetadataJSON: mustMarshalJSON(link.Metadata),
		})
	}
	eventID, err := a.rt.TelemetryRepo.PersistSecurityEventWithLinks(ctx, record, links)
	if err != nil {
		writeJSONError(w, http.StatusBadRequest, fmt.Sprintf("failed to persist telemetry event: %v", err))
		return
	}

	resp := map[string]string{
		"request_id": requestID,
		"event_id":   eventID,
		"status":     "accepted",
	}
	respBody := mustMarshalJSON(resp)
	if err := storeIdempotencyResponse(ctx, a.rt.DB, "v1/telemetry/events", idempotencyKey, http.StatusAccepted, respBody); err != nil {
		writeJSONError(w, http.StatusInternalServerError, "failed to store idempotency response")
		return
	}
	writeRawJSON(w, http.StatusAccepted, respBody)
}

type decisionWriteRequest struct {
	TenantID        *string                `json:"tenant_id"`
	WorkspaceID     *string                `json:"workspace_id"`
	Subject         string                 `json:"subject"`
	SessionID       *string                `json:"session_id"`
	PolicySetID     *string                `json:"policy_set_id"`
	PolicySetKey    *string                `json:"policy_set_key"`
	Tier            *string                `json:"tier"`
	Action          string                 `json:"action"`
	ResourceRef     string                 `json:"resource_ref"`
	Allow           bool                   `json:"allow"`
	ReasonCode      string                 `json:"reason_code"`
	MatchedRuleID   *string                `json:"matched_rule_id"`
	Trace           []decisionTraceStep    `json:"trace"`
	DecisionContext map[string]interface{} `json:"decision_context"`
	ActorType       string                 `json:"actor_type"`
	ActorID         *string                `json:"actor_id"`
	EventType       string                 `json:"event_type"`
	Severity        string                 `json:"severity"`
	Message         string                 `json:"message"`
	Event           map[string]interface{} `json:"event"`
}

type decisionTraceStep struct {
	StepOrder int    `json:"step_order"`
	RuleID    string `json:"rule_id"`
	Matched   bool   `json:"matched"`
	Outcome   string `json:"outcome"`
	Reason    string `json:"reason"`
}

type telemetryWriteRequest struct {
	TenantID    *string                  `json:"tenant_id"`
	WorkspaceID *string                  `json:"workspace_id"`
	ActorType   string                   `json:"actor_type"`
	ActorID     *string                  `json:"actor_id"`
	EventType   string                   `json:"event_type"`
	Severity    string                   `json:"severity"`
	Message     string                   `json:"message"`
	TraceHash   *string                  `json:"trace_hash"`
	Event       map[string]interface{}   `json:"event"`
	Links       []telemetryEventLinkSpec `json:"links"`
}

type telemetryEventLinkSpec struct {
	LinkKind string                 `json:"link_kind"`
	LinkedID string                 `json:"linked_id"`
	Metadata map[string]interface{} `json:"metadata"`
}

func mustMarshalJSON(v interface{}) []byte {
	out, err := json.Marshal(v)
	if err != nil {
		return []byte("{}")
	}
	return out
}

func writeJSONError(w http.ResponseWriter, status int, msg string) {
	writeJSON(w, status, map[string]string{"error": msg})
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	body := mustMarshalJSON(payload)
	writeRawJSON(w, status, body)
}

func writeRawJSON(w http.ResponseWriter, status int, body []byte) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(body)
}
