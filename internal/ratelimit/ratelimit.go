package ratelimit

import (
	"log"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/sudhanshuraheja/authllama/internal/config"
)

type RateLimiter struct {
	mu      sync.RWMutex
	path    string
	config  *config.Config
	buckets map[string]*tokenBucket
	watcher *fsnotify.Watcher
}

type tokenBucket struct {
	lastRefill time.Time
	tokens     int
	capacity   int
	interval   time.Duration
	mu         sync.Mutex
}

func (tb *tokenBucket) Allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill)
	refill := int(elapsed / tb.interval)
	if refill > 0 {
		tb.tokens = min(tb.capacity, tb.tokens+refill)
		tb.lastRefill = now
	}
	if tb.tokens > 0 {
		tb.tokens--
		return true
	}
	return false
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func NewLimiter(cfg *config.Config) *RateLimiter {
	buckets := make(map[string]*tokenBucket)
	for service, cfg := range cfg.Services {
		if cfg.TPM > 0 {
			interval := time.Minute / time.Duration(cfg.TPM)
			buckets[service] = &tokenBucket{
				lastRefill: time.Now(),
				tokens:     cfg.TPM,
				capacity:   cfg.TPM,
				interval:   interval,
			}
		}
	}

	return &RateLimiter{
		config:  cfg,
		buckets: buckets,
	}
}

func LoadRateLimiter(path string) (*RateLimiter, error) {
	rl := &RateLimiter{
		path:    path,
		buckets: map[string]*tokenBucket{},
	}

	if err := rl.load(); err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	rl.watcher = watcher

	go rl.watch()

	if err := watcher.Add(path); err != nil {
		return nil, err
	}

	return rl, nil
}

func (rl *RateLimiter) load() error {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	cfg, err := config.LoadConfig(rl.path)
	if err != nil {
		return err
	}
	rl.config = cfg
	rl.buckets = map[string]*tokenBucket{}
	for service, cfg := range cfg.Services {
		if cfg.TPM > 0 {
			interval := time.Minute / time.Duration(cfg.TPM)
			rl.buckets[service] = &tokenBucket{
				lastRefill: time.Now(),
				tokens:     cfg.TPM,
				capacity:   cfg.TPM,
				interval:   interval,
			}
		}
	}

	log.Println("Rate limit config loaded")
	return nil
}

func (rl *RateLimiter) watch() {
	for {
		select {
		case event, ok := <-rl.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				_ = rl.load()
			}
		case err, ok := <-rl.watcher.Errors:
			if !ok {
				return
			}
			log.Printf("Rate limiter watcher error: %v", err)
		}
	}
}

func (rl *RateLimiter) IsAuthorized(service, token string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	cfg, err := rl.config.GetService(service)
	return err == nil && cfg.Auth == token
}

func (rl *RateLimiter) Allow(service string) bool {
	rl.mu.RLock()
	defer rl.mu.RUnlock()
	tb, ok := rl.buckets[service]
	if !ok {
		return true // no limit for this service
	}
	return tb.Allow()
}

func (r *RateLimiter) Reload() error {
	return r.load()
}
