package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestProxyEndpoints(t *testing.T) {
	// Start fake Ollama server
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			if _, err := io.WriteString(w, `{"response":"This is a mock response for `+r.URL.Path+`"}`); err != nil {
				t.Fatalf("failed to write response: %v", err)
			}
		} else {
			http.NotFound(w, r)
		}
	}))
	defer ollama.Close()

	handler := NewProxyHandler(ollama.URL)

	testProxyEndpoint(t, handler.HandleGenerate, "/api/generate", `{"model":"gemma:2b","prompt":"Hello"}`)
	testProxyEndpoint(t, handler.HandleChat, "/api/chat", `{"model":"gemma:2b","messages":[{"role":"user","content":"Hi"}]}`)
}

func testProxyEndpoint(t *testing.T, handler func(http.ResponseWriter, *http.Request), path string, payload string) {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	handler(rr, req)

	resp := rr.Result()
	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode != http.StatusOK {
		t.Fatalf("%s: expected 200, got %d", path, resp.StatusCode)
	}

	if !strings.Contains(string(body), "mock response") {
		t.Errorf("%s: unexpected body: %s", path, string(body))
	}
}
