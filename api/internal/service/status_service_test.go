package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/model"
)

func TestStatusServiceUsesInjectedClock(t *testing.T) {
	svc := &DefaultStatusService{
		now: func() time.Time { return time.Unix(123, 0) },
	}
	status, err := svc.GetHealthStatus(context.Background())
	if err != nil {
		t.Fatalf("GetHealthStatus() error = %v", err)
	}
	if status.Timestamp != 123 {
		t.Fatalf("Timestamp = %d, want 123", status.Timestamp)
	}
}

func TestStatusService_GetHealthStatus_SkippedServicesDegraded(t *testing.T) {
	svc := NewStatusService(nil, nil, nil)

	status, err := svc.GetHealthStatus(context.Background())
	if err != nil {
		t.Fatalf("GetHealthStatus() error = %v", err)
	}
	if status.Status != model.HealthStateDegraded {
		t.Fatalf("status = %s, want degraded", status.Status)
	}
	if status.Services[model.ServiceNamePostgres].Status != model.HealthStateSkipped {
		t.Fatalf("postgres status = %s, want skipped", status.Services[model.ServiceNamePostgres].Status)
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
	if status.Status != model.HealthStateDegraded {
		t.Fatalf("status = %s, want degraded", status.Status)
	}
	if got := status.Services[model.ServiceNamePostgres].Error; !strings.Contains(got, "internal server error") {
		t.Fatalf("postgres error = %q, want wrapped internal error", got)
	}
	if got := status.Services[model.ServiceNameRedis].Error; !strings.Contains(got, "internal server error") {
		t.Fatalf("redis error = %q, want wrapped internal error", got)
	}
	if got := status.Services[model.ServiceNameOSS].Error; !strings.Contains(got, "internal server error") {
		t.Fatalf("oss error = %q, want wrapped internal error", got)
	}
}
