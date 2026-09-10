package external

import (
	"testing"
	"time"
)

func TestSearchClients_UseConfiguredTimeout(t *testing.T) {
	const timeout = 7 * time.Second

	bing := NewBingSearchClientWithTimeout("key", timeout)
	if bing.httpClient.Timeout != timeout {
		t.Fatalf("bing timeout = %s, want %s", bing.httpClient.Timeout, timeout)
	}

	google := NewGoogleSearchClientWithTimeout("key", timeout)
	if google.httpClient.Timeout != timeout {
		t.Fatalf("google timeout = %s, want %s", google.httpClient.Timeout, timeout)
	}
}

func TestSearchClients_InvalidTimeoutUsesDefault(t *testing.T) {
	bing := NewBingSearchClientWithTimeout("key", 0)
	if bing.httpClient.Timeout != 30*time.Second {
		t.Fatalf("bing timeout = %s, want 30s", bing.httpClient.Timeout)
	}

	google := NewGoogleSearchClientWithTimeout("key", -time.Second)
	if google.httpClient.Timeout != 30*time.Second {
		t.Fatalf("google timeout = %s, want 30s", google.httpClient.Timeout)
	}
}
