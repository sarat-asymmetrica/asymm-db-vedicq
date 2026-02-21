package main

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

type cachedResponse struct {
	ResponseCode int
	ResponseJSON []byte
}

func reserveIdempotencyKey(
	ctx context.Context,
	db *sql.DB,
	scope string,
	idempotencyKey string,
	requestHash string,
	ttl time.Duration,
) (bool, *cachedResponse, error) {
	if db == nil {
		return false, nil, fmt.Errorf("idempotency: nil db handle")
	}
	scope = strings.TrimSpace(scope)
	idempotencyKey = strings.TrimSpace(idempotencyKey)
	requestHash = strings.TrimSpace(requestHash)
	if scope == "" || idempotencyKey == "" || requestHash == "" {
		return false, nil, fmt.Errorf("idempotency: scope/key/hash are required")
	}
	if ttl <= 0 {
		ttl = 24 * time.Hour
	}

	fullKey := scopedIdempotencyKey(scope, idempotencyKey)
	cached, found, err := loadIdempotencyRow(ctx, db, scope, fullKey)
	if err != nil {
		return false, nil, err
	}
	if found {
		if cached.requestHash != requestHash {
			return false, nil, fmt.Errorf("idempotency key reused with different payload")
		}
		if cached.ResponseCode > 0 && len(cached.ResponseJSON) > 0 {
			return false, &cachedResponse{
				ResponseCode: cached.ResponseCode,
				ResponseJSON: cached.ResponseJSON,
			}, nil
		}
		return false, nil, nil
	}

	_, _ = db.ExecContext(
		ctx,
		`DELETE FROM ops.idempotency_keys
		 WHERE scope = $1
		   AND idempotency_key = $2
		   AND expires_at <= now()`,
		scope,
		fullKey,
	)

	ttlSeconds := int64(ttl / time.Second)
	if ttlSeconds < 1 {
		ttlSeconds = 1
	}
	res, err := db.ExecContext(
		ctx,
		`INSERT INTO ops.idempotency_keys
		 (idempotency_key, scope, request_hash, expires_at)
		 VALUES ($1, $2, $3, now() + make_interval(secs => $4))
		 ON CONFLICT (idempotency_key) DO NOTHING`,
		fullKey,
		scope,
		requestHash,
		ttlSeconds,
	)
	if err != nil {
		return false, nil, err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return false, nil, err
	}
	if affected == 1 {
		return true, nil, nil
	}

	// Conflict race. Re-read and return deterministic status.
	cached, found, err = loadIdempotencyRow(ctx, db, scope, fullKey)
	if err != nil {
		return false, nil, err
	}
	if !found {
		return false, nil, fmt.Errorf("idempotency: conflict detected but no row found")
	}
	if cached.requestHash != requestHash {
		return false, nil, fmt.Errorf("idempotency key reused with different payload")
	}
	if cached.ResponseCode > 0 && len(cached.ResponseJSON) > 0 {
		return false, &cachedResponse{
			ResponseCode: cached.ResponseCode,
			ResponseJSON: cached.ResponseJSON,
		}, nil
	}
	return false, nil, nil
}

func storeIdempotencyResponse(
	ctx context.Context,
	db *sql.DB,
	scope string,
	idempotencyKey string,
	responseCode int,
	responseJSON []byte,
) error {
	if db == nil {
		return fmt.Errorf("idempotency: nil db handle")
	}
	if strings.TrimSpace(scope) == "" || strings.TrimSpace(idempotencyKey) == "" {
		return fmt.Errorf("idempotency: scope/key are required")
	}
	if responseCode <= 0 {
		return fmt.Errorf("idempotency: response code must be > 0")
	}
	if len(responseJSON) == 0 {
		responseJSON = []byte("{}")
	}
	fullKey := scopedIdempotencyKey(scope, idempotencyKey)
	_, err := db.ExecContext(
		ctx,
		`UPDATE ops.idempotency_keys
		 SET response_code = $3,
		     response_json = $4
		 WHERE scope = $1
		   AND idempotency_key = $2`,
		scope,
		fullKey,
		responseCode,
		responseJSON,
	)
	return err
}

type idempotencyRow struct {
	requestHash  string
	ResponseCode int
	ResponseJSON []byte
}

func loadIdempotencyRow(ctx context.Context, db *sql.DB, scope, fullKey string) (idempotencyRow, bool, error) {
	var row idempotencyRow
	err := db.QueryRowContext(
		ctx,
		`SELECT request_hash, COALESCE(response_code, 0), COALESCE(response_json, '{}'::jsonb)
		 FROM ops.idempotency_keys
		 WHERE scope = $1
		   AND idempotency_key = $2
		   AND expires_at > now()`,
		scope,
		fullKey,
	).Scan(&row.requestHash, &row.ResponseCode, &row.ResponseJSON)
	if err == sql.ErrNoRows {
		return idempotencyRow{}, false, nil
	}
	if err != nil {
		return idempotencyRow{}, false, err
	}
	return row, true, nil
}

func scopedIdempotencyKey(scope, key string) string {
	return scope + ":" + key
}
