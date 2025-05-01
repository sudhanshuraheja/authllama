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
	"github.com/sudhanshuraheja/authllama/internal/config"
	"github.com/sudhanshuraheja/authllama/internal/observability"
	"github.com/sudhanshuraheja/authllama/internal/proxy"
	"github.com/sudhanshuraheja/authllama/internal/ratelimit"
)

func registerAdminHandlers(mux *http.ServeMux, store *auth.AuthStore, limiter *ratelimit.RateLimiter, adminKey string) {
	mux.HandleFunc("/admin/reload", func(w http.ResponseWriter, r *http.Request) {
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
}

func main() {
	cfg, err := config.LoadConfig("config/config.json")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	store := auth.NewStore(cfg)
	limiter := ratelimit.NewLimiter(cfg)
	proxy := proxy.NewProxy(cfg)

	adminKey := cfg.Global.AdminAPIKey

	mux := http.NewServeMux()
	registerAdminHandlers(mux, store, limiter, adminKey)

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

	routes := map[string]func(http.ResponseWriter, *http.Request){
		"/api/generate": proxy.HandleGenerate,
		"/api/chat":     proxy.HandleChat,
		"/api/tags":     proxy.HandleTags,
		"/api/show":     proxy.HandleShow,
		"/api/pull":     proxy.HandlePull,
		"/api/push":     proxy.HandlePush,
		"/api/create":   proxy.HandleCreate,
		"/api/delete":   proxy.HandleDelete,
	}

	for path, handler := range routes {
		mux.HandleFunc(path, authWrapper(func(w http.ResponseWriter, r *http.Request, _ string) {
			handler(w, r)
		}))
	}

	mux.HandleFunc("/", authWrapper(func(w http.ResponseWriter, r *http.Request, service string) {
		fmt.Fprintf(w, "Hello, %s! You are authorized.\n", service)
	}))

	go func() {
		for {
			time.Sleep(30 * time.Second)
			observability.Log()
		}
	}()

	if cfg.Global.Port == "" {
		log.Fatal("server port is not configured")
	}
	server := &http.Server{
		Addr:    ":" + cfg.Global.Port,
		Handler: mux,
	}

	go func() {
		log.Printf("Listening on :%s", cfg.Global.Port)
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
