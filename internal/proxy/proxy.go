package proxy

import (
	"bytes"
	"io"
	"log"
	"maps"
	"net/http"
	"time"
)

type ProxyHandler struct {
	OllamaURL string
	Client    *http.Client
}

func NewProxyHandler(ollamaURL string) *ProxyHandler {
	return &ProxyHandler{
		OllamaURL: ollamaURL,
		Client:    &http.Client{Timeout: 15 * time.Second},
	}
}

func (p *ProxyHandler) forward(method, path string, body io.Reader, w http.ResponseWriter, r *http.Request) {
	log.Printf("Proxying %s to %s", method, path)

	proxyReq, err := http.NewRequest(method, p.OllamaURL+path, body)
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}
	proxyReq.URL.RawQuery = r.URL.RawQuery
	proxyReq.Header = r.Header.Clone()

	resp, err := p.Client.Do(proxyReq)
	if err != nil {
		http.Error(w, "Request to Ollama failed", http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	maps.Copy(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Printf("error copying response body: %v", err)
	}
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (p *ProxyHandler) forwardPost(path string, w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()
	p.forward(http.MethodPost, path, bytes.NewReader(bodyBytes), w, r)
}

func (p *ProxyHandler) forwardGet(path string, w http.ResponseWriter, r *http.Request) {
	p.forward(http.MethodGet, path, nil, w, r)
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
