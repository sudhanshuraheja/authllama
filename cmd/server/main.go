package main

import (
	"fmt"
	"log"
	"net/http"
	"os"

	"github.com/sudhanshuraheja/authllama/internal/auth"
	"github.com/sudhanshuraheja/authllama/internal/proxy"
)

func main() {
	store, err := auth.LoadAuthConfig("config/config.json")
	if err != nil {
		log.Fatalf("failed to load auth config: %v", err)
	}

	proxyHandler := proxy.NewProxyHandler("http://localhost:11434") // Ollama default port

	authWrapper := func(handler func(w http.ResponseWriter, r *http.Request, service string)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			service := r.Header.Get("X-Service-Name")
			header := r.Header.Get("Authorization")

			if !store.IsAuthorized(service, header) {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			handler(w, r, service)
		}
	}

	http.HandleFunc("/", authWrapper(func(w http.ResponseWriter, r *http.Request, service string) {
		fmt.Fprintf(w, "Hello, %s! You are authorized.\n", service)
	}))

	http.HandleFunc("/api/generate", authWrapper(func(w http.ResponseWriter, r *http.Request, _ string) {
		proxyHandler.HandleGenerate(w, r)
	}))

	http.HandleFunc("/api/chat", authWrapper(func(w http.ResponseWriter, r *http.Request, _ string) {
		proxyHandler.HandleChat(w, r)
	}))

	port := "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}
	log.Printf("Listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, nil))
}
