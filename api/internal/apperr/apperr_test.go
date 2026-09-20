package apperr

import (
	"context"
	"errors"
	"net/http"
	"testing"
)

func TestToInternal(t *testing.T) {
	t.Run("nil passthrough", func(t *testing.T) {
		if err := ToInternal(nil); err != nil {
			t.Fatalf("ToInternal(nil) = %v, want nil", err)
		}
	})

	t.Run("wraps plain error", func(t *testing.T) {
		err := ToInternal(context.Canceled)
		if err == nil || err.Kind != KindInternal {
			t.Fatalf("ToInternal() = %+v, want internal error", err)
		}
		if !errors.Is(err, context.Canceled) {
			t.Fatal("ToInternal() should preserve cause for errors.Is")
		}
	})

	t.Run("passes through internal error", func(t *testing.T) {
		original := Internal("already internal")
		if got := ToInternal(original); got != original {
			t.Fatalf("ToInternal() = %+v, want same instance", got)
		}
	})

	t.Run("rewraps non-internal apperr", func(t *testing.T) {
		notFound := NotFound("missing")
		got := ToInternal(notFound)
		if got.Kind != KindInternal || !errors.Is(got, notFound) {
			t.Fatalf("ToInternal() = %+v, want wrapped internal keeping cause", got)
		}
	})
}

func TestKindHTTPStatus(t *testing.T) {
	tests := []struct {
		kind Kind
		want int
	}{
		{KindInvalidArgument, http.StatusBadRequest},
		{KindNotFound, http.StatusNotFound},
		{KindConflict, http.StatusConflict},
		{KindUnauthorized, http.StatusUnauthorized},
		{KindForbidden, http.StatusForbidden},
		{KindFailedPrecondition, http.StatusPreconditionFailed},
		{KindUnavailable, http.StatusServiceUnavailable},
		{KindInternal, http.StatusInternalServerError},
	}
	for _, tt := range tests {
		if got := tt.kind.HTTPStatus(); got != tt.want {
			t.Errorf("Kind(%s).HTTPStatus() = %d, want %d", tt.kind, got, tt.want)
		}
	}
}

func TestErrorMessageFormatting(t *testing.T) {
	cause := errors.New("db down")

	if got := Wrap(KindInternal, "internal server error", cause).Error(); got != "internal server error: db down" {
		t.Errorf("Error() = %q, want combined message", got)
	}
	if got := New(KindNotFound, "session missing").Error(); got != "session missing" {
		t.Errorf("Error() = %q, want msg only", got)
	}
	if got := (&Error{Cause: cause}).Error(); got != "db down" {
		t.Errorf("Error() = %q, want cause only", got)
	}
}

// TestErrorsIsMatchesByKind 保证 errors.Is 只比较 Kind，
// 调用方据此可以放心用哨兵风格判断（如 errors.Is(err, ErrSessionNotFound)）。
func TestErrorsIsMatchesByKind(t *testing.T) {
	if !errors.Is(NotFound("a"), NotFound("b")) {
		t.Error("errors.Is should match same Kind regardless Msg")
	}
	if errors.Is(NotFound("a"), Conflict("a")) {
		t.Error("errors.Is should not match different Kind")
	}
}
