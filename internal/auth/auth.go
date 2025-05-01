package auth

import (
	"context"
	"fmt"
	"log"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/sudhanshuraheja/authllama/internal/config"
)

type AuthStore struct {
	mu      sync.RWMutex
	path    string
	config  *config.Config
	watcher *fsnotify.Watcher
	ctx     context.Context
	cancel  context.CancelFunc
}

func NewStore(cfg *config.Config) *AuthStore {
	return &AuthStore{
		config: cfg,
	}
}

func LoadAuthConfig(path string) (*AuthStore, error) {
	store := &AuthStore{
		path: path,
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(context.Background())
	store.ctx = ctx
	store.cancel = cancel
	if err := store.setupWatcher(); err != nil {
		return nil, err
	}

	return store, nil
}

func (a *AuthStore) load() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	cfg, err := config.LoadConfig(a.path)
	if err != nil {
		return err
	}
	// Minimal validation: must have at least one service
	if len(cfg.Services) == 0 {
		return fmt.Errorf("invalid config: no services defined")
	}
	a.config = cfg
	log.Println("Auth config reloaded")
	return nil
}

func (a *AuthStore) Reload() error {
	return a.load()
}

func (a *AuthStore) watchFile(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-a.watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write {
				log.Println("Detected change in auth config file.")
				if err := a.load(); err != nil {
					log.Printf("failed to reload auth config: %v", err)
				}
			}
		case err, ok := <-a.watcher.Errors:
			if !ok {
				return
			}
			log.Println("Watcher error:", err)
		}
	}
}

func (a *AuthStore) setupWatcher() error {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return err
	}
	a.watcher = watcher
	go a.watchFile(a.ctx)
	return watcher.Add(a.path)
}

func (a *AuthStore) Close() error {
	if a.cancel != nil {
		a.cancel()
	}
	if a.watcher != nil {
		return a.watcher.Close()
	}
	return nil
}

func (a *AuthStore) IsAuthorized(service, header string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()

	svc, err := a.config.GetService(service)
	if err != nil {
		log.Printf("auth check failed: service %q not found", service)
		return false
	}

	log.Printf("Auth check for service=%q: expected=<%s>, received=<%s>", service, svc.Auth, header)

	return svc.Auth == header
}
