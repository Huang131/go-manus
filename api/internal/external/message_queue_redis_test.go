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
	if defaultLockExpire != 10*time.Second {
		t.Fatalf("default lock expiry = %s, want 10s", defaultLockExpire)
	}
	if lockRetryInterval != 100*time.Millisecond {
		t.Fatalf("lock retry interval = %s, want 100ms", lockRetryInterval)
	}

	queue := NewRedisStreamMessageQueue(nil)
	if queue.lockExpire != defaultLockExpire {
		t.Fatalf("queue lock expiry = %s, want %s", queue.lockExpire, defaultLockExpire)
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
