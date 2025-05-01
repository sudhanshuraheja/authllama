

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

	writeTestFile(t, path, `{
	"_global": {
		"ollama_url": "http://localhost:11434",
		"port": "8080",
		"admin_api_key": "test"
	},
	"services": {
		"service-a": {"auth": "Bearer abc123", "tpm": 10}
	}
}`)

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

	writeTestFile(t, path, `{
	"_global": {
		"ollama_url": "http://localhost:11434",
		"port": "8080",
		"admin_api_key": "test"
	},
	"services": {
		"service-x": {"auth": "Bearer old-token", "tpm": 10}
	}
}`)

	store, err := LoadAuthConfig(path)
	if err != nil {
		t.Fatalf("failed to load config: %v", err)
	}

	if !store.IsAuthorized("service-x", "Bearer old-token") {
		t.Fatal("expected old-token to be authorized")
	}

	writeTestFile(t, path, `{
	"_global": {
		"ollama_url": "http://localhost:11434",
		"port": "8080",
		"admin_api_key": "test"
	},
	"services": {
		"service-x": {"auth": "Bearer new-token", "tpm": 10}
	}
}`)

	time.Sleep(300 * time.Millisecond)

	if !store.IsAuthorized("service-x", "Bearer new-token") {
		t.Error("expected new-token to be authorized after config reload")
	}
}