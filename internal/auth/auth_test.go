

package auth

import (
	"os"
	"testing"
	"time"
)

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test config: %v", err)
	}
}

func TestIsAuthorized(t *testing.T) {
	path := "test_config.json"
	defer os.Remove(path)

	writeTestFile(t, path, `{"service-a": "Bearer abc123"}`)

	store, err := LoadAuthConfig(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	tests := []struct {
		service string
		header  string
		allowed bool
	}{
		{"service-a", "Bearer abc123", true},
		{"service-a", "wrong", false},
		{"unknown", "Bearer abc123", false},
	}

	for _, test := range tests {
		if got := store.IsAuthorized(test.service, test.header); got != test.allowed {
			t.Errorf("IsAuthorized(%q, %q) = %v; want %v", test.service, test.header, got, test.allowed)
		}
	}
}

func TestReloadConfig(t *testing.T) {
	path := "test_reload_config.json"
	defer os.Remove(path)

	writeTestFile(t, path, `{"service-x": "Bearer old-token"}`)

	store, err := LoadAuthConfig(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if !store.IsAuthorized("service-x", "Bearer old-token") {
		t.Fatal("expected old-token to be authorized")
	}

	writeTestFile(t, path, `{"service-x": "Bearer new-token"}`)

	time.Sleep(300 * time.Millisecond)

	if !store.IsAuthorized("service-x", "Bearer new-token") {
		t.Error("expected new-token to be authorized after config reload")
	}
}