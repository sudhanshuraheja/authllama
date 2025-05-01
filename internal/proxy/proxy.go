package proxy

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"maps"
	"net/http"
	"time"

	"github.com/sudhanshuraheja/authllama/internal/observability"
	"github.com/sudhanshuraheja/authllama/internal/config"
)

type ProxyHandler struct {
	OllamaURL string
	Client    *http.Client
	Config    config.GlobalConfig
}

func NewProxyHandler(cfg config.GlobalConfig) *ProxyHandler {
	return &ProxyHandler{
		OllamaURL: cfg.OllamaURL,
		Client:    &http.Client{Timeout: time.Duration(cfg.TimeoutSecs) * time.Second},
		Config:    cfg,
	}
}

func NewProxy(cfg *config.Config) *ProxyHandler {
	if cfg.Global.OllamaURL == "" {
		log.Fatal("missing ollama_url in _global config")
	}
	return NewProxyHandler(cfg.Global)
}

func (p *ProxyHandler) streamResponse(resp *http.Response, method, path string, w http.ResponseWriter, r *http.Request) {
	observability.IncStreamed()

	w.Header().Set("Transfer-Encoding", "chunked")
	w.Header().Set("X-Accel-Buffering", "no")

	flusher, ok := w.(http.Flusher)
	if !ok {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(resp.StatusCode)

	ctx, cancel := context.WithTimeout(r.Context(), p.Config.StreamTimeout())
	defer cancel()

	reader := io.LimitReader(resp.Body, int64(p.Config.MaxBodySize))
	buf := make([]byte, p.Config.StreamBufferSize)

	for {
		select {
		case <-ctx.Done():
			log.Printf("stream timeout for %s %s", method, path)
			return
		default:
			n, err := reader.Read(buf)
			if n > 0 {
				if _, writeErr := w.Write(buf[:n]); writeErr != nil {
					log.Printf("stream write error: %v", writeErr)
					return
				}
				flusher.Flush()
			}
			if err != nil {
				if err != io.EOF {
					log.Printf("stream read error: %v", err)
				}
				return
			}
		}
	}
}

func (p *ProxyHandler) forward(method, path string, body io.Reader, w http.ResponseWriter, r *http.Request, stream bool) {
	log.Printf("Proxying %s to %s", method, path)

	proxyReq, err := http.NewRequest(method, p.OllamaURL+path, body)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	proxyReq.URL.RawQuery = r.URL.RawQuery
	proxyReq.Header = r.Header.Clone()

	resp, err := p.Client.Do(proxyReq)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Request to Ollama failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	maps.Copy(w.Header(), resp.Header)
	if resp.StatusCode >= 400 {
		log.Printf("Ollama returned error %d for %s %s", resp.StatusCode, method, path)
	}

	if stream {
		p.streamResponse(resp, method, path, w, r)
		return
	}

	w.WriteHeader(resp.StatusCode)
	reader := io.LimitReader(resp.Body, int64(p.Config.MaxBodySize))
	if _, err := io.Copy(w, reader); err != nil {
		log.Printf("error copying response body: %v", err)
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (p *ProxyHandler) forwardPost(path string, w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	type StreamFlag struct {
		Stream bool `json:"stream"`
	}
	var sf StreamFlag
	if err := json.Unmarshal(bodyBytes, &sf); err != nil {
		log.Printf("failed to parse stream flag: %v", err)
	}
	// Log model and prompt after decoding StreamFlag.
	type PromptLog struct {
		Model  string `json:"model"`
		Prompt string `json:"prompt"`
	}
	var pl PromptLog
	if err := json.Unmarshal(bodyBytes, &pl); err == nil {
		log.Printf("request model=%q prompt=%q", pl.Model, pl.Prompt)
	}
	shouldStream := sf.Stream

	p.forward(http.MethodPost, path, bytes.NewReader(bodyBytes), w, r, shouldStream)
}

func (p *ProxyHandler) forwardGet(path string, w http.ResponseWriter, r *http.Request) {
	p.forward(http.MethodGet, path, nil, w, r, false)
}

// Handles POST /api/generate
func (p *ProxyHandler) HandleGenerate(w http.ResponseWriter, r *http.Request) {
	p.forwardPost("/api/generate", w, r)
}

// Handles POST /api/chat
func (p *ProxyHandler) HandleChat(w http.ResponseWriter, r *http.Request) {
	p.forwardPost("/api/chat", w, r)
}

// Handles GET /api/tags
func (p *ProxyHandler) HandleTags(w http.ResponseWriter, r *http.Request) {
	p.forwardGet("/api/tags", w, r)
}

// Handles GET /api/show
func (p *ProxyHandler) HandleShow(w http.ResponseWriter, r *http.Request) {
	p.forwardGet("/api/show", w, r)
}

// Handles POST /api/pull
func (p *ProxyHandler) HandlePull(w http.ResponseWriter, r *http.Request) {
	p.forwardPost("/api/pull", w, r)
}

// Handles POST /api/push
func (p *ProxyHandler) HandlePush(w http.ResponseWriter, r *http.Request) {
	p.forwardPost("/api/push", w, r)
}

// Handles POST /api/create
func (p *ProxyHandler) HandleCreate(w http.ResponseWriter, r *http.Request) {
	p.forwardPost("/api/create", w, r)
}

// Handles POST /api/delete
func (p *ProxyHandler) HandleDelete(w http.ResponseWriter, r *http.Request) {
	p.forwardPost("/api/delete", w, r)
}
