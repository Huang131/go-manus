package model

import (
	"testing"
)

func TestSessionStatus_Values(t *testing.T) {
	tests := []struct {
		status   SessionStatus
		expected string
	}{
		{SessionStatusPending, "pending"},
		{SessionStatusRunning, "running"},
		{SessionStatusWaiting, "waiting"},
		{SessionStatusCompleted, "completed"},
	}

	for _, tt := range tests {
		if string(tt.status) != tt.expected {
			t.Errorf("SessionStatus %v: expected %s, got %s", tt.status, tt.expected, string(tt.status))
		}
	}
}
