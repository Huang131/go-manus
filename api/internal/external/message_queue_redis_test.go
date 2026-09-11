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
	if streamMaxLen < 10000 {
		t.Fatalf("stream max len = %d, want at least 10000 for token deltas", streamMaxLen)
	}
	if streamRetention <= 0 {
		t.Fatalf("stream retention = %s, want positive", streamRetention)
	}
	if CompletedStreamRetention() <= 0 || CompletedStreamRetention() >= streamRetention {
		t.Fatalf("completed stream retention = %s, want positive and shorter than running retention %s",
			CompletedStreamRetention(), streamRetention)
	}
}
