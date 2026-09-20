package service

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/Huang131/go-manus/api/internal/infrastructure"
	"github.com/Huang131/go-manus/api/internal/model"
)

// newHealthStatusForTest 构造与 GetHealthStatus 相同初始状态的对象，
// 供直接调用 checkDependency 的测试使用。
func newHealthStatusForTest() *model.HealthStatus {
	return &model.HealthStatus{
		Status:   model.HealthStateHealthy,
		Services: map[model.ServiceName]model.ServiceStatus{},
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

// TestStatusService_GetHealthStatus_WrapsHealthErrors 验证 GetHealthStatus 确实
// 检查了全部三个依赖，且错误经 apperr 脱敏后才写入结果。
// 未初始化的基础设施结构体（Pool / Client 为 nil）会让 HealthCheck 立刻报错。
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

	names := []model.ServiceName{model.ServiceNamePostgres, model.ServiceNameRedis, model.ServiceNameOSS}
	for _, name := range names {
		if got := status.Services[name].Status; got != model.HealthStateUnhealthy {
			t.Errorf("%s status = %s, want unhealthy", name, got)
		}
		if got := status.Services[name].Error; !strings.Contains(got, "internal server error") {
			t.Errorf("%s error = %q, want wrapped internal error", name, got)
		}
	}
}

// TestCheckDependency_SuccessKeepsHealthy 覆盖唯一的成功分支：
// 检查通过时该依赖为 healthy，整体状态保持 healthy（不被误降级）。
func TestCheckDependency_SuccessKeepsHealthy(t *testing.T) {
	svc := &DefaultStatusService{}
	status := newHealthStatusForTest()

	svc.checkDependency(context.Background(), status, model.ServiceNamePostgres, func(context.Context) error {
		return nil
	})

	if got := status.Services[model.ServiceNamePostgres].Status; got != model.HealthStateHealthy {
		t.Errorf("postgres status = %s, want healthy", got)
	}
	if got := status.Services[model.ServiceNamePostgres].Error; got != "" {
		t.Errorf("postgres error = %q, want empty", got)
	}
	if status.Status != model.HealthStateHealthy {
		t.Errorf("overall status = %s, want healthy", status.Status)
	}
}

// TestCheckDependency_ProvidesBoundedContext 验证每次依赖检查都跑在有界 context 下。
//
// 这里不等待超时真正触发（那要 3s，只是在测标准库），只断言传下去的 context 带了
// 合理 deadline。这层超时是依赖侧出现网络黑洞时唯一能保证 handler 返回的机制，
// 被删掉就意味着回归。
func TestCheckDependency_ProvidesBoundedContext(t *testing.T) {
	svc := &DefaultStatusService{}
	status := newHealthStatusForTest()

	var hasDeadline bool
	var remaining time.Duration
	svc.checkDependency(context.Background(), status, model.ServiceNameRedis, func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		hasDeadline = ok
		if ok {
			remaining = time.Until(deadline)
		}
		return nil
	})

	if !hasDeadline {
		t.Fatal("依赖检查必须带 deadline，否则依赖侧卡死会拖住整个 handler")
	}
	if remaining <= 0 || remaining > dependencyCheckTimeout {
		t.Fatalf("remaining = %v, want in (0, %s]", remaining, dependencyCheckTimeout)
	}
}
