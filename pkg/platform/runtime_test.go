package platform

import (
	"context"
	"testing"
	"time"
)

func TestRuntimeHealthCheckNil(t *testing.T) {
	var r *Runtime
	err := r.HealthCheck(context.Background(), time.Second)
	if err == nil {
		t.Fatalf("expected error for nil runtime")
	}
}
