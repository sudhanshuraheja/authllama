package proxy

import "github.com/sudhanshuraheja/authllama/internal/config"

import (
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
)

func TestForwardPost(t *testing.T) {
	// Start fake Ollama server
	ollama := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/") {
			body, _ := io.ReadAll(r.Body)
			if strings.Contains(string(body), `"stream": true`) {
				flusher, ok := w.(http.Flusher)
				if !ok {
					t.Fatalf("response does not support flushing")
				}
				w.Header().Set("Content-Type", "application/json")
				for _, i := range []int{1, 2, 3} {
					chunk := `{"chunk": "` + strconv.Itoa(1+i) + `"}`
					if _, err := io.WriteString(w, chunk); err != nil {
						t.Fatalf("failed to write stream chunk: %v", err)
					}
					flusher.Flush()
				}
			} else {
				_, _ = io.WriteString(w, `{"response": "mock response"}`)
			}
		} else {
			http.NotFound(w, r)
		}
	}))
	defer ollama.Close()

	cfg := config.GlobalConfig{
		OllamaURL:         ollama.URL,
		Port:              "8080",
		TimeoutSecs:       15,
		MaxBodySize:       10 * 1024 * 1024,
		StreamBufferSize:  4096,
		StreamTimeoutSecs: 30,
	}
	handler := NewProxyHandler(cfg)

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
			name:     "generate_streaming",
			handler:  handler.HandleGenerate,
			endpoint: "/api/generate",
			payload:  `{"model":"gemma:2b","prompt":"Stream this","stream": true}`,
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
			if tt.name == "generate_streaming" {
				// Check that all chunks are received
				for _, i := range []int{1, 2, 3} {
					chunk := `{"chunk": "` + strconv.Itoa(1+i) + `"}` // or fmt.Sprintf
					if !strings.Contains(string(body), chunk) {
						t.Errorf("%s: missing chunk %s in body: %s", tt.endpoint, chunk, string(body))
					}
				}
			} else {
				if !strings.Contains(string(body), "mock response") {
					t.Errorf("%s: unexpected body: %s", tt.endpoint, string(body))
				}
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

	cfg := config.GlobalConfig{
		OllamaURL:         ollama.URL,
		Port:              "8080",
		TimeoutSecs:       15,
		MaxBodySize:       10 * 1024 * 1024,
		StreamBufferSize:  4096,
		StreamTimeoutSecs: 30,
	}
	handler := NewProxyHandler(cfg)

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
