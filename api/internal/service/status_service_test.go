package service

import (
	"context"
	"strings"
	"testing"

	"github.com/Huang131/go-manus/api/internal/apperr"
	"github.com/Huang131/go-manus/api/internal/infrastructure"
)

type fakePostgres struct {
	err error
}

func (f *fakePostgres) HealthCheck(ctx context.Context) error { return f.err }
func (f *fakePostgres) Close()                                {}

type fakeRedis struct {
	err error
}

func (f *fakeRedis) HealthCheck(ctx context.Context) error { return f.err }
func (f *fakeRedis) Close() error                          { return nil }

type fakeOSS struct {
	err error
}

func (f *fakeOSS) HealthCheck(ctx context.Context) error { return f.err }
func (f *fakeOSS) Close() error                          { return nil }

func TestStatusService_GetHealthStatus_SkippedServicesDegraded(t *testing.T) {
	svc := NewStatusService(nil, nil, nil)

	status, err := svc.GetHealthStatus(context.Background())
	if err != nil {
		t.Fatalf("GetHealthStatus() error = %v", err)
	}
	if status.Status != "degraded" {
		t.Fatalf("status = %s, want degraded", status.Status)
	}
	if status.Services["postgres"].Status != "skipped" {
		t.Fatalf("postgres status = %s, want skipped", status.Services["postgres"].Status)
	}
}

func TestStatusService_GetHealthStatus_WrapsHealthErrors(t *testing.T) {
	svc := &DefaultStatusService{
		db:    &infrastructure.Postgres{},
		redis: &infrastructure.Redis{},
		oss:   &infrastructure.OSS{},
	}

	status, err := svc.GetHealthStatus(context.Background())
	if err != nil {
		t.Fatalf("GetHealthStatus() error = %v", err)
	}
	if status.Status != "degraded" {
		t.Fatalf("status = %s, want degraded", status.Status)
	}
	if got := status.Services["postgres"].Error; !strings.Contains(got, "internal server error") {
		t.Fatalf("postgres error = %q, want wrapped internal error", got)
	}
	if got := status.Services["redis"].Error; !strings.Contains(got, "internal server error") {
		t.Fatalf("redis error = %q, want wrapped internal error", got)
	}
	if got := status.Services["oss"].Error; !strings.Contains(got, "internal server error") {
		t.Fatalf("oss error = %q, want wrapped internal error", got)
	}
}

func TestToInternal(t *testing.T) {
	err := apperr.ToInternal(nil)
	if err != nil {
		t.Fatalf("ToInternal(nil) = %v, want nil", err)
	}

	wrapped := apperr.ToInternal(context.Canceled)
	if wrapped == nil || wrapped.Error() == "" {
		t.Fatal("ToInternal should wrap non-nil error")
	}
}
