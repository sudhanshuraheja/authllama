package auth

import (
	"encoding/json"
	"log"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type AuthStore struct {
	mu      sync.RWMutex
	path    string
	data    map[string]string
	watcher *fsnotify.Watcher
}

func LoadAuthConfig(path string) (*AuthStore, error) {
	store := &AuthStore{
		path: path,
		data: map[string]string{},
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

	var newData map[string]string
	if err := json.Unmarshal(content, &newData); err != nil {
		return err
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
				_ = a.load()
			}
		case err, ok := <-a.watcher.Errors:
			if !ok {
				return
			}
			log.Println("Watcher error:", err)
		}
	}
}

func (a *AuthStore) IsAuthorized(service string, header string) bool {
	a.mu.RLock()
	defer a.mu.RUnlock()
	expected, ok := a.data[service]
	return ok && expected == header
}
