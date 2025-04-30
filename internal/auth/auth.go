package auth

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type AuthStore struct {
	mu      sync.RWMutex
	path    string
	data    map[string]ServiceConfig
	watcher *fsnotify.Watcher
}

type ServiceConfig struct {
	Auth string `json:"auth"`
	TPM  int    `json:"tpm"`
}

func LoadAuthConfig(path string) (*AuthStore, error) {
	store := &AuthStore{
		path: path,
		data: map[string]ServiceConfig{},
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	store.watcher = watcher

	go store.watchFile()

	if err := watcher.Add(path); err != nil {
		return nil, err
	}

	return store, nil
}

func (a *AuthStore) load() error {
	a.mu.Lock()
	defer a.mu.Unlock()

	content, err := os.ReadFile(a.path)
	if err != nil {
		return err
	}

	var newData map[string]ServiceConfig
	if err := json.Unmarshal(content, &newData); err != nil {
		return err
	}

	for svc, cfg := range newData {
		if cfg.Auth == "" {
			return fmt.Errorf("missing auth for service: %s", svc)
		}
	}

	a.data = newData
	log.Println("Auth config reloaded")

	return nil
}

func (a *AuthStore) Reload() error {
	return a.load()
}

func (a *AuthStore) watchFile() {
	for {
		select {
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

func (a *AuthStore) Close() error {
	if a.watcher != nil {
		return a.watcher.Close()
	}
	return nil
}

func (a *AuthStore) IsAuthorized(service string, header string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	expected, ok := a.data[service]
	if !ok {
		log.Printf("auth check failed: unknown service '%s'", service)
	}
	return ok && expected.Auth == header
}
