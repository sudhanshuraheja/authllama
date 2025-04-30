package proxy

import (
	"bytes"
	"io"
	"log"
	"maps"
	"net/http"
)

type ProxyHandler struct {
	OllamaURL string
	Client    *http.Client
}

func NewProxyHandler(ollamaURL string) *ProxyHandler {
	return &ProxyHandler{
		OllamaURL: ollamaURL,
		Client:    &http.Client{},
	}
}

// Handles POST /api/generate
func (p *ProxyHandler) HandleGenerate(w http.ResponseWriter, r *http.Request) {
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	r.Body.Close()

	proxyReq, err := http.NewRequest(http.MethodPost, p.OllamaURL+"/api/generate", io.NopCloser(bytes.NewReader(bodyBytes)))
	if err != nil {
		http.Error(w, "Failed to create request", http.StatusInternalServerError)
		return
	}

	proxyReq.Header = r.Header.Clone() // Forward headers

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
}
