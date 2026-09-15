package external

import (
	"testing"
	"time"
)

func TestSearchClients_UseConfiguredTimeout(t *testing.T) {
	const timeout = 7 * time.Second

	tavily := NewTavilySearchClientWithTimeout("key", timeout)
	if tavily.httpClient.Timeout != timeout {
		t.Fatalf("tavily timeout = %s, want %s", tavily.httpClient.Timeout, timeout)
	}

	bocha := NewBochaSearchClientWithTimeout("key", timeout)
	if bocha.httpClient.Timeout != timeout {
		t.Fatalf("bocha timeout = %s, want %s", bocha.httpClient.Timeout, timeout)
	}

	google := NewGoogleSearchClientWithTimeout("key", timeout)
	if google.httpClient.Timeout != timeout {
		t.Fatalf("google timeout = %s, want %s", google.httpClient.Timeout, timeout)
	}
}

func TestSearchClients_InvalidTimeoutUsesDefault(t *testing.T) {
	tavily := NewTavilySearchClientWithTimeout("key", 0)
	if tavily.httpClient.Timeout != 30*time.Second {
		t.Fatalf("tavily timeout = %s, want 30s", tavily.httpClient.Timeout)
	}

	bocha := NewBochaSearchClientWithTimeout("key", -time.Second)
	if bocha.httpClient.Timeout != 30*time.Second {
		t.Fatalf("bocha timeout = %s, want 30s", bocha.httpClient.Timeout)
	}

	google := NewGoogleSearchClientWithTimeout("key", -time.Second)
	if google.httpClient.Timeout != 30*time.Second {
		t.Fatalf("google timeout = %s, want 30s", google.httpClient.Timeout)
	}
}
