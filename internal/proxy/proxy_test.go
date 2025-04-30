package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestForwardPost(t *testing.T) {
	// Start fake Ollama server
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/") {
			if _, err := io.WriteString(w, `{"response":"This is a mock response for `+r.URL.Path+`"}`); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		} else {
			http.NotFound(w, r)
		}
	}))
	defer ollama.Close()

	handler := NewProxyHandler(ollama.URL)

	tests := []struct {
		name     string
		handler  func(http.ResponseWriter, *http.Request)
		endpoint string
		payload  string
	}{
		{
			name:     "generate",
			handler:  handler.HandleGenerate,
			endpoint: "/api/generate",
			payload:  `{"model":"gemma:2b","prompt":"Hello"}`,
		},
		{
			name:     "chat",
			handler:  handler.HandleChat,
			endpoint: "/api/chat",
			payload:  `{"model":"gemma:2b","messages":[{"role":"user","content":"Hi"}]}`,
		},
		{
			name:     "pull",
			handler:  handler.HandlePull,
			endpoint: "/api/pull",
			payload:  `{"model":"gemma:2b"}`,
		},
		{
			name:     "push",
			handler:  handler.HandlePush,
			endpoint: "/api/push",
			payload:  `{"model":"gemma:2b"}`,
		},
		{
			name:     "create",
			handler:  handler.HandleCreate,
			endpoint: "/api/create",
			payload:  `{"model":"gemma:2b"}`,
		},
		{
			name:     "delete",
			handler:  handler.HandleDelete,
			endpoint: "/api/delete",
			payload:  `{"model":"gemma:2b"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, tt.endpoint, strings.NewReader(tt.payload))
			req.Header.Set("Content-Type", "application/json")
			rr := httptest.NewRecorder()

			tt.handler(rr, req)

			resp := rr.Result()
			body, _ := io.ReadAll(resp.Body)

			if resp.StatusCode != http.StatusOK {
				t.Fatalf("%s: expected 200, got %d", tt.endpoint, resp.StatusCode)
			}
			if !strings.Contains(string(body), "mock response") {
				t.Errorf("%s: unexpected body: %s", tt.endpoint, string(body))
			}
		})
	}
}

func TestForwardGet(t *testing.T) {
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/tags" {
			_, _ = io.WriteString(w, `["gemma:2b","llama2"]`)
		} else if r.Method == http.MethodGet && r.URL.Path == "/api/show" {
			_, _ = io.WriteString(w, `{"model":"gemma:2b"}`)
		} else {
			http.NotFound(w, r)
		}
	}))
	defer ollama.Close()

	handler := NewProxyHandler(ollama.URL)

	req := httptest.NewRequest(http.MethodGet, "/api/tags", nil)
	rr := httptest.NewRecorder()

	handler.HandleTags(rr, req)

	resp := rr.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/tags: expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "gemma:2b") {
		t.Errorf("GET /api/tags: unexpected body: %s", string(body))
	}

	req = httptest.NewRequest(http.MethodGet, "/api/show", nil)
	rr = httptest.NewRecorder()
	handler.HandleShow(rr, req)
	resp = rr.Result()
	body, _ = io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/show: expected 200, got %d", resp.StatusCode)
	}
	if !strings.Contains(string(body), "gemma:2b") {
		t.Errorf("GET /api/show: unexpected body: %s", string(body))
	}
}
