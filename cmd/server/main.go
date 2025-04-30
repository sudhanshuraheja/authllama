package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/sudhanshuraheja/authllama/internal/auth"
	"github.com/sudhanshuraheja/authllama/internal/observability"
	"github.com/sudhanshuraheja/authllama/internal/proxy"
	"github.com/sudhanshuraheja/authllama/internal/ratelimit"
)

func main() {
	store, err := auth.LoadAuthConfig("config/config.json")
	if err != nil {
		log.Fatalf("failed to load auth config: %v", err)
	}

	limiter, err := ratelimit.LoadRateLimiter("config/config.json")
	if err != nil {
		log.Fatalf("failed to load rate limiter config: %v", err)
	}

	proxyHandler := proxy.NewProxyHandler("http://localhost:11434") // Ollama default port

	adminKey := os.Getenv("ADMIN_API_KEY")

	http.HandleFunc("/admin/reload", func(w http.ResponseWriter, r *http.Request) {
		apiKey := r.Header.Get("X-Admin-Key")
		if apiKey != adminKey {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if err := store.Reload(); err != nil {
			http.Error(w, "Failed to reload auth config", http.StatusInternalServerError)
			return
		}
		if err := limiter.Reload(); err != nil {
			http.Error(w, "Failed to reload rate limit config", http.StatusInternalServerError)
			return
		}
		log.Println("Manual config reload successful")
		w.WriteHeader(http.StatusOK)
		if _, err := w.Write([]byte("Reloaded config")); err != nil {
			log.Printf("failed to write reload confirmation: %v", err)
		}
	})

	authWrapper := func(handler func(w http.ResponseWriter, r *http.Request, service string)) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			service := r.Header.Get("X-Service-Name")
			header := r.Header.Get("Authorization")

			if !store.IsAuthorized(service, header) {
				observability.IncAuthFails()
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			if !limiter.Allow(service) {
				log.Printf("Rate limit exceeded for service %s", service)
				observability.IncRateLimited()
				http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
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

	http.HandleFunc("/api/tags", authWrapper(func(w http.ResponseWriter, r *http.Request, _ string) {
		proxyHandler.HandleTags(w, r)
	}))

	http.HandleFunc("/api/show", authWrapper(func(w http.ResponseWriter, r *http.Request, _ string) {
		proxyHandler.HandleShow(w, r)
	}))

	http.HandleFunc("/api/pull", authWrapper(func(w http.ResponseWriter, r *http.Request, _ string) {
		proxyHandler.HandlePull(w, r)
	}))

	http.HandleFunc("/api/push", authWrapper(func(w http.ResponseWriter, r *http.Request, _ string) {
		proxyHandler.HandlePush(w, r)
	}))

	http.HandleFunc("/api/create", authWrapper(func(w http.ResponseWriter, r *http.Request, _ string) {
		proxyHandler.HandleCreate(w, r)
	}))

	http.HandleFunc("/api/delete", authWrapper(func(w http.ResponseWriter, r *http.Request, _ string) {
		proxyHandler.HandleDelete(w, r)
	}))

	go func() {
		for {
			time.Sleep(30 * time.Second)
			observability.Log()
		}
	}()

	port := "8080"
	if os.Getenv("PORT") != "" {
		port = os.Getenv("PORT")
	}

	server := &http.Server{
		Addr: ":" + port,
	}

	go func() {
		log.Printf("Listening on :%s", port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	// Graceful shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)

	<-quit
	log.Println("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatalf("Failed to gracefully shutdown: %v", err)
	}

	log.Println("Server shutdown complete")
}
