package external

import (
	"testing"
	"time"
)

func TestRedisMessageQueueDefaultTimeoutsUseDurations(t *testing.T) {
	if defaultBlockTimeout != 3*time.Second {
		t.Fatalf("default block timeout = %s, want 3s", defaultBlockTimeout)
	}
	if maxBlockTimeout != 5*time.Second {
		t.Fatalf("max block timeout = %s, want 5s", maxBlockTimeout)
	}
}

func TestRedisMessageQueueStreamRetentionPolicy(t *testing.T) {
	if streamMaxLen <= 0 {
		t.Fatalf("stream max len = %d, want positive", streamMaxLen)
	}
	if streamRetention <= 0 {
		t.Fatalf("stream retention = %s, want positive", streamRetention)
	}
}
