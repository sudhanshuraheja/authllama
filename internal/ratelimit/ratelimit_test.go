package ratelimit

import (
	"os"
	"testing"
	"time"
)

func writeTestConfig(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
}

func TestLoadRateLimiterAndIsAuthorized(t *testing.T) {
	path := "test_rate_limit_config.json"
	defer os.Remove(path)

	writeTestConfig(t, path, `{
		"service-a": {"auth": "Bearer abc123", "tpm": 5},
		"service-b": {"auth": "Bearer xyz456", "tpm": 0}
	}`)

	rl, err := LoadRateLimiter(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if !rl.IsAuthorized("service-a", "Bearer abc123") {
		t.Error("expected service-a to be authorized")
	}
	if rl.IsAuthorized("service-a", "wrong-token") {
		t.Error("expected service-a to be unauthorized with wrong token")
	}
	if rl.IsAuthorized("unknown", "Bearer abc123") {
		t.Error("expected unknown service to be unauthorized")
	}
}

func TestRateLimitEnforcement(t *testing.T) {
	path := "test_rate_limit_limit.json"
	defer os.Remove(path)

	writeTestConfig(t, path, `{
		"service-a": {"auth": "Bearer abc123", "tpm": 2}
	}`)

	rl, err := LoadRateLimiter(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	// First 2 should pass
	if !rl.Allow("service-a") {
		t.Error("expected first request to be allowed")
	}
	if !rl.Allow("service-a") {
		t.Error("expected second request to be allowed")
	}
	// Third should fail
	if rl.Allow("service-a") {
		t.Error("expected third request to be blocked")
	}

	// Override bucket to use fast refill for test
	rl.mu.Lock()
	rl.buckets["service-a"] = &tokenBucket{
		lastRefill: time.Now(),
		tokens:     0,
		capacity:   2,
		interval:   100 * time.Millisecond,
	}
	rl.mu.Unlock()

	time.Sleep(250 * time.Millisecond)

	if !rl.Allow("service-a") {
		t.Error("expected request to be allowed after refill")
	}
}
