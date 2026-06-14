package config

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type Manager struct {
	mu   sync.RWMutex
	cfg  Config
	path string
}

type ReloadCallback func(Config, string, error)

func NewManager(path string) (*Manager, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}
	cfg, err := Load(abs)
	if err != nil {
		return nil, err
	}
	return &Manager{cfg: cfg, path: abs}, nil
}

func (m *Manager) Config() Config {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.cfg
}

func (m *Manager) Reload() (Config, error) {
	cfg, err := Load(m.path)
	if err != nil {
		return m.Config(), err
	}
	m.mu.Lock()
	m.cfg = cfg
	m.mu.Unlock()
	return cfg, nil
}

func (m *Manager) StartWatch(ctx context.Context, watchPaths []string, cb ReloadCallback) {
	go func() {
		watcher, err := fsnotify.NewWatcher()
		if err != nil {
			log.Printf("[hot-reload] watcher init failed: %v", err)
			return
		}
		defer watcher.Close()

		seen := map[string]struct{}{}
		addPath := func(p string) {
			if p == "" {
				return
			}
			normalized := filepath.Clean(p)
			if _, ok := seen[normalized]; ok {
				return
			}
			if err := watcher.Add(normalized); err != nil {
				log.Printf("[hot-reload] add watch failed: %s %v", normalized, err)
				return
			}
			seen[normalized] = struct{}{}
		}

		for _, p := range watchPaths {
			addPath(p)
		}
		// also watch parent dir for deleted/renamed watched files.
		for _, p := range watchPaths {
			if strings.HasSuffix(p, string(filepath.Separator)) {
				continue
			}
			dir := filepath.Dir(p)
			addPath(dir)
		}

		mu := sync.Mutex{}
		pending := map[string]struct{}{}
		var timer *time.Timer
		trigger := func(path string) {
			mu.Lock()
			pending[path] = struct{}{}
			mu.Unlock()
			if timer != nil {
				timer.Reset(150 * time.Millisecond)
				return
			}
			timer = time.AfterFunc(150*time.Millisecond, func() {
				mu.Lock()
				keys := make([]string, 0, len(pending))
				for p := range pending {
					keys = append(keys, p)
				}
				pending = map[string]struct{}{}
				mu.Unlock()
				for _, p := range keys {
					if strings.HasSuffix(p, filepath.Base(m.path)) {
						cfg, err := m.Reload()
						cb(cfg, p, err)
						continue
					}
					cb(m.Config(), p, nil)
				}
				mu.Lock()
				timer = nil
				mu.Unlock()
			})
		}

		for {
			select {
			case <-ctx.Done():
				return
			case ev := <-watcher.Events:
				if ev.Has(fsnotify.Write) || ev.Has(fsnotify.Create) || ev.Has(fsnotify.Remove) || ev.Has(fsnotify.Rename) {
					trigger(ev.Name)
				}
			case err := <-watcher.Errors:
				log.Printf("[hot-reload] watcher error: %v", err)
			}
		}
	}()
}
